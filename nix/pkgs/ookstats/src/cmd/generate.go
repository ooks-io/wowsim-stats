package cmd

import (
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
    "unicode"

    "github.com/spf13/cobra"
    _ "github.com/tursodatabase/go-libsql"
    "ookstats/internal/database"
)

// JSON models (trimmed to what we need client-side)
type PlayerJSON struct {
    ID                   int64   `json:"id"`
    Name                 string  `json:"name"`
    RealmSlug            string  `json:"realm_slug"`
    RealmName            string  `json:"realm_name"`
    Region               string  `json:"region"`
    ClassName            string  `json:"class_name,omitempty"`
    ActiveSpecName       string  `json:"active_spec_name,omitempty"`
    MainSpecID           *int    `json:"main_spec_id,omitempty"`
    DungeonsCompleted    int     `json:"dungeons_completed"`
    TotalRuns            int     `json:"total_runs"`
    CombinedBestTime     *int64  `json:"combined_best_time,omitempty"`
    GlobalRanking        *int    `json:"global_ranking,omitempty"`
    RegionalRanking      *int    `json:"regional_ranking,omitempty"`
    RealmRanking         *int    `json:"realm_ranking,omitempty"`
    GlobalBracket        string  `json:"global_ranking_bracket,omitempty"`
    RegionalBracket      string  `json:"regional_ranking_bracket,omitempty"`
    RealmBracket         string  `json:"realm_ranking_bracket,omitempty"`
    AvatarURL            string  `json:"avatar_url,omitempty"`
    LastUpdated          *int64  `json:"last_updated,omitempty"`
    GuildName            string  `json:"guild_name,omitempty"`
    RaceName             string  `json:"race_name,omitempty"`
    AverageItemLevel     *int    `json:"average_item_level,omitempty"`
    EquippedItemLevel    *int    `json:"equipped_item_level,omitempty"`
}

type TeamMemberJSON struct {
    Name      string `json:"name"`
    SpecID    *int   `json:"spec_id,omitempty"`
    Region    string `json:"region"`
    RealmSlug string `json:"realm_slug"`
}

type BestRunJSON struct {
    DungeonID                 int     `json:"dungeon_id"`
    DungeonName               string  `json:"dungeon_name"`
    DungeonSlug               string  `json:"dungeon_slug"`
    RunID                     int64   `json:"run_id"`
    Duration                  int64   `json:"duration"`
    CompletedTimestamp        int64   `json:"completed_timestamp"`
    GlobalRankingFiltered     *int    `json:"global_ranking_filtered,omitempty"`
    RegionalRankingFiltered   *int    `json:"regional_ranking_filtered,omitempty"`
    RealmRankingFiltered      *int    `json:"realm_ranking_filtered,omitempty"`
    GlobalBracket             string  `json:"global_percentile_bracket,omitempty"`
    RegionalBracket           string  `json:"regional_percentile_bracket,omitempty"`
    RealmBracket              string  `json:"realm_percentile_bracket,omitempty"`
    TeamMembers               []TeamMemberJSON `json:"team_members"`
}

type PlayerPageJSON struct {
    Player   PlayerJSON              `json:"player"`
    BestRuns map[string]BestRunJSON  `json:"bestRuns"`
    Equipment map[string]any         `json:"equipment"`
    GeneratedAt int64                `json:"generated_at"`
    Version     string               `json:"version"`
}

// Optimized data structures for batch loading
type PlayerData struct {
    ID                   int64
    Name                 string
    RealmSlug            string
    RealmName            string
    Region               string
    ClassName            sql.NullString
    ActiveSpecName       sql.NullString
    MainSpecID           sql.NullInt64
    DungeonsCompleted    int
    TotalRuns            int
    CombinedBest         sql.NullInt64
    GlobalRanking        sql.NullInt64
    RegionalRanking      sql.NullInt64
    RealmRanking         sql.NullInt64
    GlobalBracket        sql.NullString
    RegionalBracket      sql.NullString
    RealmBracket         sql.NullString
    AvatarURL            string
    LastUpdated          sql.NullInt64
    GuildName            sql.NullString
    RaceName             sql.NullString
    AverageItemLevel     sql.NullInt64
    EquippedItemLevel    sql.NullInt64
}

type BestRunData struct {
    DungeonID                int64
    DungeonName              string
    DungeonSlug              string
    RunID                    int64
    Duration                 int64
    CompletedTimestamp       int64
    GlobalRankingFiltered    sql.NullInt64
    RegionalRankingFiltered  sql.NullInt64
    RealmRankingFiltered     sql.NullInt64
    GlobalBracket            string
    RegionalBracket          string
    RealmBracket             string
}

type TeamMemberData struct {
    RunID     int64
    Name      string
    SpecID    sql.NullInt64
    Region    string
    RealmSlug string
}

type EquipmentData struct {
    ID            int64
    SlotType      string
    ItemID        sql.NullInt64
    UpgradeID     sql.NullInt64
    Quality       string
    ItemName      string
    SnapshotTs    int64
    ItemIcon      sql.NullString
    ItemType      sql.NullString
}

type EnchantmentData struct {
    EquipmentID    int64
    EnchantmentID  sql.NullInt64
    SlotID         sql.NullInt64
    SlotType       sql.NullString
    DisplayString  sql.NullString
    SourceItemID   sql.NullInt64
    SourceItemName sql.NullString
    SpellID        sql.NullInt64
    GemIconSlug    sql.NullString
}

// Leaderboards
type LeaderboardMember struct {
    Name      string `json:"name"`
    SpecID    *int   `json:"spec_id,omitempty"`
    Region    string `json:"region"`
    RealmSlug string `json:"realm_slug"`
}

type LeaderboardRow struct {
    ID                 int64               `json:"id"`
    Duration           int64               `json:"duration"`
    CompletedTimestamp int64               `json:"completed_timestamp"`
    KeystoneLevel      int                 `json:"keystone_level"`
    DungeonName        string              `json:"dungeon_name"`
    RealmName          string              `json:"realm_name"`
    Region             string              `json:"region"`
    Members            []LeaderboardMember `json:"members"`
}

type LeaderboardPage struct {
    LeadingGroups []LeaderboardRow `json:"leading_groups"`
    Map           struct{ Name map[string]any `json:"name"` } `json:"map"`
    ConnectedRealm *struct{ Name string `json:"name"` } `json:"connected_realm,omitempty"`
    Pagination    struct{
        CurrentPage int `json:"currentPage"`
        PageSize    int `json:"pageSize"`
        TotalRuns   int `json:"totalRuns"`
        TotalPages  int `json:"totalPages"`
        HasNextPage bool `json:"hasNextPage"`
        HasPrevPage bool `json:"hasPrevPage"`
    } `json:"pagination"`
}

var generateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate static artifacts",
    Long:  `Generate static API endpoints and other build artifacts from the local database.`,
}

var generateAPICmd = &cobra.Command{
    Use:   "api",
    Short: "Generate static JSON API endpoints",
    RunE: func(cmd *cobra.Command, args []string) error {
        outDir, _ := cmd.Flags().GetString("out")
        onlyPlayers, _ := cmd.Flags().GetBool("players")
        doLeaderboards, _ := cmd.Flags().GetBool("leaderboards")
        doSearch, _ := cmd.Flags().GetBool("search")
        pageSize, _ := cmd.Flags().GetInt("page-size")
        shardSize, _ := cmd.Flags().GetInt("shard-size")
        regionsCSV, _ := cmd.Flags().GetString("regions")

        if strings.TrimSpace(outDir) == "" {
            return errors.New("--out is required")
        }
        // Connect to local DB (file:)
        db, err := database.Connect()
        if err != nil {
            return fmt.Errorf("failed to connect to db: %w", err)
        }
        defer db.Close()

        base := filepath.Join(outDir, "api")
        if err := os.MkdirAll(base, 0o755); err != nil {
            return fmt.Errorf("mkdir base: %w", err)
        }

        if onlyPlayers {
            if err := generatePlayers(db, filepath.Join(base, "player"), ""); err != nil {
                return err
            }
        }

        if doLeaderboards {
            regions := []string{}
            if strings.TrimSpace(regionsCSV) != "" {
                for _, r := range strings.Split(regionsCSV, ",") {
                    rr := strings.TrimSpace(r)
                    if rr != "" { regions = append(regions, rr) }
                }
            }
            if err := generateLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil {
                return err
            }
            if err := generatePlayerLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil {
                return err
            }
        }

        if doSearch {
            if err := generateSearchIndex(db, filepath.Join(base, "search"), shardSize); err != nil {
                return err
            }
        }

        fmt.Printf("\nStatic API generated at %s\n", base)
        return nil
    },
}

func generatePlayers(db *sql.DB, out string, version string) error {
    fmt.Println("Generating player JSON endpoints...")
    if err := os.MkdirAll(out, 0o755); err != nil {
        return fmt.Errorf("mkdir players out: %w", err)
    }

    // Step 1: Load all players with complete coverage
    fmt.Printf("Loading players with complete coverage...\n")
    players, err := loadAllCompleteCoveragePlayers(db)
    if err != nil {
        return fmt.Errorf("load players: %w", err)
    }
    fmt.Printf("[OK] Loaded %d players with complete coverage\n", len(players))

    if len(players) == 0 {
        fmt.Println("No players with complete coverage found")
        return nil
    }

    // Step 2: Batch load all supporting data
    fmt.Printf("Loading best runs data...\n")
    bestRunsMap, allRunIDs, err := loadAllBestRuns(db, getPlayerIDs(players))
    if err != nil {
        return fmt.Errorf("load best runs: %w", err)
    }
    fmt.Printf("[OK] Loaded best runs for %d players (%d total runs)\n", len(bestRunsMap), len(allRunIDs))

    fmt.Printf("Loading team members...\n")
    teamMembersMap, err := loadAllTeamMembers(db, allRunIDs)
    if err != nil {
        return fmt.Errorf("load team members: %w", err)
    }
    fmt.Printf("[OK] Loaded team members for %d runs\n", len(teamMembersMap))

    fmt.Printf("Loading equipment data...\n")
    equipmentMap, enchantmentsMap, err := loadAllEquipment(db, getPlayerIDs(players))
    if err != nil {
        return fmt.Errorf("load equipment: %w", err)
    }
    fmt.Printf("[OK] Loaded equipment for %d players\n", len(equipmentMap))

    // Step 3: Process players concurrently
    fmt.Printf("Generating JSON files concurrently...\n")
    return generatePlayerJSONsConcurrently(players, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, out, version)
}

func loadAllCompleteCoveragePlayers(db *sql.DB) ([]PlayerData, error) {
    rows, err := db.Query(`
        SELECT p.id, p.name, r.slug, r.name, r.region,
               pd.class_name, pd.active_spec_name,
               pp.main_spec_id, pp.dungeons_completed, pp.total_runs,
               pp.combined_best_time, pp.global_ranking, pp.regional_ranking, pp.realm_ranking,
               pp.global_ranking_bracket, pp.regional_ranking_bracket, pp.realm_ranking_bracket,
               COALESCE(pd.avatar_url, ''),
               COALESCE(pp.last_updated, 0),
               pd.guild_name, pd.race_name, pd.average_item_level, pd.equipped_item_level
        FROM players p
        JOIN realms r ON p.realm_id = r.id
        JOIN player_profiles pp ON p.id = pp.player_id
        LEFT JOIN player_details pd ON p.id = pd.player_id
        WHERE pp.has_complete_coverage = 1
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var players []PlayerData
    for rows.Next() {
        var player PlayerData
        if err := rows.Scan(
            &player.ID, &player.Name, &player.RealmSlug, &player.RealmName, &player.Region,
            &player.ClassName, &player.ActiveSpecName,
            &player.MainSpecID, &player.DungeonsCompleted, &player.TotalRuns,
            &player.CombinedBest, &player.GlobalRanking, &player.RegionalRanking, &player.RealmRanking,
            &player.GlobalBracket, &player.RegionalBracket, &player.RealmBracket,
            &player.AvatarURL, &player.LastUpdated,
            &player.GuildName, &player.RaceName, &player.AverageItemLevel, &player.EquippedItemLevel); err != nil {
            return nil, fmt.Errorf("scan player: %w", err)
        }
        players = append(players, player)
    }
    return players, nil
}

func getPlayerIDs(players []PlayerData) []int64 {
    ids := make([]int64, len(players))
    for i, p := range players {
        ids[i] = p.ID
    }
    return ids
}

func loadAllBestRuns(db *sql.DB, playerIDs []int64) (map[int64][]BestRunData, []int64, error) {
    if len(playerIDs) == 0 {
        return make(map[int64][]BestRunData), []int64{}, nil
    }

    // Build IN clause
    placeholders := make([]string, len(playerIDs))
    args := make([]any, len(playerIDs))
    for i, id := range playerIDs {
        placeholders[i] = "?"
        args[i] = id
    }

    query := fmt.Sprintf(`
        WITH realm_rankings AS (
            SELECT cr.id as run_id, cr.dungeon_id, cr.realm_id,
                   ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id, cr.realm_id ORDER BY cr.duration ASC, cr.completed_timestamp ASC, cr.id ASC) as realm_ranking,
                   COUNT(*) OVER (PARTITION BY cr.dungeon_id, cr.realm_id) as total_in_realm_dungeon
            FROM challenge_runs cr
        )
        SELECT pbr.player_id, pbr.dungeon_id, d.name, d.slug, pbr.run_id, pbr.duration, pbr.completed_timestamp,
               rr_global_filtered.ranking as global_ranking_filtered,
               rr_regional_filtered.ranking as regional_ranking_filtered, 
               realm_rankings.realm_ranking as realm_ranking_filtered,
               COALESCE(rr_global_filtered.percentile_bracket, '') as global_percentile_bracket,
               COALESCE(rr_regional_filtered.percentile_bracket, '') as regional_percentile_bracket,
               CASE 
                   WHEN realm_rankings.realm_ranking = 1 THEN 'artifact'
                   ELSE 
                       CASE 
                           WHEN (CAST(realm_rankings.realm_ranking AS REAL) / CAST(realm_rankings.total_in_realm_dungeon AS REAL) * 100) <= 1.0 THEN 'excellent'
                           WHEN (CAST(realm_rankings.realm_ranking AS REAL) / CAST(realm_rankings.total_in_realm_dungeon AS REAL) * 100) <= 5.0 THEN 'legendary'
                           WHEN (CAST(realm_rankings.realm_ranking AS REAL) / CAST(realm_rankings.total_in_realm_dungeon AS REAL) * 100) <= 20.0 THEN 'epic'  
                           WHEN (CAST(realm_rankings.realm_ranking AS REAL) / CAST(realm_rankings.total_in_realm_dungeon AS REAL) * 100) <= 40.0 THEN 'rare'
                           WHEN (CAST(realm_rankings.realm_ranking AS REAL) / CAST(realm_rankings.total_in_realm_dungeon AS REAL) * 100) <= 60.0 THEN 'uncommon'
                           ELSE 'common'
                       END
               END as realm_percentile_bracket
        FROM player_best_runs pbr
        JOIN dungeons d ON pbr.dungeon_id = d.id
        JOIN players p ON pbr.player_id = p.id
        JOIN realms r ON p.realm_id = r.id
        LEFT JOIN run_rankings rr_global_filtered ON pbr.run_id = rr_global_filtered.run_id 
            AND rr_global_filtered.ranking_type = 'global' AND rr_global_filtered.ranking_scope = 'filtered'
        LEFT JOIN run_rankings rr_regional_filtered ON pbr.run_id = rr_regional_filtered.run_id 
            AND rr_regional_filtered.ranking_type = 'regional' AND rr_regional_filtered.ranking_scope = r.region || '_filtered'
        LEFT JOIN realm_rankings ON pbr.run_id = realm_rankings.run_id 
            AND pbr.dungeon_id = realm_rankings.dungeon_id AND r.id = realm_rankings.realm_id
        WHERE pbr.player_id IN (%s)
        ORDER BY pbr.player_id, d.name
    `, strings.Join(placeholders, ","))

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    bestRunsMap := make(map[int64][]BestRunData)
    runIDSet := make(map[int64]bool)
    var allRunIDs []int64

    for rows.Next() {
        var playerID int64
        var run BestRunData
        if err := rows.Scan(
            &playerID, &run.DungeonID, &run.DungeonName, &run.DungeonSlug, &run.RunID, &run.Duration, &run.CompletedTimestamp,
            &run.GlobalRankingFiltered, &run.RegionalRankingFiltered, &run.RealmRankingFiltered,
            &run.GlobalBracket, &run.RegionalBracket, &run.RealmBracket); err != nil {
            return nil, nil, fmt.Errorf("scan best run: %w", err)
        }
        bestRunsMap[playerID] = append(bestRunsMap[playerID], run)
        
        // Deduplicate run IDs
        if !runIDSet[run.RunID] {
            runIDSet[run.RunID] = true
            allRunIDs = append(allRunIDs, run.RunID)
        }
    }
    return bestRunsMap, allRunIDs, nil
}

func loadAllTeamMembers(db *sql.DB, runIDs []int64) (map[int64][]TeamMemberData, error) {
    if len(runIDs) == 0 {
        return make(map[int64][]TeamMemberData), nil
    }

    teamMembersMap := make(map[int64][]TeamMemberData)
    
    // Process in batches to avoid SQL limits (max ~32k parameters)
    const batchSize = 10000
    for i := 0; i < len(runIDs); i += batchSize {
        end := i + batchSize
        if end > len(runIDs) {
            end = len(runIDs)
        }
        
        batch := runIDs[i:end]
        
        // Build IN clause for this batch
        placeholders := make([]string, len(batch))
        args := make([]any, len(batch))
        for j, id := range batch {
            placeholders[j] = "?"
            args[j] = id
        }

        query := fmt.Sprintf(`
            SELECT rm.run_id, p.name, rm.spec_id, r.region, r.slug
            FROM run_members rm
            JOIN players p ON rm.player_id = p.id
            JOIN realms r ON p.realm_id = r.id
            WHERE rm.run_id IN (%s)
            ORDER BY rm.run_id, p.name
        `, strings.Join(placeholders, ","))

        rows, err := db.Query(query, args...)
        if err != nil {
            return nil, fmt.Errorf("batch %d: %w", i/batchSize+1, err)
        }

        for rows.Next() {
            var member TeamMemberData
            if err := rows.Scan(
                &member.RunID, &member.Name, &member.SpecID, &member.Region, &member.RealmSlug); err != nil {
                rows.Close()
                return nil, fmt.Errorf("scan team member: %w", err)
            }
            teamMembersMap[member.RunID] = append(teamMembersMap[member.RunID], member)
        }
        rows.Close()
    }
    
    return teamMembersMap, nil
}

func loadAllEquipment(db *sql.DB, playerIDs []int64) (map[int64]map[int64][]EquipmentData, map[int64][]EnchantmentData, error) {
    if len(playerIDs) == 0 {
        return make(map[int64]map[int64][]EquipmentData), make(map[int64][]EnchantmentData), nil
    }

    // Build IN clause
    placeholders := make([]string, len(playerIDs))
    args := make([]any, len(playerIDs))
    for i, id := range playerIDs {
        placeholders[i] = "?"
        args[i] = id
    }

    // Get latest timestamp per player first
    latestQuery := fmt.Sprintf(`
        SELECT player_id, MAX(snapshot_timestamp) 
        FROM player_equipment 
        WHERE player_id IN (%s) 
        GROUP BY player_id
    `, strings.Join(placeholders, ","))

    latestRows, err := db.Query(latestQuery, args...)
    if err != nil {
        return nil, nil, err
    }
    defer latestRows.Close()

    playerTimestamps := make(map[int64]int64)
    for latestRows.Next() {
        var playerID, timestamp int64
        if err := latestRows.Scan(&playerID, &timestamp); err != nil {
            return nil, nil, err
        }
        playerTimestamps[playerID] = timestamp
    }

    if len(playerTimestamps) == 0 {
        return make(map[int64]map[int64][]EquipmentData), make(map[int64][]EnchantmentData), nil
    }

    // Load equipment for latest timestamps
    equipmentMap := make(map[int64]map[int64][]EquipmentData)
    var allEquipmentIDs []int64

    for playerID, timestamp := range playerTimestamps {
        rows, err := db.Query(`
            SELECT e.id, e.slot_type, e.item_id, e.upgrade_id, e.quality, e.item_name, e.snapshot_timestamp,
                   i.icon AS item_icon_slug, i.type AS item_type
            FROM player_equipment e
            LEFT JOIN items i ON e.item_id = i.id
            WHERE e.player_id = ? AND e.snapshot_timestamp = ?
            ORDER BY e.slot_type
        `, playerID, timestamp)
        if err != nil {
            return nil, nil, err
        }

        if equipmentMap[playerID] == nil {
            equipmentMap[playerID] = make(map[int64][]EquipmentData)
        }

        for rows.Next() {
            var eq EquipmentData
            if err := rows.Scan(
                &eq.ID, &eq.SlotType, &eq.ItemID, &eq.UpgradeID, &eq.Quality, &eq.ItemName, &eq.SnapshotTs,
                &eq.ItemIcon, &eq.ItemType); err != nil {
                rows.Close()
                return nil, nil, fmt.Errorf("scan equipment: %w", err)
            }
            equipmentMap[playerID][timestamp] = append(equipmentMap[playerID][timestamp], eq)
            allEquipmentIDs = append(allEquipmentIDs, eq.ID)
        }
        rows.Close()
    }

    // Load enchantments in batches
    enchantmentsMap := make(map[int64][]EnchantmentData)
    if len(allEquipmentIDs) > 0 {
        const enchBatchSize = 10000
        for i := 0; i < len(allEquipmentIDs); i += enchBatchSize {
            end := i + enchBatchSize
            if end > len(allEquipmentIDs) {
                end = len(allEquipmentIDs)
            }
            
            batch := allEquipmentIDs[i:end]
            placeholders := make([]string, len(batch))
            args := make([]any, len(batch))
            for j, id := range batch {
                placeholders[j] = "?"
                args[j] = id
            }

            enchQuery := fmt.Sprintf(`
                SELECT pee.equipment_id, pee.enchantment_id, pee.slot_id, pee.slot_type, pee.display_string,
                       pee.source_item_id, pee.source_item_name, pee.spell_id, i.icon as gem_icon_slug
                FROM player_equipment_enchantments pee
                LEFT JOIN items i ON pee.source_item_id = i.id
                WHERE pee.equipment_id IN (%s)
                ORDER BY pee.equipment_id, pee.slot_id
            `, strings.Join(placeholders, ","))

            enchRows, err := db.Query(enchQuery, args...)
            if err != nil {
                return nil, nil, fmt.Errorf("enchantments batch %d: %w", i/enchBatchSize+1, err)
            }

            for enchRows.Next() {
                var ench EnchantmentData
                if err := enchRows.Scan(
                    &ench.EquipmentID, &ench.EnchantmentID, &ench.SlotID, &ench.SlotType, &ench.DisplayString,
                    &ench.SourceItemID, &ench.SourceItemName, &ench.SpellID, &ench.GemIconSlug); err != nil {
                    enchRows.Close()
                    return nil, nil, fmt.Errorf("scan enchantment: %w", err)
                }
                enchantmentsMap[ench.EquipmentID] = append(enchantmentsMap[ench.EquipmentID], ench)
            }
            enchRows.Close()
        }
    }

    return equipmentMap, enchantmentsMap, nil
}

func generatePlayerJSONsConcurrently(players []PlayerData, bestRunsMap map[int64][]BestRunData, teamMembersMap map[int64][]TeamMemberData, equipmentMap map[int64]map[int64][]EquipmentData, enchantmentsMap map[int64][]EnchantmentData, out, version string) error {
    
    startTime := time.Now()
    const batchSize = 100
    const numWorkers = 10
    
    // Channel for work items
    type workItem struct {
        player PlayerData
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
                if err := generateSinglePlayerJSON(item.player, bestRunsMap, teamMembersMap, equipmentMap, enchantmentsMap, out, version); err != nil {
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

func generateSinglePlayerJSON(player PlayerData, bestRunsMap map[int64][]BestRunData, teamMembersMap map[int64][]TeamMemberData, equipmentMap map[int64]map[int64][]EquipmentData, enchantmentsMap map[int64][]EnchantmentData, out, version string) error {
    
    // Build PlayerJSON
    pj := PlayerJSON{
        ID:                player.ID,
        Name:              player.Name,
        RealmSlug:         player.RealmSlug,
        RealmName:         player.RealmName,
        Region:            player.Region,
        ClassName:         player.ClassName.String,
        ActiveSpecName:    player.ActiveSpecName.String,
        DungeonsCompleted: player.DungeonsCompleted,
        TotalRuns:         player.TotalRuns,
        GlobalBracket:     player.GlobalBracket.String,
        RegionalBracket:   player.RegionalBracket.String,
        RealmBracket:      player.RealmBracket.String,
        AvatarURL:         player.AvatarURL,
        GuildName:         player.GuildName.String,
        RaceName:          player.RaceName.String,
    }
    
    if player.MainSpecID.Valid { v := int(player.MainSpecID.Int64); pj.MainSpecID = &v }
    if player.CombinedBest.Valid { v := player.CombinedBest.Int64; pj.CombinedBestTime = &v }
    if player.GlobalRanking.Valid { v := int(player.GlobalRanking.Int64); pj.GlobalRanking = &v }
    if player.RegionalRanking.Valid { v := int(player.RegionalRanking.Int64); pj.RegionalRanking = &v }
    if player.RealmRanking.Valid { v := int(player.RealmRanking.Int64); pj.RealmRanking = &v }
    if player.AverageItemLevel.Valid { v := int(player.AverageItemLevel.Int64); pj.AverageItemLevel = &v }
    if player.EquippedItemLevel.Valid { v := int(player.EquippedItemLevel.Int64); pj.EquippedItemLevel = &v }
    if player.LastUpdated.Valid { v := player.LastUpdated.Int64; pj.LastUpdated = &v }
    
    // Build best runs
    bestRuns := make(map[string]BestRunJSON)
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
        
        if run.GlobalRankingFiltered.Valid { v := int(run.GlobalRankingFiltered.Int64); br.GlobalRankingFiltered = &v }
        if run.RegionalRankingFiltered.Valid { v := int(run.RegionalRankingFiltered.Int64); br.RegionalRankingFiltered = &v }
        if run.RealmRankingFiltered.Valid { v := int(run.RealmRankingFiltered.Int64); br.RealmRankingFiltered = &v }
        
        // Add team members
        for _, member := range teamMembersMap[run.RunID] {
            tm := TeamMemberJSON{
                Name:      member.Name,
                Region:    member.Region,
                RealmSlug: member.RealmSlug,
            }
            if member.SpecID.Valid { v := int(member.SpecID.Int64); tm.SpecID = &v }
            br.TeamMembers = append(br.TeamMembers, tm)
        }
        
        bestRuns[run.DungeonSlug] = br
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
            
            if eq.ItemID.Valid { eqData["item_id"] = int(eq.ItemID.Int64) }
            if eq.UpgradeID.Valid { eqData["upgrade_id"] = int(eq.UpgradeID.Int64) }
            
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
                
                if ench.EnchantmentID.Valid { enchData["enchantment_id"] = int(ench.EnchantmentID.Int64) }
                if ench.SlotID.Valid { enchData["slot_id"] = int(ench.SlotID.Int64) }
                if ench.SourceItemID.Valid { enchData["source_item_id"] = int(ench.SourceItemID.Int64) }
                if ench.SpellID.Valid { enchData["spell_id"] = int(ench.SpellID.Int64) }
                if ench.GemIconSlug.Valid { enchData["gem_icon_slug"] = ench.GemIconSlug.String }
                
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
        BestRuns:    bestRuns,
        Equipment:   equipment,
        GeneratedAt: time.Now().UnixMilli(),
        Version:     version,
    }
    
    // Write file
    dir := filepath.Join(out, pj.Region, pj.RealmSlug)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return fmt.Errorf("mkdir player dir: %w", err)
    }
    fname := filepath.Join(dir, safeSlugName(pj.Name)+".json")
    return writeJSONFile(fname, page)
}

// safeSlugName converts an arbitrary player name to a safe lowercase filename without path separators.
// It preserves Unicode letters/numbers (including diacritics), matching frontend expectations
// that URLs may contain non-ASCII characters. Only path separators are removed and spaces -> '-'.
func safeSlugName(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    // replace path separators explicitly
    s = strings.ReplaceAll(s, "/", "-")
    s = strings.ReplaceAll(s, "\\", "-")
    // allow unicode letters/digits and '-', '_' ; replace spaces with '-'
    out := make([]rune, 0, len(s))
    for _, r := range s {
        if r == ' ' {
            out = append(out, '-')
            continue
        }
        if r == '-' || r == '_' {
            out = append(out, r)
            continue
        }
        // Keep unicode letters and digits; drop other punctuation/symbols
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            out = append(out, r)
            continue
        }
        // else: drop
    }
    // collapse multiple dashes
    cleaned := make([]rune, 0, len(out))
    prevDash := false
    for _, r := range out {
        if r == '-' {
            if !prevDash { cleaned = append(cleaned, r) }
            prevDash = true
        } else {
            cleaned = append(cleaned, r)
            prevDash = false
        }
    }
    if len(cleaned) == 0 { return "player" }
    // trim leading/trailing '-'
    i := 0
    j := len(cleaned)
    for i < j && cleaned[i] == '-' { i++ }
    for j > i && cleaned[j-1] == '-' { j-- }
    return string(cleaned[i:j])
}

func writeJSONFile(path string, v any) error {
    // Ensure parent directory exists (robust for all callers)
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
    }

    // Create a temp file in the target directory to avoid cross-filesystem issues
    tmpFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
    if err != nil {
        return fmt.Errorf("create temp for %s: %w", path, err)
    }
    tmp := tmpFile.Name()

    enc := json.NewEncoder(tmpFile)
    enc.SetEscapeHTML(false)
    enc.SetIndent("", "  ")
    if err := enc.Encode(v); err != nil {
        tmpFile.Close()
        os.Remove(tmp)
        return fmt.Errorf("encode json %s: %w", path, err)
    }
    // Flush to disk
    if err := tmpFile.Sync(); err != nil {
        tmpFile.Close()
        os.Remove(tmp)
        return fmt.Errorf("sync temp %s: %w", tmp, err)
    }
    if err := tmpFile.Close(); err != nil {
        os.Remove(tmp)
        return fmt.Errorf("close temp %s: %w", tmp, err)
    }

    // Atomic replace; add a short retry in case of transient fs race
    if err := os.Rename(tmp, path); err != nil {
        // Retry once after ensuring parent dir again
        _ = os.MkdirAll(filepath.Dir(path), 0o755)
        if err2 := os.Rename(tmp, path); err2 != nil {
            os.Remove(tmp)
            return fmt.Errorf("rename %s: %w", path, err2)
        }
    }
    return nil
}

// ---------------- Leaderboards ----------------
func generateLeaderboards(db *sql.DB, out string, pageSize int, regions []string) error {
    if pageSize <= 0 { pageSize = 25 }
    if err := os.MkdirAll(out, 0o755); err != nil { return err }

    // Load dungeons
    drows, err := db.Query(`SELECT id, slug, name FROM dungeons ORDER BY name`)
    if err != nil { return fmt.Errorf("dungeons query: %w", err) }
    type Dungeon struct{ ID int; Slug, Name string }
    dungeons := []Dungeon{}
    for drows.Next() { var d Dungeon; if err := drows.Scan(&d.ID, &d.Slug, &d.Name); err != nil { drows.Close(); return err }; dungeons = append(dungeons, d) }
    drows.Close()

    // helper: write global for each dungeon
    writeGlobal := func(d Dungeon) error {
        dir := filepath.Join(out, "global", d.Slug)
        if err := os.MkdirAll(dir, 0o755); err != nil { return err }
        // Count distinct teams
        var total int
        err := db.QueryRow(`SELECT COUNT(DISTINCT team_signature) FROM challenge_runs WHERE dungeon_id = ?`, d.ID).Scan(&total)
        if err != nil { return fmt.Errorf("global count: %w", err) }
        pages := (total + pageSize - 1) / pageSize
        for p := 1; p <= pages; p++ {
            rows, err := selectCanonicalRuns(db, d.ID, "", "", pageSize, (p-1)*pageSize)
            if err != nil { return err }
            page := LeaderboardPage{ LeadingGroups: rows }
            page.Map.Name = map[string]any{"en_US": d.Name}
            page.Pagination.CurrentPage = p
            page.Pagination.PageSize = pageSize
            page.Pagination.TotalRuns = total
            page.Pagination.TotalPages = pages
            page.Pagination.HasNextPage = p < pages
            page.Pagination.HasPrevPage = p > 1
            if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil { return err }
        }
        return nil
    }

    // helper: write regional for each region/dungeon
    writeRegional := func(region string, d Dungeon) error {
        dir := filepath.Join(out, region, "all", d.Slug)
        if err := os.MkdirAll(dir, 0o755); err != nil { return err }
        var total int
        err := db.QueryRow(`
            SELECT COUNT(*) FROM (
              SELECT team_signature
              FROM challenge_runs cr
              JOIN realms r ON cr.realm_id = r.id
              WHERE cr.dungeon_id = ? AND r.region = ?
              GROUP BY team_signature
            ) x
        `, d.ID, region).Scan(&total)
        if err != nil { return fmt.Errorf("regional count: %w", err) }
        pages := (total + pageSize - 1) / pageSize
        for p := 1; p <= pages; p++ {
            rows, err := selectCanonicalRuns(db, d.ID, region, "", pageSize, (p-1)*pageSize)
            if err != nil { return err }
            page := LeaderboardPage{ LeadingGroups: rows }
            page.Map.Name = map[string]any{"en_US": d.Name}
            page.Pagination.CurrentPage = p
            page.Pagination.PageSize = pageSize
            page.Pagination.TotalRuns = total
            page.Pagination.TotalPages = pages
            page.Pagination.HasNextPage = p < pages
            page.Pagination.HasPrevPage = p > 1
            if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil { return err }
        }
        return nil
    }

    // Process globals
    fmt.Println("Generating global leaderboards...")
    for _, d := range dungeons { if err := writeGlobal(d); err != nil { return err } }

    // Regions
    if len(regions) == 0 { regions = []string{"us","eu","kr","tw"} }
    for _, reg := range regions {
        fmt.Printf("Generating %s leaderboards...\n", strings.ToUpper(reg))
        for _, d := range dungeons { if err := writeRegional(reg, d); err != nil { return err } }
    }
    // Realm leaderboards for each region+realm
    for _, reg := range regions {
        // realms for region
        rrows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, reg)
        if err != nil { return fmt.Errorf("realms list: %w", err) }
        slugs := []string{}
        for rrows.Next() { var s string; if err := rrows.Scan(&s); err != nil { rrows.Close(); return err }; slugs = append(slugs, s) }
        rrows.Close()
        for _, rslug := range slugs {
            fmt.Printf("Generating realm %s/%s leaderboards...\n", reg, rslug)
            for _, d := range dungeons {
                dir := filepath.Join(out, reg, rslug, d.Slug)
                if err := os.MkdirAll(dir, 0o755); err != nil { return err }
                // realm display name for payload shape compatibility
                var realmName string
                if err := db.QueryRow(`SELECT name FROM realms WHERE region = ? AND slug = ?`, reg, rslug).Scan(&realmName); err != nil { realmName = rslug }
                var total int
                // count distinct teams in this realm
                if err := db.QueryRow(`
                  SELECT COUNT(*) FROM (
                    SELECT team_signature
                    FROM challenge_runs cr
                    JOIN realms rr ON cr.realm_id = rr.id
                    WHERE cr.dungeon_id = ? AND rr.region = ? AND rr.slug = ?
                    GROUP BY team_signature
                  ) x
                `, d.ID, reg, rslug).Scan(&total); err != nil { return fmt.Errorf("realm count: %w", err) }
                pages := (total + pageSize - 1) / pageSize
                for p := 1; p <= pages; p++ {
                    rows, err := selectCanonicalRuns(db, d.ID, reg, rslug, pageSize, (p-1)*pageSize)
                    if err != nil { return err }
                    page := LeaderboardPage{ LeadingGroups: rows }
                    page.Map.Name = map[string]any{"en_US": d.Name}
                    page.ConnectedRealm = &struct{ Name string `json:"name"` }{ Name: realmName }
                    page.Pagination.CurrentPage = p
                    page.Pagination.PageSize = pageSize
                    page.Pagination.TotalRuns = total
                    page.Pagination.TotalPages = pages
                    page.Pagination.HasNextPage = p < pages
                    page.Pagination.HasPrevPage = p > 1
                    if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil { return err }
                }
            }
        }
    }
    fmt.Println("[OK] Leaderboards generated")
    return nil
}

// selectCanonicalRuns returns one canonical run per team_signature, ordered, with members
func selectCanonicalRuns(db *sql.DB, dungeonID int, region string, realmSlug string, limit, offset int) ([]LeaderboardRow, error) {
    // Use window function to rank runs per team_signature, picking best per team
    where := "WHERE cr.dungeon_id = ?"
    args := []any{dungeonID}
    if region != "" {
        where += " AND r.region = ?"
        args = append(args, region)
    }
    if realmSlug != "" {
        where += " AND r.slug = ?"
        args = append(args, realmSlug)
    }
    q := fmt.Sprintf(`
      WITH ranked AS (
        SELECT cr.id, cr.duration, cr.completed_timestamp,
               ROW_NUMBER() OVER (PARTITION BY cr.team_signature ORDER BY cr.duration ASC, cr.completed_timestamp ASC, cr.id ASC) AS rn
        FROM challenge_runs cr
        JOIN realms r ON cr.realm_id = r.id
        %s
      )
      SELECT id FROM ranked WHERE rn = 1
      ORDER BY duration ASC, completed_timestamp ASC, id ASC
      LIMIT %d OFFSET %d
    `, where, limit, offset)

    idRows, err := db.Query(q, args...)
    if err != nil { return nil, err }
    ids := []int64{}
    for idRows.Next() { var id int64; if err := idRows.Scan(&id); err != nil { idRows.Close(); return nil, err }; ids = append(ids, id) }
    idRows.Close()
    if len(ids) == 0 { return []LeaderboardRow{}, nil }

    // Load rows
    placeholders := make([]string, len(ids))
    iargs := make([]any, len(ids))
    for i, id := range ids { placeholders[i] = "?"; iargs[i] = id }
    rQuery := fmt.Sprintf(`
      SELECT cr.id, cr.duration, cr.completed_timestamp, cr.keystone_level,
             d.name, rr.name, rr.region
      FROM challenge_runs cr
      JOIN dungeons d ON cr.dungeon_id = d.id
      JOIN realms rr ON cr.realm_id = rr.id
      WHERE cr.id IN (%s)
    `, strings.Join(placeholders, ","))
    rrows, err := db.Query(rQuery, iargs...)
    if err != nil { return nil, err }
    byID := map[int64]LeaderboardRow{}
    for rrows.Next() {
        var row LeaderboardRow
        if err := rrows.Scan(&row.ID, &row.Duration, &row.CompletedTimestamp, &row.KeystoneLevel, &row.DungeonName, &row.RealmName, &row.Region); err != nil { rrows.Close(); return nil, err }
        byID[row.ID] = row
    }
    rrows.Close()

    // Members
    mQuery := fmt.Sprintf(`
      SELECT rm.run_id, p.name, rm.spec_id, rr.region, rr.slug
      FROM run_members rm
      JOIN players p ON rm.player_id = p.id
      JOIN realms rr ON p.realm_id = rr.id
      WHERE rm.run_id IN (%s)
      ORDER BY rm.run_id, p.name
    `, strings.Join(placeholders, ","))
    mrows, err := db.Query(mQuery, iargs...)
    if err != nil { return nil, err }
    for mrows.Next() {
        var runID int64
        var name, region, rslug string
        var spec sql.NullInt64
        if err := mrows.Scan(&runID, &name, &spec, &region, &rslug); err != nil { mrows.Close(); return nil, err }
        row := byID[runID]
        var specPtr *int
        if spec.Valid { v := int(spec.Int64); specPtr = &v }
        row.Members = append(row.Members, LeaderboardMember{ Name: name, SpecID: specPtr, Region: region, RealmSlug: rslug })
        byID[runID] = row
    }
    mrows.Close()

    // Order back as ids
    out := make([]LeaderboardRow, 0, len(ids))
    for _, id := range ids { out = append(out, byID[id]) }
    return out, nil
}

// ---------------- Player leaderboards ----------------
func generatePlayerLeaderboards(db *sql.DB, out string, pageSize int, regions []string) error {
    if pageSize <= 0 { pageSize = 25 }
    // global scope
    writeScope := func(scope string, region string) error {
        var dir string
        if scope == "global" {
            dir = filepath.Join(out, "players", "global")
        } else {
            dir = filepath.Join(out, "players", "regional", region)
        }
        if err := os.MkdirAll(dir, 0o755); err != nil { return err }

        // total count
        var total int
        if scope == "global" {
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM player_profiles
                WHERE has_complete_coverage = 1 AND combined_best_time IS NOT NULL
            `).Scan(&total); err != nil { return fmt.Errorf("players total (global): %w", err) }
        } else {
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM player_profiles pp
                JOIN players p ON pp.player_id = p.id
                JOIN realms r ON p.realm_id = r.id
                WHERE pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL AND r.region = ?
            `, region).Scan(&total); err != nil { return fmt.Errorf("players total (regional): %w", err) }
        }
        pages := (total + pageSize - 1) / pageSize

        for p := 1; p <= pages; p++ {
            offset := (p-1) * pageSize
            // select rows
            var rows *sql.Rows
            var err error
            if scope == "global" {
                rows, err = db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, pageSize, offset)
            } else {
                rows, err = db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE r.region = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, region, pageSize, offset)
            }
            if err != nil { return err }
            type PlayerRow struct {
                ID int64
                Name string
                RealmSlug string
                RealmName string
                Region string
                ClassName string
                ActiveSpecName string
                MainSpecID sql.NullInt64
                CombinedBestTime sql.NullInt64
                DungeonsCompleted int
                TotalRuns int
            }
            list := []map[string]any{}
            for rows.Next() {
                var r PlayerRow
                if err := rows.Scan(&r.ID, &r.Name, &r.RealmSlug, &r.RealmName, &r.Region, &r.ClassName, &r.ActiveSpecName, &r.MainSpecID, &r.CombinedBestTime, &r.DungeonsCompleted, &r.TotalRuns); err != nil { rows.Close(); return err }
                obj := map[string]any{
                    "player_id": r.ID,
                    "name": r.Name,
                    "realm_slug": r.RealmSlug,
                    "realm_name": r.RealmName,
                    "region": r.Region,
                    "class_name": r.ClassName,
                    "active_spec_name": r.ActiveSpecName,
                    "dungeons_completed": r.DungeonsCompleted,
                    "total_runs": r.TotalRuns,
                }
                if r.MainSpecID.Valid { obj["main_spec_id"] = int(r.MainSpecID.Int64) }
                if r.CombinedBestTime.Valid { obj["combined_best_time"] = r.CombinedBestTime.Int64 }
                list = append(list, obj)
            }
            rows.Close()
            page := map[string]any{
                "leaderboard": list,
                "title": func() string { if scope == "global" { return "Global Player Rankings" } ; return strings.ToUpper(region) + " Player Rankings" }(),
                "generated_timestamp": time.Now().UnixMilli(),
                "pagination": map[string]any{
                    "currentPage": p,
                    "pageSize": pageSize,
                    "totalPlayers": total,
                    "totalPages": pages,
                    "hasNextPage": p < pages,
                    "hasPrevPage": p > 1,
                    // keep compatibility with frontend expecting totalRuns
                    "totalRuns": total,
                },
            }
            if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil { return err }
        }
        return nil
    }

    // global
    if err := writeScope("global", ""); err != nil { return err }
    // regional
    if len(regions) == 0 { regions = []string{"us","eu","kr","tw"} }
    for _, reg := range regions {
        if err := writeScope("regional", reg); err != nil { return err }
    }
    // realm scope (per region -> realm)
    for _, reg := range regions {
        rrows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, reg)
        if err != nil { return fmt.Errorf("players realms list: %w", err) }
        slugs := []string{}
        for rrows.Next() { var s string; if err := rrows.Scan(&s); err != nil { rrows.Close(); return err }; slugs = append(slugs, s) }
        rrows.Close()
        for _, rslug := range slugs {
            dir := filepath.Join(out, "players", "realm", reg, rslug)
            if err := os.MkdirAll(dir, 0o755); err != nil { return err }
            // count total
            var total int
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM players p
                JOIN realms r ON p.realm_id = r.id
                JOIN player_profiles pp ON p.id = pp.player_id
                LEFT JOIN player_details pd ON p.id = pd.player_id
                WHERE r.region = ? AND r.slug = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
            `, reg, rslug).Scan(&total); err != nil { return fmt.Errorf("players total (realm): %w", err) }
            pages := (total + pageSize - 1) / pageSize
            for p := 1; p <= pages; p++ {
                offset := (p-1) * pageSize
                rows, err := db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE r.region = ? AND r.slug = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, reg, rslug, pageSize, offset)
                if err != nil { return err }
                type PlayerRow struct {
                    ID int64; Name string; RealmSlug string; RealmName string; Region string; ClassName string; ActiveSpecName string; MainSpecID sql.NullInt64; CombinedBestTime sql.NullInt64; DungeonsCompleted int; TotalRuns int
                }
                list := []map[string]any{}
                for rows.Next() {
                    var r PlayerRow
                    if err := rows.Scan(&r.ID, &r.Name, &r.RealmSlug, &r.RealmName, &r.Region, &r.ClassName, &r.ActiveSpecName, &r.MainSpecID, &r.CombinedBestTime, &r.DungeonsCompleted, &r.TotalRuns); err != nil { rows.Close(); return err }
                    obj := map[string]any{
                        "player_id": r.ID,
                        "name": r.Name,
                        "realm_slug": r.RealmSlug,
                        "realm_name": r.RealmName,
                        "region": r.Region,
                        "class_name": r.ClassName,
                        "active_spec_name": r.ActiveSpecName,
                        "dungeons_completed": r.DungeonsCompleted,
                        "total_runs": r.TotalRuns,
                    }
                    if r.MainSpecID.Valid { obj["main_spec_id"] = int(r.MainSpecID.Int64) }
                    if r.CombinedBestTime.Valid { obj["combined_best_time"] = r.CombinedBestTime.Int64 }
                    list = append(list, obj)
                }
                rows.Close()
                page := map[string]any{
                    "leaderboard": list,
                    "title": strings.ToUpper(reg) + "/" + rslug + " Player Rankings",
                    "generated_timestamp": time.Now().UnixMilli(),
                    "pagination": map[string]any{
                        "currentPage": p,
                        "pageSize": pageSize,
                        "totalPlayers": total,
                        "totalPages": pages,
                        "hasNextPage": p < pages,
                        "hasPrevPage": p > 1,
                        "totalRuns": total,
                    },
                }
                if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), page); err != nil { return err }
            }
        }
    }

    // class-filtered variants
    classKeys := []string{"death_knight","druid","hunter","mage","monk","paladin","priest","rogue","shaman","warlock","warrior"}

    // helper to write class-scoped pages
    writeClassScope := func(scope string, region string, realmSlug string, classKey string) error {
        var dir string
        if scope == "global" {
            dir = filepath.Join(out, "players", "class", classKey, "global")
        } else if scope == "regional" {
            dir = filepath.Join(out, "players", "class", classKey, "regional", region)
        } else {
            dir = filepath.Join(out, "players", "class", classKey, "realm", region, realmSlug)
        }
        if err := os.MkdirAll(dir, 0o755); err != nil { return err }

        // totals
        var total int
        if scope == "global" {
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM players p
                JOIN player_profiles pp ON p.id = pp.player_id
                LEFT JOIN player_details pd ON p.id = pd.player_id
                WHERE pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                  AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
            `, classKey).Scan(&total); err != nil { return fmt.Errorf("players total (class global): %w", err) }
        } else if scope == "regional" {
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM players p
                JOIN realms r ON p.realm_id = r.id
                JOIN player_profiles pp ON p.id = pp.player_id
                LEFT JOIN player_details pd ON p.id = pd.player_id
                WHERE r.region = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                  AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
            `, region, classKey).Scan(&total); err != nil { return fmt.Errorf("players total (class regional): %w", err) }
        } else {
            if err := db.QueryRow(`
                SELECT COUNT(*)
                FROM players p
                JOIN realms r ON p.realm_id = r.id
                JOIN player_profiles pp ON p.id = pp.player_id
                LEFT JOIN player_details pd ON p.id = pd.player_id
                WHERE r.region = ? AND r.slug = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                  AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
            `, region, realmSlug, classKey).Scan(&total); err != nil { return fmt.Errorf("players total (class realm): %w", err) }
        }
        pages := (total + pageSize - 1) / pageSize

        for p := 1; p <= pages; p++ {
            offset := (p-1) * pageSize
            var rows *sql.Rows
            var err error
            if scope == "global" {
                rows, err = db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                      AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, classKey, pageSize, offset)
            } else if scope == "regional" {
                rows, err = db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE r.region = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                      AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, region, classKey, pageSize, offset)
            } else {
                rows, err = db.Query(`
                    SELECT p.id, p.name, r.slug, r.name, r.region,
                           COALESCE(pd.class_name,''), COALESCE(pd.active_spec_name,''), pp.main_spec_id,
                           pp.combined_best_time, pp.dungeons_completed, pp.total_runs
                    FROM players p
                    JOIN realms r ON p.realm_id = r.id
                    JOIN player_profiles pp ON p.id = pp.player_id
                    LEFT JOIN player_details pd ON p.id = pd.player_id
                    WHERE r.region = ? AND r.slug = ? AND pp.has_complete_coverage = 1 AND pp.combined_best_time IS NOT NULL
                      AND REPLACE(LOWER(pd.class_name),' ', '_') = ?
                    ORDER BY pp.combined_best_time ASC, p.name ASC
                    LIMIT ? OFFSET ?
                `, region, realmSlug, classKey, pageSize, offset)
            }
            if err != nil { return err }
            type PlayerRow struct { ID int64; Name, RealmSlug, RealmName, Region, ClassName, ActiveSpecName string; MainSpecID sql.NullInt64; CombinedBestTime sql.NullInt64; DungeonsCompleted, TotalRuns int }
            list := []map[string]any{}
            for rows.Next() {
                var r PlayerRow
                if err := rows.Scan(&r.ID, &r.Name, &r.RealmSlug, &r.RealmName, &r.Region, &r.ClassName, &r.ActiveSpecName, &r.MainSpecID, &r.CombinedBestTime, &r.DungeonsCompleted, &r.TotalRuns); err != nil { rows.Close(); return err }
                obj := map[string]any{
                    "player_id": r.ID,
                    "name": r.Name,
                    "realm_slug": r.RealmSlug,
                    "realm_name": r.RealmName,
                    "region": r.Region,
                    "class_name": r.ClassName,
                    "active_spec_name": r.ActiveSpecName,
                    "dungeons_completed": r.DungeonsCompleted,
                    "total_runs": r.TotalRuns,
                }
                if r.MainSpecID.Valid { obj["main_spec_id"] = int(r.MainSpecID.Int64) }
                if r.CombinedBestTime.Valid { obj["combined_best_time"] = r.CombinedBestTime.Int64 }
                list = append(list, obj)
            }
            rows.Close()
            pageObj := map[string]any{
                "leaderboard": list,
                "title": func() string {
                    if scope == "global" { return "Global Player Rankings" }
                    if scope == "regional" { return strings.ToUpper(region) + " Player Rankings" }
                    return strings.ToUpper(region) + "/" + realmSlug + " Player Rankings"
                }(),
                "generated_timestamp": time.Now().UnixMilli(),
                "pagination": map[string]any{
                    "currentPage": p,
                    "pageSize": pageSize,
                    "totalPlayers": total,
                    "totalPages": pages,
                    "hasNextPage": p < pages,
                    "hasPrevPage": p > 1,
                    "totalRuns": total,
                },
            }
            if err := writeJSONFile(filepath.Join(dir, fmt.Sprintf("%d.json", p)), pageObj); err != nil { return err }
        }
        return nil
    }

    // Generate class pages
    for _, cls := range classKeys {
        // global
        if err := writeClassScope("global", "", "", cls); err != nil { return err }
        // regional
        for _, reg := range regions {
            if err := writeClassScope("regional", reg, "", cls); err != nil { return err }
            // realm
            rrows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, reg)
            if err != nil { return fmt.Errorf("players class realms list: %w", err) }
            slugs := []string{}
            for rrows.Next() { var s string; if err := rrows.Scan(&s); err != nil { rrows.Close(); return err }; slugs = append(slugs, s) }
            rrows.Close()
            for _, rslug := range slugs {
                if err := writeClassScope("realm", reg, rslug, cls); err != nil { return err }
            }
        }
    }
    fmt.Println("[OK] Player leaderboards generated")
    return nil
}

// ---------------- Search index ----------------
type SearchEntry struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Region    string `json:"region"`
    RealmSlug string `json:"realm_slug"`
    RealmName string `json:"realm_name"`
    ClassName string `json:"class_name,omitempty"`
    GlobalRanking *int `json:"global_ranking,omitempty"`
    GlobalBracket string `json:"global_ranking_bracket,omitempty"`
}

func generateSearchIndex(db *sql.DB, out string, shardSize int) error {
    if shardSize <= 0 { shardSize = 5000 }
    if err := os.MkdirAll(out, 0o755); err != nil { return err }
    rows, err := db.Query(`
        SELECT p.id, p.name, r.region, r.slug, r.name,
               COALESCE(pd.class_name, ''), pp.global_ranking, COALESCE(pp.global_ranking_bracket,'')
        FROM players p
        JOIN player_profiles pp ON p.id = pp.player_id
        JOIN realms r ON p.realm_id = r.id
        LEFT JOIN player_details pd ON p.id = pd.player_id
        WHERE pp.has_complete_coverage = 1
        ORDER BY pp.global_ranking ASC, p.name ASC
    `)
    if err != nil { return err }
    defer rows.Close()
    shard := 0
    count := 0
    buf := []SearchEntry{}
    // Precompute total
    var totalPlayers int
    if err := db.QueryRow(`
      SELECT COUNT(*)
      FROM players p
      JOIN player_profiles pp ON p.id = pp.player_id
      JOIN realms r ON p.realm_id = r.id
      WHERE pp.has_complete_coverage = 1
    `).Scan(&totalPlayers); err != nil { return err }
    flush := func() error {
        if len(buf) == 0 { return nil }
        path := filepath.Join(out, fmt.Sprintf("players-%03d.json", shard))
        meta := map[string]any{
            "total_players": totalPlayers,
            "returned_players": len(buf),
            "offset": shard*shardSize,
            "limit": shardSize,
            "last_updated": time.Now().Format(time.RFC3339),
        }
        if err := writeJSONFile(path, map[string]any{"players": buf, "metadata": meta}); err != nil { return err }
        shard++; buf = buf[:0]
        return nil
    }
    for rows.Next() {
        var e SearchEntry
        var className sql.NullString
        var gr sql.NullInt64
        var gb string
        if err := rows.Scan(&e.ID, &e.Name, &e.Region, &e.RealmSlug, &e.RealmName, &className, &gr, &gb); err != nil { return err }
        e.ClassName = className.String
        e.GlobalBracket = gb
        if gr.Valid { v := int(gr.Int64); e.GlobalRanking = &v }
        buf = append(buf, e)
        count++
        if len(buf) >= shardSize { if err := flush(); err != nil { return err } }
    }
    if err := rows.Err(); err != nil { return err }
    if err := flush(); err != nil { return err }
    fmt.Printf("[OK] Generated search index: %d players in %d shards\n", count, shard)
    return nil
}

func init() {
    rootCmd.AddCommand(generateCmd)
    generateCmd.AddCommand(generateAPICmd)
    generateAPICmd.Flags().String("out", "public", "Output directory for static API")
    generateAPICmd.Flags().Bool("players", true, "Generate player profile JSON endpoints")
    generateAPICmd.Flags().Bool("leaderboards", true, "Generate leaderboard JSON endpoints")
    generateAPICmd.Flags().Bool("search", true, "Generate search index JSON shards")
    generateAPICmd.Flags().Int("page-size", 25, "Leaderboard page size")
    generateAPICmd.Flags().Int("shard-size", 5000, "Search index shard size")
    generateAPICmd.Flags().String("regions", "us,eu,kr,tw", "Regions to include for regional leaderboards")
}
