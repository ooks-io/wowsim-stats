package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/utils"
	"ookstats/internal/wow"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GeneratePlayerLeaderboards generates player ranking JSON files for all scopes per season
func GeneratePlayerLeaderboards(db *sql.DB, out string, pageSize int, regions []string) error {
	if pageSize <= 0 {
		pageSize = 25
	}

	// Default regions
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	// Load seasons
	seasons, err := loadSeasons(db)
	if err != nil {
		return err
	}

	if len(seasons) == 0 {
		fmt.Println("Warning: No seasons found - skipping player leaderboard generation")
		return nil
	}

	// Generate leaderboards for each season
	for _, season := range seasons {
		fmt.Printf("\n=== Generating Player Leaderboards for Season %d (%s) ===\n", season.ID, season.Name)
		seasonOut := filepath.Join(out, "season", fmt.Sprintf("%d", season.ID))

		// Generate global and regional scopes for this season
		if err := generateGlobalPlayerLeaderboard(db, seasonOut, pageSize, season.ID); err != nil {
			return err
		}

		for _, reg := range regions {
			if err := generateRegionalPlayerLeaderboard(db, seasonOut, reg, pageSize, season.ID); err != nil {
				return err
			}
		}

		// Generate realm scopes for this season
		for _, reg := range regions {
			if err := generateRealmPlayerLeaderboards(db, seasonOut, reg, pageSize, season.ID); err != nil {
				return err
			}
		}

		// Generate class-filtered variants for this season
		classKeys := []string{"death_knight", "druid", "hunter", "mage", "monk", "paladin", "priest", "rogue", "shaman", "warlock", "warrior"}
		for _, cls := range classKeys {
			if err := generateClassPlayerLeaderboards(db, seasonOut, cls, pageSize, regions, season.ID); err != nil {
				return err
			}
		}
	}

	fmt.Println("[OK] Player leaderboards generated for all seasons")
	return nil
}

// generateGlobalPlayerLeaderboard generates global player rankings for a season
func generateGlobalPlayerLeaderboard(db *sql.DB, out string, pageSize int, seasonID int) error {
	dir := filepath.Join(out, "players", "global")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var total int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM player_profiles
		WHERE season_id = ? AND has_complete_coverage = 1 AND combined_best_time IS NOT NULL
	`, seasonID).Scan(&total); err != nil {
		return fmt.Errorf("players total (global, season %d): %w", seasonID, err)
	}

	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		offset := (p - 1) * pageSize
		rows, err := db.Query(`
			SELECT p.id, p.name, r.slug, r.name, r.region,
				   COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
				   pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
				   COALESCE(pp.global_ranking_bracket, '')
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
			ORDER BY pp.combined_best_time ASC, p.name ASC
			LIMIT ? OFFSET ?
		`, seasonID, pageSize, offset)
		if err != nil {
			return err
		}

		list, err := scanPlayerRows(rows)
		if err != nil {
			return err
		}

		page := buildPlayerLeaderboardPage(list, "Global Player Rankings", total, pages, p, pageSize)
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// generateRegionalPlayerLeaderboard generates regional player rankings
func generateRegionalPlayerLeaderboard(db *sql.DB, out, region string, pageSize int, seasonID int) error {
	dir := filepath.Join(out, "players", "regional", region)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var total int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM player_profiles pp
		JOIN players p ON pp.player_id = p.id
		JOIN realms r ON p.realm_id = r.id
		WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL AND r.region = ?
	`, seasonID, region).Scan(&total); err != nil {
		return fmt.Errorf("players total (regional, season %d): %w", seasonID, err)
	}

	pages := (total + pageSize - 1) / pageSize
	for p := 1; p <= pages; p++ {
		offset := (p - 1) * pageSize
		rows, err := db.Query(`
			SELECT p.id, p.name, r.slug, r.name, r.region,
				   COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
				   pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
				   COALESCE(pp.regional_ranking_bracket, '')
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.season_id = ? AND r.region = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
			ORDER BY pp.combined_best_time ASC, p.name ASC
			LIMIT ? OFFSET ?
		`, seasonID, region, pageSize, offset)
		if err != nil {
			return err
		}

		list, err := scanPlayerRows(rows)
		if err != nil {
			return err
		}

		title := strings.ToUpper(region) + " Player Rankings"
		page := buildPlayerLeaderboardPage(list, title, total, pages, p, pageSize)
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// generateRealmPlayerLeaderboards generates realm-specific player rankings for a season
func generateRealmPlayerLeaderboards(db *sql.DB, out, region string, pageSize int, seasonID int) error {
	rrows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, region)
	if err != nil {
		return fmt.Errorf("players realms list: %w", err)
	}
	defer rrows.Close()

	var slugs []string
	for rrows.Next() {
		var s string
		if err := rrows.Scan(&s); err != nil {
			return err
		}
		slugs = append(slugs, s)
	}

	groups := groupRealmSlugs(region, slugs)
	if len(groups) == 0 {
		return nil
	}

	for _, group := range groups {
		if len(group.Slugs) == 0 {
			continue
		}
		dir := filepath.Join(out, "players", "realm", region, group.Parent)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		placeholder := makeSQLPlaceholders(len(group.Slugs))
		baseArgs := append([]interface{}{seasonID, region}, stringSliceToInterface(group.Slugs)...)

		var total int
		countQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			WHERE pp.season_id = ? AND r.region = ? AND r.slug IN (%s) AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
		`, placeholder)
		if err := db.QueryRow(countQuery, baseArgs...).Scan(&total); err != nil {
			return fmt.Errorf("players total (realm, season %d): %w", seasonID, err)
		}

		pages := (total + pageSize - 1) / pageSize
		for p := 1; p <= pages; p++ {
			offset := (p - 1) * pageSize
			dataArgs := append(append([]interface{}{}, baseArgs...), pageSize, offset)
			rows, err := db.Query(fmt.Sprintf(`
				SELECT p.id, p.name, r.slug, r.name, r.region,
				       COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
				       pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
				       COALESCE(pp.realm_ranking_bracket, '')
				FROM players p
				JOIN realms r ON p.realm_id = r.id
				JOIN player_profiles pp ON p.id = pp.player_id
				LEFT JOIN player_details pd ON p.id = pd.player_id
				WHERE pp.season_id = ? AND r.region = ? AND r.slug IN (%s) AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
				ORDER BY pp.combined_best_time ASC, p.name ASC
				LIMIT ? OFFSET ?
			`, placeholder), dataArgs...)
			if err != nil {
				return err
			}

			list, err := scanPlayerRows(rows)
			if err != nil {
				return err
			}

			applyPercentileBrackets(list, offset, total)

			title := strings.ToUpper(region) + "/" + group.Parent + " Player Rankings"
			page := buildPlayerLeaderboardPage(list, title, total, pages, p, pageSize)
			if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateClassPlayerLeaderboards generates class-filtered player rankings for a season
func generateClassPlayerLeaderboards(db *sql.DB, out, classKey string, pageSize int, regions []string, seasonID int) error {
	// Global class leaderboard
	if err := generateClassScope(db, out, "global", "", "", classKey, pageSize, seasonID); err != nil {
		return err
	}

	// Regional class leaderboards
	for _, reg := range regions {
		if err := generateClassScope(db, out, "regional", reg, "", classKey, pageSize, seasonID); err != nil {
			return err
		}

		// Realm class leaderboards
		rrows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, reg)
		if err != nil {
			return fmt.Errorf("players class realms list: %w", err)
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

		seenParents := make(map[string]struct{})
		for _, rslug := range slugs {
			parentSlug := effectiveRealmSlug(reg, rslug)
			if parentSlug == "" {
				parentSlug = rslug
			}
			if _, exists := seenParents[parentSlug]; exists {
				continue
			}
			seenParents[parentSlug] = struct{}{}

			if err := generateClassScope(db, out, "realm", reg, parentSlug, classKey, pageSize, seasonID); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateClassScope generates a class-filtered leaderboard for a specific scope and season
func generateClassScope(db *sql.DB, out, scope, region, realmSlug, classKey string, pageSize int, seasonID int) error {
	var dir string
	if scope == "global" {
		dir = filepath.Join(out, "players", "class", classKey, "global")
	} else if scope == "regional" {
		dir = filepath.Join(out, "players", "class", classKey, "regional", region)
	} else {
		effectiveSlug := effectiveRealmSlug(region, realmSlug)
		if effectiveSlug == "" {
			effectiveSlug = realmSlug
		}
		realmSlug = effectiveSlug
		dir = filepath.Join(out, "players", "class", classKey, "realm", region, effectiveSlug)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Fetch ALL players for this season, then filter by class in Go (to handle fallback logic)
	var rows *sql.Rows
	var err error
	var bracketColumn string
	if scope == "global" {
		bracketColumn = "pp.global_ranking_bracket"
		rows, err = db.Query(`
			SELECT p.id, p.name, r.slug, r.name, r.region,
				   COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
				   pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
				   COALESCE(pp.global_ranking_bracket, '')
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.season_id = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
			ORDER BY pp.combined_best_time ASC, p.name ASC
		`, seasonID)
	} else if scope == "regional" {
		bracketColumn = "pp.regional_ranking_bracket"
		rows, err = db.Query(`
			SELECT p.id, p.name, r.slug, r.name, r.region,
				   COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
				   pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
				   COALESCE(pp.regional_ranking_bracket, '')
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.season_id = ? AND r.region = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
			ORDER BY pp.combined_best_time ASC, p.name ASC
		`, seasonID, region)
	} else {
		bracketColumn = "pp.realm_ranking_bracket"
		realmSlugs := realmGroupSlugs(region, realmSlug)
		if len(realmSlugs) == 0 {
			realmSlugs = []string{realmSlug}
		}
		placeholder := makeSQLPlaceholders(len(realmSlugs))
		args := append([]interface{}{seasonID, region}, stringSliceToInterface(realmSlugs)...)
		rows, err = db.Query(fmt.Sprintf(`
			SELECT p.id, p.name, r.slug, r.name, r.region,
			       COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
			       pp.combined_best_time, pp.dungeons_completed, pp.total_runs,
			       COALESCE(pp.realm_ranking_bracket, '')
			FROM players p
			JOIN realms r ON p.realm_id = r.id
			JOIN player_profiles pp ON p.id = pp.player_id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.season_id = ? AND r.region = ? AND r.slug IN (%s) AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
			ORDER BY pp.combined_best_time ASC, p.name ASC
		`, placeholder), args...)
	}

	_ = bracketColumn // unused for now, but documents intent
	if err != nil {
		return err
	}
	defer rows.Close()

	type PlayerRow struct {
		ID                                                            int64
		Name, RealmSlug, RealmName, Region, ClassName, ActiveSpecName string
		MainSpecID                                                    sql.NullInt64
		CombinedBestTime                                              sql.NullInt64
		DungeonsCompleted, TotalRuns                                  int
		RankingBracket                                                string
	}

	// Collect all matching players (filter by class in Go)
	var allMatching []PlayerRow
	for rows.Next() {
		var r PlayerRow
		if err := rows.Scan(&r.ID, &r.Name, &r.RealmSlug, &r.RealmName, &r.Region, &r.ClassName, &r.ActiveSpecName, &r.MainSpecID, &r.CombinedBestTime, &r.DungeonsCompleted, &r.TotalRuns, &r.RankingBracket); err != nil {
			return err
		}

		// Derive class from main_spec_id if missing
		derivedClass := r.ClassName
		derivedSpec := r.ActiveSpecName
		if (derivedClass == "" || derivedSpec == "") && r.MainSpecID.Valid {
			if cls, spec, ok := wow.GetClassAndSpec(int(r.MainSpecID.Int64)); ok {
				if derivedClass == "" {
					derivedClass = cls
				}
				if derivedSpec == "" {
					derivedSpec = spec
				}
			}
		}

		// Filter by class (case-insensitive, space -> underscore)
		playerClassKey := strings.ReplaceAll(strings.ToLower(derivedClass), " ", "_")
		if playerClassKey == classKey {
			r.ClassName = derivedClass
			r.ActiveSpecName = derivedSpec
			allMatching = append(allMatching, r)
		}
	}

	// Paginate the filtered results
	total := len(allMatching)
	pages := (total + pageSize - 1) / pageSize

	for p := 1; p <= pages; p++ {
		offset := (p - 1) * pageSize
		end := offset + pageSize
		if end > total {
			end = total
		}

		var list []map[string]any
		for _, r := range allMatching[offset:end] {
			obj := map[string]any{
				"player_id":          r.ID,
				"name":               r.Name,
				"realm_slug":         r.RealmSlug,
				"realm_name":         r.RealmName,
				"region":             r.Region,
				"class_name":         r.ClassName,
				"active_spec_name":   r.ActiveSpecName,
				"dungeons_completed": r.DungeonsCompleted,
				"total_runs":         r.TotalRuns,
			}
			if r.RankingBracket != "" {
				obj["ranking_percentile"] = r.RankingBracket
			}
			if r.MainSpecID.Valid {
				obj["main_spec_id"] = int(r.MainSpecID.Int64)
			}
			if r.CombinedBestTime.Valid {
				obj["combined_best_time"] = r.CombinedBestTime.Int64
			}
			list = append(list, obj)
		}

		if scope == "realm" {
			applyPercentileBrackets(list, offset, total)
		}

		title := "Global Player Rankings"
		if scope == "regional" {
			title = strings.ToUpper(region) + " Player Rankings"
		} else if scope == "realm" {
			title = strings.ToUpper(region) + "/" + realmSlug + " Player Rankings"
		}

		page := buildPlayerLeaderboardPage(list, title, total, pages, p, pageSize)
		if err := writer.WriteJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil {
			return err
		}
	}
	return nil
}

// scanPlayerRows scans player rows and applies class/spec fallback
func scanPlayerRows(rows *sql.Rows) ([]map[string]any, error) {
	defer rows.Close()

	type PlayerRow struct {
		ID                int64
		Name              string
		RealmSlug         string
		RealmName         string
		Region            string
		ClassName         string
		ActiveSpecName    string
		MainSpecID        sql.NullInt64
		CombinedBestTime  sql.NullInt64
		DungeonsCompleted int
		TotalRuns         int
		RankingBracket    string
	}

	var list []map[string]any
	for rows.Next() {
		var r PlayerRow
		if err := rows.Scan(&r.ID, &r.Name, &r.RealmSlug, &r.RealmName, &r.Region, &r.ClassName, &r.ActiveSpecName, &r.MainSpecID, &r.CombinedBestTime, &r.DungeonsCompleted, &r.TotalRuns, &r.RankingBracket); err != nil {
			return nil, err
		}

		obj := map[string]any{
			"player_id":          r.ID,
			"name":               r.Name,
			"realm_slug":         r.RealmSlug,
			"realm_name":         r.RealmName,
			"region":             r.Region,
			"class_name":         r.ClassName,
			"active_spec_name":   r.ActiveSpecName,
			"dungeons_completed": r.DungeonsCompleted,
			"total_runs":         r.TotalRuns,
		}

		if r.RankingBracket != "" {
			obj["ranking_percentile"] = r.RankingBracket
		}

		// Apply class/spec fallback if missing
		if r.MainSpecID.Valid {
			v := int(r.MainSpecID.Int64)
			obj["class_name"], obj["active_spec_name"] = wow.FallbackClassAndSpec(r.ClassName, r.ActiveSpecName, &v)
			obj["main_spec_id"] = v
		}

		if r.CombinedBestTime.Valid {
			obj["combined_best_time"] = r.CombinedBestTime.Int64
		}
		list = append(list, obj)
	}
	return list, nil
}

// buildPlayerLeaderboardPage builds a player leaderboard page structure
func buildPlayerLeaderboardPage(leaderboard []map[string]any, title string, total, totalPages, currentPage, pageSize int) map[string]any {
	return map[string]any{
		"leaderboard":         leaderboard,
		"title":               title,
		"generated_timestamp": time.Now().UnixMilli(),
		"pagination": map[string]any{
			"currentPage":  currentPage,
			"pageSize":     pageSize,
			"totalPlayers": total,
			"totalPages":   totalPages,
			"hasNextPage":  currentPage < totalPages,
			"hasPrevPage":  currentPage > 1,
			"totalRuns":    total, // keep compatibility with frontend
		},
	}
}

func applyPercentileBrackets(entries []map[string]any, offset int, total int) {
	if total <= 0 {
		return
	}
	for i, entry := range entries {
		rank := offset + i + 1
		if rank <= 0 {
			continue
		}
		bracket := utils.CalculatePercentileBracket(rank, total)
		if bracket == "" {
			delete(entry, "ranking_percentile")
		} else {
			entry["ranking_percentile"] = bracket
		}
	}
}
