package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type accountAllTimeJob struct {
	scope     string
	region    string
	realmSlug string
	out       string
	pageSize  int
}

func GenerateAllTimeAccountLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 4
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	root := filepath.Join(out, "all-time", "players", "by-time")
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clean all-time accounts tree: %w", err)
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

	aggs, err := loader.LoadAllTimeAccountAggregates(db)
	if err != nil {
		return err
	}

	totalJobs := 1 + len(regions)
	for _, reg := range regions {
		totalJobs += len(realmSlugs[reg])
	}
	fmt.Printf("Generating all-time account leaderboards with %d workers (%d jobs)...\n", workers, totalJobs)

	jobs := make(chan accountAllTimeJob, 100)
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
				if err := writeAllTimeAccountScope(job, aggs, realmParents); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%10 == 0 {
					fmt.Printf("  ... %d/%d all-time account scopes generated\n", c, totalJobs)
				}
			}
		}()
	}

	jobs <- accountAllTimeJob{scope: "global", out: out, pageSize: pageSize}
	for _, reg := range regions {
		jobs <- accountAllTimeJob{scope: "regional", region: reg, out: out, pageSize: pageSize}
	}
	for _, reg := range regions {
		for _, rslug := range realmSlugs[reg] {
			jobs <- accountAllTimeJob{scope: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d all-time account leaderboards\n", completed.Load())
	return nil
}

func writeAllTimeAccountScope(job accountAllTimeJob, all []*loader.AccountAggregate, realmParents map[string]string) error {
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

	dir := filepath.Join(job.out, "all-time", "players", "by-time", job.scope)
	switch job.scope {
	case "regional":
		dir = filepath.Join(job.out, "all-time", "players", "by-time", "regional", job.region)
	case "realm":
		dir = filepath.Join(job.out, "all-time", "players", "by-time", "realm", job.region, job.realmSlug)
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
		title := "All-Time Global Account Rankings"
		switch job.scope {
		case "regional":
			title = "All-Time " + strings.ToUpper(job.region) + " Account Rankings"
		case "realm":
			title = "All-Time " + strings.ToUpper(job.region) + " " + job.realmSlug + " Account Rankings"
		}
		page := map[string]any{
			"leaderboard":         out,
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
