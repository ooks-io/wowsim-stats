package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/loader"
	"ookstats/internal/utils"
	"ookstats/internal/wow"
	"ookstats/internal/writer"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PlayerJSON represents the JSON structure for a player profile
type PlayerJSON struct {
	ID                int64                       `json:"id"`
	Name              string                      `json:"name"`
	RealmSlug         string                      `json:"realm_slug"`
	RealmName         string                      `json:"realm_name"`
	Region            string                      `json:"region"`
	ClassName         string                      `json:"class_name,omitempty"`
	ActiveSpecName    string                      `json:"active_spec_name,omitempty"`
	AvatarURL         string                      `json:"avatar_url,omitempty"`
	GuildName         string                      `json:"guild_name,omitempty"`
	RaceName          string                      `json:"race_name,omitempty"`
	AverageItemLevel  *int                        `json:"average_item_level,omitempty"`
	EquippedItemLevel *int                        `json:"equipped_item_level,omitempty"`
	Seasons           map[string]PlayerSeasonJSON `json:"seasons"`
}

// PlayerSeasonJSON represents a player's stats for a specific season
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

// TeamMemberJSON represents a team member in a run
type TeamMemberJSON struct {
	Name      string `json:"name"`
	SpecID    *int   `json:"spec_id,omitempty"`
	Region    string `json:"region"`
	RealmSlug string `json:"realm_slug"`
}

// BestRunJSON represents a player's best run for a dungeon
type BestRunJSON struct {
	DungeonID               int              `json:"dungeon_id"`
	DungeonName             string           `json:"dungeon_name"`
	DungeonSlug             string           `json:"dungeon_slug"`
	RunID                   int64            `json:"run_id"`
	Duration                int64            `json:"duration"`
	CompletedTimestamp      int64            `json:"completed_timestamp"`
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

	// Step 4: Process players concurrently
	fmt.Printf("Generating JSON files concurrently...\n")
	return GeneratePlayerJSONs(players, playerSeasonsMap, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, out, version)
}

// GeneratePlayerJSONs generates JSON files for all players concurrently
func GeneratePlayerJSONs(players []loader.PlayerData, playerSeasonsMap map[int64][]loader.PlayerSeasonData, bestRunsMap map[int64][]loader.BestRunData, teamMembersMap map[int64][]loader.TeamMemberData, equipmentMap map[int64]map[int64][]loader.EquipmentData, enchantmentsMap map[int64][]loader.EnchantmentData, out, version string) error {
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
				if err := generateSinglePlayerJSON(item.player, playerSeasonsMap, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, out, version); err != nil {
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
func generateSinglePlayerJSON(player loader.PlayerData, playerSeasonsMap map[int64][]loader.PlayerSeasonData, bestRunsMap map[int64][]loader.BestRunData, teamMembersMap map[int64][]loader.TeamMemberData, equipmentMap map[int64]map[int64][]loader.EquipmentData, enchantmentsMap map[int64][]loader.EnchantmentData, out, version string) error {
	// Build PlayerJSON with base info
	pj := PlayerJSON{
		ID:             player.ID,
		Name:           player.Name,
		RealmSlug:      player.RealmSlug,
		RealmName:      player.RealmName,
		Region:         player.Region,
		ClassName:      player.ClassName.String,
		ActiveSpecName: player.ActiveSpecName.String,
		AvatarURL:      player.AvatarURL,
		GuildName:      player.GuildName.String,
		RaceName:       player.RaceName.String,
		Seasons:        make(map[string]PlayerSeasonJSON),
	}

	if player.AverageItemLevel.Valid {
		v := int(player.AverageItemLevel.Int64)
		pj.AverageItemLevel = &v
	}
	if player.EquippedItemLevel.Valid {
		v := int(player.EquippedItemLevel.Int64)
		pj.EquippedItemLevel = &v
	}

	// Build seasons data
	for _, seasonData := range playerSeasonsMap[player.ID] {
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

	// Build equipment
	equipment := make(map[string]any)
	for _, eqList := range equipmentMap[player.ID] {
		for _, eq := range eqList {
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
		break // Only process the latest timestamp
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
