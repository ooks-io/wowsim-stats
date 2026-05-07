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

type playerSeasonRunsJob struct {
	seasonID  int
	scope     string
	region    string
	realmSlug string
	classKey  string
	classWith string // "global" | "regional" | "realm" when scope=class
	out       string
	pageSize  int
}

func GeneratePlayerSeasonRunsLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 8
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	seasons, err := loadSeasons(db)
	if err != nil {
		return err
	}
	if len(seasons) == 0 {
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
	classKeys := []string{"death_knight", "druid", "hunter", "mage", "monk", "paladin", "priest", "rogue", "shaman", "warlock", "warrior"}

	// share per-season load across scope workers
	type cached struct {
		stats []*loader.CharacterSeasonRuns
		err   error
	}
	cache := make(map[int]*cached)
	var cacheMu sync.Mutex
	getStats := func(seasonID int) ([]*loader.CharacterSeasonRuns, error) {
		cacheMu.Lock()
		c, ok := cache[seasonID]
		cacheMu.Unlock()
		if ok {
			return c.stats, c.err
		}
		ss, err := loader.LoadCharacterSeasonRuns(db, seasonID)
		if err == nil {
			for _, s := range ss {
				var specID *int
				if s.MainSpecID.Valid {
					v := int(s.MainSpecID.Int64)
					specID = &v
				}
				s.ClassName, s.ActiveSpecName = wow.FallbackClassAndSpec(s.ClassName, s.ActiveSpecName, specID)
			}
		}
		cacheMu.Lock()
		cache[seasonID] = &cached{stats: ss, err: err}
		cacheMu.Unlock()
		return ss, err
	}

	totalJobs := 0
	for range seasons {
		totalJobs += 1 + len(regions)
		for _, reg := range regions {
			totalJobs += len(realmSlugs[reg])
		}
		for range classKeys {
			totalJobs += 1 + len(regions)
			for _, reg := range regions {
				totalJobs += len(realmSlugs[reg])
			}
		}
	}
	fmt.Printf("Generating per-season per-char Total Runs leaderboards with %d workers (%d jobs)...\n", workers, totalJobs)

	jobs := make(chan playerSeasonRunsJob, 200)
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
				ss, err := getStats(job.seasonID)
				if err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				if err := writePlayerSeasonRunsScope(job, ss, realmParents); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%200 == 0 {
					fmt.Printf("  ... %d/%d per-char season-runs scopes generated\n", c, totalJobs)
				}
			}
		}()
	}

	for _, season := range seasons {
		jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "global", out: out, pageSize: pageSize}
		for _, reg := range regions {
			jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "regional", region: reg, out: out, pageSize: pageSize}
		}
		for _, reg := range regions {
			for _, rslug := range realmSlugs[reg] {
				jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
			}
		}
		for _, cls := range classKeys {
			jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "class", classKey: cls, classWith: "global", out: out, pageSize: pageSize}
			for _, reg := range regions {
				jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "class", classKey: cls, classWith: "regional", region: reg, out: out, pageSize: pageSize}
			}
			for _, reg := range regions {
				for _, rslug := range realmSlugs[reg] {
					jobs <- playerSeasonRunsJob{seasonID: season.ID, scope: "class", classKey: cls, classWith: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
				}
			}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d per-char season-runs leaderboards\n", completed.Load())
	return nil
}

func writePlayerSeasonRunsScope(job playerSeasonRunsJob, all []*loader.CharacterSeasonRuns, realmParents map[string]string) error {
	classNameKey := func(s string) string {
		return strings.ReplaceAll(strings.ToLower(s), " ", "_")
	}

	matches := func(s *loader.CharacterSeasonRuns) bool {
		if job.scope == "class" && classNameKey(s.ClassName) != job.classKey {
			return false
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
			if !strings.EqualFold(s.Region, job.region) {
				return false
			}
			parent := loader.ResolveParentRealm(realmParents, s.Region, s.RealmSlug)
			return strings.EqualFold(parent, job.realmSlug)
		}
		return false
	}

	rows := make([]*loader.CharacterSeasonRuns, 0, len(all)/8)
	for _, s := range all {
		if matches(s) {
			rows = append(rows, s)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalRuns != rows[j].TotalRuns {
			return rows[i].TotalRuns > rows[j].TotalRuns
		}
		return rows[i].Name < rows[j].Name
	})

	dir := filepath.Join(job.out, fmt.Sprintf("season%d", job.seasonID), "characters", "by-runs")
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
				"id":               s.PlayerID,
				"name":             s.Name,
				"realm_slug":       s.RealmSlug,
				"realm_name":       s.RealmName,
				"region":           s.Region,
				"class_name":       s.ClassName,
				"active_spec_name": s.ActiveSpecName,
				"total_runs":       s.TotalRuns,
			}
			if s.MainSpecID.Valid {
				entry["main_spec_id"] = int(s.MainSpecID.Int64)
			}
			entries = append(entries, entry)
		}
		title := fmt.Sprintf("Season %d Total Runs", job.seasonID)
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
