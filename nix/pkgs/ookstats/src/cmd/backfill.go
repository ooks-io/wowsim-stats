package cmd

import (
	"fmt"
	"ookstats/internal/database"

	"github.com/spf13/cobra"
)

var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Backfill missing data",
	Long:  `Backfill and update historical data with new fields.`,
}

var backfillSeasonsCmd = &cobra.Command{
	Use:   "seasons",
	Short: "Assign season_id to all challenge runs",
	Long: `Assigns season_id to all existing challenge runs based on their completed_timestamp
and the season start/end timestamps for their region.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Backfill Season Assignments ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		dbService := database.NewDatabaseService(db)

		fmt.Println("\nAssigning seasons to challenge runs based on timestamps...")
		err = dbService.AssignRunsToSeasons()
		if err != nil {
			return fmt.Errorf("failed to assign seasons: %w", err)
		}

		// show stats
		fmt.Println("\n Season Assignment Statistics")
		rows, err := db.Query(`
			SELECT
				s.region,
				COALESCE(s.season_name, 'Season ' || s.season_number) as season_display,
				COUNT(cr.id) as run_count
			FROM seasons s
			LEFT JOIN challenge_runs cr ON cr.season_id = s.season_number AND cr.realm_id IN (
				SELECT id FROM realms WHERE region = s.region
			)
			GROUP BY s.id, s.region, s.season_name, s.season_number
			ORDER BY s.region, s.season_number
		`)
		if err != nil {
			return fmt.Errorf("failed to query stats: %w", err)
		}
		defer rows.Close()

		fmt.Printf("\n%-8s %-20s %10s\n", "Region", "Season", "Runs")
		fmt.Println("----------------------------------------")
		for rows.Next() {
			var region, seasonName string
			var runCount int
			if err := rows.Scan(&region, &seasonName, &runCount); err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}
			fmt.Printf("%-8s %-20s %10d\n", region, seasonName, runCount)
		}

		fmt.Printf("\n[OK] Season backfill complete!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backfillCmd)
	backfillCmd.AddCommand(backfillSeasonsCmd)
}
