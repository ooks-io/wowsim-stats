package pipeline

import (
	"database/sql"
	"time"

	"github.com/charmbracelet/log"
)

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
