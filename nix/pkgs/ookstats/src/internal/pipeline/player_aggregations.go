package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/wow"
)

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
		INNER JOIN realms r ON cr.realm_id = r.id
		LEFT JOIN realms parent_r ON r.parent_realm_slug = parent_r.slug AND r.region = parent_r.region
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
			AND rr_lf.ranking_type = 'realm' AND rr_lf.ranking_scope = COALESCE(parent_r.slug, r.slug) || '_filtered'
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
