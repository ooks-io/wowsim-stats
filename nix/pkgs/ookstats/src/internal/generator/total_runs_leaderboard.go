package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/wow"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// cross-season total runs: SUM(pp.total_runs) per character, gated by
// has_complete_coverage = 1 in any season (kept loose so a top grinder
// isn't dropped for skipping one dungeon in the latest season).

type totalRunsJob struct {
	scope     string
	region    string
	realmSlug string
	classKey  string
	out       string
	pageSize  int
	regions   []string
}

// writes to <out>/players/total-runs/...
func GenerateTotalRunsLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 10
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	root := filepath.Join(out, "players", "total-runs")
	// wipe so realms dropping out of the data don't leave stale JSON
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clean total-runs tree: %w", err)
	}

	realmSlugs := make(map[string][]string)
	for _, reg := range regions {
		rrows, err := db.Query(`
			SELECT slug
			FROM realms
			WHERE region = ? AND (parent_realm_slug IS NULL OR parent_realm_slug = '')
			ORDER BY slug
		`, reg)
		if err != nil {
			return fmt.Errorf("load realm slugs: %w", err)
		}
		var slugs []string
		for rrows.Next() {
			var s string
			if err := rrows.Scan(&s); err != nil {
				rrows.Close()
				return err
			}
			slugs = append(slugs, s)
		}
		rrows.Close()
		realmSlugs[reg] = slugs
	}

	classKeys := []string{"death_knight", "druid", "hunter", "mage", "monk", "paladin", "priest", "rogue", "shaman", "warlock", "warrior"}

	totalJobs := 1
	totalJobs += len(regions)
	for _, reg := range regions {
		totalJobs += len(realmSlugs[reg])
	}
	totalJobs += len(classKeys)

	fmt.Printf("Generating Total Runs leaderboard with %d workers (%d total jobs)...\n", workers, totalJobs)

	jobs := make(chan totalRunsJob, 100)
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
				var err error
				switch job.scope {
				case "global":
					err = generateTotalRunsGlobal(db, job.out, job.pageSize)
				case "regional":
					err = generateTotalRunsRegional(db, job.out, job.region, job.pageSize)
				case "realm":
					err = generateTotalRunsRealm(db, job.out, job.region, job.realmSlug, job.pageSize)
				case "class":
					err = generateTotalRunsClass(db, job.out, job.classKey, job.pageSize, job.regions)
				}
				if err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%20 == 0 {
					fmt.Printf("  ... %d/%d total-runs leaderboards generated\n", c, totalJobs)
				}
			}
		}()
	}

	jobs <- totalRunsJob{scope: "global", out: root, pageSize: pageSize}
	for _, reg := range regions {
		jobs <- totalRunsJob{scope: "regional", region: reg, out: root, pageSize: pageSize}
	}
	for _, reg := range regions {
		for _, rslug := range realmSlugs[reg] {
			jobs <- totalRunsJob{scope: "realm", region: reg, realmSlug: rslug, out: root, pageSize: pageSize}
		}
	}
	for _, cls := range classKeys {
		jobs <- totalRunsJob{scope: "class", classKey: cls, out: root, pageSize: pageSize, regions: regions}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d total-runs leaderboards\n", completed.Load())
	return nil
}

func totalRunsBaseSelect() string {
	return `
		SELECT p.id,
		       p.name,
		       r.slug         AS realm_slug,
		       r.name         AS realm_name,
		       r.region,
		       COALESCE(pd.class_name, '')       AS class_name,
		       COALESCE(pd.active_spec_name, '') AS active_spec_name,
		       MAX(pp.main_spec_id)              AS main_spec_id,
		       SUM(pp.total_runs)                AS all_time_runs
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		JOIN player_profiles pp ON p.id = pp.player_id
		LEFT JOIN player_details pd ON p.id = pd.player_id
	`
}

const totalRunsHaving = `
	GROUP BY p.id, p.name, r.slug, r.name, r.region, pd.class_name, pd.active_spec_name
	HAVING MAX(pp.has_complete_coverage) = 1 AND SUM(pp.total_runs) > 0
`

func generateTotalRunsGlobal(db *sql.DB, root string, pageSize int) error {
	dir := filepath.Join(root, "global")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rows, err := db.Query(totalRunsBaseSelect() + totalRunsHaving + `
		ORDER BY all_time_runs DESC, p.name ASC
	`)
	if err != nil {
		return fmt.Errorf("total-runs query (global): %w", err)
	}
	all, err := scanTotalRunsRows(rows)
	if err != nil {
		return err
	}
	return writeTotalRunsPages(dir, all, "Global Total Runs", pageSize)
}

func generateTotalRunsRegional(db *sql.DB, root, region string, pageSize int) error {
	dir := filepath.Join(root, "regional", region)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rows, err := db.Query(totalRunsBaseSelect()+`
		WHERE r.region = ?
	`+totalRunsHaving+`
		ORDER BY all_time_runs DESC, p.name ASC
	`, region)
	if err != nil {
		return fmt.Errorf("total-runs query (regional %s): %w", region, err)
	}
	all, err := scanTotalRunsRows(rows)
	if err != nil {
		return err
	}
	return writeTotalRunsPages(dir, all, strings.ToUpper(region)+" Total Runs", pageSize)
}

// pool = parent + child realms (matches per-season realm leaderboard)
func generateTotalRunsRealm(db *sql.DB, root, region, rslug string, pageSize int) error {
	dir := filepath.Join(root, "realm", region, rslug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rows, err := db.Query(totalRunsBaseSelect()+`
		WHERE r.region = ? AND (r.slug = ? OR r.parent_realm_slug = ?)
	`+totalRunsHaving+`
		ORDER BY all_time_runs DESC, p.name ASC
	`, region, rslug, rslug)
	if err != nil {
		return fmt.Errorf("total-runs query (realm %s/%s): %w", region, rslug, err)
	}
	all, err := scanTotalRunsRows(rows)
	if err != nil {
		return err
	}
	title := strings.ToUpper(region) + "/" + rslug + " Total Runs"
	return writeTotalRunsPages(dir, all, title, pageSize)
}

func writeTotalRunsPages(dir string, all []map[string]any, title string, pageSize int) error {
	total := len(all)
	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		start := (p - 1) * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		page := buildTotalRunsPage(all[start:end], title, total, pages, p, pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// filters in Go so we can reuse the wow.GetClassAndSpec fallback when
// pd.class_name is missing
func generateTotalRunsClass(db *sql.DB, root, classKey string, pageSize int, regions []string) error {
	if err := generateTotalRunsClassScope(db, root, "global", "", "", classKey, pageSize); err != nil {
		return err
	}
	for _, reg := range regions {
		if err := generateTotalRunsClassScope(db, root, "regional", reg, "", classKey, pageSize); err != nil {
			return err
		}
		rrows, err := db.Query(`
			SELECT slug
			FROM realms
			WHERE region = ? AND (parent_realm_slug IS NULL OR parent_realm_slug = '')
			ORDER BY slug
		`, reg)
		if err != nil {
			return fmt.Errorf("total-runs class realms list: %w", err)
		}
		var slugs []string
		for rrows.Next() {
			var s string
			if err := rrows.Scan(&s); err != nil {
				rrows.Close()
				return err
			}
			slugs = append(slugs, s)
		}
		rrows.Close()
		for _, rslug := range slugs {
			if err := generateTotalRunsClassScope(db, root, "realm", reg, rslug, classKey, pageSize); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateTotalRunsClassScope(db *sql.DB, root, scope, region, rslug, classKey string, pageSize int) error {
	var dir string
	switch scope {
	case "global":
		dir = filepath.Join(root, "class", classKey, "global")
	case "regional":
		dir = filepath.Join(root, "class", classKey, "regional", region)
	default:
		dir = filepath.Join(root, "class", classKey, "realm", region, rslug)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var rows *sql.Rows
	var err error
	switch scope {
	case "global":
		rows, err = db.Query(totalRunsBaseSelect() + totalRunsHaving + `
			ORDER BY all_time_runs DESC, p.name ASC
		`)
	case "regional":
		rows, err = db.Query(totalRunsBaseSelect()+`
			WHERE r.region = ?
		`+totalRunsHaving+`
			ORDER BY all_time_runs DESC, p.name ASC
		`, region)
	default:
		rows, err = db.Query(totalRunsBaseSelect()+`
			WHERE r.region = ? AND (r.slug = ? OR r.parent_realm_slug = ?)
		`+totalRunsHaving+`
			ORDER BY all_time_runs DESC, p.name ASC
		`, region, rslug, rslug)
	}
	if err != nil {
		return err
	}
	all, err := scanTotalRunsRows(rows)
	if err != nil {
		return err
	}

	matching := make([]map[string]any, 0, len(all)/4)
	for _, r := range all {
		cls, _ := r["class_name"].(string)
		key := strings.ReplaceAll(strings.ToLower(cls), " ", "_")
		if key == classKey {
			matching = append(matching, r)
		}
	}

	total := len(matching)
	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		offset := (p - 1) * pageSize
		end := offset + pageSize
		if end > total {
			end = total
		}
		title := "Global Total Runs"
		switch scope {
		case "regional":
			title = strings.ToUpper(region) + " Total Runs"
		case "realm":
			title = strings.ToUpper(region) + "/" + rslug + " Total Runs"
		}
		page := buildTotalRunsPage(matching[offset:end], title, total, pages, p, pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// field names mirror scanPlayerRows so the frontend table component can
// render either tree without branching
func scanTotalRunsRows(rows *sql.Rows) ([]map[string]any, error) {
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var (
			id                                int64
			name, realmSlug, realmName        string
			region, className, activeSpecName string
			mainSpecID                        sql.NullInt64
			allTimeRuns                       sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &realmSlug, &realmName, &region, &className, &activeSpecName, &mainSpecID, &allTimeRuns); err != nil {
			return nil, err
		}
		obj := map[string]any{
			"player_id":  id,
			"name":       name,
			"realm_slug": realmSlug,
			"realm_name": realmName,
			"region":     region,
			"class_name": className,
			"total_runs": allTimeRuns.Int64,
		}
		if mainSpecID.Valid {
			v := int(mainSpecID.Int64)
			cls, spec := wow.FallbackClassAndSpec(className, activeSpecName, &v)
			obj["class_name"] = cls
			obj["active_spec_name"] = spec
			obj["main_spec_id"] = v
		} else if activeSpecName != "" {
			obj["active_spec_name"] = activeSpecName
		}
		out = append(out, obj)
	}
	return out, nil
}

// shape mirrors buildPlayerLeaderboardPage so Pagination and the table
// component can render either tree without branching
func buildTotalRunsPage(list []map[string]any, title string, total, totalPages, currentPage, pageSize int) map[string]any {
	return map[string]any{
		"leaderboard": list,
		"title":       title,
		"pagination": map[string]any{
			"currentPage":  currentPage,
			"pageSize":     pageSize,
			"totalPlayers": total,
			"totalPages":   totalPages,
			"hasNextPage":  currentPage < totalPages,
			"hasPrevPage":  currentPage > 1,
		},
	}
}
