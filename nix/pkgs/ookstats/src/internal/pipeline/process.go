package pipeline

// season assignment has been migrated to use cr.season_id directly instead of period_seasons lookups.
// all queries now use timestamp-based season assignment from the challenge_runs table.

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/wow"
)

// ProcessPlayersOptions contains options for player processing
type ProcessPlayersOptions struct {
	Verbose bool
}

// ProcessPlayers processes player aggregations and rankings
func ProcessPlayers(db *sql.DB, opts ProcessPlayersOptions) (profilesCreated int, qualifiedPlayers int, err error) {
	log.Info("player aggregation")

	// check if we have data
	var runCount, playerCount int
	db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)
	db.QueryRow("SELECT COUNT(*) FROM players").Scan(&playerCount)

	log.Info("found data in database", "runs", runCount, "players", playerCount)

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
	log.Info("checking season configuration")
	var seasonCount int
	tx.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&seasonCount)
	if seasonCount == 0 {
		log.Warn("no seasons found in database - proceeding with legacy all-time processing")
	} else {
		log.Info("found seasons configured", "count", seasonCount)
	}

	// step 1: create player aggregations (season-aware if seasons exist)
	log.Info("creating player aggregations")
	profilesCreated, err = createPlayerAggregations(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create player aggregations: %w", err)
	}

	// step 2: compute player rankings (global, regional, realm) per season
	log.Info("computing player rankings")
	qualifiedPlayers, err = computePlayerRankings(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compute player rankings: %w", err)
	}

	// step 3: compute class-specific rankings per season
	log.Info("computing class-specific rankings")
	if err = computePlayerClassRankings(tx); err != nil {
		return 0, 0, fmt.Errorf("failed to compute class rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit player aggregations: %w", err)
	}

	// optimize database
	log.Info("optimizing database")
	if _, err := db.Exec("VACUUM"); err != nil {
		log.Warn("database optimization failed", "error", err)
	}

	log.Info("player aggregation complete",
		"profiles", profilesCreated,
		"qualified_players", qualifiedPlayers)

	return profilesCreated, qualifiedPlayers, nil
}

// ProcessRunRankingsOptions contains options for run ranking processing
type ProcessRunRankingsOptions struct {
	Verbose bool
}

// ProcessRunRankings computes global, regional, and realm rankings for all runs
func ProcessRunRankings(db *sql.DB, opts ProcessRunRankingsOptions) error {
	log.Info("run ranking processor")

	// check if we have data
	var runCount int
	db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)

	if runCount == 0 {
		return fmt.Errorf("no runs found in database - run 'fetch cm' first")
	}

	log.Info("found runs in database", "runs", runCount)

	// begin transaction for all ranking operations
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// step 1: compute global run rankings
	log.Info("computing global run rankings")
	if err := computeGlobalRankings(tx); err != nil {
		return fmt.Errorf("failed to compute global rankings: %w", err)
	}

	// step 2: compute regional run rankings
	log.Info("computing regional run rankings")
	if err := computeRegionalRankings(tx); err != nil {
		return fmt.Errorf("failed to compute regional rankings: %w", err)
	}

	// step 3: compute realm run rankings (pool-based for connected realms)
	log.Info("computing realm run rankings (pool-based)")
	if err := computeRealmRankings(tx); err != nil {
		return fmt.Errorf("failed to compute realm rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit run rankings: %w", err)
	}

	// optimize database
	log.Info("optimizing database")
	if _, err := db.Exec("VACUUM"); err != nil {
		log.Warn("database optimization failed", "error", err)
	}

	log.Info("run ranking computation complete")
	return nil
}

// createPlayerAggregations creates player profiles and best runs
func createPlayerAggregations(tx *sql.Tx) (int, error) {
	log.Info("computing player aggregations")

	// clear existing aggregation data
	if _, err := tx.Exec("DELETE FROM player_profiles"); err != nil {
		return 0, err
	}
	if _, err := tx.Exec("DELETE FROM player_best_runs"); err != nil {
		return 0, err
	}
	log.Info("cleared existing player aggregation data")

	currentTime := time.Now().UnixMilli()

	// step 1: find best run per player per dungeon per season WITH rankings in one efficient query
	log.Info("computing best runs per player per dungeon per season with rankings")
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
			cr.season_id,
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
				cr2.season_id,
				MIN(cr2.duration) as best_duration
			FROM run_members rm2
			INNER JOIN challenge_runs cr2 ON rm2.run_id = cr2.id
			GROUP BY rm2.player_id, cr2.dungeon_id, cr2.season_id
		) best_times ON rm.player_id = best_times.player_id
					 AND cr.dungeon_id = best_times.dungeon_id
					 AND cr.season_id = best_times.season_id
					 AND cr.duration = best_times.best_duration
		LEFT JOIN run_rankings rr_gf ON cr.id = rr_gf.run_id
			AND rr_gf.ranking_type = 'global' AND rr_gf.ranking_scope = 'filtered'
			AND rr_gf.season_id = cr.season_id
		LEFT JOIN run_rankings rr_rf ON cr.id = rr_rf.run_id
			AND rr_rf.ranking_type = 'regional' AND rr_rf.ranking_scope = 'filtered'
			AND rr_rf.season_id = cr.season_id
		LEFT JOIN run_rankings rr_lf ON cr.id = rr_lf.run_id
			AND rr_lf.ranking_type = 'realm' AND rr_lf.ranking_scope = 'filtered'
			AND rr_lf.season_id = cr.season_id
		GROUP BY rm.player_id, cr.dungeon_id, cr.season_id
		HAVING cr.id = MIN(cr.id)
	`)
	if err != nil {
		return 0, err
	}

	var bestRunsCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_best_runs").Scan(&bestRunsCount)
	log.Info("computed best runs with rankings", "count", bestRunsCount)

	log.Info("creating player profiles per season")
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
			SELECT rm.player_id, cr.season_id, COUNT(*) as run_count
			FROM run_members rm
			INNER JOIN challenge_runs cr ON rm.run_id = cr.id
			GROUP BY rm.player_id, cr.season_id
		) season_runs ON p.id = season_runs.player_id AND pbr.season_id = season_runs.season_id
		GROUP BY p.id, pbr.season_id, p.name, p.realm_id, season_runs.run_count
	`, currentTime)
	if err != nil {
		return 0, err
	}

	var profilesCount int
	tx.QueryRow("SELECT COUNT(*) FROM player_profiles").Scan(&profilesCount)
	log.Info("created player profiles", "count", profilesCount)

	// step 4: determine main spec for each player per season based on best runs
	log.Info("computing main specs per season")
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
	log.Info("updated main specs")

	// step 5: derive class from main spec
	log.Info("deriving class names from main specs")
	if err := deriveClassFromMainSpec(tx); err != nil {
		return 0, fmt.Errorf("failed to derive class names: %w", err)
	}
	log.Info("updated class names")

	return profilesCount, nil
}

// deriveClassFromMainSpec derives class_name from main_spec_id for all player profiles
func deriveClassFromMainSpec(tx *sql.Tx) error {
	// Query all player profiles with a main_spec_id
	rows, err := tx.Query(`
		SELECT player_id, season_id, main_spec_id
		FROM player_profiles
		WHERE main_spec_id IS NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to query player profiles: %w", err)
	}
	defer rows.Close()

	// Prepare update statement
	updateStmt, err := tx.Prepare(`
		UPDATE player_profiles
		SET class_name = ?
		WHERE player_id = ? AND season_id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	updatedCount := 0
	for rows.Next() {
		var playerID, seasonID, mainSpecID int
		if err := rows.Scan(&playerID, &seasonID, &mainSpecID); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Use wow package to get class from spec
		className, _, ok := wow.GetClassAndSpec(mainSpecID)
		if !ok {
			// If spec not found, skip this player (leave class_name NULL)
			continue
		}

		// Update player_profiles with derived class
		if _, err := updateStmt.Exec(className, playerID, seasonID); err != nil {
			return fmt.Errorf("failed to update class for player %d season %d: %w", playerID, seasonID, err)
		}
		updatedCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Debug("derived class for player profiles", "count", updatedCount)
	return nil
}

// computePlayerRankings computes rankings for players with complete coverage per season
func computePlayerRankings(tx *sql.Tx) (int, error) {
	log.Info("computing player rankings per season")

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
		log.Warn("no seasons found in player profiles - skipping rankings")
		return 0, nil
	}

	totalQualified := 0
	for _, seasonID := range seasons {
		log.Info("processing season", "season_id", seasonID)

		// get qualified players count for this season
		var qualifiedCount int
		err := tx.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE season_id = ? AND has_complete_coverage = 1", seasonID).Scan(&qualifiedCount)
		if err != nil {
			return 0, err
		}

		log.Info("found players with complete coverage",
			"count", qualifiedCount,
			"season_id", seasonID)

		if qualifiedCount == 0 {
			log.Warn("no qualified players found - skipping season", "season_id", seasonID)
			continue
		}

		totalQualified += qualifiedCount

		// step 1: global rankings for this season
		log.Info("computing global rankings", "season_id", seasonID)
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
			INSERT OR REPLACE INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
			SELECT
				player_id, 'best_overall', 'global', global_ranking, combined_best_time, ?
			FROM player_profiles
			WHERE season_id = ? AND has_complete_coverage = 1 AND global_ranking IS NOT NULL
		`, currentTime, seasonID)
		if err != nil {
			return 0, err
		}

		// update global ranking brackets for this season
		log.Info("computing global ranking brackets", "season_id", seasonID)
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
		log.Info("computing regional rankings", "season_id", seasonID)
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
			INSERT OR REPLACE INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
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
		log.Info("computing regional ranking brackets", "season_id", seasonID)
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
		log.Info("computing realm rankings (using realm pools)", "season_id", seasonID)
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
			INSERT OR REPLACE INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
			SELECT
				player_id, 'best_overall', CAST(realm_id AS TEXT), realm_ranking, combined_best_time, ?
			FROM player_profiles
			WHERE season_id = ? AND has_complete_coverage = 1 AND realm_ranking IS NOT NULL
		`, currentTime, seasonID)
		if err != nil {
			return 0, err
		}

		// update realm ranking brackets for this season (pool-based for connected realms)
		log.Info("computing realm ranking brackets (using realm pools)", "season_id", seasonID)
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

		log.Info("computed rankings for season with percentile brackets",
			"season_id", seasonID,
			"qualified_players", qualifiedCount)
	}

	log.Info("computed rankings for all seasons",
		"total_qualified_players", totalQualified)
	return totalQualified, nil
}

// computePlayerClassRankings computes class-specific rankings for players per season
func computePlayerClassRankings(tx *sql.Tx) error {
	log.Info("computing class-specific player rankings per season")

	// get all distinct season numbers
	seasonRows, err := tx.Query("SELECT DISTINCT season_number FROM seasons ORDER BY season_number")
	if err != nil {
		return fmt.Errorf("failed to query seasons: %w", err)
	}
	defer seasonRows.Close()

	var seasons []int
	for seasonRows.Next() {
		var seasonNumber int
		if err := seasonRows.Scan(&seasonNumber); err != nil {
			return err
		}
		seasons = append(seasons, seasonNumber)
	}

	if len(seasons) == 0 {
		log.Warn("no seasons found, skipping class rankings")
		return nil
	}

	for _, seasonID := range seasons {
		log.Info("processing class rankings for season", "season_id", seasonID)

		// step 1: global class rankings
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET global_class_rank = (
				SELECT ranking FROM (
					SELECT
						player_id,
						ROW_NUMBER() OVER (PARTITION BY class_name ORDER BY combined_best_time ASC) as ranking
					FROM player_profiles
					WHERE season_id = ? AND has_complete_coverage = 1 AND class_name IS NOT NULL
				) class_ranks
				WHERE class_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1 AND class_name IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute global class rankings: %w", err)
		}

		// step 2: global class ranking brackets
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET global_class_bracket = (
				CASE
					WHEN counts.combined_best_time = counts.class_min_time THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.global_class_rank AS REAL) / CAST(counts.class_total AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.global_class_rank AS REAL) / CAST(counts.class_total AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.global_class_rank AS REAL) / CAST(counts.class_total AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.global_class_rank AS REAL) / CAST(counts.class_total AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.global_class_rank AS REAL) / CAST(counts.class_total AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					pp.player_id,
					pp.global_class_rank,
					pp.combined_best_time,
					MIN(pp.combined_best_time) OVER (PARTITION BY pp.class_name) as class_min_time,
					COUNT(*) OVER (PARTITION BY pp.class_name) as class_total
				FROM player_profiles pp
				WHERE pp.season_id = ? AND pp.has_complete_coverage = 1
					AND pp.global_class_rank IS NOT NULL AND pp.class_name IS NOT NULL
			) counts
			WHERE player_profiles.player_id = counts.player_id
				AND player_profiles.season_id = ?
				AND player_profiles.has_complete_coverage = 1
				AND player_profiles.global_class_rank IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute global class brackets: %w", err)
		}

		// step 3: regional class rankings
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET region_class_rank = (
				SELECT ranking FROM (
					SELECT
						pp.player_id,
						ROW_NUMBER() OVER (PARTITION BY r.region, pp.class_name ORDER BY pp.combined_best_time ASC) as ranking
					FROM player_profiles pp
					INNER JOIN realms r ON pp.realm_id = r.id
					WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.class_name IS NOT NULL
				) class_ranks
				WHERE class_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1 AND class_name IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute regional class rankings: %w", err)
		}

		// step 4: regional class ranking brackets
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET region_class_bracket = (
				CASE
					WHEN counts.combined_best_time = counts.class_regional_min_time THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.region_class_rank AS REAL) / CAST(counts.class_regional_total AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.region_class_rank AS REAL) / CAST(counts.class_regional_total AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.region_class_rank AS REAL) / CAST(counts.class_regional_total AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.region_class_rank AS REAL) / CAST(counts.class_regional_total AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.region_class_rank AS REAL) / CAST(counts.class_regional_total AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					pp.player_id,
					pp.region_class_rank,
					pp.combined_best_time,
					MIN(pp.combined_best_time) OVER (PARTITION BY r.region, pp.class_name) as class_regional_min_time,
					COUNT(*) OVER (PARTITION BY r.region, pp.class_name) as class_regional_total
				FROM player_profiles pp
				INNER JOIN realms r ON pp.realm_id = r.id
				WHERE pp.season_id = ? AND pp.has_complete_coverage = 1
					AND pp.region_class_rank IS NOT NULL AND pp.class_name IS NOT NULL
			) counts
			WHERE player_profiles.player_id = counts.player_id
				AND player_profiles.season_id = ?
				AND player_profiles.has_complete_coverage = 1
				AND player_profiles.region_class_rank IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute regional class brackets: %w", err)
		}

		// step 5: realm class rankings (pool-based for connected realms)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET realm_class_rank = (
				SELECT ranking FROM (
					SELECT
						pp.player_id,
						ROW_NUMBER() OVER (
							PARTITION BY COALESCE(parent_r.id, r.id), pp.class_name
							ORDER BY pp.combined_best_time ASC
						) as ranking
					FROM player_profiles pp
					JOIN realms r ON pp.realm_id = r.id
					LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
					WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.class_name IS NOT NULL
				) class_ranks
				WHERE class_ranks.player_id = player_profiles.player_id
			)
			WHERE season_id = ? AND has_complete_coverage = 1 AND class_name IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute realm class rankings: %w", err)
		}

		// step 6: realm class ranking brackets (pool-based)
		_, err = tx.Exec(`
			UPDATE player_profiles
			SET realm_class_bracket = (
				CASE
					WHEN counts.combined_best_time = counts.class_pool_min_time THEN 'artifact'
					ELSE
						CASE
							WHEN (CAST(counts.realm_class_rank AS REAL) / CAST(counts.class_pool_total AS REAL) * 100) <= 1.0 THEN 'excellent'
							WHEN (CAST(counts.realm_class_rank AS REAL) / CAST(counts.class_pool_total AS REAL) * 100) <= 5.0 THEN 'legendary'
							WHEN (CAST(counts.realm_class_rank AS REAL) / CAST(counts.class_pool_total AS REAL) * 100) <= 20.0 THEN 'epic'
							WHEN (CAST(counts.realm_class_rank AS REAL) / CAST(counts.class_pool_total AS REAL) * 100) <= 40.0 THEN 'rare'
							WHEN (CAST(counts.realm_class_rank AS REAL) / CAST(counts.class_pool_total AS REAL) * 100) <= 60.0 THEN 'uncommon'
							ELSE 'common'
						END
				END
			)
			FROM (
				SELECT
					pp.player_id,
					pp.realm_class_rank,
					pp.combined_best_time,
					COALESCE(parent_r.id, r.id) as pool_id,
					MIN(pp.combined_best_time) OVER (PARTITION BY COALESCE(parent_r.id, r.id), pp.class_name) as class_pool_min_time,
					COUNT(*) OVER (PARTITION BY COALESCE(parent_r.id, r.id), pp.class_name) as class_pool_total
				FROM player_profiles pp
				JOIN realms r ON pp.realm_id = r.id
				LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
				WHERE pp.season_id = ? AND pp.has_complete_coverage = 1
					AND pp.realm_class_rank IS NOT NULL AND pp.class_name IS NOT NULL
			) counts
			WHERE player_profiles.player_id = counts.player_id
				AND player_profiles.season_id = ?
				AND player_profiles.has_complete_coverage = 1
				AND player_profiles.realm_class_rank IS NOT NULL
		`, seasonID, seasonID)
		if err != nil {
			return fmt.Errorf("failed to compute realm class brackets: %w", err)
		}

		log.Info("computed class rankings for season", "season_id", seasonID)
	}

	log.Info("computed class rankings for all seasons")
	return nil
}

// computeGlobalRankings computes global rankings for all runs (per season)
func computeGlobalRankings(tx *sql.Tx) error {
	log.Info("computing global rankings per season")

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
			ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id, cr.season_id ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
			cr.season_id as season_id,
			? as computed_at
		FROM challenge_runs cr
	`, currentTime)
	if err != nil {
		return err
	}

	// update percentile brackets for unfiltered global rankings using efficient SQL (per season)
	log.Info("computing global ranking brackets per season")
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
		SELECT DISTINCT cr.season_id as season_id
		FROM challenge_runs cr
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
					WHERE cr.dungeon_id = ? AND cr.season_id = ?
					GROUP BY cr.team_signature
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
					WHERE cr.dungeon_id = ? AND cr.season_id = ?
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
	log.Info("computing filtered global ranking brackets per season")
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

	log.Info("computed global rankings with percentile brackets (all and filtered)")
	return nil
}

// computeRegionalRankings computes regional rankings for all runs
func computeRegionalRankings(tx *sql.Tx) error {
	log.Info("computing regional rankings")

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
				ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id, cr.season_id ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
				cr.season_id as season_id,
				? as computed_at
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			WHERE r.region = ?
		`, region, currentTime, region)

		if err != nil {
			return err
		}

		// update percentile brackets for unfiltered regional rankings using efficient SQL (per season)
		log.Info("computing unfiltered regional ranking brackets per season", "region", region)
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
			SELECT DISTINCT cr.season_id as season_id
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
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
						WHERE cr.dungeon_id = ? AND r.region = ? AND cr.season_id = ?
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
						WHERE cr.dungeon_id = ? AND r.region = ? AND cr.season_id = ?
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
		log.Info("computing filtered regional ranking brackets per season", "region", region)
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

	log.Info("computed regional rankings with percentile brackets", "regions", len(regions))
	return nil
}

// computeRealmRankings computes realm rankings per realm pool (connected realms grouped together)
func computeRealmRankings(tx *sql.Tx) error {
	log.Info("computing realm rankings (pool-based for connected realms)")

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

	log.Info("found realm pools to process", "pools", len(pools))

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
					PARTITION BY cr.dungeon_id, cr.season_id
					ORDER BY cr.duration ASC, cr.completed_timestamp ASC
				) as ranking,
				cr.season_id as season_id,
				? as computed_at
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
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
			SELECT DISTINCT cr.season_id as season_id
			FROM challenge_runs cr
			INNER JOIN realms r ON cr.realm_id = r.id
			LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
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
						WHERE cr.dungeon_id = ?
							AND r.region = ?
							AND COALESCE(parent_r.slug, r.slug) = ?
							AND cr.season_id = ?
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
						INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
														AND cr.duration = btr.best_duration
						WHERE cr.dungeon_id = ?
							AND r.region = ?
							AND COALESCE(parent_r.slug, r.slug) = ?
							AND cr.season_id = ?
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

	log.Info("computed realm rankings with percentile brackets", "pools", len(pools))
	return nil
}
