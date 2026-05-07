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

type accountTotalRunsJob struct {
	scope     string
	region    string
	realmSlug string
	out       string
	pageSize  int
}

type totalRunsCharRow struct {
	PlayerID         int64
	Name             string
	RealmSlug        string
	RealmName        string
	Region           string
	ClassName        string
	ActiveSpecName   string
	MainSpecID       sql.NullInt64
	TotalRuns        int
	CombinedBestTime sql.NullInt64
}

type accountTotalRunsRow struct {
	AccountID      int64
	Region         string
	RealmSlugMain  string
	TotalRuns      int
	CharacterCount int
	Main           map[string]any
	Alts           []map[string]any
}

func GenerateAccountTotalRunsLeaderboard(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 4
	}
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	root := filepath.Join(out, "players", "total-runs", "accounts")
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clean account total-runs tree: %w", err)
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

	rows, err := loadAccountTotalRunsRows(db, realmParents)
	if err != nil {
		return err
	}

	totalJobs := 1 + len(regions)
	for _, reg := range regions {
		totalJobs += len(realmSlugs[reg])
	}
	fmt.Printf("Generating account Total Runs leaderboard with %d workers (%d total jobs)...\n", workers, totalJobs)

	jobs := make(chan accountTotalRunsJob, 100)
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
				if err := writeAccountTotalRunsScope(job, rows); err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				c := completed.Add(1)
				if c%20 == 0 {
					fmt.Printf("  ... %d/%d account total-runs scopes generated\n", c, totalJobs)
				}
			}
		}()
	}

	jobs <- accountTotalRunsJob{scope: "global", out: out, pageSize: pageSize}
	for _, reg := range regions {
		jobs <- accountTotalRunsJob{scope: "regional", region: reg, out: out, pageSize: pageSize}
	}
	for _, reg := range regions {
		for _, rslug := range realmSlugs[reg] {
			jobs <- accountTotalRunsJob{scope: "realm", region: reg, realmSlug: rslug, out: out, pageSize: pageSize}
		}
	}
	close(jobs)
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		return err.(error)
	}
	fmt.Printf("\n[OK] Generated %d account Total Runs leaderboards\n", completed.Load())
	return nil
}

func loadAccountTotalRunsRows(db *sql.DB, realmParents map[string]string) ([]accountTotalRunsRow, error) {
	chars, err := db.Query(`
		SELECT p.account_id, p.id, p.name, r.slug, r.name, r.region,
		       COALESCE(pd.class_name, ''),
		       COALESCE(pd.active_spec_name, ''),
		       (SELECT pp.main_spec_id FROM player_profiles pp
		         WHERE pp.player_id = p.id
		         ORDER BY pp.season_id DESC LIMIT 1) AS main_spec_id,
		       (SELECT COALESCE(SUM(pp.total_runs), 0) FROM player_profiles pp
		         WHERE pp.player_id = p.id) AS total_runs,
		       (SELECT MAX(pp.combined_best_time) FROM player_profiles pp
		         WHERE pp.player_id = p.id) AS combined_best
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE p.account_id IS NOT NULL AND p.is_valid = 1
	`)
	if err != nil {
		return nil, fmt.Errorf("load account total runs chars: %w", err)
	}
	defer chars.Close()

	byAccount := make(map[int64][]*totalRunsCharRow)
	for chars.Next() {
		var accountID int64
		var c totalRunsCharRow
		if err := chars.Scan(&accountID, &c.PlayerID, &c.Name, &c.RealmSlug, &c.RealmName, &c.Region,
			&c.ClassName, &c.ActiveSpecName, &c.MainSpecID, &c.TotalRuns, &c.CombinedBestTime); err != nil {
			return nil, err
		}
		byAccount[accountID] = append(byAccount[accountID], &c)
	}
	if err := chars.Err(); err != nil {
		return nil, err
	}

	out := make([]accountTotalRunsRow, 0, len(byAccount))
	for accountID, cs := range byAccount {
		// main = top runner (not the combined-time leaderboard's fastest 9/9)
		var main *totalRunsCharRow
		var totalRuns int
		for _, c := range cs {
			totalRuns += c.TotalRuns
			if main == nil || c.TotalRuns > main.TotalRuns ||
				(c.TotalRuns == main.TotalRuns && c.PlayerID < main.PlayerID) {
				main = c
			}
		}
		if totalRuns == 0 || main == nil {
			continue
		}

		row := accountTotalRunsRow{
			AccountID:      accountID,
			Region:         main.Region,
			TotalRuns:      totalRuns,
			CharacterCount: len(cs),
			Main:           projectTotalRunsChar(main),
		}
		row.RealmSlugMain = loader.ResolveParentRealm(realmParents, main.Region, main.RealmSlug)

		alts := make([]map[string]any, 0, len(cs)-1)
		for _, c := range cs {
			if c.PlayerID == main.PlayerID {
				continue
			}
			// phantom alt
			if c.TotalRuns == 0 && c.ClassName == "" && !c.CombinedBestTime.Valid {
				continue
			}
			alts = append(alts, projectTotalRunsChar(c))
		}
		sort.Slice(alts, func(i, j int) bool {
			na, _ := alts[i]["name"].(string)
			nb, _ := alts[j]["name"].(string)
			return na < nb
		})
		row.Alts = alts

		out = append(out, row)
	}

	return out, nil
}

func projectTotalRunsChar(c *totalRunsCharRow) map[string]any {
	obj := map[string]any{
		"player_id":        c.PlayerID,
		"name":             c.Name,
		"realm_slug":       c.RealmSlug,
		"realm_name":       c.RealmName,
		"region":           c.Region,
		"class_name":       c.ClassName,
		"active_spec_name": c.ActiveSpecName,
		"total_runs":       c.TotalRuns,
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

func writeAccountTotalRunsScope(job accountTotalRunsJob, all []accountTotalRunsRow) error {
	rows := make([]accountTotalRunsRow, 0, len(all))
	for _, r := range all {
		if (job.scope == "regional" || job.scope == "realm") && !strings.EqualFold(r.Region, job.region) {
			continue
		}
		if job.scope == "realm" && !strings.EqualFold(r.RealmSlugMain, job.realmSlug) {
			continue
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalRuns != rows[j].TotalRuns {
			return rows[i].TotalRuns > rows[j].TotalRuns
		}
		return rows[i].AccountID < rows[j].AccountID
	})

	dir := filepath.Join(job.out, "players", "total-runs", "accounts", job.scope)
	switch job.scope {
	case "regional":
		dir = filepath.Join(job.out, "players", "total-runs", "accounts", "regional", job.region)
	case "realm":
		dir = filepath.Join(job.out, "players", "total-runs", "accounts", "realm", job.region, job.realmSlug)
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
			entries = append(entries, map[string]any{
				"rank":            i + 1,
				"account_id":      r.AccountID,
				"region":          r.Region,
				"total_runs":      r.TotalRuns,
				"character_count": r.CharacterCount,
				"main":            r.Main,
				"alts":            r.Alts,
			})
		}
		title := "Global Account Total Runs"
		switch job.scope {
		case "regional":
			title = strings.ToUpper(job.region) + " Account Total Runs"
		case "realm":
			title = strings.ToUpper(job.region) + " " + job.realmSlug + " Account Total Runs"
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
				"totalRuns":     total, // legacy compat
			},
		}
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}
