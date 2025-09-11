package cmd

import (
	"context"
	"fmt"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch data from Blizzard API",
	Long:  `Fetch challenge mode leaderboards and player profiles from Blizzard API.`,
}

var fetchCMCmd = &cobra.Command{
	Use:   "cm",
	Short: "Fetch challenge mode leaderboards",
	Long:  `Fetch challenge mode leaderboard data for all realms and dungeons.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Fetching challenge mode leaderboards...")

    db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Blizzard API client: %w", err)
		}

		// enable verbose logging if requested
		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		client.Verbose = verbose

		// optional: reduce fallback depth for incremental runs to speed up
		fallbackDepth, _ := cmd.Flags().GetInt("fallback-depth")
		if fallbackDepth > 0 {
			client.FallbackLimit = fallbackDepth
		}

		fmt.Println("Connected to local database")
		fmt.Println("Blizzard API client initialized")

		// initialize database service
		dbService := database.NewDatabaseService(db)

		// set database verbose based on flag
		database.SetVerbose(verbose)

		// ensure auxiliary tables for incremental markers exist
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS api_fetch_markers (
				realm_slug TEXT NOT NULL,
				dungeon_id INTEGER NOT NULL,
				period_id INTEGER NOT NULL,
				last_completed_ts INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (realm_slug, dungeon_id, period_id)
			)`); err != nil {
			return fmt.Errorf("failed to ensure api_fetch_markers table: %w", err)
		}

		// check for incremental mode and force flags
		incremental, _ := cmd.Flags().GetBool("incremental")
		force, _ := cmd.Flags().GetBool("force")

		// check last fetch info
		lastFetch, prevRuns, prevPlayers, err := dbService.GetLastFetchInfo("challenge_mode_leaderboard")
		if err != nil {
			return fmt.Errorf("failed to get last fetch info: %w", err)
		}

		if lastFetch != nil {
			lastFetchTime := time.UnixMilli(*lastFetch)
			fmt.Printf("Last fetch: %s\n", lastFetchTime.UTC().Format("2006-01-02 15:04:05 UTC"))
			fmt.Printf("Previous fetch: %d runs, %d players\n", prevRuns, prevPlayers)

			if incremental && !force {
				timeSinceLastFetch := time.Since(lastFetchTime)
				if timeSinceLastFetch < 1*time.Hour {
					fmt.Printf("Incremental mode: Last fetch was %.1f minutes ago\n", timeSinceLastFetch.Minutes())
					fmt.Printf("Skipping fetch - use --force to override\n")
					return nil
				}
				fmt.Printf("Incremental mode: Last fetch was %.1f hours ago - proceeding\n", timeSinceLastFetch.Hours())
				// If user didn't explicitly set fallback-depth, prefer a shallow fallback for incremental runs
				if fallbackDepth == 0 {
					client.FallbackLimit = 2
				}
			} else if force {
				fmt.Printf("Force mode: Proceeding despite recent fetch\n")
			}
		} else {
			if incremental {
				fmt.Println("Incremental mode: First time running - full database population")
			} else {
				fmt.Println("First time running - full database population")
			}
		}

    _, dungeons := blizzard.GetHardcodedPeriodAndDungeons()
    allRealms := blizzard.GetAllRealms()
    fmt.Printf("Dungeons: %d, Realms: %d\n", len(dungeons), len(allRealms))

		// Optional filtering: regions/realms/dungeons
		regionsCSV, _ := cmd.Flags().GetString("regions")
		realmsCSV, _ := cmd.Flags().GetString("realms")
		dungeonsCSV, _ := cmd.Flags().GetString("dungeons")

		if strings.TrimSpace(regionsCSV) != "" {
			allowed := map[string]bool{}
			for _, r := range strings.Split(regionsCSV, ",") {
				allowed[strings.TrimSpace(r)] = true
			}
			for slug, info := range allRealms {
				if !allowed[info.Region] {
					delete(allRealms, slug)
				}
			}
		}
		if strings.TrimSpace(realmsCSV) != "" {
			allowed := map[string]bool{}
			for _, s := range strings.Split(realmsCSV, ",") {
				allowed[strings.TrimSpace(s)] = true
			}
			for slug := range allRealms {
				if !allowed[slug] {
					delete(allRealms, slug)
				}
			}
		}
		if strings.TrimSpace(dungeonsCSV) != "" {
			// parse list of ids or slugs
			allowed := map[string]bool{}
			for _, s := range strings.Split(dungeonsCSV, ",") {
				allowed[strings.TrimSpace(s)] = true
			}
			filtered := make([]blizzard.DungeonInfo, 0, len(dungeons))
			for _, d := range dungeons {
				idStr := fmt.Sprintf("%d", d.ID)
				if allowed[idStr] || allowed[d.Slug] {
					filtered = append(filtered, d)
				}
			}
			if len(filtered) > 0 {
				dungeons = filtered
			}
		}

        // pre-populate reference data (optimized): insert all dungeons once, then batch insert realms
        fmt.Printf("Pre-populating reference data...\n")
        fmt.Printf("  • Ensuring dungeons (%d)\n", len(dungeons))
        // ensure slugs are set on realmInfo entries
        for realmSlug, realmInfo := range allRealms {
            realmInfo.Slug = realmSlug
            allRealms[realmSlug] = realmInfo
        }
        if err := dbService.EnsureDungeonsOnce(dungeons); err != nil {
            return fmt.Errorf("failed to ensure dungeons: %w", err)
        }
        fmt.Printf("  ✓ Dungeons ensured\n")
        fmt.Printf("  • Ensuring realms (%d)\n", len(allRealms))
        if err := dbService.EnsureRealmsBatch(allRealms); err != nil {
            return fmt.Errorf("failed to ensure realms: %w", err)
        }
        fmt.Printf("  ✓ Realms ensured\n")
        fmt.Printf("Reference data populated for %d realms and %d dungeons\n", len(allRealms), len(dungeons))

        // Determine mode and periods
        sweep, _ := cmd.Flags().GetBool("sweep-periods")
        fallbackMode, _ := cmd.Flags().GetBool("fallback-mode")
        periodsCSV, _ := cmd.Flags().GetString("periods")
        ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
        defer cancel()

        totalRuns := 0
        totalPlayers := 0

        if sweep && !fallbackMode {
            // Global sweep: iterate periods newest→oldest, fetch all realms×dungeons per period
            var periods []string
            if strings.TrimSpace(periodsCSV) != "" {
                for _, p := range strings.Split(periodsCSV, ",") { if v := strings.TrimSpace(p); v != "" { periods = append(periods, v) } }
            } else {
                periods = blizzard.GetGlobalPeriods()
            }
            fmt.Printf("\nStarting global period sweep: %v\n", periods)
            for _, period := range periods {
                fmt.Printf("\n--- Period %s ---\n", period)
                res := client.FetchAllRealmsConcurrent(ctx, allRealms, dungeons, period)
                runs, players, berr := dbService.BatchProcessFetchResults(ctx, res)
                if berr != nil { fmt.Printf("Batch errors in period %s: %v\n", period, berr) }
                fmt.Printf("Period %s → inserted runs: %d, new players: %d\n", period, runs, players)
                totalRuns += runs
                totalPlayers += players
            }
        } else {
            // Legacy per-dungeon fallback mode
            fallbackPeriods := blizzard.GetFallbackPeriods()
            fmt.Printf("Using multi-period fallback strategy: %v\n", fallbackPeriods)
            fmt.Printf("\nStarting concurrent API fetching (fallback mode)...\n")
            startTime := time.Now()
            results := client.FetchAllRealmsConcurrentWithFallback(ctx, allRealms, dungeons)
            runs, players, batchErr := dbService.BatchProcessFetchResults(ctx, results)
            if batchErr != nil { fmt.Printf("Batch processing encountered errors: %v\n", batchErr) }
            elapsed := time.Since(startTime)
            if verbose {
                req, nf, avg := client.Stats()
                fmt.Printf("HTTP metrics: requests=%d, 404s=%d, avg=%.1fms\n", req, nf, avg)
            }
            fmt.Printf("Completed fallback run in %v\n", elapsed)
            totalRuns += runs
            totalPlayers += players
        }

        // update fetch metadata
        if err := dbService.UpdateFetchMetadata("challenge_mode_leaderboard", totalRuns, totalPlayers); err != nil {
            return fmt.Errorf("failed to update fetch metadata: %w", err)
        }

        fmt.Printf("\nSuccessfully inserted %d runs and %d new players into local database\n", totalRuns, totalPlayers)
        fmt.Printf("Database saved to: %s\n", database.DBFilePath())

        return nil
    },
}

var fetchProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Fetch detailed player profiles",
	Long:  `Fetch detailed player profile data including equipment and character information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Player Profile Fetcher (Go Implementation) ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Blizzard API client: %w", err)
		}

		fmt.Println("Connected to local database")
		fmt.Println("Blizzard API client initialized")

		// initialize database service
		dbService := database.NewDatabaseService(db)

		// get eligible players (9/9 completion)
		fmt.Println("\nFinding eligible players with complete coverage (9/9 dungeons)...")
		players, err := dbService.GetEligiblePlayersForProfileFetch()
		if err != nil {
			return fmt.Errorf("failed to get eligible players: %w", err)
		}

    if len(players) == 0 {
        fmt.Println("No eligible players found. Run 'ookstats process players' first to generate player profiles.")
        return nil
    }

		fmt.Printf("Found %d eligible players with 9/9 completion\n", len(players))

		// check batch size flag
		batchSize, _ := cmd.Flags().GetInt("batch-size")
		if batchSize <= 0 {
			batchSize = 20 // default batch size
		}

		maxPlayers, _ := cmd.Flags().GetInt("max-players")
		if maxPlayers > 0 && len(players) > maxPlayers {
			players = players[:maxPlayers]
			fmt.Printf("Limited to first %d players due to --max-players flag\n", maxPlayers)
		}

		fmt.Printf("Processing %d players in batches of %d with 20 concurrent requests\n", len(players), batchSize)

		// start concurrent fetching with batch processing
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		fmt.Printf("\nStarting player profile fetching...\n")
		startTime := time.Now()

		totalProfiles := 0
		totalEquipment := 0
		processedCount := 0

		// process in batches to avoid overwhelming the API
		for i := 0; i < len(players); i += batchSize {
			end := i + batchSize
			if end > len(players) {
				end = len(players)
			}
			batch := players[i:end]

			batchNumber := (i / batchSize) + 1
			totalBatches := (len(players) + batchSize - 1) / batchSize

			fmt.Printf("\n--- Batch %d/%d (%d players) ---\n", batchNumber, totalBatches, len(batch))

			// fetch profiles concurrently for this batch
			results := client.FetchPlayerProfilesConcurrent(ctx, batch)

			// process results
			batchProfiles := 0
			batchEquipment := 0
			timestamp := time.Now().UnixMilli()

			for result := range results {
				processedCount++

				if result.Error != nil {
					fmt.Printf("  ❌ %s (%s): %v\n", result.PlayerName, result.Region, result.Error)
					continue
				}

				// insert profile data
				profiles, equipment, err := dbService.InsertPlayerProfileData(result, timestamp)
				if err != nil {
					fmt.Printf("  ❌ %s (%s): DB error - %v\n", result.PlayerName, result.Region, err)
					continue
				}

				batchProfiles += profiles
				batchEquipment += equipment

				// show success status
				statusParts := []string{}
				if result.Summary != nil {
					statusParts = append(statusParts, "profile")
				}
				if result.Equipment != nil {
					statusParts = append(statusParts, fmt.Sprintf("%d items", equipment))
				}
				if result.Media != nil {
					statusParts = append(statusParts, "avatar")
				}

				if len(statusParts) > 0 {
					fmt.Printf("  ✅ %s (%s): %s\n", result.PlayerName, result.Region, strings.Join(statusParts, ", "))
				}
			}

			totalProfiles += batchProfiles
			totalEquipment += batchEquipment

			elapsed := time.Since(startTime)
			fmt.Printf("  → Batch %d complete: %d profiles, %d items (Total: %d/%d players, %.1f players/min)\n",
				batchNumber, batchProfiles, batchEquipment, processedCount, len(players),
				float64(processedCount)/elapsed.Minutes())

			// small delay between batches to be respectful to the API
			if i+batchSize < len(players) {
				fmt.Printf("  ⏳ Waiting 1 second before next batch...\n")
				time.Sleep(1 * time.Second)
			}
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\n✅ Player profile fetching complete!\n")
		fmt.Printf("   Processed: %d/%d players in %v\n", processedCount, len(players), elapsed)
		fmt.Printf("   Updated: %d player profiles\n", totalProfiles)
		fmt.Printf("   Updated: %d equipment items\n", totalEquipment)

		if processedCount > 0 {
			rate := float64(processedCount) / elapsed.Minutes()
			fmt.Printf("   Rate: %.1f players/minute\n", rate)
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Run 'ookstats sync' to push profile data to Turso")
		fmt.Println("  2. Rebuild the website to include new player profile data")

		return nil
	},
}

func init() {
    rootCmd.AddCommand(fetchCmd)
    fetchCmd.AddCommand(fetchCMCmd)
    fetchCmd.AddCommand(fetchProfilesCmd)

	// add incremental processing flags for CM fetching
	fetchCMCmd.Flags().Bool("incremental", false, "Skip fetching if last fetch was recent")
	fetchCMCmd.Flags().Bool("force", false, "Force fetch even in incremental mode")
	fetchCMCmd.Flags().Int("fallback-depth", 0, "Limit number of fallback periods to try per dungeon (0 = default)")
	fetchCMCmd.Flags().Int("concurrency", 20, "Max concurrent API requests")
	fetchCMCmd.Flags().Int("api-timeout-seconds", 15, "HTTP client timeout in seconds")
	fetchCMCmd.Flags().String("regions", "", "Comma-separated regions to include (us,eu,kr)")
	fetchCMCmd.Flags().String("realms", "", "Comma-separated realm slugs to include")
    fetchCMCmd.Flags().String("dungeons", "", "Comma-separated dungeon IDs or slugs to include")
    // period strategy
    fetchCMCmd.Flags().Bool("sweep-periods", true, "Sweep periods globally (newest→oldest) and insert any new runs")
    fetchCMCmd.Flags().String("periods", "", "Comma-separated period IDs to sweep (overrides default)")
    fetchCMCmd.Flags().Bool("fallback-mode", false, "Use per-dungeon fallback mode instead of global sweep")

	// add player profile fetching flags
	fetchProfilesCmd.Flags().Int("batch-size", 20, "Number of players to process per batch")
	fetchProfilesCmd.Flags().Int("max-players", 0, "Maximum number of players to process (0 = no limit)")
}
