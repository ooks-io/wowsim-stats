package generator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/utils"
	"ookstats/internal/wow"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type PlayerJSON struct {
	ID                int64                        `json:"id"`
	Name              string                       `json:"name"`
	RealmSlug         string                       `json:"realm_slug"`
	RealmName         string                       `json:"realm_name"`
	Region            string                       `json:"region"`
	ClassName         string                       `json:"class_name,omitempty"`
	ActiveSpecName    string                       `json:"active_spec_name,omitempty"`
	AvatarURL         string                       `json:"avatar_url,omitempty"`
	GuildName         string                       `json:"guild_name,omitempty"`
	RaceName          string                       `json:"race_name,omitempty"`
	AverageItemLevel  *int                         `json:"average_item_level,omitempty"`
	EquippedItemLevel *int                         `json:"equipped_item_level,omitempty"`
	AllTimeTotalRuns  int                          `json:"all_time_total_runs"`
	Seasons           map[string]PlayerSeasonJSON  `json:"seasons"`
	Alts              []AccountCharJSON            `json:"alts,omitempty"`
	Account           map[string]AccountSeasonJSON `json:"account,omitempty"`
	// not season-scoped, matches the Total Runs leaderboard
	AccountLifetimeTotalRuns int `json:"account_lifetime_total_runs,omitempty"`
}

type AccountSeasonJSON struct {
	AccountID         int64  `json:"account_id"`
	CombinedBestTime  *int64 `json:"combined_best_time,omitempty"`
	DungeonsCompleted int    `json:"dungeons_completed"`
	TotalRuns         int    `json:"total_runs"`
	CharacterCount    int    `json:"character_count"`
	HasFullCoverage   bool   `json:"has_full_coverage"`
	GlobalRanking     *int   `json:"global_ranking,omitempty"`
	RegionalRanking   *int   `json:"regional_ranking,omitempty"`
	RealmRanking      *int   `json:"realm_ranking,omitempty"`
	GlobalBracket     string `json:"global_ranking_bracket,omitempty"`
	RegionalBracket   string `json:"regional_ranking_bracket,omitempty"`
	RealmBracket      string `json:"realm_ranking_bracket,omitempty"`
	Region            string `json:"region,omitempty"`
	RealmSlug         string `json:"realm_slug,omitempty"`
	RealmName         string `json:"realm_name,omitempty"`
	MainPlayerID      int64  `json:"main_player_id,omitempty"`
}

// keep JSON tags in sync with AccountCharSummary on the frontend
type AccountCharJSON struct {
	PlayerID       int64  `json:"player_id"`
	Name           string `json:"name"`
	RealmSlug      string `json:"realm_slug"`
	RealmName      string `json:"realm_name,omitempty"`
	Region         string `json:"region"`
	ClassName      string `json:"class_name,omitempty"`
	ActiveSpecName string `json:"active_spec_name,omitempty"`
	MainSpecID     *int   `json:"main_spec_id,omitempty"`
}

type PlayerSeasonJSON struct {
	MainSpecID        *int                   `json:"main_spec_id,omitempty"`
	DungeonsCompleted int                    `json:"dungeons_completed"`
	TotalRuns         int                    `json:"total_runs"`
	CombinedBestTime  *int64                 `json:"combined_best_time,omitempty"`
	GlobalRanking     *int                   `json:"global_ranking,omitempty"`
	RegionalRanking   *int                   `json:"regional_ranking,omitempty"`
	RealmRanking      *int                   `json:"realm_ranking,omitempty"`
	GlobalBracket     string                 `json:"global_ranking_bracket,omitempty"`
	RegionalBracket   string                 `json:"regional_ranking_bracket,omitempty"`
	RealmBracket      string                 `json:"realm_ranking_bracket,omitempty"`
	LastUpdated       *int64                 `json:"last_updated,omitempty"`
	BestRuns          map[string]BestRunJSON `json:"best_runs"`
}

type TeamMemberJSON struct {
	Name      string `json:"name"`
	SpecID    *int   `json:"spec_id,omitempty"`
	Region    string `json:"region"`
	RealmSlug string `json:"realm_slug"`
}

type BestRunJSON struct {
	DungeonID          int    `json:"dungeon_id"`
	DungeonName        string `json:"dungeon_name"`
	DungeonSlug        string `json:"dungeon_slug"`
	RunID              int64  `json:"run_id"`
	Duration           int64  `json:"duration"`
	CompletedTimestamp int64  `json:"completed_timestamp"`
	// always populated; only meaningful to consumers in All Time view
	SeasonID                int              `json:"season_id,omitempty"`
	GlobalRankingFiltered   *int             `json:"global_ranking_filtered,omitempty"`
	RegionalRankingFiltered *int             `json:"regional_ranking_filtered,omitempty"`
	RealmRankingFiltered    *int             `json:"realm_ranking_filtered,omitempty"`
	GlobalBracket           string           `json:"global_percentile_bracket,omitempty"`
	RegionalBracket         string           `json:"regional_percentile_bracket,omitempty"`
	RealmBracket            string           `json:"realm_percentile_bracket,omitempty"`
	TeamMembers             []TeamMemberJSON `json:"team_members"`
}

// PlayerPageJSON represents the complete player page output
type PlayerPageJSON struct {
	Player      PlayerJSON     `json:"player"`
	Equipment   map[string]any `json:"equipment"`
	GeneratedAt int64          `json:"generated_at"`
	Version     string         `json:"version"`
}

// GeneratePlayers orchestrates the full player JSON generation pipeline
func GeneratePlayers(db *sql.DB, out string, version string) error {
	fmt.Println("Generating player JSON endpoints...")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("mkdir players out: %w", err)
	}

	// Step 1: Load all players with complete coverage
	fmt.Printf("Loading players with complete coverage...\n")
	players, err := loader.LoadAllCompleteCoveragePlayers(db)
	if err != nil {
		return fmt.Errorf("load players: %w", err)
	}
	fmt.Printf("[OK] Loaded %d players with complete coverage\n", len(players))

	if len(players) == 0 {
		fmt.Println("No players with complete coverage found")
		return nil
	}

	// Step 2: Load player season data
	fmt.Printf("Loading player season data...\n")
	playerSeasonsMap, err := loader.LoadAllPlayerSeasons(db, loader.GetPlayerIDs(players))
	if err != nil {
		return fmt.Errorf("load player seasons: %w", err)
	}
	fmt.Printf("[OK] Loaded season data for %d players\n", len(playerSeasonsMap))

	// Step 3: Batch load all supporting data
	fmt.Printf("Loading best runs data...\n")
	bestRunsMap, allRunIDs, err := loader.LoadAllBestRuns(db, loader.GetPlayerIDs(players))
	if err != nil {
		return fmt.Errorf("load best runs: %w", err)
	}
	fmt.Printf("[OK] Loaded best runs for %d players (%d total runs)\n", len(bestRunsMap), len(allRunIDs))

	fmt.Printf("Loading team members...\n")
	teamMembersMap, err := loader.LoadAllTeamMembers(db, allRunIDs)
	if err != nil {
		return fmt.Errorf("load team members: %w", err)
	}
	fmt.Printf("[OK] Loaded team members for %d runs\n", len(teamMembersMap))

	fmt.Printf("Loading equipment data...\n")
	equipmentMap, enchantmentsMap, err := loader.LoadAllEquipment(db, loader.GetPlayerIDs(players))
	if err != nil {
		return fmt.Errorf("load equipment: %w", err)
	}
	fmt.Printf("[OK] Loaded equipment for %d players\n", len(equipmentMap))

	fmt.Printf("Loading account alts...\n")
	altsMap, err := loader.LoadAllAccountAlts(db)
	if err != nil {
		return fmt.Errorf("load account alts: %w", err)
	}
	fmt.Printf("[OK] Loaded alts for %d players\n", len(altsMap))

	fmt.Printf("Loading account season rankings...\n")
	regions := []string{"us", "eu", "kr", "tw"}
	seasons, err := loadSeasons(db)
	if err != nil {
		return fmt.Errorf("load seasons for account rankings: %w", err)
	}
	accountSeasonStats := make(map[int]map[int64]*loader.AccountSeasonStats, len(seasons))
	for _, season := range seasons {
		stats, err := loader.LoadAccountSeasonStats(db, season.ID, regions)
		if err != nil {
			return fmt.Errorf("load account season %d stats: %w", season.ID, err)
		}
		accountSeasonStats[season.ID] = stats
	}
	fmt.Printf("[OK] Loaded account rankings across %d seasons\n", len(seasons))

	fmt.Printf("Loading all-time account rankings...\n")
	allTimeAccountStats, err := loader.LoadAllTimeAccountStats(db, regions)
	if err != nil {
		return fmt.Errorf("load all-time account stats: %w", err)
	}
	fmt.Printf("[OK] Loaded all-time account rankings (%d accounts)\n", len(allTimeAccountStats))

	fmt.Printf("Loading all-time character rankings...\n")
	allTimeCharRanks, err := loader.LoadAllTimeCharacterRanks(db, regions)
	if err != nil {
		return fmt.Errorf("load all-time char ranks: %w", err)
	}
	fmt.Printf("[OK] Loaded all-time character rankings (%d chars)\n", len(allTimeCharRanks))

	accountLifetimeRuns := make(map[int64]int)
	for _, byAccount := range accountSeasonStats {
		for accountID, stats := range byAccount {
			accountLifetimeRuns[accountID] += stats.TotalRuns
		}
	}

	playerAccountIDs, err := loader.LoadPlayerAccountIDs(db)
	if err != nil {
		return fmt.Errorf("load player account ids: %w", err)
	}

	// Step 4: Process players concurrently
	fmt.Printf("Generating JSON files concurrently...\n")
	return GeneratePlayerJSONs(players, playerSeasonsMap, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, altsMap, accountSeasonStats, accountLifetimeRuns, allTimeAccountStats, allTimeCharRanks, playerAccountIDs, out, version)
}

// GeneratePlayerJSONs generates JSON files for all players concurrently
func GeneratePlayerJSONs(players []loader.PlayerData, playerSeasonsMap map[int64][]loader.PlayerSeasonData, bestRunsMap map[int64][]loader.BestRunData, teamMembersMap map[int64][]loader.TeamMemberData, equipmentMap map[int64][]loader.EquipmentData, enchantmentsMap map[int64][]loader.EnchantmentData, altsMap map[int64][]loader.AltSummary, accountSeasonStats map[int]map[int64]*loader.AccountSeasonStats, accountLifetimeRuns map[int64]int, allTimeAccountStats map[int64]*loader.AccountSeasonStats, allTimeCharRanks map[int64]*loader.CharacterAllTimeRanks, playerAccountIDs map[int64]int64, out, version string) error {
	startTime := time.Now()
	const batchSize = 100
	const numWorkers = 10

	// Channel for work items
	type workItem struct {
		player loader.PlayerData
		index  int
	}

	workChan := make(chan workItem, batchSize)
	errChan := make(chan error, numWorkers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workChan {
				if err := generateSinglePlayerJSON(item.player, playerSeasonsMap, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, altsMap, accountSeasonStats, accountLifetimeRuns, allTimeAccountStats, allTimeCharRanks, playerAccountIDs, out, version); err != nil {
					errChan <- fmt.Errorf("player %s: %w", item.player.Name, err)
					return
				}
				if (item.index+1)%500 == 0 {
					fmt.Printf("  ... %d players generated\n", item.index+1)
				}
			}
		}()
	}

	// Send work items
	go func() {
		defer close(workChan)
		for i, player := range players {
			workChan <- workItem{player: player, index: i}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		return err
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[OK] Generated %d player JSON files in %v\n", len(players), elapsed)
	return nil
}

// generateSinglePlayerJSON generates a JSON file for a single player
func generateSinglePlayerJSON(player loader.PlayerData, playerSeasonsMap map[int64][]loader.PlayerSeasonData, bestRunsMap map[int64][]loader.BestRunData, teamMembersMap map[int64][]loader.TeamMemberData, equipmentMap map[int64][]loader.EquipmentData, enchantmentsMap map[int64][]loader.EnchantmentData, altsMap map[int64][]loader.AltSummary, accountSeasonStats map[int]map[int64]*loader.AccountSeasonStats, accountLifetimeRuns map[int64]int, allTimeAccountStats map[int64]*loader.AccountSeasonStats, allTimeCharRanks map[int64]*loader.CharacterAllTimeRanks, playerAccountIDs map[int64]int64, out, version string) error {
	accountID := playerAccountIDs[player.ID]
	// Build PlayerJSON with base info
	pj := PlayerJSON{
		ID:                       player.ID,
		Name:                     player.Name,
		RealmSlug:                player.RealmSlug,
		RealmName:                player.RealmName,
		Region:                   player.Region,
		ClassName:                player.ClassName.String,
		ActiveSpecName:           player.ActiveSpecName.String,
		AvatarURL:                player.AvatarURL,
		GuildName:                player.GuildName.String,
		RaceName:                 player.RaceName.String,
		Seasons:                  make(map[string]PlayerSeasonJSON),
		Alts:                     projectAlts(altsMap[player.ID]),
		Account:                  projectAccount(accountID, accountSeasonStats),
		AccountLifetimeTotalRuns: accountLifetimeRuns[accountID],
	}

	if player.AverageItemLevel.Valid {
		v := int(player.AverageItemLevel.Int64)
		pj.AverageItemLevel = &v
	}
	if player.EquippedItemLevel.Valid {
		v := int(player.EquippedItemLevel.Int64)
		pj.EquippedItemLevel = &v
	}

	for _, seasonData := range playerSeasonsMap[player.ID] {
		pj.AllTimeTotalRuns += seasonData.TotalRuns
		seasonJSON := PlayerSeasonJSON{
			DungeonsCompleted: seasonData.DungeonsCompleted,
			TotalRuns:         seasonData.TotalRuns,
			GlobalBracket:     seasonData.GlobalBracket.String,
			RegionalBracket:   seasonData.RegionalBracket.String,
			RealmBracket:      seasonData.RealmBracket.String,
			BestRuns:          make(map[string]BestRunJSON),
		}

		if seasonData.MainSpecID.Valid {
			v := int(seasonData.MainSpecID.Int64)
			seasonJSON.MainSpecID = &v

			// Fallback: if class/spec missing at player level, derive from main_spec_id
			if pj.ClassName == "" || pj.ActiveSpecName == "" {
				if cls, spec, ok := wow.GetClassAndSpec(v); ok {
					if pj.ClassName == "" {
						pj.ClassName = cls
					}
					if pj.ActiveSpecName == "" {
						pj.ActiveSpecName = spec
					}
				}
			}
		}
		if seasonData.CombinedBest.Valid {
			v := seasonData.CombinedBest.Int64
			seasonJSON.CombinedBestTime = &v
		}
		if seasonData.GlobalRanking.Valid {
			v := int(seasonData.GlobalRanking.Int64)
			seasonJSON.GlobalRanking = &v
		}
		if seasonData.RegionalRanking.Valid {
			v := int(seasonData.RegionalRanking.Int64)
			seasonJSON.RegionalRanking = &v
		}
		if seasonData.RealmRanking.Valid {
			v := int(seasonData.RealmRanking.Int64)
			seasonJSON.RealmRanking = &v
		}
		if seasonData.LastUpdated.Valid {
			v := seasonData.LastUpdated.Int64
			seasonJSON.LastUpdated = &v
		}

		pj.Seasons[fmt.Sprintf("%d", seasonData.SeasonID)] = seasonJSON
	}

	// Build best runs and organize by season
	for _, run := range bestRunsMap[player.ID] {
		br := BestRunJSON{
			DungeonID:          int(run.DungeonID),
			DungeonName:        run.DungeonName,
			DungeonSlug:        run.DungeonSlug,
			RunID:              run.RunID,
			Duration:           run.Duration,
			CompletedTimestamp: run.CompletedTimestamp,
			SeasonID:           run.SeasonID,
			GlobalBracket:      run.GlobalBracket,
			RegionalBracket:    run.RegionalBracket,
			RealmBracket:       run.RealmBracket,
		}

		if run.GlobalRankingFiltered.Valid {
			v := int(run.GlobalRankingFiltered.Int64)
			br.GlobalRankingFiltered = &v
		}
		if run.RegionalRankingFiltered.Valid {
			v := int(run.RegionalRankingFiltered.Int64)
			br.RegionalRankingFiltered = &v
		}
		if run.RealmRankingFiltered.Valid {
			v := int(run.RealmRankingFiltered.Int64)
			br.RealmRankingFiltered = &v
		}

		// Add team members
		for _, member := range teamMembersMap[run.RunID] {
			tm := TeamMemberJSON{
				Name:      member.Name,
				Region:    member.Region,
				RealmSlug: member.RealmSlug,
			}
			if member.SpecID.Valid {
				v := int(member.SpecID.Int64)
				tm.SpecID = &v
			}
			br.TeamMembers = append(br.TeamMembers, tm)
		}

		// Add best run to the appropriate season
		seasonKey := fmt.Sprintf("%d", run.SeasonID)
		if season, exists := pj.Seasons[seasonKey]; exists {
			season.BestRuns[run.DungeonSlug] = br
			pj.Seasons[seasonKey] = season
		}
	}

	if rk, ok := allTimeCharRanks[player.ID]; ok {
		allBest := buildAllTimeBestRuns(bestRunsMap[player.ID], teamMembersMap)
		atSeason := PlayerSeasonJSON{
			DungeonsCompleted: len(allBest),
			TotalRuns:         pj.AllTimeTotalRuns,
			BestRuns:          allBest,
			GlobalBracket:     rk.GlobalBracket,
			RegionalBracket:   rk.RegionalBracket,
			RealmBracket:      rk.RealmBracket,
		}
		if rk.Stats != nil {
			t := rk.Stats.CombinedBest
			atSeason.CombinedBestTime = &t
			if rk.Stats.MainSpecID.Valid {
				v := int(rk.Stats.MainSpecID.Int64)
				atSeason.MainSpecID = &v
			}
		}
		if rk.GlobalRanking > 0 {
			r := rk.GlobalRanking
			atSeason.GlobalRanking = &r
		}
		if rk.RegionalRanking > 0 {
			r := rk.RegionalRanking
			atSeason.RegionalRanking = &r
		}
		if rk.RealmRanking > 0 {
			r := rk.RealmRanking
			atSeason.RealmRanking = &r
		}
		pj.Seasons["all-time"] = atSeason
	}

	if accountID := playerAccountIDs[player.ID]; accountID != 0 {
		if s, ok := allTimeAccountStats[accountID]; ok {
			entry := AccountSeasonJSON{
				AccountID:         s.AccountID,
				DungeonsCompleted: s.DungeonsCompleted,
				TotalRuns:         s.TotalRuns,
				CharacterCount:    s.CharacterCount,
				HasFullCoverage:   s.HasFullCoverage,
				GlobalBracket:     s.GlobalBracket,
				RegionalBracket:   s.RegionalBracket,
				RealmBracket:      s.RealmBracket,
				Region:            s.Region,
				RealmSlug:         s.RealmSlug,
				RealmName:         s.RealmName,
				MainPlayerID:      s.MainPlayerID,
			}
			if s.HasFullCoverage {
				t := s.CombinedBestTime
				entry.CombinedBestTime = &t
			}
			if s.GlobalRanking > 0 {
				r := s.GlobalRanking
				entry.GlobalRanking = &r
			}
			if s.RegionalRanking > 0 {
				r := s.RegionalRanking
				entry.RegionalRanking = &r
			}
			if s.RealmRanking > 0 {
				r := s.RealmRanking
				entry.RealmRanking = &r
			}
			if pj.Account == nil {
				pj.Account = make(map[string]AccountSeasonJSON)
			}
			pj.Account["all-time"] = entry
		}
	}

	// Build equipment
	equipment := make(map[string]any)
	for _, eq := range equipmentMap[player.ID] {
		eqData := map[string]any{
			"id":                 eq.ID,
			"slot_type":          eq.SlotType,
			"item_id":            nil,
			"upgrade_id":         nil,
			"quality":            eq.Quality,
			"item_name":          eq.ItemName,
			"snapshot_timestamp": eq.SnapshotTs,
			"item_icon_slug":     eq.ItemIcon.String,
			"item_type":          eq.ItemType.String,
			"enchantments":       []map[string]any{},
		}

		if eq.ItemID.Valid {
			eqData["item_id"] = int(eq.ItemID.Int64)
		}
		if eq.UpgradeID.Valid {
			eqData["upgrade_id"] = int(eq.UpgradeID.Int64)
		}
		if eq.ItemStats.Valid && eq.ItemStats.String != "" && eq.ItemStats.String != "{}" {
			var scalingOpts any
			if err := json.Unmarshal([]byte(eq.ItemStats.String), &scalingOpts); err == nil {
				eqData["scaling_options"] = scalingOpts
			}
		}
		if eq.ItemEffect.Valid && eq.ItemEffect.String != "" {
			var itemEffect any
			if err := json.Unmarshal([]byte(eq.ItemEffect.String), &itemEffect); err == nil {
				eqData["item_effect"] = itemEffect
			}
		}
		if eq.SpellDescription.Valid && eq.SpellDescription.String != "" {
			eqData["spell_description"] = eq.SpellDescription.String
		}

		// Add enchantments
		for _, ench := range enchantmentsMap[eq.ID] {
			enchData := map[string]any{
				"enchantment_id":   nil,
				"slot_id":          nil,
				"slot_type":        ench.SlotType.String,
				"display_string":   ench.DisplayString.String,
				"source_item_id":   nil,
				"source_item_name": ench.SourceItemName.String,
				"spell_id":         nil,
			}

			if ench.EnchantmentID.Valid {
				enchData["enchantment_id"] = int(ench.EnchantmentID.Int64)
			}
			if ench.SlotID.Valid {
				enchData["slot_id"] = int(ench.SlotID.Int64)
			}
			if ench.SourceItemID.Valid {
				enchData["source_item_id"] = int(ench.SourceItemID.Int64)
			}
			if ench.SpellID.Valid {
				enchData["spell_id"] = int(ench.SpellID.Int64)
			}
			if ench.GemIconSlug.Valid {
				enchData["gem_icon_slug"] = ench.GemIconSlug.String
			}

			if arr, ok := eqData["enchantments"].([]map[string]any); ok {
				eqData["enchantments"] = append(arr, enchData)
			} else {
				eqData["enchantments"] = []map[string]any{enchData}
			}
		}

		equipment[eq.SlotType] = eqData
	}

	// Create final JSON
	page := PlayerPageJSON{
		Player:      pj,
		Equipment:   equipment,
		GeneratedAt: time.Now().UnixMilli(),
		Version:     version,
	}

	// Write file
	dir := filepath.Join(out, pj.Region, pj.RealmSlug)
	fname := filepath.Join(dir, utils.SafeSlugName(pj.Name)+".json")
	return writer.WriteJSONFileCompact(fname, page)
}

func buildAllTimeBestRuns(runs []loader.BestRunData, teamMembersMap map[int64][]loader.TeamMemberData) map[string]BestRunJSON {
	winners := make(map[string]loader.BestRunData)
	for _, r := range runs {
		cur, ok := winners[r.DungeonSlug]
		if !ok || r.Duration < cur.Duration {
			winners[r.DungeonSlug] = r
		}
	}
	out := make(map[string]BestRunJSON, len(winners))
	for slug, r := range winners {
		br := BestRunJSON{
			DungeonID:          int(r.DungeonID),
			DungeonName:        r.DungeonName,
			DungeonSlug:        r.DungeonSlug,
			RunID:              r.RunID,
			Duration:           r.Duration,
			CompletedTimestamp: r.CompletedTimestamp,
			SeasonID:           r.SeasonID,
			GlobalBracket:      r.GlobalBracket,
			RegionalBracket:    r.RegionalBracket,
			RealmBracket:       r.RealmBracket,
		}
		if r.GlobalRankingFiltered.Valid {
			v := int(r.GlobalRankingFiltered.Int64)
			br.GlobalRankingFiltered = &v
		}
		if r.RegionalRankingFiltered.Valid {
			v := int(r.RegionalRankingFiltered.Int64)
			br.RegionalRankingFiltered = &v
		}
		if r.RealmRankingFiltered.Valid {
			v := int(r.RealmRankingFiltered.Int64)
			br.RealmRankingFiltered = &v
		}
		for _, m := range teamMembersMap[r.RunID] {
			tm := TeamMemberJSON{Name: m.Name, Region: m.Region, RealmSlug: m.RealmSlug}
			if m.SpecID.Valid {
				v := int(m.SpecID.Int64)
				tm.SpecID = &v
			}
			br.TeamMembers = append(br.TeamMembers, tm)
		}
		out[slug] = br
	}
	return out
}

func projectAccount(accountID int64, accountSeasonStats map[int]map[int64]*loader.AccountSeasonStats) map[string]AccountSeasonJSON {
	if accountID == 0 {
		return nil
	}
	out := make(map[string]AccountSeasonJSON)
	for seasonID, byAccount := range accountSeasonStats {
		stats, ok := byAccount[accountID]
		if !ok {
			continue
		}
		entry := AccountSeasonJSON{
			AccountID:         stats.AccountID,
			DungeonsCompleted: stats.DungeonsCompleted,
			TotalRuns:         stats.TotalRuns,
			CharacterCount:    stats.CharacterCount,
			HasFullCoverage:   stats.HasFullCoverage,
			GlobalBracket:     stats.GlobalBracket,
			RegionalBracket:   stats.RegionalBracket,
			RealmBracket:      stats.RealmBracket,
			Region:            stats.Region,
			RealmSlug:         stats.RealmSlug,
			RealmName:         stats.RealmName,
			MainPlayerID:      stats.MainPlayerID,
		}
		if stats.HasFullCoverage {
			t := stats.CombinedBestTime
			entry.CombinedBestTime = &t
		}
		if stats.GlobalRanking > 0 {
			r := stats.GlobalRanking
			entry.GlobalRanking = &r
		}
		if stats.RegionalRanking > 0 {
			r := stats.RegionalRanking
			entry.RegionalRanking = &r
		}
		if stats.RealmRanking > 0 {
			r := stats.RealmRanking
			entry.RealmRanking = &r
		}
		out[fmt.Sprintf("%d", seasonID)] = entry
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func projectAlts(alts []loader.AltSummary) []AccountCharJSON {
	if len(alts) == 0 {
		return nil
	}
	out := make([]AccountCharJSON, 0, len(alts))
	for _, a := range alts {
		row := AccountCharJSON{
			PlayerID:       a.PlayerID,
			Name:           a.Name,
			RealmSlug:      a.RealmSlug,
			RealmName:      a.RealmName,
			Region:         a.Region,
			ClassName:      a.ClassName.String,
			ActiveSpecName: a.ActiveSpecName.String,
		}
		if a.MainSpecID.Valid {
			v := int(a.MainSpecID.Int64)
			row.MainSpecID = &v
			if cls, spec, ok := wow.GetClassAndSpec(v); ok {
				if row.ClassName == "" {
					row.ClassName = cls
				}
				if row.ActiveSpecName == "" {
					row.ActiveSpecName = spec
				}
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
