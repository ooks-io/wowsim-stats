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
    "ookstats/internal/generator"
    "ookstats/internal/pipeline"
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

        // 3) Initialize Blizzard API client
        client, err := blizzard.NewClient()
        if err != nil {
            return fmt.Errorf("blizzard client: %w", err)
        }
        client.Verbose = verbose
        if concurrency > 0 {
            client.SetConcurrency(concurrency)
        }

        // 4) Sync season metadata from API
        fmt.Println("\n=== Syncing season metadata ===")
        if err := syncSeasons(db, client, regionsCSV); err != nil {
            return fmt.Errorf("sync seasons: %w", err)
        }

        // 5) Fetch CM runs using pipeline (includes child realm filtering)
        fmt.Println("\n=== Fetching Challenge Mode leaderboards (global period sweep) ===")

        dbService := database.NewDatabaseService(db)

        // control database-internal verbosity (hide 404 noise unless verbose)
        database.SetVerbose(verbose)

        // Parse regions for filter
        var regions []string
        if strings.TrimSpace(regionsCSV) != "" {
            for _, r := range strings.Split(regionsCSV, ",") {
                if trimmed := strings.TrimSpace(r); trimmed != "" {
                    regions = append(regions, trimmed)
                }
            }
        }

        // Parse periods (if provided)
        var periods []string
        if strings.TrimSpace(periodsCSV) != "" {
            var err error
            periods, err = blizzard.ParsePeriods(periodsCSV)
            if err != nil {
                return fmt.Errorf("failed to parse periods: %w", err)
            }
        }

        // Use pipeline function (handles child realm filtering automatically)
        fetchOpts := pipeline.FetchCMOptions{
            Verbose:     verbose,
            Regions:     regions,
            Realms:      []string{}, // no realm filter
            Dungeons:    []string{}, // no dungeon filter
            Periods:     periods,    // empty means fetch dynamically
            Concurrency: concurrency,
            Timeout:     45 * time.Minute,
        }

        result, err := pipeline.FetchChallengeMode(dbService, client, fetchOpts)
        if err != nil {
            return fmt.Errorf("fetch challenge mode: %w", err)
        }

        fmt.Printf("\n[OK] Fetch complete: %d runs, %d players in %v\n",
            result.TotalRuns, result.TotalPlayers, result.Duration)

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

        // 8) Generate status API via analyze
        fmt.Println("\n=== Generating status API (analyze) ===")
        statusDir := filepath.Join(normalizedOut, "api", "status")
        outPath := filepath.Join(statusDir, "latest-runs.json")
        // Get realms and dungeons for analyze
        _, dungeons := blizzard.GetHardcodedPeriodAndDungeons()
        allRealms := blizzard.GetAllRealms()
        if err := runAnalyze(db, client, allRealms, dungeons, periodsCSV, outPath, statusDir); err != nil {
            return fmt.Errorf("analyze status: %w", err)
        }

        // Print summary
        summarizeBuild(db, normalizedOut)

        fmt.Printf("\nBuild complete. Static API at %s/api\n", normalizedOut)
        return nil
    },
}

// processPlayersOnce runs the same steps as `process players`
func processPlayersOnce(db *sql.DB) error {
    opts := pipeline.ProcessPlayersOptions{
        Verbose: false,
    }
    _, _, err := pipeline.ProcessPlayers(db, opts)
    return err
}

// processRunRankingsOnce runs the same steps as `process rankings`
func processRunRankingsOnce(db *sql.DB) error {
    opts := pipeline.ProcessRunRankingsOptions{
        Verbose: false,
    }
    return pipeline.ProcessRunRankings(db, opts)
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
    if err := generator.GeneratePlayers(db, filepath.Join(base, "player"), ""); err != nil { return err }

    // leaderboards (+ players rankings)
    regions := []string{}
    if strings.TrimSpace(regionsCSV) != "" {
        for _, r := range strings.Split(regionsCSV, ",") {
            rr := strings.TrimSpace(r)
            if rr != "" { regions = append(regions, rr) }
        }
    }
    if err := generator.GenerateLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil { return err }
    if err := generator.GeneratePlayerLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil { return err }

    // search index
    if err := generator.GenerateSearchIndex(db, filepath.Join(base, "search"), shardSize); err != nil { return err }

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

// syncSeasons syncs season metadata from Blizzard API for all regions
func syncSeasons(db *sql.DB, client *blizzard.Client, regionsCSV string) error {
	dbService := database.NewDatabaseService(db)

	// Parse regions from CSV (seasons and periods are region-specific)
	var regions []string
	if strings.TrimSpace(regionsCSV) != "" {
		for _, r := range strings.Split(regionsCSV, ",") {
			if trimmed := strings.TrimSpace(r); trimmed != "" {
				regions = append(regions, trimmed)
			}
		}
	}

	// Default to all regions if none specified
	if len(regions) == 0 {
		regions = []string{"us", "eu", "kr", "tw"}
	}

	fmt.Printf("Fetching season metadata from %d regions...\n", len(regions))

	// Process each region
	for _, region := range regions {
		fmt.Printf("\n=== Region: %s ===\n", strings.ToUpper(region))

		// Fetch season index for this region
		seasonIndex, err := client.FetchSeasonIndex(region)
		if err != nil {
			fmt.Printf("Failed to fetch season index for %s: %v - skipping region\n", strings.ToUpper(region), err)
			continue
		}

		if len(seasonIndex.Seasons) == 0 {
			fmt.Printf("No seasons found for %s\n", strings.ToUpper(region))
			continue
		}

		fmt.Printf("Found %d seasons in %s\n", len(seasonIndex.Seasons), strings.ToUpper(region))

		// Process each season for this region
		for _, seasonRef := range seasonIndex.Seasons {
			seasonID := seasonRef.ID
			fmt.Printf("  Season %d: ", seasonID)

			// Fetch season details
			seasonDetail, err := client.FetchSeasonDetail(region, seasonID)
			if err != nil {
				fmt.Printf("error fetching details - %v\n", err)
				continue
			}

			// Upsert season with region
			dbSeasonID, err := dbService.UpsertSeason(seasonDetail.ID, region, seasonDetail.SeasonName, seasonDetail.StartTimestamp)
			if err != nil {
				fmt.Printf("error upserting - %v\n", err)
				continue
			}

			// Link periods to season
			if len(seasonDetail.Periods) > 0 {
				firstPeriod := seasonDetail.Periods[0].ID
				lastPeriod := seasonDetail.Periods[len(seasonDetail.Periods)-1].ID

				// Update period range
				err = dbService.UpdateSeasonPeriodRange(dbSeasonID, firstPeriod, lastPeriod)
				if err != nil {
					fmt.Printf("error updating period range - %v\n", err)
				}

				// Link each period
				for _, periodRef := range seasonDetail.Periods {
					err = dbService.LinkPeriodToSeason(periodRef.ID, dbSeasonID)
					if err != nil {
						fmt.Printf("error linking period %d - %v\n", periodRef.ID, err)
					}
				}
			}
			fmt.Printf("%s (%d periods)\n", seasonDetail.SeasonName, len(seasonDetail.Periods))
		}
	}

	fmt.Println("\n[OK] Season metadata synced for all regions")
	return nil
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
    buildCmd.Flags().String("periods", "", "Period specification: comma-separated list or ranges (e.g., '1020-1036' or '1020,1025,1030-1036'). Default: fetch all periods from API")
    buildCmd.Flags().Int("concurrency", 20, "Max concurrent API requests")
}
