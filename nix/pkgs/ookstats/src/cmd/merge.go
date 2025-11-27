package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"ookstats/internal/database"
	"ookstats/internal/merge"
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge duplicate player records",
	Long:  `Merge duplicate player records using a configuration file that maps old identities to new ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			return fmt.Errorf("--config flag is required")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		log.Info("player merge tool", "config", configPath, "dry_run", dryRun)

		// Load config
		config, err := merge.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		log.Info("loaded merge configuration", "entries", len(config.Merges))

		// Connect to database
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		dbService := database.NewDatabaseService(db)

		totalMerged := 0
		totalRuns := 0

		for i, entry := range config.Merges {
			log.Info("processing merge entry", "index", i+1, "total", len(config.Merges))

			// Look up "from" player
			fromID, err := dbService.GetPlayerByNameRealmRegion(
				entry.From.Name,
				entry.From.Realm,
				entry.From.Region,
			)
			if err != nil {
				log.Error("failed to find source player", "error", err, "entry", i+1)
				continue
			}

			// Look up "to" player
			toID, err := dbService.GetPlayerByNameRealmRegion(
				entry.To.Name,
				entry.To.Realm,
				entry.To.Region,
			)
			if err != nil {
				log.Error("failed to find target player", "error", err, "entry", i+1)
				continue
			}

			if fromID == toID {
				log.Warn("source and target are the same player, skipping", "player_id", fromID)
				continue
			}

			log.Info("merge plan",
				"from_player", fmt.Sprintf("%s-%s (%s)", entry.From.Name, entry.From.Realm, entry.From.Region),
				"from_id", fromID,
				"to_player", fmt.Sprintf("%s-%s (%s)", entry.To.Name, entry.To.Realm, entry.To.Region),
				"to_id", toID)

			if dryRun {
				log.Info("dry-run mode: skipping actual merge")
				continue
			}

			// Migrate runs
			runsMigrated, err := dbService.MigratePlayerRuns(fromID, toID)
			if err != nil {
				log.Error("failed to migrate runs", "error", err, "from", fromID, "to", toID)
				continue
			}

			// Invalidate target player's profile for rebuild
			if err := dbService.InvalidatePlayerProfile(toID); err != nil {
				log.Warn("failed to invalidate profile", "player_id", toID, "error", err)
			}

			// Mark source player as invalid
			if err := dbService.UpdatePlayerStatus(fromID, false, 0, nil); err != nil {
				log.Error("failed to mark source player invalid", "player_id", fromID, "error", err)
				continue
			}

			log.Info("merge complete",
				"from_id", fromID,
				"to_id", toID,
				"runs_migrated", runsMigrated)

			totalMerged++
			totalRuns += runsMigrated
		}

		log.Info("merge operation complete",
			"total_entries", len(config.Merges),
			"successful_merges", totalMerged,
			"total_runs_migrated", totalRuns)

		if !dryRun && totalMerged > 0 {
			log.Info("next step: run 'ookstats build profiles' to rebuild player rankings")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().String("config", "", "Path to merge configuration JSON file (required)")
	mergeCmd.Flags().Bool("dry-run", false, "Show what would be merged without executing")
	mergeCmd.MarkFlagRequired("config")
}
