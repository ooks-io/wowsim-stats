package pipeline

import (
	"database/sql"
	"fmt"
	"time"
)

// ProcessPlayersOptions contains options for player processing
type ProcessPlayersOptions struct {
	Verbose bool
}

// ProcessPlayers processes player aggregations and rankings
func ProcessPlayers(db *sql.DB, opts ProcessPlayersOptions) (profilesCreated int, qualifiedPlayers int, err error) {
	fmt.Println("=== Player Aggregation ===")

	// check if we have data
	var runCount, playerCount int
	db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)
	db.QueryRow("SELECT COUNT(*) FROM players").Scan(&playerCount)

	fmt.Printf("Found %d runs and %d players in database\n", runCount, playerCount)

	if runCount == 0 {
		return 0, 0, fmt.Errorf("no runs found in database - run 'fetch cm' first")
	}

	// begin transaction for all player operations
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// step 0: ensure seasons are properly configured
	fmt.Println("\n0. Checking season configuration...")
	var seasonCount int
	tx.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&seasonCount)
	if seasonCount == 0 {
		fmt.Printf("Warning: No seasons found in database. Proceeding with legacy all-time processing.\n")
	} else {
		fmt.Printf("Found %d seasons configured\n", seasonCount)
	}

	// step 1: create player aggregations (season-aware if seasons exist)
	fmt.Println("\n1. Creating player aggregations...")
	profilesCreated, err = createPlayerAggregations(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create player aggregations: %w", err)
	}

	// step 2: compute player rankings (global, regional, realm) per season
	fmt.Println("\n2. Computing player rankings...")
	qualifiedPlayers, err = computePlayerRankings(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compute player rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit player aggregations: %w", err)
	}

	// optimize database
	fmt.Println("\n3. Optimizing database...")
	if _, err := db.Exec("VACUUM"); err != nil {
		fmt.Printf("Warning: database optimization failed: %v\n", err)
	}

	fmt.Printf("\nPlayer aggregation complete!\n")
	fmt.Printf("   Created %d player profiles\n", profilesCreated)
	fmt.Printf("   Computed rankings for %d qualified players\n", qualifiedPlayers)

	return profilesCreated, qualifiedPlayers, nil
}

// ProcessRunRankingsOptions contains options for run ranking processing
type ProcessRunRankingsOptions struct {
	Verbose bool
}

// ProcessRunRankings computes global, regional, and realm rankings for all runs
func ProcessRunRankings(db *sql.DB, opts ProcessRunRankingsOptions) error {
	fmt.Println("=== Run Ranking Processor ===")

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

	// step 3: compute realm run rankings (pool-based for connected realms)
	fmt.Println("\n3. Computing realm run rankings (pool-based)...")
	if err := computeRealmRankings(tx); err != nil {
		return fmt.Errorf("failed to compute realm rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit run rankings: %w", err)
	}

	// optimize database
	fmt.Println("\n4. Optimizing database...")
	if _, err := db.Exec("VACUUM"); err != nil {
		fmt.Printf("Warning: database optimization failed: %v\n", err)
	}

	fmt.Printf("\nRun ranking computation complete!\n")
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

	// step 1: find best run per player per dungeon per season WITH rankings in one efficient query
	fmt.Printf("Step 1: Computing best runs per player per dungeon per season with rankings...\n")
	_, err := tx.Exec(`
		INSERT INTO player_best_runs (
			player_id, dungeon_id, run_id, duration, season_id, completed_timestamp,
			global_ranking_filtered, regional_ranking_filtered, realm_ranking_filtered,
			global_percentile_bracket, regional_percentile_bracket, realm_percentile_bracket
		)
		SELECT
			rm.player_id,
			cr.dungeon_id,
			cr.id as run_id,
			cr.duration,
			COALESCE(s_agg.season_number, 1) as season_id,
			cr.completed_timestamp,
			rr_gf.ranking as global_ranking_filtered,
			rr_rf.ranking as regional_ranking_filtered,
			rr_lf.ranking as realm_ranking_filtered,
			rr_gf.percentile_bracket as global_percentile_bracket,
			rr_rf.percentile_bracket as regional_percentile_bracket,
			rr_lf.percentile_bracket as realm_percentile_bracket
		FROM run_members rm
		INNER JOIN challenge_runs cr ON rm.run_id = cr.id
		LEFT JOIN (
			SELECT ps.period_id, MIN(s.season_number) as season_number
			FROM period_seasons ps
			JOIN seasons s ON ps.season_id = s.id
			GROUP BY ps.period_id
		) s_agg ON cr.period_id = s_agg.period_id
		INNER JOIN (
			SELECT
				rm2.player_id,
				cr2.dungeon_id,
				COALESCE(s2_agg.season_number, 1) as season_id,
				MIN(cr2.duration) as best_duration
			FROM run_members rm2
			INNER JOIN challenge_runs cr2 ON rm2.run_id = cr2.id
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s2_agg ON cr2.period_id = s2_agg.period_id
			GROUP BY rm2.player_id, cr2.dungeon_id, COALESCE(s2_agg.season_number, 1)
		) best_times ON rm.player_id = best_times.player_id
					 AND cr.dungeon_id = best_times.dungeon_id
					 AND COALESCE(s_agg.season_number, 1) = best_times.season_id
					 AND cr.duration = best_times.best_duration
		LEFT JOIN run_rankings rr_gf ON cr.id = rr_gf.run_id
			AND rr_gf.ranking_type = 'global' AND rr_gf.ranking_scope = 'filtered'
			AND rr_gf.season_id = COALESCE(s_agg.season_number, 1)
		LEFT JOIN run_rankings rr_rf ON cr.id = rr_rf.run_id
			AND rr_rf.ranking_type = 'regional' AND rr_rf.ranking_scope = 'filtered'
			AND rr_rf.season_id = COALESCE(s_agg.season_number, 1)
		LEFT JOIN run_rankings rr_lf ON cr.id = rr_lf.run_id
			AND rr_lf.ranking_type = 'realm' AND rr_lf.ranking_scope = 'filtered'
			AND rr_lf.season_id = COALESCE(s_agg.season_number, 1)
		GROUP BY rm.player_id, cr.dungeon_id, COALESCE(s_agg.season_number, 1)
		HAVING cr.id = MIN(cr.id)
	`)
	if err != nil {
		return 0, err
	}

	var bestRunsCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_best_runs").Scan(&bestRunsCount)
	fmt.Printf("[OK] Computed %d best runs with rankings in single query\n", bestRunsCount)

	fmt.Printf("Step 3: Creating player profiles per season...\n")
	_, err = tx.Exec(`
		INSERT INTO player_profiles (
			player_id, season_id, name, realm_id, dungeons_completed, total_runs,
			combined_best_time, average_best_time, has_complete_coverage, last_updated
		)
		SELECT
			p.id as player_id,
			pbr.season_id,
			p.name,
			p.realm_id,
			COUNT(pbr.dungeon_id) as dungeons_completed,
			season_runs.run_count as total_runs,
			COALESCE(SUM(pbr.duration), 0) as combined_best_time,
			CASE
				WHEN COUNT(pbr.dungeon_id) > 0
				THEN CAST(SUM(pbr.duration) AS REAL) / COUNT(pbr.dungeon_id)
				ELSE 0
			END as average_best_time,
			CASE WHEN COUNT(pbr.dungeon_id) = (SELECT COUNT(*) FROM dungeons) THEN 1 ELSE 0 END as has_complete_coverage,
			? as last_updated
		FROM players p
		INNER JOIN player_best_runs pbr ON p.id = pbr.player_id
		INNER JOIN (
			SELECT rm.player_id, COALESCE(s_agg.season_number, 1) as season_id, COUNT(*) as run_count
			FROM run_members rm
			INNER JOIN challenge_runs cr ON rm.run_id = cr.id
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s_agg ON cr.period_id = s_agg.period_id
			GROUP BY rm.player_id, COALESCE(s_agg.season_number, 1)
		) season_runs ON p.id = season_runs.player_id AND pbr.season_id = season_runs.season_id
		GROUP BY p.id, pbr.season_id, p.name, p.realm_id, season_runs.run_count
	`, currentTime)
	if err != nil {
		return 0, err
	}

	var profilesCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_profiles").Scan(&profilesCount)
	fmt.Printf("[OK] Created %d player profiles\n", profilesCount)

	// step 4: determine main spec for each player per season based on best runs
	fmt.Printf("Step 4: Computing main specs per season...\n")
	_, err = tx.Exec(`
		UPDATE player_profiles
		SET main_spec_id = (
			SELECT spec_counts.spec_id
			FROM (
				SELECT
					rm.player_id,
					pbr.season_id,
					rm.spec_id,
					COUNT(*) as spec_count,
					ROW_NUMBER() OVER (PARTITION BY rm.player_id, pbr.season_id ORDER BY COUNT(*) DESC, rm.spec_id ASC) as rank
				FROM run_members rm
				INNER JOIN player_best_runs pbr ON rm.run_id = pbr.run_id AND rm.player_id = pbr.player_id
				WHERE rm.spec_id IS NOT NULL
				GROUP BY rm.player_id, pbr.season_id, rm.spec_id
			) spec_counts
			WHERE spec_counts.player_id = player_profiles.player_id
				AND spec_counts.season_id = player_profiles.season_id
				AND spec_counts.rank = 1
		)
	`)
	if err != nil {
		return 0, err
	}
	fmt.Printf("[OK] Updated main specs\n")

	return profilesCount, nil
}

// computePlayerRankings computes rankings for players with complete coverage per season
func computePlayerRankings(tx *sql.Tx) (int, error) {
	fmt.Printf("Computing player rankings per season...\n")

	currentTime := time.Now().UnixMilli()

	// Get all distinct season numbers (global seasons across all regions)
	seasonRows, err := tx.Query("SELECT DISTINCT season_number FROM seasons ORDER BY season_number")
	if err != nil {
		return 0, fmt.Errorf("failed to query seasons: %w", err)
	}
	defer seasonRows.Close()

	var seasons []int
	for seasonRows.Next() {
		var seasonNumber int
		if err := seasonRows.Scan(&seasonNumber); err != nil {
			return 0, err
		}
		seasons = append(seasons, seasonNumber)
	}

	if len(seasons) == 0 {
		fmt.Printf("No seasons found in player profiles, skipping rankings\n")
		return 0, nil
	}

	totalQualified := 0
	for _, seasonID := range seasons {
		fmt.Printf("\n=== Processing Season %d ===\n", seasonID)

		// get qualified players count for this season
		var qualifiedCount int
		err := tx.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE season_id = ? AND has_complete_coverage = 1", seasonID).Scan(&qualifiedCount)
		if err != nil {
			return 0, err
		}

		fmt.Printf("Found %d players with complete coverage in season %d\n", qualifiedCount, seasonID)

		if qualifiedCount == 0 {
			fmt.Printf("No qualified players found for season %d, skipping\n", seasonID)
			continue
		}

		totalQualified += qualifiedCount

		// step 1: global rankings for this season
		fmt.Printf("Computing global rankings for season %d...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET global_ranking = (
				SELECT ranking FROM (
					SELECT
						player_id,
						ROW_NUMBER() OVER (ORDER BY combined_best_time ASC) as ranking
					FROM player_profiles
					WHERE season_id = ? AND has_complete_coverage = 1
				) global_ranks
				WHERE global_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1
		`, seasonID, seasonID)
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec(`
			INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
			SELECT
				player_id, 'best_overall', 'global', global_ranking, combined_best_time, ?
			FROM player_profiles
			WHERE season_id = ? AND has_complete_coverage = 1 AND global_ranking IS NOT NULL
		`, currentTime, seasonID)
		if err != nil {
			return 0, err
		}

		// update global ranking brackets for this season
		fmt.Printf("Computing global ranking brackets for season %d...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET global_ranking_bracket = (
				CASE
					WHEN player_profiles.combined_best_time = (SELECT MIN(combined_best_time) FROM player_profiles WHERE season_id = ? AND has_complete_coverage = 1) THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(player_profiles.global_ranking AS REAL) / CAST(? AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			WHERE season_id = ? AND has_complete_coverage = 1 AND global_ranking IS NOT NULL
		`, seasonID, qualifiedCount, qualifiedCount, qualifiedCount, qualifiedCount, qualifiedCount, seasonID)
		if err != nil {
			return 0, err
		}

		// step 2: regional rankings for this season
		fmt.Printf("Computing regional rankings for season %d...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET regional_ranking = (
				SELECT ranking FROM (
					SELECT
						pp.player_id,
						ROW_NUMBER() OVER (PARTITION BY r.region ORDER BY pp.combined_best_time ASC) as ranking
					FROM player_profiles pp
					INNER JOIN realms r ON pp.realm_id = r.id
					WHERE pp.season_id = ? AND pp.has_complete_coverage = 1
				) regional_ranks
				WHERE regional_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1
		`, seasonID, seasonID)
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec(`
			INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
			SELECT
				pp.player_id, 'best_overall', r.region, pp.regional_ranking, pp.combined_best_time, ?
			FROM player_profiles pp
			INNER JOIN realms r ON pp.realm_id = r.id
			WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.regional_ranking IS NOT NULL
		`, currentTime, seasonID)
		if err != nil {
			return 0, err
		}

		// update regional ranking brackets for this season
		fmt.Printf("Computing regional ranking brackets for season %d...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET regional_ranking_bracket = (
				CASE
					WHEN counts.combined_best_time = counts.regional_min_time THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.regional_ranking AS REAL) / CAST(counts.regional_total AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					pp.player_id,
					pp.regional_ranking,
					pp.combined_best_time,
					MIN(pp.combined_best_time) OVER (PARTITION BY r.region) as regional_min_time,
					COUNT(*) OVER (PARTITION BY r.region) as regional_total
				FROM player_profiles pp
				INNER JOIN realms r ON pp.realm_id = r.id
				WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.regional_ranking IS NOT NULL
			) counts
			WHERE player_profiles.player_id = counts.player_id
			AND player_profiles.season_id = ?
			AND player_profiles.has_complete_coverage = 1
			AND player_profiles.regional_ranking IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return 0, err
		}

		// step 3: realm rankings for this season (pool-based for connected realms)
		fmt.Printf("Computing realm rankings for season %d (using realm pools)...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET realm_ranking = (
				SELECT ranking FROM (
					SELECT
						pp.player_id,
						ROW_NUMBER() OVER (
							PARTITION BY COALESCE(parent_r.id, r.id)
							ORDER BY pp.combined_best_time ASC
						) as ranking
					FROM player_profiles pp
					JOIN realms r ON pp.realm_id = r.id
					LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
					WHERE pp.season_id = ? AND pp.has_complete_coverage = 1
				) realm_ranks
				WHERE realm_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1
		`, seasonID, seasonID)
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec(`
			INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
			SELECT
				player_id, 'best_overall', CAST(realm_id AS TEXT), realm_ranking, combined_best_time, ?
			FROM player_profiles
			WHERE season_id = ? AND has_complete_coverage = 1 AND realm_ranking IS NOT NULL
		`, currentTime, seasonID)
		if err != nil {
			return 0, err
		}

		// update realm ranking brackets for this season (pool-based for connected realms)
		fmt.Printf("Computing realm ranking brackets for season %d (using realm pools)...\n", seasonID)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET realm_ranking_bracket = (
				CASE
					WHEN counts.combined_best_time = counts.pool_min_time THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.realm_ranking AS REAL) / CAST(counts.pool_total AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.realm_ranking AS REAL) / CAST(counts.pool_total AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.realm_ranking AS REAL) / CAST(counts.pool_total AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.realm_ranking AS REAL) / CAST(counts.pool_total AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.realm_ranking AS REAL) / CAST(counts.pool_total AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					pp.player_id,
					pp.realm_ranking,
					pp.combined_best_time,
					COALESCE(parent_r.id, r.id) as pool_id,
					MIN(pp.combined_best_time) OVER (PARTITION BY COALESCE(parent_r.id, r.id)) as pool_min_time,
					COUNT(*) OVER (PARTITION BY COALESCE(parent_r.id, r.id)) as pool_total
				FROM player_profiles pp
				JOIN realms r ON pp.realm_id = r.id
				LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
				WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.realm_ranking IS NOT NULL
			) counts
			WHERE player_profiles.player_id = counts.player_id
			AND player_profiles.season_id = ?
			AND player_profiles.has_complete_coverage = 1
			AND player_profiles.realm_ranking IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return 0, err
		}

		fmt.Printf("[OK] Computed rankings for season %d with percentile brackets (%d qualified players)\n", seasonID, qualifiedCount)
	}

	fmt.Printf("\n[OK] Computed rankings for all seasons (total: %d qualified players across all seasons)\n", totalQualified)
	return totalQualified, nil
}

// computeGlobalRankings computes global rankings for all runs (per season)
func computeGlobalRankings(tx *sql.Tx) error {
	fmt.Printf("Computing global rankings per season...\n")

	currentTime := time.Now().UnixMilli()

	// clear existing global rankings
	if _, err := tx.Exec("DELETE FROM run_rankings WHERE ranking_type = 'global'"); err != nil {
		return err
	}

	// unfiltered global rankings - first insert without brackets, partitioned by season
	_, err := tx.Exec(`
		INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
		SELECT
			cr.id as run_id,
			cr.dungeon_id,
			'global' as ranking_type,
			'all' as ranking_scope,
			ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id, COALESCE(s_agg.season_number, 1) ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
			COALESCE(s_agg.season_number, 1) as season_id,
			? as computed_at
		FROM challenge_runs cr
		LEFT JOIN (
			SELECT ps.period_id, MIN(s.season_number) as season_number
			FROM period_seasons ps
			JOIN seasons s ON ps.season_id = s.id
			GROUP BY ps.period_id
		) s_agg ON cr.period_id = s_agg.period_id
	`, currentTime)
	if err != nil {
		return err
	}

	// update percentile brackets for unfiltered global rankings using efficient SQL (per season)
	fmt.Printf("Computing global ranking brackets per season...\n")
	_, err = tx.Exec(`
		UPDATE run_rankings
		SET percentile_bracket = (
			CASE
				WHEN counts.duration = counts.min_duration THEN 'artifact'
				ELSE
					CASE
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT
				rr.run_id,
				rr.dungeon_id,
				rr.season_id,
				rr.ranking,
				cr.duration,
				MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id) as min_duration,
				COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id) as total_in_season_dungeon
			FROM run_rankings rr
			INNER JOIN challenge_runs cr ON rr.run_id = cr.id
			WHERE rr.ranking_type = 'global' AND rr.ranking_scope = 'all'
		) counts
		WHERE run_rankings.run_id = counts.run_id
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.season_id = counts.season_id
		AND run_rankings.ranking_type = 'global'
		AND run_rankings.ranking_scope = 'all'
	`)
	if err != nil {
		return err
	}

	// filtered global rankings (best time per team, per season)
	// get all dungeon IDs and season IDs for filtered rankings
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

	// Get all seasons (including fallback season 1 for unmapped periods)
	seasonRows, err := tx.Query(`
		SELECT DISTINCT COALESCE(s_agg.season_number, 1) as season_id
		FROM challenge_runs cr
		LEFT JOIN (
			SELECT ps.period_id, MIN(s.season_number) as season_number
			FROM period_seasons ps
			JOIN seasons s ON ps.season_id = s.id
			GROUP BY ps.period_id
		) s_agg ON cr.period_id = s_agg.period_id
	`)
	if err != nil {
		return err
	}
	defer seasonRows.Close()

	var seasonIDs []int
	for seasonRows.Next() {
		var id int
		if err := seasonRows.Scan(&id); err != nil {
			return err
		}
		seasonIDs = append(seasonIDs, id)
	}

	// Process each dungeon x season combination
	for _, dungeonID := range dungeonIDs {
		for _, seasonID := range seasonIDs {
			_, err := tx.Exec(`
				WITH best_team_runs AS (
					SELECT
						cr.team_signature,
						MIN(cr.duration) as best_duration
					FROM challenge_runs cr
					LEFT JOIN (
						SELECT ps.period_id, MIN(s.season_number) as season_number
						FROM period_seasons ps
						JOIN seasons s ON ps.season_id = s.id
						GROUP BY ps.period_id
					) s_agg ON cr.period_id = s_agg.period_id
					WHERE cr.dungeon_id = ? AND COALESCE(s_agg.season_number, 1) = ?
					GROUP BY cr.team_signature
				),
				filtered_runs AS (
					SELECT
						cr.id as run_id,
						cr.duration,
						cr.completed_timestamp,
						ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
					FROM challenge_runs cr
					LEFT JOIN (
						SELECT ps.period_id, MIN(s.season_number) as season_number
						FROM period_seasons ps
						JOIN seasons s ON ps.season_id = s.id
						GROUP BY ps.period_id
					) s_agg ON cr.period_id = s_agg.period_id
					INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
												 AND cr.duration = btr.best_duration
					WHERE cr.dungeon_id = ? AND COALESCE(s_agg.season_number, 1) = ?
					GROUP BY cr.team_signature
					HAVING cr.id = MIN(cr.id)
				)
				INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
				SELECT
					run_id,
					? as dungeon_id,
					'global' as ranking_type,
					'filtered' as ranking_scope,
					filtered_rank as ranking,
					? as season_id,
					? as computed_at
				FROM filtered_runs
			`, dungeonID, seasonID, dungeonID, seasonID, dungeonID, seasonID, currentTime)

			if err != nil {
				return err
			}
		}
	}

	// update percentile brackets for filtered global rankings using efficient SQL (per season)
	fmt.Printf("Computing filtered global ranking brackets per season...\n")
	_, err = tx.Exec(`
		UPDATE run_rankings
		SET percentile_bracket = (
			CASE
				WHEN counts.duration = counts.min_duration THEN 'artifact'
				ELSE
					CASE
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
						WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
						ELSE 'common'
					END
			END
		)
		FROM (
			SELECT
				rr.run_id,
				rr.dungeon_id,
				rr.season_id,
				rr.ranking,
				cr.duration,
				MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id) as min_duration,
				COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id) as total_in_season_dungeon
			FROM run_rankings rr
			INNER JOIN challenge_runs cr ON rr.run_id = cr.id
			WHERE rr.ranking_type = 'global' AND rr.ranking_scope = 'filtered'
		) counts
		WHERE run_rankings.run_id = counts.run_id
		AND run_rankings.dungeon_id = counts.dungeon_id
		AND run_rankings.season_id = counts.season_id
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
		// unfiltered regional rankings (per season)
		_, err := tx.Exec(`
			INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
			SELECT
				cr.id as run_id,
				cr.dungeon_id,
				'regional' as ranking_type,
				? as ranking_scope,
				ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id, COALESCE(s_agg.season_number, 1) ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
				COALESCE(s_agg.season_number, 1) as season_id,
				? as computed_at
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s_agg ON cr.period_id = s_agg.period_id
			WHERE r.region = ?
		`, region, currentTime, region)

		if err != nil {
			return err
		}

		// update percentile brackets for unfiltered regional rankings using efficient SQL (per season)
		fmt.Printf("Computing unfiltered regional ranking brackets for %s per season...\n", region)
		_, err = tx.Exec(`
			UPDATE run_rankings
			SET percentile_bracket = (
				CASE
					WHEN counts.duration = counts.min_duration THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					rr.run_id,
					rr.dungeon_id,
					rr.season_id,
					rr.ranking,
					cr.duration,
					MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as min_duration,
					COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as total_in_region_season_dungeon
				FROM run_rankings rr
				INNER JOIN challenge_runs cr ON rr.run_id = cr.id
				WHERE rr.ranking_type = 'regional' AND rr.ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.season_id = counts.season_id
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

		// Get all seasons for this region
		seasonRows, err := tx.Query(`
			SELECT DISTINCT COALESCE(s_agg.season_number, 1) as season_id
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s_agg ON cr.period_id = s_agg.period_id
			WHERE r.region = ?
		`, region)
		if err != nil {
			return err
		}

		var seasonIDs []int
		for seasonRows.Next() {
			var id int
			if err := seasonRows.Scan(&id); err != nil {
				seasonRows.Close()
				return err
			}
			seasonIDs = append(seasonIDs, id)
		}
		seasonRows.Close()

		// filtered regional rankings - per dungeon x season
		for _, dungeonID := range dungeonIDs {
			for _, seasonID := range seasonIDs {
				_, err := tx.Exec(`
					WITH best_team_runs AS (
						SELECT
							cr.team_signature,
							MIN(cr.duration) as best_duration
						FROM challenge_runs cr
						INNER JOIN realms r ON cr.realm_id = r.id
						LEFT JOIN (
							SELECT ps.period_id, MIN(s.season_number) as season_number
							FROM period_seasons ps
							JOIN seasons s ON ps.season_id = s.id
							GROUP BY ps.period_id
						) s_agg ON cr.period_id = s_agg.period_id
						WHERE cr.dungeon_id = ? AND r.region = ? AND COALESCE(s_agg.season_number, 1) = ?
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
						LEFT JOIN (
							SELECT ps.period_id, MIN(s.season_number) as season_number
							FROM period_seasons ps
							JOIN seasons s ON ps.season_id = s.id
							GROUP BY ps.period_id
						) s_agg ON cr.period_id = s_agg.period_id
						INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
														AND cr.duration = btr.best_duration
						WHERE cr.dungeon_id = ? AND r.region = ? AND COALESCE(s_agg.season_number, 1) = ?
						GROUP BY cr.team_signature
						HAVING cr.id = MIN(cr.id)
					)
					INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
					SELECT
						run_id,
						? as dungeon_id,
						'regional' as ranking_type,
						? as ranking_scope,
						filtered_rank as ranking,
						? as season_id,
						? as computed_at
					FROM filtered_runs
				`, dungeonID, region, seasonID, dungeonID, region, seasonID, dungeonID, region+"_filtered", seasonID, currentTime)

				if err != nil {
					return err
				}
			}
		}

		// update percentile brackets for filtered regional rankings using efficient SQL (per season)
		filteredScope := region + "_filtered"
		fmt.Printf("Computing filtered regional ranking brackets for %s per season...\n", region)
		_, err = tx.Exec(`
			UPDATE run_rankings
			SET percentile_bracket = (
				CASE
					WHEN counts.duration = counts.min_duration THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_region_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					rr.run_id,
					rr.dungeon_id,
					rr.season_id,
					rr.ranking,
					cr.duration,
					MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as min_duration,
					COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as total_in_region_season_dungeon
				FROM run_rankings rr
				INNER JOIN challenge_runs cr ON rr.run_id = cr.id
				WHERE rr.ranking_type = 'regional' AND rr.ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.season_id = counts.season_id
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

// computeRealmRankings computes realm rankings per realm pool (connected realms grouped together)
func computeRealmRankings(tx *sql.Tx) error {
	fmt.Printf("Computing realm rankings (pool-based for connected realms)...\n")

	currentTime := time.Now().UnixMilli()

	// clear existing realm rankings
	if _, err := tx.Exec("DELETE FROM run_rankings WHERE ranking_type = 'realm'"); err != nil {
		return err
	}

	// get all realm pools (parent realms or independent realms that will be used as pool identifiers)
	type RealmPool struct {
		PoolSlug string
		Region   string
	}

	poolRows, err := tx.Query(`
		SELECT DISTINCT
			COALESCE(parent_r.slug, r.slug) as pool_slug,
			r.region
		FROM realms r
		LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
		ORDER BY r.region, pool_slug
	`)
	if err != nil {
		return err
	}
	defer poolRows.Close()

	var pools []RealmPool
	for poolRows.Next() {
		var pool RealmPool
		if err := poolRows.Scan(&pool.PoolSlug, &pool.Region); err != nil {
			return err
		}
		pools = append(pools, pool)
	}

	fmt.Printf("Found %d realm pools to process\n", len(pools))

	for _, pool := range pools {
		// unfiltered realm rankings (per season) using pool-based partitioning
		_, err := tx.Exec(`
			INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
			SELECT
				cr.id as run_id,
				cr.dungeon_id,
				'realm' as ranking_type,
				? as ranking_scope,
				ROW_NUMBER() OVER (
					PARTITION BY cr.dungeon_id, COALESCE(s_agg.season_number, 1)
					ORDER BY cr.duration ASC, cr.completed_timestamp ASC
				) as ranking,
				COALESCE(s_agg.season_number, 1) as season_id,
				? as computed_at
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s_agg ON cr.period_id = s_agg.period_id
			WHERE r.region = ? AND COALESCE(parent_r.slug, r.slug) = ?
		`, pool.PoolSlug, currentTime, pool.Region, pool.PoolSlug)

		if err != nil {
			return err
		}

		// update percentile brackets for unfiltered realm rankings
		_, err = tx.Exec(`
			UPDATE run_rankings
			SET percentile_bracket = (
				CASE
					WHEN counts.duration = counts.min_duration THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					rr.run_id,
					rr.dungeon_id,
					rr.season_id,
					rr.ranking,
					cr.duration,
					MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as min_duration,
					COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as total_in_pool_season_dungeon
				FROM run_rankings rr
				INNER JOIN challenge_runs cr ON rr.run_id = cr.id
				WHERE rr.ranking_type = 'realm' AND rr.ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.season_id = counts.season_id
			AND run_rankings.ranking_type = 'realm'
			AND run_rankings.ranking_scope = ?
		`, pool.PoolSlug, pool.PoolSlug)
		if err != nil {
			return err
		}
	}

	// now compute filtered rankings per pool x dungeon x season
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

	for _, pool := range pools {
		// Get seasons for this pool
		seasonRows, err := tx.Query(`
			SELECT DISTINCT COALESCE(s_agg.season_number, 1) as season_id
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
			LEFT JOIN (
				SELECT ps.period_id, MIN(s.season_number) as season_number
				FROM period_seasons ps
				JOIN seasons s ON ps.season_id = s.id
				GROUP BY ps.period_id
			) s_agg ON cr.period_id = s_agg.period_id
			WHERE r.region = ? AND COALESCE(parent_r.slug, r.slug) = ?
		`, pool.Region, pool.PoolSlug)
		if err != nil {
			return err
		}

		var seasonIDs []int
		for seasonRows.Next() {
			var id int
			if err := seasonRows.Scan(&id); err != nil {
				seasonRows.Close()
				return err
			}
			seasonIDs = append(seasonIDs, id)
		}
		seasonRows.Close()

		// filtered realm rankings - per dungeon x season using pool-based partitioning
		for _, dungeonID := range dungeonIDs {
			for _, seasonID := range seasonIDs {
				_, err := tx.Exec(`
					WITH best_team_runs AS (
						SELECT
							cr.team_signature,
							MIN(cr.duration) as best_duration
						FROM challenge_runs cr
						INNER JOIN realms r ON cr.realm_id = r.id
						LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
						LEFT JOIN (
							SELECT ps.period_id, MIN(s.season_number) as season_number
							FROM period_seasons ps
							JOIN seasons s ON ps.season_id = s.id
							GROUP BY ps.period_id
						) s_agg ON cr.period_id = s_agg.period_id
						WHERE cr.dungeon_id = ?
							AND r.region = ?
							AND COALESCE(parent_r.slug, r.slug) = ?
							AND COALESCE(s_agg.season_number, 1) = ?
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
						LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
						LEFT JOIN (
							SELECT ps.period_id, MIN(s.season_number) as season_number
							FROM period_seasons ps
							JOIN seasons s ON ps.season_id = s.id
							GROUP BY ps.period_id
						) s_agg ON cr.period_id = s_agg.period_id
						INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
														AND cr.duration = btr.best_duration
						WHERE cr.dungeon_id = ?
							AND r.region = ?
							AND COALESCE(parent_r.slug, r.slug) = ?
							AND COALESCE(s_agg.season_number, 1) = ?
						GROUP BY cr.team_signature
						HAVING cr.id = MIN(cr.id)
					)
					INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, season_id, computed_at)
					SELECT
						run_id,
						? as dungeon_id,
						'realm' as ranking_type,
						? as ranking_scope,
						filtered_rank as ranking,
						? as season_id,
						? as computed_at
					FROM filtered_runs
				`, dungeonID, pool.Region, pool.PoolSlug, seasonID,
					dungeonID, pool.Region, pool.PoolSlug, seasonID,
					dungeonID, pool.PoolSlug+"_filtered", seasonID, currentTime)

				if err != nil {
					return err
				}
			}
		}

		// update percentile brackets for filtered realm rankings
		filteredScope := pool.PoolSlug + "_filtered"
		_, err = tx.Exec(`
			UPDATE run_rankings
			SET percentile_bracket = (
				CASE
					WHEN counts.duration = counts.min_duration THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.ranking AS REAL) / CAST(counts.total_in_pool_season_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					rr.run_id,
					rr.dungeon_id,
					rr.season_id,
					rr.ranking,
					cr.duration,
					MIN(cr.duration) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as min_duration,
					COUNT(*) OVER (PARTITION BY rr.dungeon_id, rr.season_id, rr.ranking_scope) as total_in_pool_season_dungeon
				FROM run_rankings rr
				INNER JOIN challenge_runs cr ON rr.run_id = cr.id
				WHERE rr.ranking_type = 'realm' AND rr.ranking_scope = ?
			) counts
			WHERE run_rankings.run_id = counts.run_id
			AND run_rankings.dungeon_id = counts.dungeon_id
			AND run_rankings.season_id = counts.season_id
			AND run_rankings.ranking_type = 'realm'
			AND run_rankings.ranking_scope = ?
		`, filteredScope, filteredScope)
		if err != nil {
			return err
		}
	}

	fmt.Printf("[OK] Computed realm rankings with percentile brackets for %d realm pools\n", len(pools))
	return nil
}
