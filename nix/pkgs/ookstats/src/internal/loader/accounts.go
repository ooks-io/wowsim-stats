package loader

import (
	"database/sql"
	"fmt"
	"strings"
)

const DungeonsForFullCoverage = 9

type AccountChar struct {
	PlayerID         int64
	Name             string
	RealmSlug        string
	RealmName        string
	Region           string
	ClassName        string
	ActiveSpecName   string
	MainSpecID       sql.NullInt64
	CombinedBestTime sql.NullInt64
	TotalRuns        int
	HasFullCoverage  bool
}

type AccountAggregate struct {
	AccountID      int64
	Region         string
	Chars          []*AccountChar
	BestPerDungeon map[int]AccountBestRun
}

type AccountBestRun struct {
	DungeonID int
	PlayerID  int64
	Duration  int64
}

func LoadAccountAggregates(db *sql.DB, seasonID int) ([]*AccountAggregate, error) {
	charRows, err := db.Query(`
		SELECT p.account_id, p.id, p.name, r.slug, r.name, r.region,
		       COALESCE(pd.class_name, ''),
		       COALESCE(pd.active_spec_name, ''),
		       pp.main_spec_id,
		       pp.combined_best_time,
		       COALESCE(pp.total_runs, 0),
		       COALESCE(pp.has_complete_coverage, 0)
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_profiles pp ON p.id = pp.player_id AND pp.season_id = ?
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE p.account_id IS NOT NULL AND p.is_valid = 1
	`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("load account chars: %w", err)
	}
	defer charRows.Close()

	byAccount := make(map[int64]*AccountAggregate)
	for charRows.Next() {
		var (
			accountID int64
			c         AccountChar
			covered   int
		)
		if err := charRows.Scan(&accountID, &c.PlayerID, &c.Name, &c.RealmSlug, &c.RealmName, &c.Region,
			&c.ClassName, &c.ActiveSpecName, &c.MainSpecID, &c.CombinedBestTime, &c.TotalRuns, &covered); err != nil {
			return nil, err
		}
		c.HasFullCoverage = covered == 1
		agg, ok := byAccount[accountID]
		if !ok {
			agg = &AccountAggregate{
				AccountID:      accountID,
				Region:         c.Region,
				BestPerDungeon: make(map[int]AccountBestRun),
			}
			byAccount[accountID] = agg
		}
		agg.Chars = append(agg.Chars, &c)
	}
	if err := charRows.Err(); err != nil {
		return nil, err
	}

	bestRows, err := db.Query(`
		SELECT pbr.player_id, pbr.dungeon_id, pbr.duration, p.account_id
		FROM player_best_runs pbr
		JOIN players p ON p.id = pbr.player_id
		WHERE pbr.season_id = ? AND p.account_id IS NOT NULL
	`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("load best runs: %w", err)
	}
	defer bestRows.Close()

	for bestRows.Next() {
		var playerID int64
		var dungeonID int
		var duration int64
		var accountID int64
		if err := bestRows.Scan(&playerID, &dungeonID, &duration, &accountID); err != nil {
			return nil, err
		}
		agg := byAccount[accountID]
		if agg == nil {
			continue
		}
		cur, ok := agg.BestPerDungeon[dungeonID]
		if !ok || duration < cur.Duration {
			agg.BestPerDungeon[dungeonID] = AccountBestRun{DungeonID: dungeonID, PlayerID: playerID, Duration: duration}
		}
	}
	if err := bestRows.Err(); err != nil {
		return nil, err
	}

	out := make([]*AccountAggregate, 0, len(byAccount))
	for _, agg := range byAccount {
		out = append(out, agg)
	}
	return out, nil
}

func LoadAllTimeAccountAggregates(db *sql.DB) ([]*AccountAggregate, error) {
	// main_spec_id from latest season so the UI stays current
	charRows, err := db.Query(`
		SELECT p.account_id, p.id, p.name, r.slug, r.name, r.region,
		       COALESCE(pd.class_name, ''),
		       COALESCE(pd.active_spec_name, ''),
		       (SELECT pp.main_spec_id FROM player_profiles pp
		         WHERE pp.player_id = p.id
		         ORDER BY pp.season_id DESC LIMIT 1) AS main_spec_id,
		       (SELECT COALESCE(SUM(pp.total_runs), 0) FROM player_profiles pp
		         WHERE pp.player_id = p.id) AS total_runs
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE p.account_id IS NOT NULL AND p.is_valid = 1
	`)
	if err != nil {
		return nil, fmt.Errorf("load all-time account chars: %w", err)
	}
	defer charRows.Close()

	byAccount := make(map[int64]*AccountAggregate)
	playerByID := make(map[int64]*AccountChar)
	for charRows.Next() {
		var (
			accountID int64
			c         AccountChar
		)
		if err := charRows.Scan(&accountID, &c.PlayerID, &c.Name, &c.RealmSlug, &c.RealmName, &c.Region,
			&c.ClassName, &c.ActiveSpecName, &c.MainSpecID, &c.TotalRuns); err != nil {
			return nil, err
		}
		agg, ok := byAccount[accountID]
		if !ok {
			agg = &AccountAggregate{
				AccountID:      accountID,
				Region:         c.Region,
				BestPerDungeon: make(map[int]AccountBestRun),
			}
			byAccount[accountID] = agg
		}
		agg.Chars = append(agg.Chars, &c)
		playerByID[c.PlayerID] = &c
	}
	if err := charRows.Err(); err != nil {
		return nil, err
	}

	charBest, err := db.Query(`
		SELECT pbr.player_id, pbr.dungeon_id, MIN(pbr.duration), p.account_id
		FROM player_best_runs pbr
		JOIN players p ON p.id = pbr.player_id
		WHERE p.account_id IS NOT NULL
		GROUP BY pbr.player_id, pbr.dungeon_id
	`)
	if err != nil {
		return nil, fmt.Errorf("load char all-time best: %w", err)
	}
	defer charBest.Close()

	type charDungeon struct {
		Sum  int64
		Done int
	}
	charSums := make(map[int64]*charDungeon)
	for charBest.Next() {
		var playerID int64
		var dungeonID int
		var duration int64
		var accountID int64
		if err := charBest.Scan(&playerID, &dungeonID, &duration, &accountID); err != nil {
			return nil, err
		}
		s, ok := charSums[playerID]
		if !ok {
			s = &charDungeon{}
			charSums[playerID] = s
		}
		s.Sum += duration
		s.Done++
		agg := byAccount[accountID]
		if agg == nil {
			continue
		}
		cur, ok := agg.BestPerDungeon[dungeonID]
		if !ok || duration < cur.Duration {
			agg.BestPerDungeon[dungeonID] = AccountBestRun{DungeonID: dungeonID, PlayerID: playerID, Duration: duration}
		}
	}
	if err := charBest.Err(); err != nil {
		return nil, err
	}

	// stamp char fields so PickMainCharacter works on all-time data
	for _, c := range playerByID {
		s, ok := charSums[c.PlayerID]
		if !ok {
			continue
		}
		c.CombinedBestTime = sql.NullInt64{Int64: s.Sum, Valid: s.Done >= DungeonsForFullCoverage}
		c.HasFullCoverage = s.Done >= DungeonsForFullCoverage
	}

	out := make([]*AccountAggregate, 0, len(byAccount))
	for _, agg := range byAccount {
		out = append(out, agg)
	}
	return out, nil
}

// prefer the 9/9 alt with best time, else the best alt overall
func PickMainCharacter(agg *AccountAggregate) *AccountChar {
	var nineNine *AccountChar
	var anyChar *AccountChar
	for _, c := range agg.Chars {
		if c.HasFullCoverage && c.CombinedBestTime.Valid {
			if nineNine == nil || c.CombinedBestTime.Int64 < nineNine.CombinedBestTime.Int64 {
				nineNine = c
			}
		}
		if c.CombinedBestTime.Valid {
			if anyChar == nil || c.CombinedBestTime.Int64 < anyChar.CombinedBestTime.Int64 {
				anyChar = c
			}
		}
	}
	if nineNine != nil {
		return nineNine
	}
	if anyChar != nil {
		return anyChar
	}
	if len(agg.Chars) > 0 {
		return agg.Chars[0]
	}
	return nil
}

type CharacterAllTimeStats struct {
	PlayerID       int64
	Name           string
	RealmSlug      string
	RealmName      string
	Region         string
	ClassName      string
	ActiveSpecName string
	MainSpecID     sql.NullInt64
	CombinedBest   int64
	TotalRuns      int
}

func LoadAllTimeCharacterStats(db *sql.DB) ([]*CharacterAllTimeStats, error) {
	rows, err := db.Query(`
		WITH per_dungeon AS (
			SELECT player_id, dungeon_id, MIN(duration) AS best_dur
			FROM player_best_runs
			GROUP BY player_id, dungeon_id
		),
		per_player AS (
			SELECT player_id,
			       SUM(best_dur) AS combined_best,
			       COUNT(*)      AS dungeons
			FROM per_dungeon
			GROUP BY player_id
			HAVING COUNT(*) >= 9
		)
		SELECT p.id, p.name, r.slug, r.name, r.region,
		       COALESCE(pd.class_name, ''),
		       COALESCE(pd.active_spec_name, ''),
		       (SELECT pp.main_spec_id FROM player_profiles pp
		         WHERE pp.player_id = p.id
		         ORDER BY pp.season_id DESC LIMIT 1) AS main_spec_id,
		       pp.combined_best,
		       (SELECT COALESCE(SUM(prof.total_runs), 0) FROM player_profiles prof
		         WHERE prof.player_id = p.id) AS total_runs
		FROM per_player pp
		JOIN players p ON p.id = pp.player_id AND p.is_valid = 1
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
	`)
	if err != nil {
		return nil, fmt.Errorf("load all-time char stats: %w", err)
	}
	defer rows.Close()

	out := make([]*CharacterAllTimeStats, 0, 25000)
	for rows.Next() {
		var c CharacterAllTimeStats
		if err := rows.Scan(&c.PlayerID, &c.Name, &c.RealmSlug, &c.RealmName, &c.Region,
			&c.ClassName, &c.ActiveSpecName, &c.MainSpecID, &c.CombinedBest, &c.TotalRuns); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

type CharacterSeasonRuns struct {
	PlayerID       int64
	Name           string
	RealmSlug      string
	RealmName      string
	Region         string
	ClassName      string
	ActiveSpecName string
	MainSpecID     sql.NullInt64
	TotalRuns      int
}

func LoadCharacterSeasonRuns(db *sql.DB, seasonID int) ([]*CharacterSeasonRuns, error) {
	rows, err := db.Query(`
		SELECT p.id, p.name, r.slug, r.name, r.region,
		       COALESCE(pd.class_name, ''),
		       COALESCE(pd.active_spec_name, ''),
		       pp.main_spec_id,
		       pp.total_runs
		FROM player_profiles pp
		JOIN players p ON p.id = pp.player_id AND p.is_valid = 1
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE pp.season_id = ? AND pp.total_runs > 0
	`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("load char season %d runs: %w", seasonID, err)
	}
	defer rows.Close()

	out := make([]*CharacterSeasonRuns, 0, 30000)
	for rows.Next() {
		var c CharacterSeasonRuns
		if err := rows.Scan(&c.PlayerID, &c.Name, &c.RealmSlug, &c.RealmName, &c.Region,
			&c.ClassName, &c.ActiveSpecName, &c.MainSpecID, &c.TotalRuns); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// "region/slug" -> parent slug; parent + standalone realms map to themselves
func LoadRealmParents(db *sql.DB, regions []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, reg := range regions {
		rows, err := db.Query(`
			SELECT slug, COALESCE(NULLIF(parent_realm_slug, ''), slug)
			FROM realms
			WHERE region = ?
		`, reg)
		if err != nil {
			return nil, fmt.Errorf("load realm parents: %w", err)
		}
		for rows.Next() {
			var slug, parent string
			if err := rows.Scan(&slug, &parent); err != nil {
				rows.Close()
				return nil, err
			}
			out[reg+"/"+slug] = parent
		}
		rows.Close()
	}
	return out, nil
}

func ResolveParentRealm(parents map[string]string, region, realmSlug string) string {
	if parent, ok := parents[strings.ToLower(region)+"/"+strings.ToLower(realmSlug)]; ok && parent != "" {
		return parent
	}
	return realmSlug
}
