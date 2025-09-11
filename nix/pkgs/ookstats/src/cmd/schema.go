package cmd

import (
	"fmt"
	"ookstats/internal/database"

	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Database schema management",
	Long:  `Manage database schema creation and initialization.`,
}

var schemaInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize database schema",
	Long:  `Create all required tables and indexes for the ookstats database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Database Schema Initialization ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		// Initialize complete schema
		err = database.EnsureCompleteSchema(db)
		if err != nil {
			return fmt.Errorf("failed to initialize schema: %w", err)
		}

		fmt.Printf("\n✅ Database schema initialization complete!\n")
		fmt.Printf("Database ready at: %s\n", database.DBFilePath())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaInitCmd)
}