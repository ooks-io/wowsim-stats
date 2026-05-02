package generator

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ookstats/internal/wow"
	"ookstats/internal/writer"
)

// HomeJSON is the top-level structure for web/public/api/home.json.
// Layout: a cross-season recent-top-runs feed, plus per-season blocks
// containing top-3 dungeon records and top-10 player lists per scope.
type HomeJSON struct {
	GeneratedAt   int64                     `json:"generated_at"`
	RecentTopRuns []HomeRunEntry            `json:"recent_top_runs"`
	Seasons       map[string]HomeSeasonJSON `json:"seasons"`
}

// HomeSeasonJSON holds per-season top-3 runs and top-10 player lists.
type HomeSeasonJSON struct {
	SeasonID          int                       `json:"season_id"`
	SeasonName        string                    `json:"season_name,omitempty"`
	TopRunsPerDungeon map[string][]HomeRunEntry `json:"top_runs_per_dungeon"`
	TopPlayers        HomePlayerLists           `json:"top_players"`
}

// HomePlayerLists groups the top-10 player lists by scope.
type HomePlayerLists struct {
	Global []HomePlayerEntry `json:"global"`
	US     []HomePlayerEntry `json:"us"`
	EU     []HomePlayerEntry `json:"eu"`
	KR     []HomePlayerEntry `json:"kr"`
	TW     []HomePlayerEntry `json:"tw"`
}

// HomeRunEntry is one top run, used both for per-dungeon top-3 and the recent feed.
type HomeRunEntry struct {
	Rank               int              `json:"rank,omitempty"`
	Bracket            string           `json:"bracket,omitempty"`
	RunID              int64            `json:"run_id"`
	DungeonID          int              `json:"dungeon_id"`
	DungeonName        string           `json:"dungeon_name"`
	DungeonSlug        string           `json:"dungeon_slug"`
	DurationMs         int64            `json:"duration_ms"`
	CompletedTimestamp int64            `json:"completed_timestamp"`
	SeasonID           int              `json:"season_id,omitempty"`
	Rankings           *HomeRunRankings `json:"rankings,omitempty"`
	TeamMembers        []HomeRunMember  `json:"team_members"`
}

// HomeRunRankings carries the rank-bucket(s) a run qualified for.
// Only populated on entries in RecentTopRuns. Currently global-only by design.
type HomeRunRankings struct {
	Global int `json:"global"`
}

// HomeRunMember is one of the 5 players in a run.
// Field shapes mirror the canonical web/src/lib/types.ts TeamMember (id, name, spec_id, etc.)
// so the TeamComposition component can consume these without adapter mapping.
type HomeRunMember struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	RealmSlug string `json:"realm_slug"`
	Region    string `json:"region"`
	ClassName string `json:"class_name,omitempty"`
	SpecID    int    `json:"spec_id,omitempty"`
	SpecName  string `json:"spec_name,omitempty"`
	Faction   string `json:"faction,omitempty"`
}

// HomePlayerEntry is one player in a top-10 list.
type HomePlayerEntry struct {
	Rank                   int    `json:"rank"`
	PlayerID               int64  `json:"player_id"`
	Name                   string `json:"name"`
	RealmSlug              string `json:"realm_slug"`
	RealmName              string `json:"realm_name,omitempty"`
	Region                 string `json:"region"`
	ClassName              string `json:"class_name,omitempty"`
	ActiveSpecID           int    `json:"active_spec_id,omitempty"`
	ActiveSpecName         string `json:"active_spec_name,omitempty"`
	CombinedBestTimeMs     int64  `json:"combined_best_time_ms"`
	GlobalRanking          int    `json:"global_ranking,omitempty"`
	GlobalRankingBracket   string `json:"global_ranking_bracket,omitempty"`
	RegionalRanking        int    `json:"regional_ranking,omitempty"`
	RegionalRankingBracket string `json:"regional_ranking_bracket,omitempty"`
	AvatarURL              string `json:"avatar_url,omitempty"`
}

// Seasons we surface on the home page. Mirrors run_rankings.season_id values.
var homeSeasons = []int{1, 2}

// regions for the regional top-player lists
var homeRegions = []string{"us", "eu", "kr", "tw"}

// GenerateHome writes outDir/api/home.json with the home page payload.
func GenerateHome(db *sql.DB, outDir string) error {
	home := HomeJSON{
		GeneratedAt: time.Now().UnixMilli(),
		Seasons:     make(map[string]HomeSeasonJSON),
	}

	for _, seasonID := range homeSeasons {
		topRuns, err := loadTopRunsPerDungeon(db, seasonID, 3)
		if err != nil {
			return fmt.Errorf("top runs for season %d: %w", seasonID, err)
		}

		topPlayers, err := loadTopPlayers(db, seasonID, 10)
		if err != nil {
			return fmt.Errorf("top players for season %d: %w", seasonID, err)
		}

		seasonName, err := loadSeasonName(db, seasonID)
		if err != nil {
			return fmt.Errorf("season name for %d: %w", seasonID, err)
		}

		home.Seasons[fmt.Sprintf("%d", seasonID)] = HomeSeasonJSON{
			SeasonID:          seasonID,
			SeasonName:        seasonName,
			TopRunsPerDungeon: topRuns,
			TopPlayers:        topPlayers,
		}
	}

	recent, err := loadRecentTopGlobalRuns(db, 10)
	if err != nil {
		return fmt.Errorf("recent top global runs: %w", err)
	}
	home.RecentTopRuns = recent

	outPath := filepath.Join(outDir, "api", "home.json")
	if err := writer.WriteJSONFileCompact(outPath, home); err != nil {
		return fmt.Errorf("write home.json: %w", err)
	}
	return nil
}

// loadTopRunsPerDungeon returns up to topN team-filtered global runs per dungeon for the season,
// keyed by dungeon slug.
func loadTopRunsPerDungeon(db *sql.DB, seasonID, topN int) (map[string][]HomeRunEntry, error) {
	rows, err := db.Query(`
		SELECT
			rr.run_id, rr.ranking, rr.percentile_bracket,
			cr.dungeon_id, cr.duration, cr.completed_timestamp,
			d.slug, d.name
		FROM run_rankings rr
		JOIN challenge_runs cr ON rr.run_id = cr.id
		JOIN dungeons d ON cr.dungeon_id = d.id
		WHERE rr.ranking_type = 'global'
		  AND rr.ranking_scope = 'filtered'
		  AND rr.season_id = ?
		  AND rr.ranking <= ?
		ORDER BY cr.dungeon_id, rr.ranking
	`, seasonID, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]HomeRunEntry)
	var runIDs []int64
	for rows.Next() {
		var e HomeRunEntry
		var bracket sql.NullString
		if err := rows.Scan(&e.RunID, &e.Rank, &bracket, &e.DungeonID, &e.DurationMs, &e.CompletedTimestamp, &e.DungeonSlug, &e.DungeonName); err != nil {
			return nil, err
		}
		if bracket.Valid {
			e.Bracket = bracket.String
		}
		out[e.DungeonSlug] = append(out[e.DungeonSlug], e)
		runIDs = append(runIDs, e.RunID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	members, err := loadTeamMembersForRuns(db, runIDs)
	if err != nil {
		return nil, err
	}
	for slug, entries := range out {
		for i := range entries {
			entries[i].TeamMembers = members[entries[i].RunID]
		}
		out[slug] = entries
	}
	return out, nil
}

// loadRecentTopGlobalRuns returns up to limit recent runs that hit top-10 globally
// (team-filtered) across either season, ordered newest-first by completion time.
func loadRecentTopGlobalRuns(db *sql.DB, limit int) ([]HomeRunEntry, error) {
	rows, err := db.Query(`
		SELECT
			rr.run_id, rr.ranking, rr.percentile_bracket, rr.season_id,
			cr.dungeon_id, cr.duration, cr.completed_timestamp,
			d.slug, d.name
		FROM run_rankings rr
		JOIN challenge_runs cr ON rr.run_id = cr.id
		JOIN dungeons d ON cr.dungeon_id = d.id
		WHERE rr.ranking_type = 'global'
		  AND rr.ranking_scope = 'filtered'
		  AND rr.ranking <= 10
		ORDER BY cr.completed_timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HomeRunEntry
	var runIDs []int64
	for rows.Next() {
		var e HomeRunEntry
		var globalRank int
		var bracket sql.NullString
		if err := rows.Scan(&e.RunID, &globalRank, &bracket, &e.SeasonID, &e.DungeonID, &e.DurationMs, &e.CompletedTimestamp, &e.DungeonSlug, &e.DungeonName); err != nil {
			return nil, err
		}
		e.Rankings = &HomeRunRankings{Global: globalRank}
		if bracket.Valid {
			e.Bracket = bracket.String
		}
		out = append(out, e)
		runIDs = append(runIDs, e.RunID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	members, err := loadTeamMembersForRuns(db, runIDs)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].TeamMembers = members[out[i].RunID]
	}
	return out, nil
}

// loadTeamMembersForRuns fetches member metadata for the given run IDs in one batched query,
// returning a map keyed by run_id.
func loadTeamMembersForRuns(db *sql.DB, runIDs []int64) (map[int64][]HomeRunMember, error) {
	out := make(map[int64][]HomeRunMember)
	if len(runIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(runIDs))
	args := make([]any, len(runIDs))
	for i, id := range runIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	q := fmt.Sprintf(`
		SELECT
			rm.run_id, rm.spec_id, rm.faction,
			p.id, p.name,
			r.slug, r.region,
			pd.class_name
		FROM run_members rm
		JOIN players p ON rm.player_id = p.id
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE rm.run_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var runID int64
		var specID sql.NullInt64
		var faction sql.NullString
		var className sql.NullString
		var m HomeRunMember
		if err := rows.Scan(&runID, &specID, &faction, &m.ID, &m.Name, &m.RealmSlug, &m.Region, &className); err != nil {
			return nil, err
		}
		if specID.Valid {
			m.SpecID = int(specID.Int64)
		}
		if faction.Valid {
			m.Faction = faction.String
		}
		if className.Valid {
			m.ClassName = className.String
		}
		// Backfill class+spec name from spec_id if either is missing
		if m.SpecID > 0 {
			specPtr := m.SpecID
			cls, spec := wow.FallbackClassAndSpec(m.ClassName, m.SpecName, &specPtr)
			m.ClassName, m.SpecName = cls, spec
		}
		out[runID] = append(out[runID], m)
	}
	return out, rows.Err()
}

// loadTopPlayers returns the global top-N plus per-region top-N player lists for the season.
func loadTopPlayers(db *sql.DB, seasonID, topN int) (HomePlayerLists, error) {
	var lists HomePlayerLists
	global, err := loadTopGlobalPlayers(db, seasonID, topN)
	if err != nil {
		return lists, fmt.Errorf("global: %w", err)
	}
	lists.Global = global

	for _, region := range homeRegions {
		regional, err := loadTopRegionalPlayers(db, seasonID, region, topN)
		if err != nil {
			return lists, fmt.Errorf("region %s: %w", region, err)
		}
		switch region {
		case "us":
			lists.US = regional
		case "eu":
			lists.EU = regional
		case "kr":
			lists.KR = regional
		case "tw":
			lists.TW = regional
		}
	}
	return lists, nil
}

func loadTopGlobalPlayers(db *sql.DB, seasonID, topN int) ([]HomePlayerEntry, error) {
	return queryTopPlayers(db, `
		SELECT
			pp.player_id, pp.name, pp.class_name, pp.main_spec_id,
			pp.global_ranking, pp.global_ranking_bracket,
			pp.regional_ranking, pp.regional_ranking_bracket,
			pp.combined_best_time,
			r.slug, r.name, r.region,
			pd.avatar_url
		FROM player_profiles pp
		JOIN realms r ON pp.realm_id = r.id
		LEFT JOIN player_details pd ON pp.player_id = pd.player_id
		WHERE pp.season_id = ?
		  AND pp.has_complete_coverage = 1
		  AND pp.global_ranking IS NOT NULL
		  AND pp.global_ranking <= ?
		ORDER BY pp.global_ranking
	`, seasonID, topN)
}

func loadTopRegionalPlayers(db *sql.DB, seasonID int, region string, topN int) ([]HomePlayerEntry, error) {
	return queryTopPlayers(db, `
		SELECT
			pp.player_id, pp.name, pp.class_name, pp.main_spec_id,
			pp.global_ranking, pp.global_ranking_bracket,
			pp.regional_ranking, pp.regional_ranking_bracket,
			pp.combined_best_time,
			r.slug, r.name, r.region,
			pd.avatar_url
		FROM player_profiles pp
		JOIN realms r ON pp.realm_id = r.id
		LEFT JOIN player_details pd ON pp.player_id = pd.player_id
		WHERE pp.season_id = ?
		  AND pp.has_complete_coverage = 1
		  AND r.region = ?
		  AND pp.regional_ranking IS NOT NULL
		  AND pp.regional_ranking <= ?
		ORDER BY pp.regional_ranking
	`, seasonID, region, topN)
}

// queryTopPlayers executes a player-list query whose columns match the SELECT list above
// and returns enriched HomePlayerEntry rows. Last positional arg is interpreted as the rank
// to populate Rank from (global_ranking for the global list, regional_ranking for regional).
func queryTopPlayers(db *sql.DB, q string, args ...any) ([]HomePlayerEntry, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HomePlayerEntry
	for rows.Next() {
		var (
			e              HomePlayerEntry
			className      sql.NullString
			mainSpecID     sql.NullInt64
			globalRank     sql.NullInt64
			globalBracket  sql.NullString
			regionalRank   sql.NullInt64
			regionalBracket sql.NullString
			combinedBest   sql.NullInt64
			realmName      sql.NullString
			avatarURL      sql.NullString
		)
		if err := rows.Scan(
			&e.PlayerID, &e.Name, &className, &mainSpecID,
			&globalRank, &globalBracket,
			&regionalRank, &regionalBracket,
			&combinedBest,
			&e.RealmSlug, &realmName, &e.Region,
			&avatarURL,
		); err != nil {
			return nil, err
		}
		if className.Valid {
			e.ClassName = className.String
		}
		if mainSpecID.Valid {
			e.ActiveSpecID = int(mainSpecID.Int64)
		}
		if globalRank.Valid {
			e.GlobalRanking = int(globalRank.Int64)
		}
		if globalBracket.Valid {
			e.GlobalRankingBracket = globalBracket.String
		}
		if regionalRank.Valid {
			e.RegionalRanking = int(regionalRank.Int64)
		}
		if regionalBracket.Valid {
			e.RegionalRankingBracket = regionalBracket.String
		}
		if combinedBest.Valid {
			e.CombinedBestTimeMs = combinedBest.Int64
		}
		if realmName.Valid {
			e.RealmName = realmName.String
		}
		if avatarURL.Valid {
			e.AvatarURL = avatarURL.String
		}
		// Resolve spec name; class_name comes from player_profiles directly
		if e.ActiveSpecID > 0 {
			specPtr := e.ActiveSpecID
			cls, spec := wow.FallbackClassAndSpec(e.ClassName, "", &specPtr)
			e.ClassName, e.ActiveSpecName = cls, spec
		}
		// Rank for the entry: regional ranking takes precedence (the per-region lists
		// query for it); the global list will populate from GlobalRanking via the
		// fallback below.
		if e.RegionalRanking > 0 && isRegionalQuery(q) {
			e.Rank = e.RegionalRanking
		} else {
			e.Rank = e.GlobalRanking
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// isRegionalQuery is a tiny heuristic on the SELECT/ORDER text so queryTopPlayers can
// decide which rank value to surface as `rank`. Cheap and safe — keeps queryTopPlayers
// reusable for both global and regional list queries.
func isRegionalQuery(q string) bool {
	return strings.Contains(q, "ORDER BY pp.regional_ranking")
}

// loadSeasonName picks a representative season_name for the season number.
// Seasons are stored per-region; names tend to match across regions, so any non-empty
// name for that season number is fine.
func loadSeasonName(db *sql.DB, seasonNumber int) (string, error) {
	var name sql.NullString
	err := db.QueryRow(`
		SELECT season_name FROM seasons
		WHERE season_number = ? AND season_name IS NOT NULL AND season_name != ''
		LIMIT 1
	`, seasonNumber).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if name.Valid {
		return name.String, nil
	}
	return "", nil
}
