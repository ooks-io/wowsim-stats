package cmd

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"
    "ookstats/internal/blizzard"
    "ookstats/internal/database"
)

// buildCmd orchestrates a full from-scratch rebuild + static API generation
var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Full rebuild of database and static API",
    Long:  `From-scratch hourly rebuild: init schema, fetch CM runs (period sweep), process players + rankings, fetch profiles, and generate static API.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        outDir, _ := cmd.Flags().GetString("out")
        if strings.TrimSpace(outDir) == "" {
            return errors.New("--out is required (e.g. web/public or web/public/api)")
        }

        // Normalize output: our generator writes under "+/api". Accept either web/public or web/public/api and normalize to parent.
        normalizedOut := strings.TrimRight(outDir, string(os.PathSeparator))
        if filepath.Base(normalizedOut) == "api" {
            normalizedOut = filepath.Dir(normalizedOut)
        }

        fromScratch, _ := cmd.Flags().GetBool("from-scratch")
        regionsCSV, _ := cmd.Flags().GetString("regions")
        pageSize, _ := cmd.Flags().GetInt("page-size")
        shardSize, _ := cmd.Flags().GetInt("shard-size")
        wowsimsDB, _ := cmd.Flags().GetString("wowsims-db")
        skipProfiles, _ := cmd.Flags().GetBool("skip-profiles")
        periodsCSV, _ := cmd.Flags().GetString("periods")
        concurrency, _ := cmd.Flags().GetInt("concurrency")

        // optional verbose logging propagated to API client
        verbose, _ := cmd.InheritedFlags().GetBool("verbose")

        // Handle from-scratch (file DSN only)
        if fromScratch {
            dbPath := database.DBFilePath()
            // Only remove if it's a plain local file path
            if !strings.Contains(dbPath, "://") {
                if _, err := os.Stat(dbPath); err == nil {
                    fmt.Printf("Removing existing local DB: %s\n", dbPath)
                    if rmErr := os.Remove(dbPath); rmErr != nil {
                        return fmt.Errorf("failed to remove db file: %w", rmErr)
                    }
                }
            } else {
                fmt.Printf("--from-scratch requested but DSN is not a local file (%s) - skipping deletion.\n", dbPath)
            }
        }

        // 1) Schema init
        db, err := database.Connect()
        if err != nil {
            return fmt.Errorf("db connect: %w", err)
        }
        defer db.Close()

        if err := database.EnsureCompleteSchema(db); err != nil {
            return fmt.Errorf("schema init: %w", err)
        }

        // 2) Populate items (embedded default; file can override)
        fmt.Printf("\n=== Populating items (embedded or file override) ===\n")
        if err := populateItems(db, wowsimsDB); err != nil {
            return fmt.Errorf("populate items: %w", err)
        }

        // 3) Fetch CM runs using global period sweep
        fmt.Println("\n=== Fetching Challenge Mode leaderboards (global period sweep) ===")
        client, err := blizzard.NewClient()
        if err != nil {
            return fmt.Errorf("blizzard client: %w", err)
        }
        client.Verbose = verbose
        if concurrency > 0 {
            client.SetConcurrency(concurrency)
        }

        // Realms and dungeons
        _, dungeons := blizzard.GetHardcodedPeriodAndDungeons()
        allRealms := blizzard.GetAllRealms()
        fmt.Printf("Dungeons: %d, Realms: %d\n", len(dungeons), len(allRealms))

        // Optional region filter
        if strings.TrimSpace(regionsCSV) != "" {
            allowed := map[string]bool{}
            for _, r := range strings.Split(regionsCSV, ",") { allowed[strings.TrimSpace(r)] = true }
            for slug, info := range allRealms {
                if !allowed[info.Region] { delete(allRealms, slug) }
            }
        }

        // Ensure dungeons/realms exist
        dbService := database.NewDatabaseService(db)
        if err := dbService.EnsureDungeonsOnce(dungeons); err != nil {
            return fmt.Errorf("ensure dungeons: %w", err)
        }
        if err := dbService.EnsureRealmsBatch(allRealms); err != nil {
            return fmt.Errorf("ensure realms: %w", err)
        }
        
        // control database-internal verbosity (hide 404 noise unless verbose)
        database.SetVerbose(verbose)

        // Determine periods to sweep
        periods := []string{}
        if strings.TrimSpace(periodsCSV) != "" {
            for _, p := range strings.Split(periodsCSV, ",") {
                v := strings.TrimSpace(p)
                if v != "" { periods = append(periods, v) }
            }
        } else {
            periods = blizzard.GetGlobalPeriods()
        }
        fmt.Printf("Sweeping periods (newest -> oldest): %v\n", periods)

        totalRuns := 0
        totalPlayers := 0

        ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
        defer cancel()

        sweepStart := time.Now()
        for _, period := range periods {
            fmt.Printf("\n--- Period %s ---\n", period)
            realmResults := client.FetchAllRealmsConcurrent(ctx, allRealms, dungeons, period)
            runs, players, err := dbService.BatchProcessFetchResults(ctx, realmResults)
            if err != nil {
                fmt.Printf("Batch processing errors in period %s: %v\n", period, err)
            }
            fmt.Printf("Period %s -> inserted runs: %d, new players: %d\n", period, runs, players)
            totalRuns += runs
            totalPlayers += players
        }
        fmt.Printf("\nSweep complete in %v\n", time.Since(sweepStart))

        if err := dbService.UpdateFetchMetadata("challenge_mode_leaderboard", totalRuns, totalPlayers); err != nil {
            return fmt.Errorf("update fetch metadata: %w", err)
        }
        fmt.Printf("Total inserted runs: %d, new players: %d\n", totalRuns, totalPlayers)

        // 4) Process players (aggregations + rankings)
        fmt.Println("\n=== Processing Players (aggregations + rankings) ===")
        if err := processPlayersOnce(db); err != nil {
            return err
        }

        // 5) Fetch detailed player profiles (optional)
        if !skipProfiles {
            fmt.Println("\n=== Fetching detailed player profiles (9/9) ===")
            if err := fetchProfilesOnce(db, client); err != nil {
                return err
            }
        } else {
            fmt.Println("Skipping player profile fetch per flag")
        }

        // 6) Process run rankings (global/regional)
        fmt.Println("\n=== Processing Run Rankings (global + regional) ===")
        if err := processRunRankingsOnce(db); err != nil {
            return err
        }

        // 7) Generate static API
        fmt.Println("\n=== Generating static API ===")
        if err := generateAllAPI(db, normalizedOut, pageSize, shardSize, regionsCSV); err != nil {
            return err
        }

        // Print summary
        summarizeBuild(db, normalizedOut)

        fmt.Printf("\nBuild complete. Static API at %s/api\n", normalizedOut)
        return nil
    },
}

// processPlayersOnce runs the same steps as `process players`
func processPlayersOnce(db *sql.DB) error {
    tx, err := db.Begin()
    if err != nil { return fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback()

    if _, err := createPlayerAggregations(tx); err != nil {
        return fmt.Errorf("create player aggregations: %w", err)
    }
    if _, err := computePlayerRankings(tx); err != nil {
        return fmt.Errorf("compute player rankings: %w", err)
    }
    if err := tx.Commit(); err != nil { return fmt.Errorf("commit players: %w", err) }
    if _, err := db.Exec("VACUUM"); err != nil { fmt.Printf("VACUUM warning: %v\n", err) }
    return nil
}

// processRunRankingsOnce runs the same steps as `process rankings`
func processRunRankingsOnce(db *sql.DB) error {
    tx, err := db.Begin()
    if err != nil { return fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback()

    if err := computeGlobalRankings(tx); err != nil { return fmt.Errorf("global rankings: %w", err) }
    if err := computeRegionalRankings(tx); err != nil { return fmt.Errorf("regional rankings: %w", err) }
    if err := tx.Commit(); err != nil { return fmt.Errorf("commit run rankings: %w", err) }
    if _, err := db.Exec("VACUUM"); err != nil { fmt.Printf("VACUUM warning: %v\n", err) }
    return nil
}

// fetchProfilesOnce runs the same logic as `fetch profiles`
func fetchProfilesOnce(db *sql.DB, client *blizzard.Client) error {
    dbService := database.NewDatabaseService(db)
    players, err := dbService.GetEligiblePlayersForProfileFetch()
    if err != nil { return fmt.Errorf("eligible players: %w", err) }
    if len(players) == 0 { fmt.Println("No eligible players (9/9). Skipping profiles."); return nil }

    // batch in reasonable size
    batchSize := 20
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    totalProfiles := 0
    totalItems := 0
    processed := 0
    start := time.Now()

    for i := 0; i < len(players); i += batchSize {
        end := i + batchSize
        if end > len(players) { end = len(players) }
        batch := players[i:end]
        fmt.Printf("\nProfiles batch %d/%d (%d players)\n", (i/batchSize)+1, (len(players)+batchSize-1)/batchSize, len(batch))
        results := client.FetchPlayerProfilesConcurrent(ctx, batch)
        ts := time.Now().UnixMilli()
        batchProfiles := 0
        batchItems := 0
        for res := range results {
            processed++
            if res.Error != nil { fmt.Printf("  [ERROR] %s (%s): %v\n", res.PlayerName, res.Region, res.Error); continue }
            profs, items, err := dbService.InsertPlayerProfileData(res, ts)
            if err != nil { fmt.Printf("  [ERROR] %s (%s): DB error - %v\n", res.PlayerName, res.Region, err); continue }
            batchProfiles += profs; batchItems += items
        }
        totalProfiles += batchProfiles
        totalItems += batchItems
        fmt.Printf("  -> Batch complete: %d profiles, %d items (Total %d/%d)\n", batchProfiles, batchItems, processed, len(players))
        if i+batchSize < len(players) { time.Sleep(1 * time.Second) }
    }

    fmt.Printf("Profiles complete: %d profiles, %d items in %v\n", totalProfiles, totalItems, time.Since(start))
    return nil
}

// generateAllAPI mirrors the behavior of `generate api`
func generateAllAPI(db *sql.DB, outParent string, pageSize, shardSize int, regionsCSV string) error {
    base := filepath.Join(outParent, "api")
    if err := os.MkdirAll(base, 0o755); err != nil { return fmt.Errorf("mkdir: %w", err) }

    // players
    if err := generatePlayers(db, filepath.Join(base, "player"), ""); err != nil { return err }

    // leaderboards (+ players rankings)
    regions := []string{}
    if strings.TrimSpace(regionsCSV) != "" {
        for _, r := range strings.Split(regionsCSV, ",") {
            rr := strings.TrimSpace(r)
            if rr != "" { regions = append(regions, rr) }
        }
    }
    if err := generateLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil { return err }
    if err := generatePlayerLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil { return err }

    // search index
    if err := generateSearchIndex(db, filepath.Join(base, "search"), shardSize); err != nil { return err }

    fmt.Println("[OK] Static API generated")
    return nil
}

// summarizeBuild prints DB summary and per-realm period coverage
func summarizeBuild(db *sql.DB, outParent string) {
    var runCount, playerCount, completePlayers, detailsCount int
    _ = db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)
    _ = db.QueryRow("SELECT COUNT(*) FROM players").Scan(&playerCount)
    _ = db.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE has_complete_coverage = 1").Scan(&completePlayers)
    _ = db.QueryRow("SELECT COUNT(*) FROM player_details").Scan(&detailsCount)

    fmt.Printf("\n===== Build Summary =====\n")
    fmt.Printf("Runs: %d\n", runCount)
    fmt.Printf("Players: %d\n", playerCount)
    fmt.Printf("Complete Coverage Players (9/9): %d\n", completePlayers)
    fmt.Printf("Player Details Rows: %d\n", detailsCount)

    fmt.Printf("\nPer-Realm Period Coverage:\n")
    rows, err := db.Query(`
        SELECT r.slug, GROUP_CONCAT(DISTINCT cr.period_id ORDER BY cr.period_id DESC)
        FROM challenge_runs cr
        JOIN realms r ON cr.realm_id = r.id
        GROUP BY r.slug
        ORDER BY r.region, r.slug`)
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var slug, periods string
            if err := rows.Scan(&slug, &periods); err == nil {
                if periods == "" { periods = "-" }
                fmt.Printf("  %s: [%s]\n", slug, periods)
            }
        }
    }
}

func init() {
    rootCmd.AddCommand(buildCmd)
    buildCmd.Flags().String("out", "", "Parent output directory for static API (e.g. web/public or web/public/api)")
    buildCmd.Flags().Bool("from-scratch", true, "Delete local DB file first if using a file-based DSN")
    buildCmd.Flags().String("regions", "", "Comma-separated regions to include (default: all)")
    buildCmd.Flags().Int("page-size", 25, "Leaderboard pagination size")
    buildCmd.Flags().Int("shard-size", 5000, "Search index shard size")
    buildCmd.Flags().String("wowsims-db", "", "Optional path to WoWSims items JSON for item enrichment")
    buildCmd.Flags().Bool("skip-profiles", false, "Skip fetching player detailed profiles")
    buildCmd.Flags().String("periods", "", "Comma-separated period IDs to sweep (default: newest->oldest set)")
    buildCmd.Flags().Int("concurrency", 20, "Max concurrent API requests")
}
