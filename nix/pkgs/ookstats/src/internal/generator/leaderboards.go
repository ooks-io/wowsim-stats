package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"strings"
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
	ID                  int64                   `json:"id"`
	Duration            int64                   `json:"duration"`
	CompletedTimestamp  int64                   `json:"completed_timestamp"`
	KeystoneLevel       int                     `json:"keystone_level"`
	DungeonName         string                  `json:"dungeon_name"`
	RealmName           string                  `json:"realm_name"`
	Region              string                  `json:"region"`
	RankingPercentile   string                  `json:"ranking_percentile,omitempty"`
	Members             []LeaderboardMemberJSON `json:"members"`
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

// GenerateLeaderboards generates leaderboard JSON files for all scopes (global, regional, realm) per season
func GenerateLeaderboards(db *sql.DB, out string, pageSize int, regions []string) error {
	if pageSize <= 0 {
		pageSize = 25
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

	// Generate leaderboards for each season
	for _, season := range seasons {
		fmt.Printf("\n=== Season %d (%s) ===\n", season.ID, season.Name)
		seasonOut := filepath.Join(out, "season", fmt.Sprintf("%d", season.ID))

		// Process globals for this season
		fmt.Println("Generating global leaderboards...")
		for _, d := range dungeons {
			if err := generateGlobalLeaderboard(db, seasonOut, d, season.ID, pageSize); err != nil {
				return err
			}
		}

		// Regions for this season
		for _, reg := range regions {
			fmt.Printf("Generating %s leaderboards...\n", strings.ToUpper(reg))
			for _, d := range dungeons {
				if err := generateRegionalLeaderboard(db, seasonOut, reg, d, season.ID, pageSize); err != nil {
					return err
				}
			}
		}

		// Realm leaderboards for each region+realm for this season
		for _, reg := range regions {
			slugs, err := loadRealmSlugs(db, reg)
			if err != nil {
				return err
			}
			for _, rslug := range slugs {
				fmt.Printf("Generating realm %s/%s leaderboards...\n", reg, rslug)
				for _, d := range dungeons {
					if err := generateRealmLeaderboard(db, seasonOut, reg, rslug, d, season.ID, pageSize); err != nil {
						return err
					}
				}
			}
		}
	}

	fmt.Println("\n[OK] Season-scoped leaderboards generated")
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
		LEFT JOIN period_seasons ps ON cr.period_id = ps.period_id
		WHERE cr.dungeon_id = ? AND COALESCE(ps.season_id, 1) = ?
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
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
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
			LEFT JOIN period_seasons ps ON cr.period_id = ps.period_id
			WHERE cr.dungeon_id = ? AND r.region = ? AND COALESCE(ps.season_id, 1) = ?
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
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
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
			LEFT JOIN period_seasons ps ON cr.period_id = ps.period_id
			WHERE cr.dungeon_id = ? AND rr.region = ? AND rr.slug = ? AND COALESCE(ps.season_id, 1) = ?
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
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
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
