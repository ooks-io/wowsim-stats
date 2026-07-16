package generator

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ookstats/internal/loader"
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

// regions for the regional top-player lists
var homeRegions = []string{"us", "eu", "kr", "tw"}

// GenerateHome writes outDir/api/home.json with the home page payload.
func GenerateHome(db *sql.DB, outDir string) error {
	homeSeasons, err := loadHomeSeasons(db)
	if err != nil {
		return fmt.Errorf("load home seasons: %w", err)
	}

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

func loadHomeSeasons(db *sql.DB) ([]int, error) {
	rows, err := db.Query(`SELECT DISTINCT season_number FROM seasons ORDER BY season_number ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
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

// loadTopPlayers returns the global top-N plus per-region top-N lists for the season,
// ranked over accounts like the players leaderboard pages, shown via each account's main.
func loadTopPlayers(db *sql.DB, seasonID, topN int) (HomePlayerLists, error) {
	var lists HomePlayerLists
	aggs, err := loader.LoadAccountAggregates(db, seasonID)
	if err != nil {
		return lists, err
	}

	type accountRow struct {
		accountID int64
		combined  int64
		main      *loader.AccountChar
		region    string
	}
	eligible := make([]accountRow, 0, len(aggs))
	for _, agg := range aggs {
		if len(agg.BestPerDungeon) < loader.DungeonsForFullCoverage {
			continue
		}
		main := loader.PickMainCharacter(agg)
		if main == nil {
			continue
		}
		var sum int64
		for _, br := range agg.BestPerDungeon {
			sum += br.Duration
		}
		eligible = append(eligible, accountRow{accountID: agg.AccountID, combined: sum, main: main, region: agg.Region})
	}
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].combined != eligible[j].combined {
			return eligible[i].combined < eligible[j].combined
		}
		return eligible[i].accountID < eligible[j].accountID
	})

	toEntry := func(row accountRow, rank, total int, regional bool) HomePlayerEntry {
		e := HomePlayerEntry{
			Rank:               rank,
			PlayerID:           row.main.PlayerID,
			Name:               row.main.Name,
			RealmSlug:          row.main.RealmSlug,
			RealmName:          row.main.RealmName,
			Region:             row.main.Region,
			ClassName:          row.main.ClassName,
			CombinedBestTimeMs: row.combined,
		}
		if row.main.MainSpecID.Valid {
			specID := int(row.main.MainSpecID.Int64)
			e.ActiveSpecID = specID
			e.ClassName, e.ActiveSpecName = wow.FallbackClassAndSpec(row.main.ClassName, row.main.ActiveSpecName, &specID)
		}
		bracket := loader.PercentileBracket(rank, total)
		if regional {
			e.RegionalRanking = rank
			e.RegionalRankingBracket = bracket
		} else {
			e.GlobalRanking = rank
			e.GlobalRankingBracket = bracket
		}
		return e
	}

	total := len(eligible)
	for i, row := range eligible {
		if i >= topN {
			break
		}
		lists.Global = append(lists.Global, toEntry(row, i+1, total, false))
	}

	for _, region := range homeRegions {
		regionTotal := 0
		for _, row := range eligible {
			if strings.EqualFold(row.region, region) {
				regionTotal++
			}
		}
		var regional []HomePlayerEntry
		for _, row := range eligible {
			if !strings.EqualFold(row.region, region) {
				continue
			}
			rank := len(regional) + 1
			if rank > topN {
				break
			}
			regional = append(regional, toEntry(row, rank, regionTotal, true))
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

	if err := fillAvatars(db, &lists); err != nil {
		return lists, err
	}
	return lists, nil
}

// account aggregates carry no avatar; look up the handful of displayed mains
func fillAvatars(db *sql.DB, lists *HomePlayerLists) error {
	all := [][]HomePlayerEntry{lists.Global, lists.US, lists.EU, lists.KR, lists.TW}
	idSet := map[int64]struct{}{}
	for _, l := range all {
		for _, e := range l {
			idSet[e.PlayerID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return nil
	}
	ids := make([]any, 0, len(idSet))
	ph := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
		ph = append(ph, "?")
	}
	rows, err := db.Query(`
		SELECT player_id, avatar_url
		FROM player_details
		WHERE player_id IN (`+strings.Join(ph, ",")+`)
		  AND avatar_url IS NOT NULL
	`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()
	avatars := map[int64]string{}
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return err
		}
		avatars[id] = url
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, l := range all {
		for i := range l {
			l[i].AvatarURL = avatars[l[i].PlayerID]
		}
	}
	return nil
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
