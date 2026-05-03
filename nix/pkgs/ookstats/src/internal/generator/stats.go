package generator

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"ookstats/internal/wow"
	"ookstats/internal/writer"
)

// StatsJSON is the top-level shape for web/public/api/stats.json.
// Three scopes are emitted: cross-season "all_time" plus one entry per season number.
type StatsJSON struct {
	GeneratedAt int64                            `json:"generated_at"`
	// Scopes is keyed by region (global/us/eu/kr/tw), then by season key
	// (all_time/season_1/season_2). Region "global" includes all regions.
	Scopes map[string]map[string]StatsScope `json:"scopes"`
}

// StatsScope holds aggregated stats for a single scope (all-time or one season).
type StatsScope struct {
	TotalRuns         int64                     `json:"total_runs"`
	TotalPlayers      int64                     `json:"total_players"`
	NineOfNinePlayers int64                     `json:"nine_of_nine_players"`
	CompletionTiers   CompletionTiers           `json:"completion_tiers"`
	SpecCounts        map[string]SpecCountBucket `json:"spec_counts"`
	WeeklyActivity    []WeeklyActivityEntry     `json:"weekly_activity"`
}

// SpecCountBucket — entries plus the bucket's distinct-run denominator, so the
// chart can switch between "total slot count" and "% of runs containing the
// spec" without recomputing. `ByDungeon` is the same shape grouped per-dungeon,
// for the spec chart's dungeon filter and the dungeon-distribution chart.
type SpecCountBucket struct {
	TotalRuns int64                       `json:"total_runs"`
	Entries   []SpecCountEntry            `json:"entries"`
	ByDungeon map[int]DungeonSpecBucket   `json:"by_dungeon"`
}

// DungeonSpecBucket — sub-bucket for one dungeon within a SpecCountBucket.
type DungeonSpecBucket struct {
	TotalRuns int64            `json:"total_runs"`
	Entries   []SpecCountEntry `json:"entries"`
}

// CompletionTiers groups the 9-of-9 medal-tier counts.
type CompletionTiers struct {
	NineOfNineGold     CompletionTier `json:"9_of_9_gold"`
	NineOfNinePlatinum CompletionTier `json:"9_of_9_platinum"`
	NineOfNineTitle    CompletionTier `json:"9_of_9_title"`
}

// CompletionTier — count plus three percentile denominators.
// `percentile_of_all_players` = count / scope.total_players (rarity vs scoped pool)
// `percentile_of_completed_players` = count / players_with_9_dungeon_bests (rarity among finishers)
// `percentile_of_all_time_players` = count / all_time.total_players (stable cross-scope denominator)
type CompletionTier struct {
	Count                        int64   `json:"count"`
	PercentileOfAllPlayers       float64 `json:"percentile_of_all_players"`
	PercentileOfCompletedPlayers float64 `json:"percentile_of_completed_players"`
	PercentileOfAllTimePlayers   float64 `json:"percentile_of_all_time_players"`
}

// SpecCountEntry — one row of the spec-distribution charts.
// `count` is the total number of run_member spec slots (a run with two of the
// same spec contributes 2). `runs_with_spec` is the count of distinct runs
// containing the spec (the same run with two of the spec contributes 1 — capped
// at the bucket's total_runs).
type SpecCountEntry struct {
	SpecID       int    `json:"spec_id"`
	ClassName    string `json:"class_name"`
	SpecName     string `json:"spec_name"`
	Count        int64  `json:"count"`
	RunsWithSpec int64  `json:"runs_with_spec"`
}

// WeeklyActivityEntry — one bucket on the activity timeline (Monday-start week).
type WeeklyActivityEntry struct {
	WeekStart string `json:"week_start"` // YYYY-MM-DD
	RunCount  int64  `json:"run_count"`
}

// dungeonThresholds — gold / platinum / title timer thresholds in milliseconds.
// Gold = the achievement par-time (the "beat the timer" goal).
// Platinum / title are tighter community-defined cutoffs (see /docs or original source).
type dungeonThresholds struct {
	GoldMs     int64
	PlatinumMs int64
	TitleMs    int64
}

var dungeonTimerThresholds = map[int]dungeonThresholds{
	2:  {GoldMs: 900000, PlatinumMs: 615000, TitleMs: 510000},   // Temple of the Jade Serpent (15:00 / 10:15 / 8:30)
	56: {GoldMs: 720000, PlatinumMs: 495000, TitleMs: 390000},   // Stormstout Brewery        (12:00 / 8:15 / 6:30)
	57: {GoldMs: 780000, PlatinumMs: 480000, TitleMs: 330000},   // Gate of the Setting Sun   (13:00 / 8:00 / 5:30)
	58: {GoldMs: 1260000, PlatinumMs: 840000, TitleMs: 630000},  // Shado-Pan Monastery       (21:00 / 14:00 / 10:30)
	59: {GoldMs: 1050000, PlatinumMs: 735000, TitleMs: 615000},  // Siege of Niuzao Temple    (17:30 / 12:15 / 10:15)
	60: {GoldMs: 720000, PlatinumMs: 495000, TitleMs: 405000},   // Mogu'shan Palace          (12:00 / 8:15 / 6:45)
	76: {GoldMs: 1140000, PlatinumMs: 615000, TitleMs: 435000},  // Scholomance               (19:00 / 10:15 / 7:15)
	77: {GoldMs: 780000, PlatinumMs: 480000, TitleMs: 255000},   // Scarlet Halls             (13:00 / 8:00 / 4:15)
	78: {GoldMs: 780000, PlatinumMs: 540000, TitleMs: 330000},   // Scarlet Monastery         (13:00 / 9:00 / 5:30)
}

// totalDungeons in the season — a player must clear all of them to qualify for 9/9.
var totalDungeons = len(dungeonTimerThresholds)

// In-game cutoffs are lenient: the displayed cutoff is the first second that
// still earns the medal. e.g. a 5:30.800 run beats a "5:30" title cutoff,
// because the timer reads "5:30" until it ticks over to 5:31. So the actual
// pass condition is `duration < threshold + leniencyMs`.
const leniencyMs int64 = 1000

// beatsTier reports whether duration earns at least the given threshold.
func beatsTier(duration, thresholdMs int64) bool {
	return duration < thresholdMs+leniencyMs
}

// Scopes emitted in the output. Order matters for JSON readability.
type statsScopeDef struct {
	Key       string
	SeasonNum int // 0 = all-time
}

var statsScopes = []statsScopeDef{
	{Key: "all_time", SeasonNum: 0},
	{Key: "season_1", SeasonNum: 1},
	{Key: "season_2", SeasonNum: 2},
}

// statsRegions are the region scopes emitted; "global" means no region filter.
var statsRegions = []string{"global", "us", "eu", "kr", "tw"}

// GenerateStats writes outDir/api/stats.json with aggregated stats per (region, season) scope.
func GenerateStats(db *sql.DB, outDir string) error {
	out := StatsJSON{
		GeneratedAt: time.Now().UnixMilli(),
		Scopes:      make(map[string]map[string]StatsScope, len(statsRegions)),
	}

	for _, region := range statsRegions {
		regionScopes := make(map[string]StatsScope, len(statsScopes))
		// Compute all-time first per region so its total_players is the stable
		// denominator for per-season scopes within that region.
		var allTimeTotalPlayers int64
		for _, sc := range statsScopes {
			ss, err := buildStatsScope(db, sc.SeasonNum, region, allTimeTotalPlayers)
			if err != nil {
				return fmt.Errorf("scope %s/%s: %w", region, sc.Key, err)
			}
			if sc.SeasonNum == 0 {
				allTimeTotalPlayers = ss.TotalPlayers
			}
			regionScopes[sc.Key] = ss
		}
		out.Scopes[region] = regionScopes
	}

	outPath := filepath.Join(outDir, "api", "stats.json")
	if err := writer.WriteJSONFileCompact(outPath, out); err != nil {
		return fmt.Errorf("write stats.json: %w", err)
	}
	return nil
}

// buildStatsScope assembles a single StatsScope.
// seasonNum == 0 means cross-season (all-time). region == "global" means no region filter.
// For all-time, pass allTimeTotalPlayers=0; for per-season builds, pass the all-time scope's
// TotalPlayers (within the same region) so the `percentile_of_all_time_players` field
// uses a stable denominator.
func buildStatsScope(db *sql.DB, seasonNum int, region string, allTimeTotalPlayers int64) (StatsScope, error) {
	var s StatsScope
	var err error

	if s.TotalRuns, err = countTotalRuns(db, seasonNum, region); err != nil {
		return s, fmt.Errorf("total runs: %w", err)
	}

	if s.TotalPlayers, err = countTotalPlayers(db, seasonNum, region); err != nil {
		return s, fmt.Errorf("total players: %w", err)
	}

	// On all-time, the stable denominator is just this scope's own TotalPlayers.
	allTimeDen := allTimeTotalPlayers
	if seasonNum == 0 {
		allTimeDen = s.TotalPlayers
	}

	tiers, completedCount, err := computeCompletionTiers(db, seasonNum, region)
	if err != nil {
		return s, fmt.Errorf("completion tiers: %w", err)
	}
	s.CompletionTiers = tiers
	s.NineOfNinePlayers = completedCount
	// fill percentile fields now that we know all three denominators
	s.CompletionTiers.NineOfNineGold = withPercentiles(s.CompletionTiers.NineOfNineGold, s.TotalPlayers, completedCount, allTimeDen)
	s.CompletionTiers.NineOfNinePlatinum = withPercentiles(s.CompletionTiers.NineOfNinePlatinum, s.TotalPlayers, completedCount, allTimeDen)
	s.CompletionTiers.NineOfNineTitle = withPercentiles(s.CompletionTiers.NineOfNineTitle, s.TotalPlayers, completedCount, allTimeDen)

	if s.SpecCounts, err = computeSpecCounts(db, seasonNum, region); err != nil {
		return s, fmt.Errorf("spec counts: %w", err)
	}

	if s.WeeklyActivity, err = queryWeeklyActivity(db, seasonNum, region); err != nil {
		return s, fmt.Errorf("weekly activity: %w", err)
	}

	return s, nil
}

// seasonClauseAndArg returns the SQL fragment and arg slice for filtering by challenge_runs.season_id.
// Empty string for all-time (seasonNum == 0).
func seasonClauseAndArg(seasonNum int, alias string) (string, []any) {
	if seasonNum == 0 {
		return "", nil
	}
	if alias == "" {
		alias = "challenge_runs"
	}
	return fmt.Sprintf(" AND %s.season_id = ?", alias), []any{seasonNum}
}

// regionClauseAndArg filters by realm region. region == "global" returns empty (no filter).
// `realmAlias` is the SQL alias of the realms table reference in the FROM/JOIN clause.
func regionClauseAndArg(region, realmAlias string) (string, []any) {
	if region == "" || region == "global" {
		return "", nil
	}
	if realmAlias == "" {
		realmAlias = "r"
	}
	return fmt.Sprintf(" AND %s.region = ?", realmAlias), []any{region}
}

func countTotalRuns(db *sql.DB, seasonNum int, region string) (int64, error) {
	// Region filter requires joining realms; only join when needed to keep the global
	// case as a fast plain COUNT(*) on challenge_runs.
	if region == "" || region == "global" {
		q := "SELECT COUNT(*) FROM challenge_runs cr WHERE 1=1"
		clause, args := seasonClauseAndArg(seasonNum, "cr")
		var n int64
		if err := db.QueryRow(q+clause, args...).Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}
	q := "SELECT COUNT(*) FROM challenge_runs cr JOIN realms r ON r.id = cr.realm_id WHERE 1=1"
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	var n int64
	if err := db.QueryRow(q+sClause+rClause, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// countTotalPlayers counts distinct players who appear in run_members within scope.
// "Within scope" = any player who participated in a run during the given season+region.
func countTotalPlayers(db *sql.DB, seasonNum int, region string) (int64, error) {
	// Always join through challenge_runs so we can filter by season and/or region.
	q := `SELECT COUNT(DISTINCT rm.player_id)
	      FROM run_members rm
	      JOIN challenge_runs cr ON rm.run_id = cr.id`
	if region != "" && region != "global" {
		q += " JOIN realms r ON r.id = cr.realm_id"
	}
	q += " WHERE 1=1"
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	var n int64
	if err := db.QueryRow(q+sClause+rClause, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// computeCompletionTiers loads each player's best time per (season, dungeon) and
// classifies them against the gold / platinum / title thresholds.
// Tiers are achievements earned *within a single season*, so for the all-time scope
// we take the union of per-season qualifiers — a player counts if they hit 9/9 of a
// tier in any single season, not by combining best times across seasons.
// Returns the tier counts and the count of players who completed all 9 dungeons in
// at least one season (the "completed" denominator).
func computeCompletionTiers(db *sql.DB, seasonNum int, region string) (CompletionTiers, int64, error) {
	q := `
		SELECT rm.player_id, cr.season_id, cr.dungeon_id, MIN(cr.duration) AS best_duration
		FROM run_members rm
		JOIN challenge_runs cr ON rm.run_id = cr.id
		%s
		WHERE 1=1 %s %s
		GROUP BY rm.player_id, cr.season_id, cr.dungeon_id
	`
	regionJoin := ""
	if region != "" && region != "global" {
		regionJoin = "JOIN realms r ON r.id = cr.realm_id"
	}
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	rows, err := db.Query(fmt.Sprintf(q, regionJoin, sClause, rClause), args...)
	if err != nil {
		return CompletionTiers{}, 0, err
	}
	defer rows.Close()

	// (player_id, season_id) -> per-tier dungeon-cleared count
	type key struct {
		playerID int64
		seasonID int
	}
	type counts struct{ gold, plat, title, anyDungeon int }
	per := make(map[key]*counts)
	for rows.Next() {
		var playerID int64
		var seasonID, dungeonID int
		var bestDuration int64
		if err := rows.Scan(&playerID, &seasonID, &dungeonID, &bestDuration); err != nil {
			return CompletionTiers{}, 0, err
		}
		threshold, ok := dungeonTimerThresholds[dungeonID]
		if !ok {
			continue // unknown dungeon — skip silently
		}
		k := key{playerID, seasonID}
		c := per[k]
		if c == nil {
			c = &counts{}
			per[k] = c
		}
		c.anyDungeon++
		if beatsTier(bestDuration, threshold.GoldMs) {
			c.gold++
		}
		if beatsTier(bestDuration, threshold.PlatinumMs) {
			c.plat++
		}
		if beatsTier(bestDuration, threshold.TitleMs) {
			c.title++
		}
	}
	if err := rows.Err(); err != nil {
		return CompletionTiers{}, 0, err
	}

	// Union per player: did they qualify in *any* season?
	type qualifiers struct{ gold, plat, title, completed bool }
	playerQual := make(map[int64]*qualifiers)
	for k, c := range per {
		q := playerQual[k.playerID]
		if q == nil {
			q = &qualifiers{}
			playerQual[k.playerID] = q
		}
		if c.anyDungeon == totalDungeons {
			q.completed = true
		}
		if c.gold == totalDungeons {
			q.gold = true
		}
		if c.plat == totalDungeons {
			q.plat = true
		}
		if c.title == totalDungeons {
			q.title = true
		}
	}

	var tiers CompletionTiers
	var completedCount int64
	for _, q := range playerQual {
		if q.completed {
			completedCount++
		}
		if q.gold {
			tiers.NineOfNineGold.Count++
		}
		if q.plat {
			tiers.NineOfNinePlatinum.Count++
		}
		if q.title {
			tiers.NineOfNineTitle.Count++
		}
	}
	return tiers, completedCount, nil
}

func withPercentiles(t CompletionTier, totalPlayers, completedPlayers, allTimePlayers int64) CompletionTier {
	if totalPlayers > 0 {
		t.PercentileOfAllPlayers = round2(float64(t.Count) * 100.0 / float64(totalPlayers))
	}
	if completedPlayers > 0 {
		t.PercentileOfCompletedPlayers = round2(float64(t.Count) * 100.0 / float64(completedPlayers))
	}
	if allTimePlayers > 0 {
		t.PercentileOfAllTimePlayers = round2(float64(t.Count) * 100.0 / float64(allTimePlayers))
	}
	return t
}

// computeSpecCounts returns the spec-distribution charts:
//   all_runs:      every run_member row in scope
//   gold_runs:     run_members where the run beat the gold threshold for its dungeon
//   platinum_runs: same for platinum
//   title_runs:    same for title
//   top_50_runs:   run_members where the run was top-50 in scope (globally team-filtered for
//                  region=="global", regionally team-filtered for a specific region)
// Each bucket carries both `count` (slot occurrences, includes spec stacking)
// and `runs_with_spec` (distinct run count, capped at the bucket's total_runs).
func computeSpecCounts(db *sql.DB, seasonNum int, region string) (map[string]SpecCountBucket, error) {
	out := map[string]SpecCountBucket{}

	allRuns, err := querySpecCountsForRuns(db, seasonNum, region)
	if err != nil {
		return nil, fmt.Errorf("all_runs: %w", err)
	}
	out["all_runs"] = allRuns

	for _, tier := range []struct {
		key        string
		thresholdF func(dungeonThresholds) int64
	}{
		{"gold_runs", func(t dungeonThresholds) int64 { return t.GoldMs }},
		{"platinum_runs", func(t dungeonThresholds) int64 { return t.PlatinumMs }},
		{"title_runs", func(t dungeonThresholds) int64 { return t.TitleMs }},
	} {
		tierCounts, err := querySpecCountsForTier(db, seasonNum, region, tier.thresholdF)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tier.key, err)
		}
		out[tier.key] = tierCounts
	}

	top50, err := querySpecCountsForTop50(db, seasonNum, region)
	if err != nil {
		return nil, fmt.Errorf("top_50_runs: %w", err)
	}
	out["top_50_runs"] = top50

	return out, nil
}

// querySpecCountsForRuns counts run_member spec_ids across all runs in scope.
// Returns slot count + distinct-run count per spec, plus the bucket's total
// distinct run count, plus the same broken out per dungeon.
func querySpecCountsForRuns(db *sql.DB, seasonNum int, region string) (SpecCountBucket, error) {
	regionJoin := ""
	if region != "" && region != "global" {
		regionJoin = "JOIN realms r ON r.id = cr.realm_id"
	}
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)

	// One query grouped by (dungeon, spec) — we derive both the per-dungeon
	// breakdown and the overall totals from the same result set, since each
	// challenge_run has a single dungeon_id (no double-counting across dungeons).
	q := `
		SELECT cr.dungeon_id, rm.spec_id,
		       COUNT(*) AS slot_count,
		       COUNT(DISTINCT rm.run_id) AS run_count
		FROM run_members rm
		JOIN challenge_runs cr ON rm.run_id = cr.id
		%s
		WHERE rm.spec_id IS NOT NULL %s %s
		GROUP BY cr.dungeon_id, rm.spec_id
	`
	rows, err := db.Query(fmt.Sprintf(q, regionJoin, sClause, rClause), args...)
	if err != nil {
		return SpecCountBucket{}, err
	}
	defer rows.Close()

	// dungeonId -> specId -> {count, distinctRuns}
	perDungeon := make(map[int]map[int]*specRunCount)
	overall := make(map[int]*specRunCount)
	for rows.Next() {
		var dungeonID, specID int
		var slot, runs int64
		if err := rows.Scan(&dungeonID, &specID, &slot, &runs); err != nil {
			return SpecCountBucket{}, err
		}
		dm, ok := perDungeon[dungeonID]
		if !ok {
			dm = make(map[int]*specRunCount)
			perDungeon[dungeonID] = dm
		}
		dm[specID] = &specRunCount{count: slot, distinctRuns: runs}
		o := overall[specID]
		if o == nil {
			o = &specRunCount{}
			overall[specID] = o
		}
		o.count += slot
		// Each run lives in one dungeon, so distinct runs sum cleanly across dungeons.
		o.distinctRuns += runs
	}
	if err := rows.Err(); err != nil {
		return SpecCountBucket{}, err
	}

	// Per-dungeon distinct run totals AND overall — one query, no scan loop tax.
	perDungeonTotals, overallTotal, err := queryRunCountsForRuns(db, seasonNum, region)
	if err != nil {
		return SpecCountBucket{}, err
	}

	bucket := SpecCountBucket{
		TotalRuns: overallTotal,
		Entries:   entriesFromCounts(overall),
		ByDungeon: make(map[int]DungeonSpecBucket, len(perDungeonTotals)),
	}
	for dungeonID, total := range perDungeonTotals {
		bucket.ByDungeon[dungeonID] = DungeonSpecBucket{
			TotalRuns: total,
			Entries:   entriesFromCounts(perDungeon[dungeonID]),
		}
	}
	return bucket, nil
}

// queryRunCountsForRuns returns distinct-run counts per dungeon (and the overall
// sum) within scope, used as the denominator for the spec-distribution chart's
// "Runs" metric and for the dungeon-distribution chart.
func queryRunCountsForRuns(db *sql.DB, seasonNum int, region string) (map[int]int64, int64, error) {
	regionJoin := ""
	if region != "" && region != "global" {
		regionJoin = "JOIN realms r ON r.id = cr.realm_id"
	}
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	q := fmt.Sprintf(`
		SELECT cr.dungeon_id, COUNT(DISTINCT cr.id)
		FROM challenge_runs cr
		%s
		WHERE 1=1 %s %s
		GROUP BY cr.dungeon_id
	`, regionJoin, sClause, rClause)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	perDungeon := make(map[int]int64)
	var total int64
	for rows.Next() {
		var dungeonID int
		var count int64
		if err := rows.Scan(&dungeonID, &count); err != nil {
			return nil, 0, err
		}
		perDungeon[dungeonID] = count
		total += count
	}
	return perDungeon, total, rows.Err()
}

// specRunCount aggregates slot count + distinct-run count for a single spec.
type specRunCount struct {
	count, distinctRuns int64
}

// entriesFromCounts converts a (specId -> {count, distinctRuns}) map into the
// sorted SpecCountEntry slice used by the chart, attaching class/spec metadata.
// Sorted descending by count for stable order; the chart re-orders by canonical
// class+spec layout but tooltips/iteration work nicer on a deterministic input.
func entriesFromCounts(m map[int]*specRunCount) []SpecCountEntry {
	if m == nil {
		return []SpecCountEntry{}
	}
	out := make([]SpecCountEntry, 0, len(m))
	for specID, c := range m {
		entry := SpecCountEntry{SpecID: specID, Count: c.count, RunsWithSpec: c.distinctRuns}
		if cls, spec, ok := wow.GetClassAndSpec(specID); ok {
			entry.ClassName = cls
			entry.SpecName = spec
		}
		out = append(out, entry)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Count > out[j-1].Count; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// querySpecCountsForTier counts run_member spec_ids across runs that beat a per-dungeon threshold.
// We can't easily express the per-dungeon threshold in a single SQL query, so instead we load
// (run_id, dungeon_id, duration) rows in scope, decide qualification in Go, then count specs by
// the qualifying run_id set. We track both slot occurrences and distinct-run-with-spec counts,
// and a per-dungeon breakdown for the spec-distribution chart's dungeon filter.
func querySpecCountsForTier(db *sql.DB, seasonNum int, region string, thresholdF func(dungeonThresholds) int64) (SpecCountBucket, error) {
	regionJoin := ""
	if region != "" && region != "global" {
		regionJoin = "JOIN realms r ON r.id = cr.realm_id"
	}
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT cr.id, cr.dungeon_id, cr.duration
		FROM challenge_runs cr
		%s
		WHERE 1=1 %s %s
	`, regionJoin, sClause, rClause), args...)
	if err != nil {
		return SpecCountBucket{}, err
	}
	// runId -> dungeonId for runs that beat the threshold. Per-dungeon counts
	// fall out of the same map.
	qualifyingDungeon := make(map[int64]int)
	dungeonRunCounts := make(map[int]int64)
	for rows.Next() {
		var id int64
		var dungeonID int
		var duration int64
		if err := rows.Scan(&id, &dungeonID, &duration); err != nil {
			rows.Close()
			return SpecCountBucket{}, err
		}
		threshold, ok := dungeonTimerThresholds[dungeonID]
		if !ok {
			continue
		}
		if beatsTier(duration, thresholdF(threshold)) {
			qualifyingDungeon[id] = dungeonID
			dungeonRunCounts[dungeonID]++
		}
	}
	rows.Close()
	if len(qualifyingDungeon) == 0 {
		return SpecCountBucket{TotalRuns: 0, Entries: []SpecCountEntry{}, ByDungeon: map[int]DungeonSpecBucket{}}, nil
	}

	// For qualifying runs, count slot occurrences AND distinct runs per spec,
	// both overall and per-dungeon. Per-dungeon distinct-run counting reuses
	// the qualifying map (one run -> one dungeon, no double-count).
	overall := make(map[int]*specAgg)
	perDungeon := make(map[int]map[int]*specAgg)
	memberRows, err := db.Query(`
		SELECT run_id, spec_id
		FROM run_members
		WHERE spec_id IS NOT NULL
	`)
	if err != nil {
		return SpecCountBucket{}, err
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var runID int64
		var specID int
		if err := memberRows.Scan(&runID, &specID); err != nil {
			return SpecCountBucket{}, err
		}
		dungeonID, ok := qualifyingDungeon[runID]
		if !ok {
			continue
		}
		o := overall[specID]
		if o == nil {
			o = &specAgg{runs: make(map[int64]struct{})}
			overall[specID] = o
		}
		o.count++
		o.runs[runID] = struct{}{}

		dm, ok := perDungeon[dungeonID]
		if !ok {
			dm = make(map[int]*specAgg)
			perDungeon[dungeonID] = dm
		}
		da := dm[specID]
		if da == nil {
			da = &specAgg{runs: make(map[int64]struct{})}
			dm[specID] = da
		}
		da.count++
		da.runs[runID] = struct{}{}
	}

	bucket := SpecCountBucket{
		TotalRuns: int64(len(qualifyingDungeon)),
		Entries:   specAggsToSlice(overall),
		ByDungeon: make(map[int]DungeonSpecBucket, len(perDungeon)),
	}
	for dungeonID, total := range dungeonRunCounts {
		bucket.ByDungeon[dungeonID] = DungeonSpecBucket{
			TotalRuns: total,
			Entries:   specAggsToSlice(perDungeon[dungeonID]),
		}
	}
	return bucket, nil
}

// querySpecCountsForTop50 counts spec_ids in top-50 runs.
// region "global" → top 50 globally (filtered); region "us"/"eu"/etc → top 50 within that region.
// Per-dungeon breakdown is also returned (each dungeon's top 50 contributes 50 runs to its bucket).
func querySpecCountsForTop50(db *sql.DB, seasonNum int, region string) (SpecCountBucket, error) {
	rankingType := "global"
	rankingScope := "filtered"
	if region != "" && region != "global" {
		rankingType = "regional"
		rankingScope = region + "_filtered"
	}
	// Group by (dungeon, spec) so we get both per-dungeon and overall totals
	// from a single result set. cr.dungeon_id comes from joining challenge_runs.
	q := `
		SELECT cr.dungeon_id, rm.spec_id,
		       COUNT(*) AS slot_count,
		       COUNT(DISTINCT rm.run_id) AS run_count
		FROM run_members rm
		JOIN run_rankings rr ON rm.run_id = rr.run_id
		JOIN challenge_runs cr ON cr.id = rm.run_id
		WHERE rm.spec_id IS NOT NULL
		  AND rr.ranking_type = ?
		  AND rr.ranking_scope = ?
		  AND rr.ranking <= 50
	`
	args := []any{rankingType, rankingScope}
	if seasonNum != 0 {
		q += " AND rr.season_id = ?"
		args = append(args, seasonNum)
	}
	q += " GROUP BY cr.dungeon_id, rm.spec_id"

	rows, err := db.Query(q, args...)
	if err != nil {
		return SpecCountBucket{}, err
	}
	defer rows.Close()

	perDungeon := make(map[int]map[int]*specRunCount)
	overall := make(map[int]*specRunCount)
	for rows.Next() {
		var dungeonID, specID int
		var slot, runs int64
		if err := rows.Scan(&dungeonID, &specID, &slot, &runs); err != nil {
			return SpecCountBucket{}, err
		}
		dm, ok := perDungeon[dungeonID]
		if !ok {
			dm = make(map[int]*specRunCount)
			perDungeon[dungeonID] = dm
		}
		dm[specID] = &specRunCount{count: slot, distinctRuns: runs}
		o := overall[specID]
		if o == nil {
			o = &specRunCount{}
			overall[specID] = o
		}
		o.count += slot
		o.distinctRuns += runs
	}
	if err := rows.Err(); err != nil {
		return SpecCountBucket{}, err
	}

	// Per-dungeon distinct top-50 run counts (and total).
	totalsQ := `
		SELECT cr.dungeon_id, COUNT(DISTINCT rr.run_id)
		FROM run_rankings rr
		JOIN challenge_runs cr ON cr.id = rr.run_id
		WHERE rr.ranking_type = ?
		  AND rr.ranking_scope = ?
		  AND rr.ranking <= 50
	`
	totalsArgs := []any{rankingType, rankingScope}
	if seasonNum != 0 {
		totalsQ += " AND rr.season_id = ?"
		totalsArgs = append(totalsArgs, seasonNum)
	}
	totalsQ += " GROUP BY cr.dungeon_id"
	totalRows, err := db.Query(totalsQ, totalsArgs...)
	if err != nil {
		return SpecCountBucket{}, err
	}
	defer totalRows.Close()
	dungeonTotals := make(map[int]int64)
	var grandTotal int64
	for totalRows.Next() {
		var dungeonID int
		var count int64
		if err := totalRows.Scan(&dungeonID, &count); err != nil {
			return SpecCountBucket{}, err
		}
		dungeonTotals[dungeonID] = count
		grandTotal += count
	}

	bucket := SpecCountBucket{
		TotalRuns: grandTotal,
		Entries:   entriesFromCounts(overall),
		ByDungeon: make(map[int]DungeonSpecBucket, len(dungeonTotals)),
	}
	for dungeonID, total := range dungeonTotals {
		bucket.ByDungeon[dungeonID] = DungeonSpecBucket{
			TotalRuns: total,
			Entries:   entriesFromCounts(perDungeon[dungeonID]),
		}
	}
	return bucket, nil
}

// specAgg holds in-memory aggregation state when we can't count via SQL —
// see querySpecCountsForTier.
type specAgg struct {
	count int64
	runs  map[int64]struct{}
}

// specAggsToSlice converts a Go-side aggregation (slot count + run-id set per
// spec) into a sorted slice — used by querySpecCountsForTier where the
// per-dungeon threshold makes a SQL-only aggregation awkward.
func specAggsToSlice(aggs map[int]*specAgg) []SpecCountEntry {
	out := make([]SpecCountEntry, 0, len(aggs))
	for specID, a := range aggs {
		entry := SpecCountEntry{SpecID: specID, Count: a.count, RunsWithSpec: int64(len(a.runs))}
		if cls, spec, ok := wow.GetClassAndSpec(specID); ok {
			entry.ClassName = cls
			entry.SpecName = spec
		}
		out = append(out, entry)
	}
	// simple insertion sort by count desc (small N: ~33 specs)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Count > out[j-1].Count; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// queryWeeklyActivity returns runs grouped by Monday-start week.
// Timestamps are stored in milliseconds; SQLite date functions need seconds.
func queryWeeklyActivity(db *sql.DB, seasonNum int, region string) ([]WeeklyActivityEntry, error) {
	regionJoin := ""
	if region != "" && region != "global" {
		regionJoin = "JOIN realms r ON r.id = cr.realm_id"
	}
	sClause, sArgs := seasonClauseAndArg(seasonNum, "cr")
	rClause, rArgs := regionClauseAndArg(region, "r")
	args := append(sArgs, rArgs...)
	// 'weekday 1' advances to next Monday; '-7 days' rolls back to start of current week.
	q := fmt.Sprintf(`
		SELECT
			date(cr.completed_timestamp / 1000, 'unixepoch', 'weekday 1', '-7 days') AS week_start,
			COUNT(*) AS run_count
		FROM challenge_runs cr
		%s
		WHERE cr.completed_timestamp IS NOT NULL %s %s
		GROUP BY week_start
		ORDER BY week_start
	`, regionJoin, sClause, rClause)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WeeklyActivityEntry
	for rows.Next() {
		var e WeeklyActivityEntry
		if err := rows.Scan(&e.WeekStart, &e.RunCount); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
