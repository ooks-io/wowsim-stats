package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"ookstats/internal/database"
)

var rootCmd = &cobra.Command{
	Use:   "ookstats",
	Short: "WoW Stats Database Management Tool",
	Long: `A comprehensive tool for managing WoW Challenge Mode statistics database.

Supports fetching data from Blizzard API, processing player rankings,
and managing data in local SQLite database.`,
}

// logger is the global logger instance
var logger *log.Logger

// GetLogger returns the global logger instance
func GetLogger() *log.Logger {
	if logger == nil {
		// fallback logger if not initialized
		logger = log.Default()
	}
	return logger
}

// initLogger initializes the global logger with appropriate settings
func initLogger(verbose bool) {
	level := log.InfoLevel
	if verbose {
		level = log.DebugLevel
	}

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           level,
	})

	// Set as default logger for the log package
	log.SetDefault(logger)
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
		// Initialize logger with verbose setting
		verbose, _ := cmd.Flags().GetBool("verbose")
		initLogger(verbose)

		// Set database path override
		if v, _ := cmd.Flags().GetString("db-file"); v != "" {
			database.SetDBPath(v)
		}
	}
}
