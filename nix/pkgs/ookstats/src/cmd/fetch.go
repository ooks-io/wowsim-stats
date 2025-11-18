package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
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
		log.Info("fetching challenge mode leaderboards")

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

		log.Info("connected to local database")
		log.Info("blizzard API client initialized")

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

		log.Info("successfully inserted data into local database",
			"runs", result.TotalRuns,
			"players", result.TotalPlayers)
		log.Info("database saved", "path", database.DBFilePath())

		return nil
	},
}

var fetchProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Fetch detailed player profiles",
	Long:  `Fetch detailed player profile data including equipment and character information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("player profile fetcher")

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

		log.Info("connected to local database")
		log.Info("blizzard API client initialized")

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

		log.Info("player profile fetching complete",
			"processed", result.ProcessedCount,
			"duration", result.Duration,
			"profiles", result.TotalProfiles,
			"equipment", result.TotalEquipment)

		if result.ProcessedCount > 0 {
			rate := float64(result.ProcessedCount) / result.Duration.Minutes()
			log.Info("fetch rate", "players_per_minute", rate)
		}

		log.Info("next step: run 'ookstats generate api' to rebuild the website")

		return nil
	},
}

var fetchSeasonsCmd = &cobra.Command{
	Use:   "seasons",
	Short: "Fetch and sync season metadata",
	Long:  `Fetch season metadata from Blizzard API and populate the seasons and period_seasons tables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("season metadata sync")

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

		log.Info("connected to local database")
		log.Info("blizzard API client initialized")

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

		log.Info("syncing seasons", "regions", regions)

		totalSeasons := 0
		totalPeriods := 0

		for _, region := range regions {
			log.Info("processing region", "region", strings.ToUpper(region))

			// Fetch season index
			seasonIndex, err := client.FetchSeasonIndex(region)
			if err != nil {
				log.Error("failed to fetch season index",
					"region", region,
					"error", err)
				continue
			}

			log.Info("found seasons",
				"count", len(seasonIndex.Seasons),
				"region", strings.ToUpper(region))

			// Process each season
			for _, seasonRef := range seasonIndex.Seasons {
				seasonID := seasonRef.ID
				log.Info("processing season", "season_id", seasonID)

				// Fetch season details
				seasonDetail, err := client.FetchSeasonDetail(region, seasonID)
				if err != nil {
					log.Error("failed to fetch season details",
						"season_id", seasonID,
						"error", err)
					continue
				}

				// Upsert season
				dbSeasonID, err := dbService.UpsertSeason(seasonDetail.ID, region, seasonDetail.SeasonName, seasonDetail.StartTimestamp)
				if err != nil {
					log.Error("failed to upsert season",
						"season_id", seasonID,
						"error", err)
					continue
				}
				totalSeasons++

				log.Info("season details",
					"name", seasonDetail.SeasonName,
					"id", seasonDetail.ID,
					"start", seasonDetail.StartTimestamp,
					"periods", len(seasonDetail.Periods))

				// Link periods to season
				if len(seasonDetail.Periods) > 0 {
					firstPeriod := seasonDetail.Periods[0].ID
					lastPeriod := seasonDetail.Periods[len(seasonDetail.Periods)-1].ID

					// Update period range
					err = dbService.UpdateSeasonPeriodRange(dbSeasonID, firstPeriod, lastPeriod)
					if err != nil {
						log.Error("failed to update season period range", "error", err)
					}

					// Link each period
					for _, periodRef := range seasonDetail.Periods {
						err = dbService.LinkPeriodToSeason(periodRef.ID, dbSeasonID)
						if err != nil {
							log.Error("failed to link period to season",
								"period_id", periodRef.ID,
								"season_id", seasonDetail.ID,
								"error", err)
						} else {
							totalPeriods++
						}
					}
					log.Info("linked periods to season",
						"periods", len(seasonDetail.Periods),
						"season_id", seasonDetail.ID)
				}
			}

			log.Info("synced seasons for region", "region", strings.ToUpper(region))
		}

		log.Info("sync complete",
			"total_seasons", totalSeasons,
			"total_periods", totalPeriods)

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
