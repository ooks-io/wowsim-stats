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

type playerAllTimeJob struct {
	scope     string
	region    string
	realmSlug string
	classKey  string // empty unless scope == "class"
	classWith string // sub-scope inside class: "global" | "regional" | "realm"
	out       string
	pageSize  int
}

func GenerateAllTimePlayerLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 8
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	root := filepath.Join(out, "all-time", "characters", "by-time")
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clean all-time chars tree: %w", err)
	}

	realmSlugs := make(map[string][]string, len(regions))
	for _, reg := range regions {
		slugs, err := loadParentRealmSlugs(db, reg)
		if err != nil {
			return err
		}
		realmSlugs[reg] = slugs
	}

	stats, err := loader.LoadAllTimeCharacterStats(db)
	if err != nil {
		return err
	}
	realmParents, err := loader.LoadRealmParents(db, regions)
	if err != nil {
		return err
	}

	for _, s := range stats {
		var specID *int
		if s.MainSpecID.Valid {
			v := int(s.MainSpecID.Int64)
			specID = &v
		}
		s.ClassName, s.ActiveSpecName = wow.FallbackClassAndSpec(s.ClassName, s.ActiveSpecName, specID)
	}

	classKeys := []string{"death_knight", "druid", "hunter", "mage", "monk", "paladin", "priest", "rogue", "shaman", "warlock", "warrior"}

	totalJobs := 1 + len(regions)
	for _, reg := range regions {
		totalJobs += len(realmSlugs[reg])
	}
	for range classKeys {
		totalJobs += 1 + len(regions)
		for _, reg := range regions {
			totalJobs += len(realmSlugs[reg])
		}
	}
	fmt.Printf("Generating all-time per-char leaderboards with %d workers (%d jobs)...\n", workers, totalJobs)

	jobs := make(chan playerAllTimeJob, 200)
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
				if err := writeAllTimePlayerScope(job, stats, realmParents); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%100 == 0 {
					fmt.Printf("  ... %d/%d all-time per-char scopes generated\n", c, totalJobs)
				}
			}
		}()
	}

	jobs <- playerAllTimeJob{scope: "global", out: out, pageSize: pageSize}
	for _, reg := range regions {
		jobs <- playerAllTimeJob{scope: "regional", region: reg, out: out, pageSize: pageSize}
	}
	for _, reg := range regions {
		for _, rslug := range realmSlugs[reg] {
			jobs <- playerAllTimeJob{scope: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
		}
	}
	for _, cls := range classKeys {
		jobs <- playerAllTimeJob{scope: "class", classKey: cls, classWith: "global", out: out, pageSize: pageSize}
		for _, reg := range regions {
			jobs <- playerAllTimeJob{scope: "class", classKey: cls, classWith: "regional", region: reg, out: out, pageSize: pageSize}
		}
		for _, reg := range regions {
			for _, rslug := range realmSlugs[reg] {
				jobs <- playerAllTimeJob{scope: "class", classKey: cls, classWith: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
			}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d all-time per-char leaderboards\n", completed.Load())
	return nil
}

func writeAllTimePlayerScope(job playerAllTimeJob, all []*loader.CharacterAllTimeStats, realmParents map[string]string) error {
	classNameKey := func(s string) string {
		return strings.ReplaceAll(strings.ToLower(s), " ", "_")
	}

	matchesScope := func(s *loader.CharacterAllTimeStats) bool {
		if job.scope == "class" {
			if classNameKey(s.ClassName) != job.classKey {
				return false
			}
		}
		geo := job.scope
		if job.scope == "class" {
			geo = job.classWith
		}
		switch geo {
		case "global":
			return true
		case "regional":
			return strings.EqualFold(s.Region, job.region)
		case "realm":
			// connected-realm rollup
			if !strings.EqualFold(s.Region, job.region) {
				return false
			}
			parent := loader.ResolveParentRealm(realmParents, s.Region, s.RealmSlug)
			return strings.EqualFold(parent, job.realmSlug)
		}
		return false
	}

	rows := make([]*loader.CharacterAllTimeStats, 0, len(all)/8)
	for _, s := range all {
		if matchesScope(s) {
			rows = append(rows, s)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CombinedBest != rows[j].CombinedBest {
			return rows[i].CombinedBest < rows[j].CombinedBest
		}
		return rows[i].Name < rows[j].Name
	})

	dir := filepath.Join(job.out, "all-time", "characters", "by-time")
	switch job.scope {
	case "global":
		dir = filepath.Join(dir, "global")
	case "regional":
		dir = filepath.Join(dir, "regional", job.region)
	case "realm":
		dir = filepath.Join(dir, "realm", job.region, job.realmSlug)
	case "class":
		dir = filepath.Join(dir, "class", job.classKey)
		switch job.classWith {
		case "global":
			dir = filepath.Join(dir, "global")
		case "regional":
			dir = filepath.Join(dir, "regional", job.region)
		case "realm":
			dir = filepath.Join(dir, "realm", job.region, job.realmSlug)
		}
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
			s := rows[i]
			entry := map[string]any{
				"id":                 s.PlayerID,
				"name":               s.Name,
				"realm_slug":         s.RealmSlug,
				"realm_name":         s.RealmName,
				"region":             s.Region,
				"class_name":         s.ClassName,
				"active_spec_name":   s.ActiveSpecName,
				"combined_best_time": s.CombinedBest,
				"total_runs":         s.TotalRuns,
				"ranking_percentile": loader.PercentileBracket(i+1, total),
			}
			if s.MainSpecID.Valid {
				entry["main_spec_id"] = int(s.MainSpecID.Int64)
			}
			entries = append(entries, entry)
		}
		title := titleForAllTimeScope(job)
		page := map[string]any{
			"leaderboard":         entries,
			"title":               title,
			"generated_timestamp": time.Now().UnixMilli(),
			"pagination": map[string]any{
				"currentPage":  p,
				"pageSize":     job.pageSize,
				"totalPlayers": total,
				"totalPages":   pages,
				"hasNextPage":  p < pages,
				"hasPrevPage":  p > 1,
				"totalRuns":    total,
			},
		}
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

func titleForAllTimeScope(job playerAllTimeJob) string {
	switch job.scope {
	case "global":
		return "All-Time Global Player Rankings"
	case "regional":
		return "All-Time " + strings.ToUpper(job.region) + " Player Rankings"
	case "realm":
		return "All-Time " + strings.ToUpper(job.region) + "/" + job.realmSlug + " Player Rankings"
	case "class":
		base := "All-Time " + job.classKey + " Player Rankings"
		switch job.classWith {
		case "global":
			return base + " (Global)"
		case "regional":
			return base + " (" + strings.ToUpper(job.region) + ")"
		case "realm":
			return base + " (" + strings.ToUpper(job.region) + "/" + job.realmSlug + ")"
		}
	}
	return "All-Time Player Rankings"
}
