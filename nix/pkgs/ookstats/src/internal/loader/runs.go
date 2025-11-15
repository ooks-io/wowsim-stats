package loader

import (
	"database/sql"
	"fmt"
	"strings"
)

// BestRunData represents a player's best run for a specific dungeon in a specific season
type BestRunData struct {
	DungeonID               int64
	DungeonName             string
	DungeonSlug             string
	RunID                   int64
	Duration                int64
	CompletedTimestamp      int64
	SeasonID                int
	GlobalRankingFiltered   sql.NullInt64
	RegionalRankingFiltered sql.NullInt64
	RealmRankingFiltered    sql.NullInt64
	GlobalBracket           string
	RegionalBracket         string
	RealmBracket            string
}

// TeamMemberData represents a member of a run team
type TeamMemberData struct {
	RunID     int64
	Name      string
	SpecID    sql.NullInt64
	Region    string
	RealmSlug string
}

// LoadAllBestRuns loads best runs for a set of players
// Returns: map[playerID][]BestRunData, all unique run IDs, error
func LoadAllBestRuns(db *sql.DB, playerIDs []int64) (map[int64][]BestRunData, []int64, error) {
	if len(playerIDs) == 0 {
		return make(map[int64][]BestRunData), []int64{}, nil
	}

	// Build IN clause
	placeholders := make([]string, len(playerIDs))
	args := make([]any, len(playerIDs))
	for i, id := range playerIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
        SELECT pbr.player_id, pbr.dungeon_id, d.name, d.slug, pbr.run_id, pbr.duration, pbr.completed_timestamp,
               pbr.season_id,
               rr_global_filtered.ranking as global_ranking_filtered,
               rr_regional_filtered.ranking as regional_ranking_filtered,
               rr_realm_filtered.ranking as realm_ranking_filtered,
               COALESCE(rr_global_filtered.percentile_bracket, '') as global_percentile_bracket,
               COALESCE(rr_regional_filtered.percentile_bracket, '') as regional_percentile_bracket,
               COALESCE(rr_realm_filtered.percentile_bracket, '') as realm_percentile_bracket
        FROM player_best_runs pbr
        JOIN dungeons d ON pbr.dungeon_id = d.id
        JOIN players p ON pbr.player_id = p.id
        JOIN realms r ON p.realm_id = r.id
        LEFT JOIN run_rankings rr_global_filtered ON pbr.run_id = rr_global_filtered.run_id
            AND rr_global_filtered.ranking_type = 'global' AND rr_global_filtered.ranking_scope = 'filtered'
            AND rr_global_filtered.season_id = pbr.season_id
        LEFT JOIN run_rankings rr_regional_filtered ON pbr.run_id = rr_regional_filtered.run_id
            AND rr_regional_filtered.ranking_type = 'regional' AND rr_regional_filtered.ranking_scope = r.region || '_filtered'
            AND rr_regional_filtered.season_id = pbr.season_id
        LEFT JOIN run_rankings rr_realm_filtered ON pbr.run_id = rr_realm_filtered.run_id
            AND rr_realm_filtered.ranking_type = 'realm' AND rr_realm_filtered.ranking_scope = 'filtered'
            AND rr_realm_filtered.season_id = pbr.season_id
        WHERE pbr.player_id IN (%s)
        ORDER BY pbr.player_id, pbr.season_id, d.name
    `, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	bestRunsMap := make(map[int64][]BestRunData)
	runIDSet := make(map[int64]bool)
	var allRunIDs []int64

	for rows.Next() {
		var playerID int64
		var run BestRunData
		if err := rows.Scan(
			&playerID, &run.DungeonID, &run.DungeonName, &run.DungeonSlug, &run.RunID, &run.Duration, &run.CompletedTimestamp,
			&run.SeasonID,
			&run.GlobalRankingFiltered, &run.RegionalRankingFiltered, &run.RealmRankingFiltered,
			&run.GlobalBracket, &run.RegionalBracket, &run.RealmBracket); err != nil {
			return nil, nil, fmt.Errorf("scan best run: %w", err)
		}
		bestRunsMap[playerID] = append(bestRunsMap[playerID], run)

		// Deduplicate run IDs
		if !runIDSet[run.RunID] {
			runIDSet[run.RunID] = true
			allRunIDs = append(allRunIDs, run.RunID)
		}
	}
	return bestRunsMap, allRunIDs, nil
}

// LoadAllTeamMembers loads team members for a set of runs
// Returns: map[runID][]TeamMemberData
func LoadAllTeamMembers(db *sql.DB, runIDs []int64) (map[int64][]TeamMemberData, error) {
	if len(runIDs) == 0 {
		return make(map[int64][]TeamMemberData), nil
	}

	teamMembersMap := make(map[int64][]TeamMemberData)

	// Process in batches to avoid SQL limits (max ~32k parameters)
	const batchSize = 10000
	for i := 0; i < len(runIDs); i += batchSize {
		end := i + batchSize
		if end > len(runIDs) {
			end = len(runIDs)
		}

		batch := runIDs[i:end]

		// Build IN clause for this batch
		placeholders := make([]string, len(batch))
		args := make([]any, len(batch))
		for j, id := range batch {
			placeholders[j] = "?"
			args[j] = id
		}

		query := fmt.Sprintf(`
            SELECT rm.run_id, p.name, rm.spec_id, r.region, r.slug
            FROM run_members rm
            JOIN players p ON rm.player_id = p.id
            JOIN realms r ON p.realm_id = r.id
            WHERE rm.run_id IN (%s)
            ORDER BY rm.run_id, p.name
        `, strings.Join(placeholders, ","))

		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i/batchSize+1, err)
		}

		for rows.Next() {
			var member TeamMemberData
			if err := rows.Scan(
				&member.RunID, &member.Name, &member.SpecID, &member.Region, &member.RealmSlug); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan team member: %w", err)
			}
			teamMembersMap[member.RunID] = append(teamMembersMap[member.RunID], member)
		}
		rows.Close()
	}

	return teamMembersMap, nil
}

// LeaderboardMember represents a member in a leaderboard row
type LeaderboardMember struct {
	Name      string
	SpecID    *int
	Region    string
	RealmSlug string
}

// LeaderboardRow represents a canonical run for leaderboards
type LeaderboardRow struct {
	ID                 int64
	Duration           int64
	CompletedTimestamp int64
	KeystoneLevel      int
	DungeonName        string
	RealmName          string
	Region             string
	RankingPercentile  string // percentile bracket based on scope
	Members            []LeaderboardMember
}

// LoadCanonicalRuns returns one canonical run per team_signature, ordered, with members
func LoadCanonicalRuns(db *sql.DB, dungeonID int, region string, realmSlug string, seasonID, limit, offset int) ([]LeaderboardRow, error) {
	// Use window function to rank runs per team_signature, picking best per team
	where := "WHERE cr.dungeon_id = ?"
	args := []any{dungeonID}
	if region != "" {
		where += " AND r.region = ?"
		args = append(args, region)
	}
	if realmSlug != "" {
		where += " AND r.slug = ?"
		args = append(args, realmSlug)
	}
	// Filter by season
	where += " AND COALESCE(ps.season_id, 1) = ?"
	args = append(args, seasonID)

	q := fmt.Sprintf(`
      WITH ranked AS (
        SELECT cr.id, cr.duration, cr.completed_timestamp,
               ROW_NUMBER() OVER (PARTITION BY cr.team_signature ORDER BY cr.duration ASC, cr.completed_timestamp ASC, cr.id ASC) AS rn
        FROM challenge_runs cr
        JOIN realms r ON cr.realm_id = r.id
        LEFT JOIN period_seasons ps ON cr.period_id = ps.period_id
        %s
      )
      SELECT id FROM ranked WHERE rn = 1
      ORDER BY duration ASC, completed_timestamp ASC, id ASC
      LIMIT %d OFFSET %d
    `, where, limit, offset)

	idRows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for idRows.Next() {
		var id int64
		if err := idRows.Scan(&id); err != nil {
			idRows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	idRows.Close()
	if len(ids) == 0 {
		return []LeaderboardRow{}, nil
	}

	// Determine ranking scope for percentile bracket
	var rankingType, rankingScope string
	if region == "" {
		// Global leaderboard
		rankingType = "global"
		rankingScope = "filtered"
	} else if realmSlug == "" {
		// Regional leaderboard
		rankingType = "regional"
		rankingScope = region + "_filtered"
	} else {
		// Realm leaderboard
		rankingType = "realm"
		rankingScope = "filtered"
	}

	// Load rows
	placeholders := make([]string, len(ids))
	iargs := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		iargs[i] = id
	}

	rQuery := fmt.Sprintf(`
      SELECT cr.id, cr.duration, cr.completed_timestamp, cr.keystone_level,
             d.name, rr.name, rr.region,
             COALESCE(run_rankings.percentile_bracket, '') as percentile_bracket
      FROM challenge_runs cr
      JOIN dungeons d ON cr.dungeon_id = d.id
      JOIN realms rr ON cr.realm_id = rr.id
      LEFT JOIN run_rankings ON cr.id = run_rankings.run_id
        AND run_rankings.ranking_type = ?
        AND run_rankings.ranking_scope = ?
        AND run_rankings.season_id = ?
      WHERE cr.id IN (%s)
    `, strings.Join(placeholders, ","))
	// Prepend ranking parameters
	iargs = append([]any{rankingType, rankingScope, seasonID}, iargs...)

	rrows, err := db.Query(rQuery, iargs...)
	if err != nil {
		return nil, err
	}
	byID := map[int64]LeaderboardRow{}
	for rrows.Next() {
		var row LeaderboardRow
		if err := rrows.Scan(&row.ID, &row.Duration, &row.CompletedTimestamp, &row.KeystoneLevel, &row.DungeonName, &row.RealmName, &row.Region, &row.RankingPercentile); err != nil {
			rrows.Close()
			return nil, err
		}
		byID[row.ID] = row
	}
	rrows.Close()

	// Members - use original ids array, not modified iargs
	mQuery := fmt.Sprintf(`
      SELECT rm.run_id, p.name, rm.spec_id, rr.region, rr.slug
      FROM run_members rm
      JOIN players p ON rm.player_id = p.id
      JOIN realms rr ON p.realm_id = rr.id
      WHERE rm.run_id IN (%s)
      ORDER BY rm.run_id, p.name
    `, strings.Join(placeholders, ","))
	mArgs := make([]any, len(ids))
	for i, id := range ids {
		mArgs[i] = id
	}
	mrows, err := db.Query(mQuery, mArgs...)
	if err != nil {
		return nil, err
	}
	for mrows.Next() {
		var runID int64
		var name, region, rslug string
		var spec sql.NullInt64
		if err := mrows.Scan(&runID, &name, &spec, &region, &rslug); err != nil {
			mrows.Close()
			return nil, err
		}
		row := byID[runID]
		var specPtr *int
		if spec.Valid {
			v := int(spec.Int64)
			specPtr = &v
		}
		row.Members = append(row.Members, LeaderboardMember{Name: name, SpecID: specPtr, Region: region, RealmSlug: rslug})
		byID[runID] = row
	}
	mrows.Close()

	// Order back as ids
	out := make([]LeaderboardRow, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out, nil
}
