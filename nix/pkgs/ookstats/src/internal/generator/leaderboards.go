package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// LeaderboardPageJSON represents a paginated leaderboard response
type LeaderboardPageJSON struct {
	LeadingGroups []LeaderboardRowJSON `json:"leading_groups"`
	Map           struct {
		Name map[string]any `json:"name"`
	} `json:"map"`
	ConnectedRealm *struct {
		Name string `json:"name"`
	} `json:"connected_realm,omitempty"`
	Pagination struct {
		CurrentPage int  `json:"currentPage"`
		PageSize    int  `json:"pageSize"`
		TotalRuns   int  `json:"totalRuns"`
		TotalPages  int  `json:"totalPages"`
		HasNextPage bool `json:"hasNextPage"`
		HasPrevPage bool `json:"hasPrevPage"`
	} `json:"pagination"`
}

// LeaderboardRowJSON represents a run in the leaderboard
type LeaderboardRowJSON struct {
	ID                 int64                   `json:"id"`
	Duration           int64                   `json:"duration"`
	CompletedTimestamp int64                   `json:"completed_timestamp"`
	KeystoneLevel      int                     `json:"keystone_level"`
	DungeonName        string                  `json:"dungeon_name"`
	RealmName          string                  `json:"realm_name"`
	Region             string                  `json:"region"`
	RankingPercentile  string                  `json:"ranking_percentile,omitempty"`
	Members            []LeaderboardMemberJSON `json:"members"`
}

// LeaderboardMemberJSON represents a team member in a leaderboard row
type LeaderboardMemberJSON struct {
	Name      string `json:"name"`
	SpecID    *int   `json:"spec_id,omitempty"`
	Region    string `json:"region"`
	RealmSlug string `json:"realm_slug"`
}

// dungeonInfo holds dungeon metadata
type dungeonInfo struct {
	ID         int
	Slug, Name string
}

// leaderboardJob represents a single leaderboard generation task
type leaderboardJob struct {
	seasonID   int
	seasonName string
	dungeon    dungeonInfo
	region     string // empty for global
	realmSlug  string // empty for global/regional
	out        string
	pageSize   int
}

// GenerateLeaderboards generates leaderboard JSON files for all scopes (global, regional, realm) per season
// Uses a worker pool for parallel generation
func GenerateLeaderboards(db *sql.DB, out string, pageSize int, regions []string, workers int) error {
	if pageSize <= 0 {
		pageSize = 25
	}
	if workers <= 0 {
		workers = 10
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	// Load dungeons
	dungeons, err := loadDungeons(db)
	if err != nil {
		return err
	}

	// Load seasons
	seasons, err := loadSeasons(db)
	if err != nil {
		return err
	}

	if len(seasons) == 0 {
		fmt.Println("Warning: No seasons found - skipping leaderboard generation")
		return nil
	}

	// Regions
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	// Pre-load all realm slugs per region
	realmSlugs := make(map[string][]string)
	for _, reg := range regions {
		slugs, err := loadRealmSlugs(db, reg)
		if err != nil {
			return err
		}
		realmSlugs[reg] = slugs
	}

	// Count total jobs for progress reporting
	var totalJobs int
	for range seasons {
		totalJobs += len(dungeons) // global
		for _, reg := range regions {
			totalJobs += len(dungeons)                        // regional
			totalJobs += len(realmSlugs[reg]) * len(dungeons) // realm
		}
	}

	fmt.Printf("Generating leaderboards with %d workers (%d total jobs)...\n", workers, totalJobs)

	// Create job channel and error channel
	jobs := make(chan leaderboardJob, 100)
	var firstErr atomic.Value
	var completed atomic.Int64
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Skip if we already have an error
				if firstErr.Load() != nil {
					continue
				}

				var err error
				if job.realmSlug != "" {
					err = generateRealmLeaderboard(db, job.out, job.region, job.realmSlug, job.dungeon, job.seasonID, job.pageSize)
				} else if job.region != "" {
					err = generateRegionalLeaderboard(db, job.out, job.region, job.dungeon, job.seasonID, job.pageSize)
				} else {
					err = generateGlobalLeaderboard(db, job.out, job.dungeon, job.seasonID, job.pageSize)
				}

				if err != nil {
					firstErr.CompareAndSwap(nil, err)
					continue
				}

				c := completed.Add(1)
				if c%100 == 0 {
					fmt.Printf("  ... %d/%d leaderboards generated\n", c, totalJobs)
				}
			}
		}()
	}

	// Queue all jobs
	for _, season := range seasons {
		fmt.Printf("\n=== Queuing Season %d (%s) ===\n", season.ID, season.Name)
		seasonOut := filepath.Join(out, "season", fmt.Sprintf("%d", season.ID))

		// Global leaderboards
		for _, d := range dungeons {
			jobs <- leaderboardJob{
				seasonID:   season.ID,
				seasonName: season.Name,
				dungeon:    d,
				region:     "",
				realmSlug:  "",
				out:        seasonOut,
				pageSize:   pageSize,
			}
		}

		// Regional leaderboards
		for _, reg := range regions {
			for _, d := range dungeons {
				jobs <- leaderboardJob{
					seasonID:   season.ID,
					seasonName: season.Name,
					dungeon:    d,
					region:     reg,
					realmSlug:  "",
					out:        seasonOut,
					pageSize:   pageSize,
				}
			}
		}

		// Realm leaderboards
		for _, reg := range regions {
			for _, rslug := range realmSlugs[reg] {
				for _, d := range dungeons {
					jobs <- leaderboardJob{
						seasonID:   season.ID,
						seasonName: season.Name,
						dungeon:    d,
						region:     reg,
						realmSlug:  rslug,
						out:        seasonOut,
						pageSize:   pageSize,
					}
				}
			}
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()

	// Check for errors
	if err := firstErr.Load(); err != nil {
		return err.(error)
	}

	fmt.Printf("\n[OK] Generated %d leaderboards\n", completed.Load())
	return nil
}

// loadDungeons loads all dungeons from the database
func loadDungeons(db *sql.DB) ([]dungeonInfo, error) {
	rows, err := db.Query(`SELECT id, slug, name FROM dungeons ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("dungeons query: %w", err)
	}
	defer rows.Close()

	var dungeons []dungeonInfo
	for rows.Next() {
		var d dungeonInfo
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name); err != nil {
			return nil, err
		}
		dungeons = append(dungeons, d)
	}
	return dungeons, nil
}

// seasonInfo holds season metadata
type seasonInfo struct {
	ID   int
	Name string
}

// loadSeasons loads all distinct season numbers from the database
func loadSeasons(db *sql.DB) ([]seasonInfo, error) {
	rows, err := db.Query(`
		SELECT DISTINCT season_number, season_name
		FROM seasons
		ORDER BY season_number ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("seasons query: %w", err)
	}
	defer rows.Close()

	var seasons []seasonInfo
	for rows.Next() {
		var s seasonInfo
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

// loadRealmSlugs loads realm slugs for a given region
func loadRealmSlugs(db *sql.DB, region string) ([]string, error) {
	rows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, region)
	if err != nil {
		return nil, fmt.Errorf("realms list: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, nil
}

// generateGlobalLeaderboard generates global leaderboard pages for a dungeon
func generateGlobalLeaderboard(db *sql.DB, out string, d dungeonInfo, seasonID, pageSize int) error {
	dir := filepath.Join(out, "global", d.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Count distinct teams for this season
	var total int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT team_signature)
		FROM challenge_runs cr
		
		WHERE cr.dungeon_id = ? AND cr.season_id = ?
	`, d.ID, seasonID).Scan(&total)
	if err != nil {
		return fmt.Errorf("global count: %w", err)
	}

	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		rows, err := loader.LoadCanonicalRuns(db, d.ID, "", "", seasonID, pageSize, (p-1)*pageSize)
		if err != nil {
			return err
		}

		page := buildLeaderboardPage(rows, d.Name, "", total, pages, p, pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// generateRegionalLeaderboard generates regional leaderboard pages for a dungeon
func generateRegionalLeaderboard(db *sql.DB, out, region string, d dungeonInfo, seasonID, pageSize int) error {
	dir := filepath.Join(out, region, "all", d.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var total int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT team_signature
			FROM challenge_runs cr
			JOIN realms r ON cr.realm_id = r.id
			
			WHERE cr.dungeon_id = ? AND r.region = ? AND cr.season_id = ?
			GROUP BY team_signature
		) x
	`, d.ID, region, seasonID).Scan(&total)
	if err != nil {
		return fmt.Errorf("regional count: %w", err)
	}

	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		rows, err := loader.LoadCanonicalRuns(db, d.ID, region, "", seasonID, pageSize, (p-1)*pageSize)
		if err != nil {
			return err
		}

		page := buildLeaderboardPage(rows, d.Name, "", total, pages, p, pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// generateRealmLeaderboard generates realm leaderboard pages for a dungeon
func generateRealmLeaderboard(db *sql.DB, out, region, realmSlug string, d dungeonInfo, seasonID, pageSize int) error {
	dir := filepath.Join(out, region, realmSlug, d.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// realm display name for payload shape compatibility
	var realmName string
	if err := db.QueryRow(`SELECT name FROM realms WHERE region = ? AND slug = ?`, region, realmSlug).Scan(&realmName); err != nil {
		realmName = realmSlug
	}

	var total int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT team_signature
			FROM challenge_runs cr
			JOIN realms rr ON cr.realm_id = rr.id
			
			WHERE cr.dungeon_id = ? AND rr.region = ? AND rr.slug = ? AND cr.season_id = ?
			GROUP BY team_signature
		) x
	`, d.ID, region, realmSlug, seasonID).Scan(&total); err != nil {
		return fmt.Errorf("realm count: %w", err)
	}

	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		rows, err := loader.LoadCanonicalRuns(db, d.ID, region, realmSlug, seasonID, pageSize, (p-1)*pageSize)
		if err != nil {
			return err
		}

		page := buildLeaderboardPage(rows, d.Name, realmName, total, pages, p, pageSize)
		if err := writer.WriteJSONFileCompact(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// buildLeaderboardPage converts loader data to JSON page structure
func buildLeaderboardPage(rows []loader.LeaderboardRow, dungeonName, realmName string, total, totalPages, currentPage, pageSize int) LeaderboardPageJSON {
	page := LeaderboardPageJSON{
		LeadingGroups: make([]LeaderboardRowJSON, len(rows)),
	}

	// Convert loader types to JSON types
	for i, row := range rows {
		jsonRow := LeaderboardRowJSON{
			ID:                 row.ID,
			Duration:           row.Duration,
			CompletedTimestamp: row.CompletedTimestamp,
			KeystoneLevel:      row.KeystoneLevel,
			DungeonName:        row.DungeonName,
			RealmName:          row.RealmName,
			Region:             row.Region,
			RankingPercentile:  row.RankingPercentile,
			Members:            make([]LeaderboardMemberJSON, len(row.Members)),
		}

		for j, member := range row.Members {
			jsonRow.Members[j] = LeaderboardMemberJSON{
				Name:      member.Name,
				SpecID:    member.SpecID,
				Region:    member.Region,
				RealmSlug: member.RealmSlug,
			}
		}

		page.LeadingGroups[i] = jsonRow
	}

	page.Map.Name = map[string]any{"en_US": dungeonName}

	if realmName != "" {
		page.ConnectedRealm = &struct {
			Name string `json:"name"`
		}{Name: realmName}
	}

	page.Pagination.CurrentPage = currentPage
	page.Pagination.PageSize = pageSize
	page.Pagination.TotalRuns = total
	page.Pagination.TotalPages = totalPages
	page.Pagination.HasNextPage = currentPage < totalPages
	page.Pagination.HasPrevPage = currentPage > 1

	return page
}
