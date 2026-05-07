package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/wow"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type accountLeaderboardJob struct {
	seasonID  int
	scope     string // "global" | "regional" | "realm"
	region    string // for regional / realm
	realmSlug string // for realm
	out       string
	pageSize  int
}

func GenerateAccountLeaderboards(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 4
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	seasons, err := loadSeasons(db)
	if err != nil {
		return err
	}
	if len(seasons) == 0 {
		fmt.Println("Warning: No seasons found - skipping account leaderboards")
		return nil
	}

	// realm-scoped: account placed where its main lives (with connected-realm rollup)
	realmSlugs := make(map[string][]string, len(regions))
	for _, reg := range regions {
		slugs, err := loadParentRealmSlugs(db, reg)
		if err != nil {
			return err
		}
		realmSlugs[reg] = slugs
	}
	realmParents, err := loader.LoadRealmParents(db, regions)
	if err != nil {
		return err
	}

	// share per-season load across scope workers
	type cached struct {
		aggregates []*loader.AccountAggregate
		err        error
	}
	cache := make(map[int]*cached)
	var cacheMu sync.Mutex
	getAggregates := func(seasonID int) ([]*loader.AccountAggregate, error) {
		cacheMu.Lock()
		c, ok := cache[seasonID]
		cacheMu.Unlock()
		if ok {
			return c.aggregates, c.err
		}
		aggs, err := loader.LoadAccountAggregates(db, seasonID)
		cacheMu.Lock()
		cache[seasonID] = &cached{aggregates: aggs, err: err}
		cacheMu.Unlock()
		return aggs, err
	}

	totalJobs := 0
	for range seasons {
		totalJobs += 1            // global
		totalJobs += len(regions) // regional
		for _, reg := range regions {
			totalJobs += len(realmSlugs[reg]) // realm
		}
	}
	fmt.Printf("Generating account leaderboards with %d workers (%d total jobs)...\n", workers, totalJobs)

	jobs := make(chan accountLeaderboardJob, 100)
	var firstErr atomic.Value
	var completed atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if firstErr.Load() != nil {
					continue
				}
				aggs, err := getAggregates(job.seasonID)
				if err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				if err := writeAccountScope(job, aggs, realmParents); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%10 == 0 {
					fmt.Printf("  ... %d/%d account leaderboards generated\n", c, totalJobs)
				}
			}
		}()
	}

	for _, season := range seasons {
		fmt.Printf("\n=== Queuing Account Leaderboards for Season %d (%s) ===\n", season.ID, season.Name)
		seasonOut := filepath.Join(out, "season", fmt.Sprintf("%d", season.ID))

		jobs <- accountLeaderboardJob{seasonID: season.ID, scope: "global", out: seasonOut, pageSize: pageSize}
		for _, reg := range regions {
			jobs <- accountLeaderboardJob{seasonID: season.ID, scope: "regional", region: reg, out: seasonOut, pageSize: pageSize}
		}
		for _, reg := range regions {
			for _, rslug := range realmSlugs[reg] {
				jobs <- accountLeaderboardJob{seasonID: season.ID, scope: "realm", region: reg, realmSlug: rslug, out: seasonOut, pageSize: pageSize}
			}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d account leaderboards\n", completed.Load())
	return nil
}

func writeAccountScope(job accountLeaderboardJob, all []*loader.AccountAggregate, realmParents map[string]string) error {
	type accountRow struct {
		Aggregate *loader.AccountAggregate
		Combined  int64
		Main      *loader.AccountChar
	}
	rows := make([]accountRow, 0, len(all))
	for _, agg := range all {
		if (job.scope == "regional" || job.scope == "realm") && !strings.EqualFold(agg.Region, job.region) {
			continue
		}
		if len(agg.BestPerDungeon) < loader.DungeonsForFullCoverage {
			continue
		}
		main := loader.PickMainCharacter(agg)
		if job.scope == "realm" {
			if main == nil {
				continue
			}
			parent := loader.ResolveParentRealm(realmParents, main.Region, main.RealmSlug)
			if !strings.EqualFold(parent, job.realmSlug) {
				continue
			}
		}
		var sum int64
		for _, br := range agg.BestPerDungeon {
			sum += br.Duration
		}
		rows = append(rows, accountRow{Aggregate: agg, Combined: sum, Main: main})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Combined != rows[j].Combined {
			return rows[i].Combined < rows[j].Combined
		}
		return rows[i].Aggregate.AccountID < rows[j].Aggregate.AccountID
	})

	dir := filepath.Join(job.out, "accounts", job.scope)
	switch job.scope {
	case "regional":
		dir = filepath.Join(job.out, "accounts", "regional", job.region)
	case "realm":
		dir = filepath.Join(job.out, "accounts", "realm", job.region, job.realmSlug)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	total := len(rows)
	pages := (total + job.pageSize - 1) / job.pageSize
	if pages == 0 {
		pages = 1
	}
	for p := 1; p <= pages; p++ {
		start := (p - 1) * job.pageSize
		end := start + job.pageSize
		if end > total {
			end = total
		}
		out := make([]map[string]any, 0, end-start)
		for i := start; i < end; i++ {
			rank := i + 1
			bracket := loader.PercentileBracket(rank, total)
			out = append(out, buildAccountRow(rank, bracket, rows[i].Aggregate, rows[i].Main, rows[i].Combined))
		}
		title := "Global Account Rankings"
		switch job.scope {
		case "regional":
			title = strings.ToUpper(job.region) + " Account Rankings"
		case "realm":
			title = strings.ToUpper(job.region) + " " + job.realmSlug + " Account Rankings"
		}
		page := buildAccountLeaderboardPage(out, title, total, pages, p, job.pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// caller picks main so the realm filter and rendered face stay in lockstep
func buildAccountRow(rank int, bracket string, agg *loader.AccountAggregate, main *loader.AccountChar, combined int64) map[string]any {
	alts := make([]map[string]any, 0, len(agg.Chars)-1)
	for _, c := range agg.Chars {
		if main != nil && c.PlayerID == main.PlayerID {
			continue
		}
		// phantom alt
		if !c.CombinedBestTime.Valid && c.ClassName == "" && c.TotalRuns == 0 {
			continue
		}
		alts = append(alts, charSummary(c))
	}
	sort.Slice(alts, func(i, j int) bool {
		na, _ := alts[i]["name"].(string)
		nb, _ := alts[j]["name"].(string)
		return na < nb
	})

	playerName := make(map[int64]string, len(agg.Chars))
	playerRealm := make(map[int64]string, len(agg.Chars))
	for _, c := range agg.Chars {
		playerName[c.PlayerID] = c.Name
		playerRealm[c.PlayerID] = c.RealmSlug
	}
	perDungeon := make(map[string]map[string]any, len(agg.BestPerDungeon))
	for did, br := range agg.BestPerDungeon {
		perDungeon[fmt.Sprintf("%d", did)] = map[string]any{
			"duration_ms":        br.Duration,
			"by_character":       playerName[br.PlayerID],
			"by_character_realm": playerRealm[br.PlayerID],
		}
	}

	totalRuns := 0
	for _, c := range agg.Chars {
		totalRuns += c.TotalRuns
	}

	return map[string]any{
		"rank":                       rank,
		"bracket":                    bracket,
		"account_id":                 agg.AccountID,
		"region":                     agg.Region,
		"account_combined_best_time": combined,
		"dungeons_completed":         len(agg.BestPerDungeon),
		"total_runs":                 totalRuns,
		"main":                       charSummary(main),
		"alts":                       alts,
		"per_dungeon_best":           perDungeon,
	}
}

func charSummary(c *loader.AccountChar) map[string]any {
	if c == nil {
		return nil
	}
	obj := map[string]any{
		"player_id":         c.PlayerID,
		"name":              c.Name,
		"realm_slug":        c.RealmSlug,
		"realm_name":        c.RealmName,
		"region":            c.Region,
		"class_name":        c.ClassName,
		"active_spec_name":  c.ActiveSpecName,
		"total_runs":        c.TotalRuns,
		"has_full_coverage": c.HasFullCoverage,
	}
	if c.CombinedBestTime.Valid {
		obj["combined_best_time"] = c.CombinedBestTime.Int64
	}
	if c.MainSpecID.Valid {
		v := int(c.MainSpecID.Int64)
		obj["class_name"], obj["active_spec_name"] = wow.FallbackClassAndSpec(c.ClassName, c.ActiveSpecName, &v)
		obj["main_spec_id"] = v
	}
	return obj
}

func buildAccountLeaderboardPage(rows []map[string]any, title string, total, totalPages, currentPage, pageSize int) map[string]any {
	return map[string]any{
		"leaderboard":         rows,
		"title":               title,
		"generated_timestamp": time.Now().UnixMilli(),
		"pagination": map[string]any{
			"currentPage":   currentPage,
			"pageSize":      pageSize,
			"totalAccounts": total,
			"totalPages":    totalPages,
			"hasNextPage":   currentPage < totalPages,
			"hasPrevPage":   currentPage > 1,
			"totalRuns":     total, // frontend compatibility
		},
	}
}
