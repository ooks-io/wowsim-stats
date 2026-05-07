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

type accountSeasonRunsJob struct {
	seasonID  int
	scope     string
	region    string
	realmSlug string
	out       string
	pageSize  int
}

func GenerateAccountSeasonRunsLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
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
		fmt.Println("Warning: No seasons found - skipping account season-runs leaderboard")
		return nil
	}

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
		totalJobs += 1 + len(regions)
		for _, reg := range regions {
			totalJobs += len(realmSlugs[reg])
		}
	}
	fmt.Printf("Generating per-season account Total Runs leaderboards with %d workers (%d jobs)...\n", workers, totalJobs)

	jobs := make(chan accountSeasonRunsJob, 100)
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
				if err := writeAccountSeasonRunsScope(job, aggs, realmParents); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%20 == 0 {
					fmt.Printf("  ... %d/%d account season-runs scopes generated\n", c, totalJobs)
				}
			}
		}()
	}

	for _, season := range seasons {
		jobs <- accountSeasonRunsJob{seasonID: season.ID, scope: "global", out: out, pageSize: pageSize}
		for _, reg := range regions {
			jobs <- accountSeasonRunsJob{seasonID: season.ID, scope: "regional", region: reg, out: out, pageSize: pageSize}
		}
		for _, reg := range regions {
			for _, rslug := range realmSlugs[reg] {
				jobs <- accountSeasonRunsJob{seasonID: season.ID, scope: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
			}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d per-season account Total Runs leaderboards\n", completed.Load())
	return nil
}

func writeAccountSeasonRunsScope(job accountSeasonRunsJob, all []*loader.AccountAggregate, realmParents map[string]string) error {
	type accountRow struct {
		Aggregate *loader.AccountAggregate
		TotalRuns int
		Main      *loader.AccountChar
	}
	rows := make([]accountRow, 0, len(all))
	for _, agg := range all {
		if (job.scope == "regional" || job.scope == "realm") && !strings.EqualFold(agg.Region, job.region) {
			continue
		}
		totalRuns := 0
		var main *loader.AccountChar
		for _, c := range agg.Chars {
			totalRuns += c.TotalRuns
			if c.TotalRuns > 0 {
				if main == nil || c.TotalRuns > main.TotalRuns ||
					(c.TotalRuns == main.TotalRuns && c.PlayerID < main.PlayerID) {
					main = c
				}
			}
		}
		if totalRuns == 0 || main == nil {
			continue
		}
		if job.scope == "realm" {
			parent := loader.ResolveParentRealm(realmParents, main.Region, main.RealmSlug)
			if !strings.EqualFold(parent, job.realmSlug) {
				continue
			}
		}
		rows = append(rows, accountRow{Aggregate: agg, TotalRuns: totalRuns, Main: main})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalRuns != rows[j].TotalRuns {
			return rows[i].TotalRuns > rows[j].TotalRuns
		}
		return rows[i].Aggregate.AccountID < rows[j].Aggregate.AccountID
	})

	dir := filepath.Join(job.out, fmt.Sprintf("season%d", job.seasonID), "players", "by-runs")
	switch job.scope {
	case "global":
		dir = filepath.Join(dir, "global")
	case "regional":
		dir = filepath.Join(dir, "regional", job.region)
	case "realm":
		dir = filepath.Join(dir, "realm", job.region, job.realmSlug)
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
		entries := make([]map[string]any, 0, end-start)
		for i := start; i < end; i++ {
			r := rows[i]
			alts := make([]map[string]any, 0, len(r.Aggregate.Chars)-1)
			for _, c := range r.Aggregate.Chars {
				if c.PlayerID == r.Main.PlayerID {
					continue
				}
				if c.TotalRuns == 0 && c.ClassName == "" && !c.CombinedBestTime.Valid {
					continue
				}
				alts = append(alts, accountSeasonCharSummary(c))
			}
			sort.Slice(alts, func(i, j int) bool {
				na, _ := alts[i]["name"].(string)
				nb, _ := alts[j]["name"].(string)
				return na < nb
			})
			entries = append(entries, map[string]any{
				"rank":            i + 1,
				"account_id":      r.Aggregate.AccountID,
				"region":          r.Aggregate.Region,
				"total_runs":      r.TotalRuns,
				"character_count": len(r.Aggregate.Chars),
				"main":            accountSeasonCharSummary(r.Main),
				"alts":            alts,
			})
		}
		title := fmt.Sprintf("Season %d Account Total Runs", job.seasonID)
		switch job.scope {
		case "regional":
			title += " - " + strings.ToUpper(job.region)
		case "realm":
			title += " - " + strings.ToUpper(job.region) + "/" + job.realmSlug
		}
		page := map[string]any{
			"leaderboard":         entries,
			"title":               title,
			"generated_timestamp": time.Now().UnixMilli(),
			"pagination": map[string]any{
				"currentPage":   p,
				"pageSize":      job.pageSize,
				"totalAccounts": total,
				"totalPages":    pages,
				"hasNextPage":   p < pages,
				"hasPrevPage":   p > 1,
				"totalRuns":     total,
			},
		}
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

func accountSeasonCharSummary(c *loader.AccountChar) map[string]any {
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
