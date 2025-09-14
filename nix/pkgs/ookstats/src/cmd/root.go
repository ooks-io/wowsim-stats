package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "ookstats/internal/database"
)

var rootCmd = &cobra.Command{
	Use:   "ookstats",
	Short: "WoW Stats Database Management Tool",
	Long: `A comprehensive tool for managing WoW Challenge Mode statistics database.

Supports fetching data from Blizzard API, processing player rankings, 
and managing data in Turso/libSQL database.`,
}

// execute adds all child commands to the root command and sets flags appropriately.
// this is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
    // enable completion commands (bash, zsh, fish, powershell)
    rootCmd.CompletionOptions.DisableDefaultCmd = false

    // global flag for local db path
    rootCmd.PersistentFlags().String("db-file", "", "Path to local SQLite database file (default: local.db). Also reads OOKSTATS_DB or ASTRO_DATABASE_FILE.")
    // global verbose flag
    rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging for debugging and benchmarking")

    // Set override before running any subcommand
    rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
        if v, _ := cmd.Flags().GetString("db-file"); v != "" {
            database.SetDBPath(v)
        }
    }
}
