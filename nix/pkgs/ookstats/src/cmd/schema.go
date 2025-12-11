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

		fmt.Printf("\n[OK] Database schema initialization complete!\n")
		fmt.Printf("Database ready at: %s\n", database.DBFilePath())

		return nil
	},
}

var schemaMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database schema to add season_id column",
	Long:  `Adds the season_id column to challenge_runs table if it doesn't exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Database Schema Migration ===")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		// check if season_id column already exists
		var columnExists int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM pragma_table_info('challenge_runs')
			WHERE name = 'season_id'
		`).Scan(&columnExists)
		if err != nil {
			return fmt.Errorf("failed to check column existence: %w", err)
		}

		if columnExists > 0 {
			fmt.Println("[SKIP] season_id column already exists")
			return nil
		}

		fmt.Println("Adding season_id column to challenge_runs table...")
		_, err = db.Exec("ALTER TABLE challenge_runs ADD COLUMN season_id INTEGER")
		if err != nil {
			return fmt.Errorf("failed to add season_id column: %w", err)
		}

		fmt.Println("Creating index on season_id...")
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_challenge_runs_season ON challenge_runs(season_id, dungeon_id, completed_timestamp)")
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}

		fmt.Printf("\n[OK] Schema migration complete!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaInitCmd)
	schemaCmd.AddCommand(schemaMigrateCmd)
}
