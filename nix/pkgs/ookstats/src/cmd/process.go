package cmd

import (
	"fmt"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"ookstats/internal/pipeline"

	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process and compute data",
	Long:  `Process raw data and compute player rankings, aggregations, etc.`,
}

var processAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Process all data (players + rankings)",
	Long:  `Process all data: player aggregations, player rankings, and run rankings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Complete Data Processing ===")

		// step 1: process players (aggregations + rankings)
		fmt.Println("\n=== Step 1: Processing Players ===")
		if err := processPlayersCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("player processing failed: %w", err)
		}

		// step 2: process run rankings
		fmt.Println("\n=== Step 2: Processing Run Rankings ===")
		if err := processRankingsCmd.RunE(cmd, args); err != nil {
			return fmt.Errorf("run ranking processing failed: %w", err)
		}

		fmt.Printf("\n[OK] Complete data processing finished!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Run 'ookstats generate api --out web/public' to update JSON files\n")
		fmt.Printf("  2. Build and deploy the website\n")

		return nil
	},
}

var processRankingsCmd = &cobra.Command{
	Use:   "rankings",
	Short: "Compute run rankings",
	Long:  `Compute global, regional, and realm rankings for all challenge mode runs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		opts := pipeline.ProcessRunRankingsOptions{
			Verbose: verbose,
		}

		if err := pipeline.ProcessRunRankings(db, opts); err != nil {
			return err
		}

		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Run 'ookstats generate api --out web/public' to update JSON files\n")
		fmt.Printf("  2. Build and deploy the website\n")

		return nil
	},
}

var processPlayersCmd = &cobra.Command{
	Use:   "players",
	Short: "Aggregate player statistics",
	Long:  `Aggregate player data including best runs, combined times, and rankings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		verbose, _ := cmd.InheritedFlags().GetBool("verbose")
		opts := pipeline.ProcessPlayersOptions{
			Verbose: verbose,
		}

		profilesCreated, qualifiedPlayers, err := pipeline.ProcessPlayers(db, opts)
		if err != nil {
			return err
		}

		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Run 'ookstats process profiles' to fetch player details\n")
		fmt.Printf("  2. Test your Astro DB integration\n")

		// Suppress unused variable warnings
		_ = profilesCreated
		_ = qualifiedPlayers

		return nil
	},
}

var processProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Fetch player profiles from Blizzard API",
	Long:  `Fetch detailed player information, equipment, and avatars for players with complete dungeon coverage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Player Profile Fetcher ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		// check if we have eligible players
		var eligibleCount int
		err = db.QueryRow("SELECT COUNT(*) FROM player_profiles WHERE has_complete_coverage = 1").Scan(&eligibleCount)
		if err != nil {
			return fmt.Errorf("failed to count eligible players: %w", err)
		}

		fmt.Printf("Found %d eligible players with complete coverage\n", eligibleCount)

		if eligibleCount == 0 {
			return fmt.Errorf("no eligible players found - run 'process players' first")
		}

		fmt.Printf("Player profile fetching not yet implemented\n")
		fmt.Printf("This command will fetch player details from Blizzard API for %d players\n", eligibleCount)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Implement Blizzard API client integration\n")
		fmt.Printf("  2. Add concurrent fetching with rate limiting\n")
		fmt.Printf("  3. Store player_details and player_equipment data\n")

		return nil
	},
}

var processAccountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Group characters into accounts via account-wide achievement tuples",
	Long: `Backfills account fingerprints for any character that has a per-character
fingerprint but no account fingerprint yet (one Blizzard API call per
candidate), then runs the in-memory connected-components grouping over every
stored fingerprint and rebuilds the accounts table + players.account_id.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("blizzard client: %w", err)
		}

		maxShare, _ := cmd.Flags().GetFloat64("max-tuple-share")
		minMatches, _ := cmd.Flags().GetInt("min-edge-matches")
		workers, _ := cmd.Flags().GetInt("backfill-workers")
		maxBackfill, _ := cmd.Flags().GetInt("max-backfill")
		skipBackfill, _ := cmd.Flags().GetBool("skip-backfill")
		manualLinks, _ := cmd.Flags().GetString("manual-links")

		opts := pipeline.AccountGroupingOptions{
			MaxTupleShare:   maxShare,
			MinEdgeMatches:  minMatches,
			BackfillWorkers: workers,
			MaxBackfill:     maxBackfill,
			ManualLinksPath: manualLinks,
		}

		var c *blizzard.Client
		if !skipBackfill {
			c = client
		}
		res, err := pipeline.ProcessAccounts(database.NewDatabaseService(db), c, opts)
		if err != nil {
			return err
		}
		fmt.Printf("\n=== Account grouping complete ===\n")
		fmt.Printf("Backfilled: %d (errors: %d)\n", res.Backfilled, res.BackfillErrors)
		fmt.Printf("Characters: %d\n", res.Characters)
		fmt.Printf("Accounts:   %d\n", res.Accounts)
		fmt.Printf("Multi-character accounts: %d\n", res.MultiCharAccounts)
		fmt.Printf("Duration:   %s\n", res.Duration.Round(1e9))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(processCmd)
	processCmd.AddCommand(processAllCmd)
	processCmd.AddCommand(processRankingsCmd)
	processCmd.AddCommand(processPlayersCmd)
	processCmd.AddCommand(processProfilesCmd)

	processCmd.AddCommand(processAccountsCmd)
	processAccountsCmd.Flags().Float64("max-tuple-share", 0.05, "Drop (id, ts) tuples seen in more than this fraction of population")
	processAccountsCmd.Flags().Int("min-edge-matches", 5, "Require N matching tuples between two characters before unioning them")
	processAccountsCmd.Flags().Int("backfill-workers", 20, "Concurrent achievement fetches during backfill")
	processAccountsCmd.Flags().Int("max-backfill", 0, "Cap backfill fetches per run (0 = unlimited)")
	processAccountsCmd.Flags().Bool("skip-backfill", false, "Skip the backfill pass; only re-run grouping over existing data")
	processAccountsCmd.Flags().String("manual-links", "account-manual-links.json", "Path to curator-supplied manual account links JSON (missing file is OK)")
}
