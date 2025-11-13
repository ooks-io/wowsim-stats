package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"ookstats/internal/pipeline"
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

		// Enable verbose logging if requested
		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		client.Verbose = verbose
		database.SetVerbose(verbose)

		fmt.Println("Connected to local database")
		fmt.Println("Blizzard API client initialized")

		// Initialize database service
		dbService := database.NewDatabaseService(db)

		// Parse filters from flags
		regionsCSV, _ := cmd.Flags().GetString("regions")
		realmsCSV, _ := cmd.Flags().GetString("realms")
		dungeonsCSV, _ := cmd.Flags().GetString("dungeons")
		periodsSpec, _ := cmd.Flags().GetString("periods")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		timeoutSecs, _ := cmd.Flags().GetInt("api-timeout-seconds")

		// Convert CSV strings to slices
		var regions []string
		if strings.TrimSpace(regionsCSV) != "" {
			for _, r := range strings.Split(regionsCSV, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					regions = append(regions, r)
				}
			}
		}

		var realms []string
		if strings.TrimSpace(realmsCSV) != "" {
			for _, r := range strings.Split(realmsCSV, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					realms = append(realms, r)
				}
			}
		}

		var dungeons []string
		if strings.TrimSpace(dungeonsCSV) != "" {
			for _, d := range strings.Split(dungeonsCSV, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					dungeons = append(dungeons, d)
				}
			}
		}

		// Parse periods
		var periods []string
		if strings.TrimSpace(periodsSpec) != "" {
			periods, err = blizzard.ParsePeriods(periodsSpec)
			if err != nil {
				return fmt.Errorf("failed to parse periods: %w", err)
			}
		}

		// Set client concurrency
		if concurrency > 0 {
			client.SetConcurrency(concurrency)
		}

		// Build options
		opts := pipeline.FetchCMOptions{
			Verbose:     verbose,
			Regions:     regions,
			Realms:      realms,
			Dungeons:    dungeons,
			Periods:     periods,
			Concurrency: concurrency,
			Timeout:     time.Duration(timeoutSecs) * time.Second,
		}
		if opts.Timeout == 0 {
			opts.Timeout = 45 * time.Minute
		}

		// Fetch challenge mode data
		result, err := pipeline.FetchChallengeMode(dbService, client, opts)
		if err != nil {
			return err
		}

		fmt.Printf("\nSuccessfully inserted %d runs and %d new players into local database\n", result.TotalRuns, result.TotalPlayers)
		fmt.Printf("Database saved to: %s\n", database.DBFilePath())

		return nil
	},
}

var fetchProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Fetch detailed player profiles",
	Long:  `Fetch detailed player profile data including equipment and character information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Player Profile Fetcher ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Blizzard API client: %w", err)
		}

		// Enable verbose logging if requested
		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		client.Verbose = verbose

		fmt.Println("Connected to local database")
		fmt.Println("Blizzard API client initialized")

		// Initialize database service
		dbService := database.NewDatabaseService(db)

		// Parse flags
		batchSize, _ := cmd.Flags().GetInt("batch-size")
		maxPlayers, _ := cmd.Flags().GetInt("max-players")

		// Build options
		opts := pipeline.FetchProfilesOptions{
			Verbose:    verbose,
			BatchSize:  batchSize,
			MaxPlayers: maxPlayers,
		}

		// Fetch player profiles
		result, err := pipeline.FetchPlayerProfiles(dbService, client, opts)
		if err != nil {
			return err
		}

		fmt.Printf("\n[OK] Player profile fetching complete!\n")
		fmt.Printf("   Processed: %d players in %v\n", result.ProcessedCount, result.Duration)
		fmt.Printf("   Updated: %d player profiles\n", result.TotalProfiles)
		fmt.Printf("   Updated: %d equipment items\n", result.TotalEquipment)

		if result.ProcessedCount > 0 {
			rate := float64(result.ProcessedCount) / result.Duration.Minutes()
			fmt.Printf("   Rate: %.1f players/minute\n", rate)
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  Run 'ookstats generate api' to rebuild the website with new player profile data")

		return nil
	},
}

var fetchSeasonsCmd = &cobra.Command{
	Use:   "seasons",
	Short: "Fetch and sync season metadata",
	Long:  `Fetch season metadata from Blizzard API and populate the seasons and period_seasons tables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Season Metadata Sync ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Blizzard API client: %w", err)
		}

		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		client.Verbose = verbose

		fmt.Println("Connected to local database")
		fmt.Println("Blizzard API client initialized")

		dbService := database.NewDatabaseService(db)

		// Get regions to sync
		regionsCSV, _ := cmd.Flags().GetString("regions")
		var regions []string
		if strings.TrimSpace(regionsCSV) != "" {
			for _, r := range strings.Split(regionsCSV, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					regions = append(regions, r)
				}
			}
		} else {
			// Default to all regions
			regions = []string{"us", "eu", "kr", "tw"}
		}

		fmt.Printf("Syncing seasons for regions: %v\n\n", regions)

		totalSeasons := 0
		totalPeriods := 0

		for _, region := range regions {
			fmt.Printf("=== Region: %s ===\n", strings.ToUpper(region))

			// Fetch season index
			seasonIndex, err := client.FetchSeasonIndex(region)
			if err != nil {
				fmt.Printf("Error fetching season index for %s: %v\n", region, err)
				continue
			}

			fmt.Printf("Found %d seasons in %s\n", len(seasonIndex.Seasons), strings.ToUpper(region))

			// Process each season
			for _, seasonRef := range seasonIndex.Seasons {
				seasonID := seasonRef.ID
				fmt.Printf("\n--- Season %d ---\n", seasonID)

				// Fetch season details
				seasonDetail, err := client.FetchSeasonDetail(region, seasonID)
				if err != nil {
					fmt.Printf("Error fetching season %d details: %v\n", seasonID, err)
					continue
				}

				// Upsert season
				err = dbService.UpsertSeason(seasonDetail.ID, seasonDetail.SeasonName, seasonDetail.StartTimestamp)
				if err != nil {
					fmt.Printf("Error upserting season %d: %v\n", seasonID, err)
					continue
				}
				totalSeasons++

				fmt.Printf("Season: %s (ID: %d)\n", seasonDetail.SeasonName, seasonDetail.ID)
				fmt.Printf("Start: %d\n", seasonDetail.StartTimestamp)
				fmt.Printf("Periods: %d\n", len(seasonDetail.Periods))

				// Link periods to season
				if len(seasonDetail.Periods) > 0 {
					firstPeriod := seasonDetail.Periods[0].ID
					lastPeriod := seasonDetail.Periods[len(seasonDetail.Periods)-1].ID

					// Update period range
					err = dbService.UpdateSeasonPeriodRange(seasonDetail.ID, firstPeriod, lastPeriod)
					if err != nil {
						fmt.Printf("Error updating season period range: %v\n", err)
					}

					// Link each period
					for _, periodRef := range seasonDetail.Periods {
						err = dbService.LinkPeriodToSeason(periodRef.ID, seasonDetail.ID)
						if err != nil {
							fmt.Printf("Error linking period %d to season %d: %v\n", periodRef.ID, seasonDetail.ID, err)
						} else {
							totalPeriods++
						}
					}
					fmt.Printf("Linked %d periods to season %d\n", len(seasonDetail.Periods), seasonDetail.ID)
				}
			}

			// Note: We only need to sync from one region since season IDs and periods are global
			// Breaking after first successful region to avoid redundant work
			fmt.Printf("\n[OK] Synced seasons from %s (seasons are global across regions)\n", strings.ToUpper(region))
			break
		}

		fmt.Printf("\n=== Sync Complete ===\n")
		fmt.Printf("Total seasons synced: %d\n", totalSeasons)
		fmt.Printf("Total period mappings created: %d\n", totalPeriods)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.AddCommand(fetchCMCmd)
	fetchCmd.AddCommand(fetchProfilesCmd)
	fetchCmd.AddCommand(fetchSeasonsCmd)

	// CM fetching flags
	fetchCMCmd.Flags().Int("concurrency", 20, "Max concurrent API requests")
	fetchCMCmd.Flags().Int("api-timeout-seconds", 15, "HTTP client timeout in seconds")
	fetchCMCmd.Flags().String("regions", "", "Comma-separated regions to include (us,eu,kr,tw)")
	fetchCMCmd.Flags().String("realms", "", "Comma-separated realm slugs to include")
	fetchCMCmd.Flags().String("dungeons", "", "Comma-separated dungeon IDs or slugs to include")
	fetchCMCmd.Flags().String("periods", "", "Period specification: comma-separated list or ranges (e.g., '1020-1036' or '1020,1025,1030-1036'). Default: fetch all periods from API")

	// add player profile fetching flags
	fetchProfilesCmd.Flags().Int("batch-size", 20, "Number of players to process per batch")
	fetchProfilesCmd.Flags().Int("max-players", 0, "Maximum number of players to process (0 = no limit)")

	// season syncing flags
	fetchSeasonsCmd.Flags().String("regions", "us", "Comma-separated regions to query (only one needed since seasons are global)")
}
