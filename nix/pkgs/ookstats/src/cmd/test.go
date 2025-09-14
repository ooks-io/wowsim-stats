package cmd

import (
	"fmt"
	"ookstats/internal/database"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test database connection",
	Long:  `Test connection to Turso database and verify access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Testing database connection...")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Println("[OK] Connected to Turso database successfully")

		// test a simple query to verify the connection works
		rows, err := db.Query("SELECT 1 as test")
		if err != nil {
			return fmt.Errorf("failed to execute test query: %w", err)
		}
		defer rows.Close()

		fmt.Println("[OK] Database query test passed")
		fmt.Println("[OK] Connection test complete!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
