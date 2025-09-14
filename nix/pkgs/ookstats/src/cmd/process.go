package cmd

import (
	"database/sql"
	"fmt"
	"ookstats/internal/database"
	"time"

	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process and compute data",
	Long:  `Process raw data and compute player rankings, aggregations, etc.`,
}

var processAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Process all data (players + rankings)",
	Long:  `Process all data: player aggregations, player rankings, and run rankings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Complete Data Processing ===")

		// step 1: process players (aggregations + rankings)
		fmt.Println("\n=== Step 1: Processing Players ===")
		if err := processPlayersCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("player processing failed: %w", err)
		}

		// step 2: process run rankings  
		fmt.Println("\n=== Step 2: Processing Run Rankings ===")
		if err := processRankingsCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("run ranking processing failed: %w", err)
		}

		fmt.Printf("\n[OK] Complete data processing finished!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Run 'ookstats generate api --out web/public' to update JSON files\n")
		fmt.Printf("  2. Build and deploy the website\n")

		return nil
	},
}

var processRankingsCmd = &cobra.Command{
	Use:   "rankings",
	Short: "Compute run rankings",
	Long:  `Compute global, regional, and realm rankings for all challenge mode runs.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        fmt.Println("=== Run Ranking Processor ===")

        db, err := database.Connect()
        if err != nil {
            return fmt.Errorf("failed to connect to database: %w", err)
        }
        defer db.Close()

        fmt.Printf("Connected to database: %s\n", database.DBFilePath())

        // check if we have data
        var runCount int
        db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)

        if runCount == 0 {
            return fmt.Errorf("no runs found in database - run 'fetch cm' first")
        }

        fmt.Printf("Found %d runs in database\n", runCount)

        // begin transaction for all ranking operations
        tx, err := db.Begin()
        if err != nil {
            return fmt.Errorf("failed to begin transaction: %w", err)
        }
        defer tx.Rollback()

        // step 1: compute global run rankings
        fmt.Println("\n1. Computing global run rankings...")
        if err := computeGlobalRankings(tx); err != nil {
            return fmt.Errorf("failed to compute global rankings: %w", err)
        }

        // step 2: compute regional run rankings
        fmt.Println("\n2. Computing regional run rankings...")
        if err := computeRegionalRankings(tx); err != nil {
            return fmt.Errorf("failed to compute regional rankings: %w", err)
        }

        // commit all changes
        if err := tx.Commit(); err != nil {
            return fmt.Errorf("failed to commit run rankings: %w", err)
        }

        // optimize database
        fmt.Println("\n3. Optimizing database...")
        if _, err := db.Exec("VACUUM"); err != nil {
            fmt.Printf("Warning: database optimization failed: %v\n", err)
        }

        fmt.Printf("\nRun ranking computation complete!\n")
        fmt.Printf("\nNext steps:\n")
        fmt.Printf("  1. Run 'ookstats generate api --out web/public' to update JSON files\n")
        fmt.Printf("  2. Build and deploy the website\n")

        return nil
    },
}

var processPlayersCmd = &cobra.Command{
	Use:   "players",
	Short: "Aggregate player statistics",
	Long:  `Aggregate player data including best runs, combined times, and rankings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Player Aggregation ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

        fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		// check if we have data
        var runCount, playerCount int
        db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)
        db.QueryRow("SELECT COUNT(*) FROM players").Scan(&playerCount)

		fmt.Printf("Found %d runs and %d players in database\n", runCount, playerCount)

		if runCount == 0 {
			return fmt.Errorf("no runs found in database - run 'fetch cm' first")
		}

		// begin transaction for all player operations
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		// step 1: create player aggregations
		fmt.Println("\n1. Creating player aggregations...")
		profilesCreated, err := createPlayerAggregations(tx)
		if err != nil {
			return fmt.Errorf("failed to create player aggregations: %w", err)
		}

        // step 2: compute player rankings (global, regional, realm)
        fmt.Println("\n2. Computing player rankings...")
        qualifiedPlayers, err := computePlayerRankings(tx)
        if err != nil {
			return fmt.Errorf("failed to compute player rankings: %w", err)
        }

		// commit all changes
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit player aggregations: %w", err)
		}

		// optimize database
		fmt.Println("\n3. Optimizing database...")
		if _, err := db.Exec("VACUUM"); err != nil {
			fmt.Printf("Warning: database optimization failed: %v\n", err)
		}

		fmt.Printf("\nPlayer aggregation complete!\n")
		fmt.Printf("   Created %d player profiles\n", profilesCreated)
		fmt.Printf("   Computed rankings for %d qualified players\n", qualifiedPlayers)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Run 'ookstats process profiles' to fetch player details\n")
		fmt.Printf("  2. Test your Astro DB integration\n")

		return nil
	},
}

// computeGlobalRankings computes global rankings for all runs
func computeGlobalRankings(tx *sql.Tx) error {
	fmt.Printf("Computing global rankings...\n")

	currentTime := time.Now().UnixMilli()

	// clear existing global rankings
	if _, err := tx.Exec("DELETE FROM run_rankings WHERE ranking_type = 'global'"); err != nil {
		return err
	}

	// unfiltered global rankings - first insert without brackets
	_, err := tx.Exec(`
		INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
		SELECT
			id as run_id,
			dungeon_id,
			'global' as ranking_type,
			'all' as ranking_scope,
			ROW_NUMBER() OVER (PARTITION BY dungeon_id ORDER BY duration ASC, completed_timestamp ASC) as ranking,
			? as computed_at
		FROM challenge_runs
	`, currentTime)
	if err != nil {
		return err
	}

	// update percentile brackets for unfiltered global rankings using efficient SQL
	fmt.Printf("Computing global ranking brackets...\n")
	_, err = tx.Exec(`
		UPDATE run_rankings 
		SET percentile_bracket = (
			CASE 
				WHEN counts.ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				run_id,
				dungeon_id,
				ranking,
				COUNT(*) OVER (PARTITION BY dungeon_id) as total_in_dungeon
			FROM run_rankings 
			WHERE ranking_type = 'global' AND ranking_scope = 'all'
		) counts
		WHERE run_rankings.run_id = counts.run_id 
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.ranking_type = 'global' 
		AND run_rankings.ranking_scope = 'all'
	`)
	if err != nil {
		return err
	}

	// filtered global rankings (best time per team)
	// get all dungeon IDs for filtered rankings
	dungeonRows, err := tx.Query("SELECT id FROM dungeons")
	if err != nil {
		return err
	}
	defer dungeonRows.Close()

	var dungeonIDs []int
	for dungeonRows.Next() {
		var id int
		if err := dungeonRows.Scan(&id); err != nil {
			return err
		}
		dungeonIDs = append(dungeonIDs, id)
	}

	for _, dungeonID := range dungeonIDs {
		_, err := tx.Exec(`
			WITH best_team_runs AS (
				SELECT
					team_signature,
					MIN(duration) as best_duration
				FROM challenge_runs
				WHERE dungeon_id = ?
				GROUP BY team_signature
			),
			filtered_runs AS (
				SELECT
					cr.id as run_id,
					cr.duration,
					cr.completed_timestamp,
					ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
				FROM challenge_runs cr
				INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
											 AND cr.duration = btr.best_duration
				WHERE cr.dungeon_id = ?
				GROUP BY cr.team_signature
				HAVING cr.id = MIN(cr.id)
			)
			INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
			SELECT
				run_id,
				? as dungeon_id,
				'global' as ranking_type,
				'filtered' as ranking_scope,
				filtered_rank as ranking,
				? as computed_at
			FROM filtered_runs
		`, dungeonID, dungeonID, dungeonID, currentTime)

		if err != nil {
			return err
		}
	}

	// update percentile brackets for filtered global rankings using efficient SQL
	fmt.Printf("Computing filtered global ranking brackets...\n")
	_, err = tx.Exec(`
		UPDATE run_rankings 
		SET percentile_bracket = (
			CASE 
				WHEN counts.ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.total_in_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				run_id,
				dungeon_id,
				ranking,
				COUNT(*) OVER (PARTITION BY dungeon_id) as total_in_dungeon
			FROM run_rankings 
			WHERE ranking_type = 'global' AND ranking_scope = 'filtered'
		) counts
		WHERE run_rankings.run_id = counts.run_id 
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.ranking_type = 'global' 
		AND run_rankings.ranking_scope = 'filtered'
	`)
	if err != nil {
		return err
	}

fmt.Printf("[OK] Computed global rankings with percentile brackets (all and filtered)\n")
	return nil
}

// computeRegionalRankings computes regional rankings for all runs
func computeRegionalRankings(tx *sql.Tx) error {
	fmt.Printf("Computing regional rankings...\n")

	currentTime := time.Now().UnixMilli()

	// clear existing regional rankings
	if _, err := tx.Exec("DELETE FROM run_rankings WHERE ranking_type = 'regional'"); err != nil {
		return err
	}

	// get all regions
	regionRows, err := tx.Query("SELECT DISTINCT region FROM realms")
	if err != nil {
		return err
	}
	defer regionRows.Close()

	var regions []string
	for regionRows.Next() {
		var region string
		if err := regionRows.Scan(&region); err != nil {
			return err
		}
		regions = append(regions, region)
	}

	for _, region := range regions {
		// unfiltered regional rankings
		_, err := tx.Exec(`
			INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
			SELECT
				cr.id as run_id,
				cr.dungeon_id,
				'regional' as ranking_type,
				? as ranking_scope,
				ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
				? as computed_at
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			WHERE r.region = ?
		`, region, currentTime, region)

		if err != nil {
			return err
		}

		// update percentile brackets for unfiltered regional rankings using efficient SQL
		fmt.Printf("Computing unfiltered regional ranking brackets for %s...\n", region)
		_, err = tx.Exec(`
			UPDATE run_rankings 
			SET percentile_bracket = (
				CASE 
					WHEN counts.ranking = 1 THEN 'artifact'
					ELSE 
						CASE 
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT 
					run_id,
					dungeon_id,
					ranking,
					COUNT(*) OVER (PARTITION BY dungeon_id, ranking_scope) as total_in_region_dungeon
				FROM run_rankings 
				WHERE ranking_type = 'regional' AND ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id 
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.ranking_type = 'regional' 
			AND run_rankings.ranking_scope = ?
		`, region, region)
		if err != nil {
			return err
		}

		// get dungeonIDs for filtered rankings
		dungeonRows, err := tx.Query("SELECT id FROM dungeons")
		if err != nil {
			return err
		}

		var dungeonIDs []int
		for dungeonRows.Next() {
			var id int
			if err := dungeonRows.Scan(&id); err != nil {
				dungeonRows.Close()
				return err
			}
			dungeonIDs = append(dungeonIDs, id)
		}
		dungeonRows.Close()

		// filtered regional rankings - reuse dungeonIDs from above
		for _, dungeonID := range dungeonIDs {
			_, err := tx.Exec(`
				WITH best_team_runs AS (
					SELECT
						cr.team_signature,
						MIN(cr.duration) as best_duration
					FROM challenge_runs cr
					INNER JOIN realms r ON cr.realm_id = r.id
					WHERE cr.dungeon_id = ? AND r.region = ?
					GROUP BY cr.team_signature
				),
				filtered_runs AS (
					SELECT
						cr.id as run_id,
						cr.duration,
						cr.completed_timestamp,
						ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
					FROM challenge_runs cr
					INNER JOIN realms r ON cr.realm_id = r.id
					INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
													AND cr.duration = btr.best_duration
					WHERE cr.dungeon_id = ? AND r.region = ?
					GROUP BY cr.team_signature
					HAVING cr.id = MIN(cr.id)
				)
				INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
				SELECT
					run_id,
					? as dungeon_id,
					'regional' as ranking_type,
					? as ranking_scope,
					filtered_rank as ranking,
					? as computed_at
				FROM filtered_runs
			`, dungeonID, region, dungeonID, region, dungeonID, region+"_filtered", currentTime)

			if err != nil {
				return err
			}
		}

		// update percentile brackets for filtered regional rankings using efficient SQL
		filteredScope := region + "_filtered"
		fmt.Printf("Computing filtered regional ranking brackets for %s...\n", region)
		_, err = tx.Exec(`
			UPDATE run_rankings 
			SET percentile_bracket = (
				CASE 
					WHEN counts.ranking = 1 THEN 'artifact'
					ELSE 
						CASE 
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
							WHEN (CAST(counts.total_in_region_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_region_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT 
					run_id,
					dungeon_id,
					ranking,
					COUNT(*) OVER (PARTITION BY dungeon_id, ranking_scope) as total_in_region_dungeon
				FROM run_rankings 
				WHERE ranking_type = 'regional' AND ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id 
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.ranking_type = 'regional' 
			AND run_rankings.ranking_scope = ?
		`, filteredScope, filteredScope)
		if err != nil {
			return err
		}
	}

fmt.Printf("[OK] Computed regional rankings with percentile brackets for %d regions\n", len(regions))
	return nil
}

// computeRealmRankings computes realm rankings for all runs
func computeRealmRankings(tx *sql.Tx) error {
	fmt.Printf("Computing realm rankings...\n")

	currentTime := time.Now().UnixMilli()

	// clear existing realm rankings
	if _, err := tx.Exec("DELETE FROM run_rankings WHERE ranking_type = 'realm'"); err != nil {
		return err
	}

	// get all realm IDs
	realmRows, err := tx.Query("SELECT id FROM realms")
	if err != nil {
		return err
	}
	defer realmRows.Close()

	var realmIDs []int
	for realmRows.Next() {
		var id int
		if err := realmRows.Scan(&id); err != nil {
			return err
		}
		realmIDs = append(realmIDs, id)
	}

	for _, realmID := range realmIDs {
		// unfiltered realm rankings
		_, err := tx.Exec(`
			INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
			SELECT
				id as run_id,
				dungeon_id,
				'realm' as ranking_type,
				? as ranking_scope,
				ROW_NUMBER() OVER (PARTITION BY dungeon_id ORDER BY duration ASC, completed_timestamp ASC) as ranking,
				? as computed_at
			FROM challenge_runs
			WHERE realm_id = ?
		`, fmt.Sprintf("%d", realmID), currentTime, realmID)

		if err != nil {
			return err
		}

		// filtered realm rankings
		dungeonRows, err := tx.Query("SELECT id FROM dungeons")
		if err != nil {
			return err
		}

		var dungeonIDs []int
		for dungeonRows.Next() {
			var id int
			if err := dungeonRows.Scan(&id); err != nil {
				dungeonRows.Close()
				return err
			}
			dungeonIDs = append(dungeonIDs, id)
		}
		dungeonRows.Close()

		for _, dungeonID := range dungeonIDs {
			_, err := tx.Exec(`
				WITH best_team_runs AS (
					SELECT
						team_signature,
						MIN(duration) as best_duration
					FROM challenge_runs
					WHERE dungeon_id = ? AND realm_id = ?
					GROUP BY team_signature
				),
				filtered_runs AS (
					SELECT
						cr.id as run_id,
						cr.duration,
						cr.completed_timestamp,
						ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
					FROM challenge_runs cr
					INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
													AND cr.duration = btr.best_duration
					WHERE cr.dungeon_id = ? AND cr.realm_id = ?
					GROUP BY cr.team_signature
					HAVING cr.id = MIN(cr.id)
				)
				INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
				SELECT
					run_id,
					? as dungeon_id,
					'realm' as ranking_type,
					? as ranking_scope,
					filtered_rank as ranking,
					? as computed_at
				FROM filtered_runs
			`, dungeonID, realmID, dungeonID, realmID, dungeonID, fmt.Sprintf("%d_filtered", realmID), currentTime)

			if err != nil {
				return err
			}
		}
	}

	// update percentile brackets for all realm rankings using efficient SQL
	fmt.Printf("Computing realm ranking brackets...\n")

	// update brackets for unfiltered realm rankings (all realms at once)
	_, err = tx.Exec(`
		UPDATE run_rankings 
		SET percentile_bracket = (
			CASE 
				WHEN counts.ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				run_id,
				dungeon_id,
				ranking_scope,
				ranking,
				COUNT(*) OVER (PARTITION BY dungeon_id, ranking_scope) as total_in_realm_dungeon
			FROM run_rankings 
			WHERE ranking_type = 'realm' AND ranking_scope NOT LIKE '%_filtered'
		) counts
		WHERE run_rankings.run_id = counts.run_id 
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.ranking_scope = counts.ranking_scope
		AND run_rankings.ranking_type = 'realm'
		AND run_rankings.ranking_scope NOT LIKE '%_filtered'
	`)
	if err != nil {
		return err
	}

	// update brackets for filtered realm rankings (all realms at once)
	_, err = tx.Exec(`
		UPDATE run_rankings 
		SET percentile_bracket = (
			CASE 
				WHEN counts.ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.total_in_realm_dungeon - counts.ranking AS REAL) / CAST(counts.total_in_realm_dungeon AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				run_id,
				dungeon_id,
				ranking_scope,
				ranking,
				COUNT(*) OVER (PARTITION BY dungeon_id, ranking_scope) as total_in_realm_dungeon
			FROM run_rankings 
			WHERE ranking_type = 'realm' AND ranking_scope LIKE '%_filtered'
		) counts
		WHERE run_rankings.run_id = counts.run_id 
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.ranking_scope = counts.ranking_scope
		AND run_rankings.ranking_type = 'realm'
		AND run_rankings.ranking_scope LIKE '%_filtered'
	`)
	if err != nil {
		return err
	}

fmt.Printf("[OK] Computed realm rankings with percentile brackets for %d realms\n", len(realmIDs))
	return nil
}

// createPlayerAggregations creates player profiles and best runs
func createPlayerAggregations(tx *sql.Tx) (int, error) {
	fmt.Printf("Computing player aggregations...\n")

	// clear existing aggregation data
	if _, err := tx.Exec("DELETE FROM player_profiles"); err != nil {
		return 0, err
	}
	if _, err := tx.Exec("DELETE FROM player_best_runs"); err != nil {
		return 0, err
	}
	fmt.Printf("Cleared existing player aggregation data\n")

	currentTime := time.Now().UnixMilli()

	// step 1: find best run per player per dungeon WITH rankings in one efficient query
	fmt.Printf("Step 1: Computing best runs per player per dungeon with rankings...\n")
	_, err := tx.Exec(`
		INSERT INTO player_best_runs (
			player_id, dungeon_id, run_id, duration, completed_timestamp,
			global_ranking_filtered, regional_ranking_filtered, realm_ranking_filtered,
			global_percentile_bracket, regional_percentile_bracket, realm_percentile_bracket
		)
		SELECT
			rm.player_id,
			cr.dungeon_id,
			cr.id as run_id,
			cr.duration,
			cr.completed_timestamp,
			rr_gf.ranking as global_ranking_filtered,
			rr_rf.ranking as regional_ranking_filtered,
			rr_lf.ranking as realm_ranking_filtered,
			rr_gf.percentile_bracket as global_percentile_bracket,
			rr_rf.percentile_bracket as regional_percentile_bracket,
			rr_lf.percentile_bracket as realm_percentile_bracket
		FROM run_members rm
		INNER JOIN challenge_runs cr ON rm.run_id = cr.id
		INNER JOIN (
			SELECT
				rm2.player_id,
				cr2.dungeon_id,
				MIN(cr2.duration) as best_duration
			FROM run_members rm2
			INNER JOIN challenge_runs cr2 ON rm2.run_id = cr2.id
			GROUP BY rm2.player_id, cr2.dungeon_id
		) best_times ON rm.player_id = best_times.player_id
					 AND cr.dungeon_id = best_times.dungeon_id
					 AND cr.duration = best_times.best_duration
		LEFT JOIN run_rankings rr_gf ON cr.id = rr_gf.run_id 
			AND rr_gf.ranking_type = 'global' AND rr_gf.ranking_scope = 'filtered'
		LEFT JOIN run_rankings rr_rf ON cr.id = rr_rf.run_id 
			AND rr_rf.ranking_type = 'regional' AND rr_rf.ranking_scope = 'filtered'
		LEFT JOIN run_rankings rr_lf ON cr.id = rr_lf.run_id 
			AND rr_lf.ranking_type = 'realm' AND rr_lf.ranking_scope = 'filtered'
		GROUP BY rm.player_id, cr.dungeon_id
		HAVING cr.id = MIN(cr.id)
	`)
	if err != nil {
		return 0, err
	}

	var bestRunsCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_best_runs").Scan(&bestRunsCount)
fmt.Printf("[OK] Computed %d best runs with rankings in single query\n", bestRunsCount)

fmt.Printf("Step 3: Creating player profiles...\n")
	_, err = tx.Exec(`
		INSERT INTO player_profiles (
			player_id, name, realm_id, dungeons_completed, total_runs,
			combined_best_time, average_best_time, has_complete_coverage, last_updated
		)
		SELECT
			p.id as player_id,
			p.name,
			p.realm_id,
			COUNT(pbr.dungeon_id) as dungeons_completed,
			total_runs.run_count as total_runs,
			COALESCE(SUM(pbr.duration), 0) as combined_best_time,
			CASE
				WHEN COUNT(pbr.dungeon_id) > 0
				THEN CAST(SUM(pbr.duration) AS REAL) / COUNT(pbr.dungeon_id)
				ELSE 0
			END as average_best_time,
			CASE WHEN COUNT(pbr.dungeon_id) = (SELECT COUNT(*) FROM dungeons) THEN 1 ELSE 0 END as has_complete_coverage,
			? as last_updated
		FROM players p
		LEFT JOIN player_best_runs pbr ON p.id = pbr.player_id
		INNER JOIN (
			SELECT rm.player_id, COUNT(*) as run_count
			FROM run_members rm
			GROUP BY rm.player_id
		) total_runs ON p.id = total_runs.player_id
		GROUP BY p.id, p.name, p.realm_id, total_runs.run_count
	`, currentTime)
	if err != nil {
		return 0, err
	}

	var profilesCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_profiles").Scan(&profilesCount)
fmt.Printf("[OK] Created %d player profiles\n", profilesCount)

	// step 4: determine main spec for each player based on best runs
	fmt.Printf("Step 4: Computing main specs...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles
		SET main_spec_id = (
			SELECT spec_counts.spec_id
			FROM (
				SELECT
					rm.player_id,
					rm.spec_id,
					COUNT(*) as spec_count,
					ROW_NUMBER() OVER (PARTITION BY rm.player_id ORDER BY COUNT(*) DESC, rm.spec_id ASC) as rank
				FROM run_members rm
				INNER JOIN player_best_runs pbr ON rm.run_id = pbr.run_id AND rm.player_id = pbr.player_id
				WHERE rm.spec_id IS NOT NULL
				GROUP BY rm.player_id, rm.spec_id
			) spec_counts
			WHERE spec_counts.player_id = player_profiles.player_id AND spec_counts.rank = 1
		)
	`)
	if err != nil {
		return 0, err
	}
fmt.Printf("[OK] Updated main specs\n")

	return profilesCount, nil
}

// computePlayerRankings computes rankings for players with complete coverage
func computePlayerRankings(tx *sql.Tx) (int, error) {
	fmt.Printf("Computing player rankings...\n")

	currentTime := time.Now().UnixMilli()

	// get qualified players count
	var qualifiedCount int
	err := tx.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE has_complete_coverage = 1").Scan(&qualifiedCount)
	if err != nil {
		return 0, err
	}

	fmt.Printf("Found %d players with complete coverage\n", qualifiedCount)

	if qualifiedCount == 0 {
		fmt.Printf("No qualified players found, skipping rankings\n")
		return 0, nil
	}

	// step 1: global rankings
	fmt.Printf("Computing global rankings...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles
		SET global_ranking = (
			SELECT ranking FROM (
				SELECT
					player_id,
					ROW_NUMBER() OVER (ORDER BY combined_best_time ASC) as ranking
				FROM player_profiles
				WHERE has_complete_coverage = 1
			) global_ranks
			WHERE global_ranks.player_id = player_profiles.player_id
		)
		WHERE has_complete_coverage = 1
	`)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
		SELECT
			player_id, 'best_overall', 'global', global_ranking, combined_best_time, ?
		FROM player_profiles
		WHERE has_complete_coverage = 1 AND global_ranking IS NOT NULL
	`, currentTime)
	if err != nil {
		return 0, err
	}

	// update global ranking brackets using efficient SQL
	fmt.Printf("Computing global ranking brackets...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles 
		SET global_ranking_bracket = (
			CASE 
				WHEN player_profiles.global_ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(? - player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(? - player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(? - player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(? - player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		WHERE has_complete_coverage = 1 AND global_ranking IS NOT NULL
	`, qualifiedCount, qualifiedCount, qualifiedCount, qualifiedCount, qualifiedCount, qualifiedCount)
	if err != nil {
		return 0, err
	}

	// step 2: regional rankings
	fmt.Printf("Computing regional rankings...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles
		SET regional_ranking = (
			SELECT ranking FROM (
				SELECT
					pp.player_id,
					ROW_NUMBER() OVER (PARTITION BY r.region ORDER BY pp.combined_best_time ASC) as ranking
				FROM player_profiles pp
				INNER JOIN realms r ON pp.realm_id = r.id
				WHERE pp.has_complete_coverage = 1
			) regional_ranks
			WHERE regional_ranks.player_id = player_profiles.player_id
		)
		WHERE has_complete_coverage = 1
	`)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
		SELECT
			pp.player_id, 'best_overall', r.region, pp.regional_ranking, pp.combined_best_time, ?
		FROM player_profiles pp
		INNER JOIN realms r ON pp.realm_id = r.id
		WHERE pp.has_complete_coverage = 1 AND pp.regional_ranking IS NOT NULL
	`, currentTime)
	if err != nil {
		return 0, err
	}

	// update regional ranking brackets using efficient SQL
	fmt.Printf("Computing regional ranking brackets...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles 
		SET regional_ranking_bracket = (
			CASE 
				WHEN counts.regional_ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.regional_total - counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.regional_total - counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.regional_total - counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.regional_total - counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				pp.player_id,
				pp.regional_ranking,
				COUNT(*) OVER (PARTITION BY r.region) as regional_total
			FROM player_profiles pp
			INNER JOIN realms r ON pp.realm_id = r.id
			WHERE pp.has_complete_coverage = 1 AND pp.regional_ranking IS NOT NULL
		) counts
		WHERE player_profiles.player_id = counts.player_id
		AND player_profiles.has_complete_coverage = 1 
		AND player_profiles.regional_ranking IS NOT NULL
	`)
	if err != nil {
		return 0, err
	}

	// step 3: realm rankings
	fmt.Printf("Computing realm rankings...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles
		SET realm_ranking = (
			SELECT ranking FROM (
				SELECT
					player_id,
					ROW_NUMBER() OVER (PARTITION BY realm_id ORDER BY combined_best_time ASC) as ranking
				FROM player_profiles
				WHERE has_complete_coverage = 1
			) realm_ranks
			WHERE realm_ranks.player_id = player_profiles.player_id
		)
		WHERE has_complete_coverage = 1
	`)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
		SELECT
			player_id, 'best_overall', CAST(realm_id AS TEXT), realm_ranking, combined_best_time, ?
		FROM player_profiles
		WHERE has_complete_coverage = 1 AND realm_ranking IS NOT NULL
	`, currentTime)
	if err != nil {
		return 0, err
	}

	// update realm ranking brackets using efficient SQL
	fmt.Printf("Computing realm ranking brackets...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles 
		SET realm_ranking_bracket = (
			CASE 
				WHEN counts.realm_ranking = 1 THEN 'artifact'
				ELSE 
					CASE 
						WHEN (CAST(counts.realm_total - counts.realm_ranking AS REAL) / CAST(counts.realm_total AS REAL) * 100) >= 95.0 THEN 'legendary'
						WHEN (CAST(counts.realm_total - counts.realm_ranking AS REAL) / CAST(counts.realm_total AS REAL) * 100) >= 80.0 THEN 'epic'  
						WHEN (CAST(counts.realm_total - counts.realm_ranking AS REAL) / CAST(counts.realm_total AS REAL) * 100) >= 60.0 THEN 'rare'
						WHEN (CAST(counts.realm_total - counts.realm_ranking AS REAL) / CAST(counts.realm_total AS REAL) * 100) >= 40.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT 
				player_id,
				realm_ranking,
				COUNT(*) OVER (PARTITION BY realm_id) as realm_total
			FROM player_profiles
			WHERE has_complete_coverage = 1 AND realm_ranking IS NOT NULL
		) counts
		WHERE player_profiles.player_id = counts.player_id
		AND player_profiles.has_complete_coverage = 1 
		AND player_profiles.realm_ranking IS NOT NULL
	`)
	if err != nil {
		return 0, err
	}

fmt.Printf("[OK] Computed rankings with percentile brackets for %d qualified players\n", qualifiedCount)
	return qualifiedCount, nil
}

var processProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Fetch player profiles from Blizzard API",
	Long:  `Fetch detailed player information, equipment, and avatars for players with complete dungeon coverage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Player Profile Fetcher ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

        fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		// check if we have eligible players
		var eligibleCount int
		err = db.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE has_complete_coverage = 1").Scan(&eligibleCount)
		if err != nil {
			return fmt.Errorf("failed to count eligible players: %w", err)
		}

		fmt.Printf("Found %d eligible players with complete coverage\n", eligibleCount)

		if eligibleCount == 0 {
			return fmt.Errorf("no eligible players found - run 'process players' first")
		}

		fmt.Printf("Player profile fetching not yet implemented\n")
		fmt.Printf("This command will fetch player details from Blizzard API for %d players\n", eligibleCount)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Implement Blizzard API client integration\n")
		fmt.Printf("  2. Add concurrent fetching with rate limiting\n")
		fmt.Printf("  3. Store player_details and player_equipment data\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(processCmd)
	processCmd.AddCommand(processAllCmd)
	processCmd.AddCommand(processRankingsCmd)
	processCmd.AddCommand(processPlayersCmd)
	processCmd.AddCommand(processProfilesCmd)
}
