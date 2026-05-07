package loader

import (
	"database/sql"
	"sort"
	"strings"
)

type AccountSeasonStats struct {
	AccountID         int64
	Region            string
	RealmSlug         string
	RealmName         string
	CombinedBestTime  int64
	DungeonsCompleted int
	TotalRuns         int
	CharacterCount    int
	HasFullCoverage   bool
	GlobalRanking     int // 0 = unranked
	RegionalRanking   int
	RealmRanking      int
	GlobalBracket     string
	RegionalBracket   string
	RealmBracket      string
	MainPlayerID      int64
}

func LoadAccountSeasonStats(db *sql.DB, seasonID int, regions []string) (map[int64]*AccountSeasonStats, error) {
	aggs, err := LoadAccountAggregates(db, seasonID)
	if err != nil {
		return nil, err
	}
	parents, err := LoadRealmParents(db, regions)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]*AccountSeasonStats, len(aggs))
	for _, agg := range aggs {
		stats := &AccountSeasonStats{
			AccountID:         agg.AccountID,
			Region:            agg.Region,
			DungeonsCompleted: len(agg.BestPerDungeon),
			CharacterCount:    len(agg.Chars),
		}
		for _, c := range agg.Chars {
			stats.TotalRuns += c.TotalRuns
		}
		main := PickMainCharacter(agg)
		if main != nil {
			stats.MainPlayerID = main.PlayerID
			stats.RealmSlug = ResolveParentRealm(parents, main.Region, main.RealmSlug)
			stats.RealmName = main.RealmName
		}
		stats.HasFullCoverage = stats.DungeonsCompleted >= DungeonsForFullCoverage
		if stats.HasFullCoverage {
			var sum int64
			for _, br := range agg.BestPerDungeon {
				sum += br.Duration
			}
			stats.CombinedBestTime = sum
		}
		out[agg.AccountID] = stats
	}

	assignRankings(out)
	return out, nil
}

func assignRankings(stats map[int64]*AccountSeasonStats) {
	type entry struct {
		stats *AccountSeasonStats
	}

	{
		ranked := make([]*AccountSeasonStats, 0, len(stats))
		for _, s := range stats {
			if s.HasFullCoverage {
				ranked = append(ranked, s)
			}
		}
		sortByCombined(ranked)
		total := len(ranked)
		for i, s := range ranked {
			rank := i + 1
			s.GlobalRanking = rank
			s.GlobalBracket = PercentileBracket(rank, total)
		}
	}

	byRegion := make(map[string][]*AccountSeasonStats)
	for _, s := range stats {
		if !s.HasFullCoverage || s.Region == "" {
			continue
		}
		byRegion[strings.ToLower(s.Region)] = append(byRegion[strings.ToLower(s.Region)], s)
	}
	for _, group := range byRegion {
		sortByCombined(group)
		total := len(group)
		for i, s := range group {
			rank := i + 1
			s.RegionalRanking = rank
			s.RegionalBracket = PercentileBracket(rank, total)
		}
	}

	// realm group key = main's parent realm (connected-realm rollup)
	byRealm := make(map[string][]*AccountSeasonStats)
	for _, s := range stats {
		if !s.HasFullCoverage || s.RealmSlug == "" || s.Region == "" {
			continue
		}
		key := strings.ToLower(s.Region) + "/" + strings.ToLower(s.RealmSlug)
		byRealm[key] = append(byRealm[key], s)
	}
	for _, group := range byRealm {
		sortByCombined(group)
		total := len(group)
		for i, s := range group {
			rank := i + 1
			s.RealmRanking = rank
			s.RealmBracket = PercentileBracket(rank, total)
		}
	}
}

func sortByCombined(rows []*AccountSeasonStats) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CombinedBestTime != rows[j].CombinedBestTime {
			return rows[i].CombinedBestTime < rows[j].CombinedBestTime
		}
		return rows[i].AccountID < rows[j].AccountID
	})
}

func LoadAllTimeAccountStats(db *sql.DB, regions []string) (map[int64]*AccountSeasonStats, error) {
	aggs, err := LoadAllTimeAccountAggregates(db)
	if err != nil {
		return nil, err
	}
	parents, err := LoadRealmParents(db, regions)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]*AccountSeasonStats, len(aggs))
	for _, agg := range aggs {
		stats := &AccountSeasonStats{
			AccountID:         agg.AccountID,
			Region:            agg.Region,
			DungeonsCompleted: len(agg.BestPerDungeon),
			CharacterCount:    len(agg.Chars),
		}
		for _, c := range agg.Chars {
			stats.TotalRuns += c.TotalRuns
		}
		main := PickMainCharacter(agg)
		if main != nil {
			stats.MainPlayerID = main.PlayerID
			stats.RealmSlug = ResolveParentRealm(parents, main.Region, main.RealmSlug)
			stats.RealmName = main.RealmName
		}
		stats.HasFullCoverage = stats.DungeonsCompleted >= DungeonsForFullCoverage
		if stats.HasFullCoverage {
			var sum int64
			for _, br := range agg.BestPerDungeon {
				sum += br.Duration
			}
			stats.CombinedBestTime = sum
		}
		out[agg.AccountID] = stats
	}

	assignRankings(out)
	return out, nil
}

type CharacterAllTimeRanks struct {
	Stats           *CharacterAllTimeStats
	GlobalRanking   int
	RegionalRanking int
	RealmRanking    int
	GlobalBracket   string
	RegionalBracket string
	RealmBracket    string
}

func LoadAllTimeCharacterRanks(db *sql.DB, regions []string) (map[int64]*CharacterAllTimeRanks, error) {
	stats, err := LoadAllTimeCharacterStats(db)
	if err != nil {
		return nil, err
	}
	parents, err := LoadRealmParents(db, regions)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]*CharacterAllTimeRanks, len(stats))
	for _, s := range stats {
		out[s.PlayerID] = &CharacterAllTimeRanks{Stats: s}
	}

	globalSorted := make([]*CharacterAllTimeStats, len(stats))
	copy(globalSorted, stats)
	sort.Slice(globalSorted, func(i, j int) bool {
		if globalSorted[i].CombinedBest != globalSorted[j].CombinedBest {
			return globalSorted[i].CombinedBest < globalSorted[j].CombinedBest
		}
		return globalSorted[i].PlayerID < globalSorted[j].PlayerID
	})
	for i, s := range globalSorted {
		rank := i + 1
		out[s.PlayerID].GlobalRanking = rank
		out[s.PlayerID].GlobalBracket = PercentileBracket(rank, len(globalSorted))
	}

	byRegion := make(map[string][]*CharacterAllTimeStats)
	for _, s := range stats {
		key := strings.ToLower(s.Region)
		byRegion[key] = append(byRegion[key], s)
	}
	for _, group := range byRegion {
		sort.Slice(group, func(i, j int) bool {
			if group[i].CombinedBest != group[j].CombinedBest {
				return group[i].CombinedBest < group[j].CombinedBest
			}
			return group[i].PlayerID < group[j].PlayerID
		})
		for i, s := range group {
			rank := i + 1
			out[s.PlayerID].RegionalRanking = rank
			out[s.PlayerID].RegionalBracket = PercentileBracket(rank, len(group))
		}
	}

	// realm group key = parent realm (connected-realm rollup)
	byRealm := make(map[string][]*CharacterAllTimeStats)
	for _, s := range stats {
		parent := ResolveParentRealm(parents, s.Region, s.RealmSlug)
		key := strings.ToLower(s.Region) + "/" + strings.ToLower(parent)
		byRealm[key] = append(byRealm[key], s)
	}
	for _, group := range byRealm {
		sort.Slice(group, func(i, j int) bool {
			if group[i].CombinedBest != group[j].CombinedBest {
				return group[i].CombinedBest < group[j].CombinedBest
			}
			return group[i].PlayerID < group[j].PlayerID
		})
		for i, s := range group {
			rank := i + 1
			out[s.PlayerID].RealmRanking = rank
			out[s.PlayerID].RealmBracket = PercentileBracket(rank, len(group))
		}
	}

	return out, nil
}

func PercentileBracket(rank, total int) string {
	if total <= 0 {
		return ""
	}
	if rank == 1 {
		return "artifact"
	}
	pct := float64(rank) / float64(total) * 100
	switch {
	case pct <= 1.0:
		return "excellent"
	case pct <= 5.0:
		return "legendary"
	case pct <= 20.0:
		return "epic"
	case pct <= 40.0:
		return "rare"
	case pct <= 60.0:
		return "uncommon"
	default:
		return "common"
	}
}
