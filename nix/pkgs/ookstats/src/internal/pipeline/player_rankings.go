package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
)

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
					MIN(pp.combined_best_time) OVER (PARTITION BY pp.season_id, COALESCE(parent_r.id, r.id)) as pool_min_time,
					COUNT(*) OVER (PARTITION BY pp.season_id, COALESCE(parent_r.id, r.id)) as pool_total
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
