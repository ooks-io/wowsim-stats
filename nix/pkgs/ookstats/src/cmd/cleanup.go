package cmd

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"ookstats/internal/database"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Database cleanup and maintenance operations",
	Long:  `Perform various database cleanup and maintenance operations to optimize space and performance.`,
}

var cleanupPlayerRankingsCmd = &cobra.Command{
	Use:   "player-rankings",
	Short: "Remove duplicate player_rankings entries",
	Long: `Removes duplicate player_rankings entries, keeping only the most recent ranking
for each (player_id, ranking_type, ranking_scope) combination.

This cleanup is safe to run and will significantly reduce database size if duplicates exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		vacuum, _ := cmd.Flags().GetBool("vacuum")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("db connect: %w", err)
		}
		defer db.Close()

		return runCleanupPlayerRankings(db, dryRun, vacuum)
	},
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.AddCommand(cleanupPlayerRankingsCmd)

	cleanupPlayerRankingsCmd.Flags().Bool("dry-run", false, "Show what would be deleted without actually deleting")
	cleanupPlayerRankingsCmd.Flags().Bool("vacuum", true, "Run VACUUM after cleanup to reclaim space")
}

func runCleanupPlayerRankings(db *sql.DB, dryRun bool, vacuum bool) error {
	log.Info("analyzing player_rankings table")

	// count total rows
	var totalRows int64
	if err := db.QueryRow("SELECT COUNT(*) FROM player_rankings").Scan(&totalRows); err != nil {
		return fmt.Errorf("count total rows: %w", err)
	}
	log.Info("current state", "total_rows", totalRows)

	// Count unique combinations
	var uniqueCombos int64
	if err := db.QueryRow(`
		SELECT COUNT(DISTINCT player_id || '|' || ranking_type || '|' || ranking_scope)
		FROM player_rankings
	`).Scan(&uniqueCombos); err != nil {
		return fmt.Errorf("count unique combos: %w", err)
	}

	duplicateRows := totalRows - uniqueCombos
	log.Info("duplicate analysis",
		"unique_combinations", uniqueCombos,
		"duplicate_rows", duplicateRows,
		"duplicate_percentage", fmt.Sprintf("%.1f%%", float64(duplicateRows)/float64(totalRows)*100))

	if duplicateRows == 0 {
		log.Info("no duplicates found - table is clean!")
		return nil
	}

	// count exact duplicates (same player, type, scope, AND rank)
	var exactDuplicates int64
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT player_id, ranking_type, ranking_scope, ranking, COUNT(*) as cnt
			FROM player_rankings
			GROUP BY player_id, ranking_type, ranking_scope, ranking
			HAVING cnt > 1
		)
	`).Scan(&exactDuplicates); err != nil {
		return fmt.Errorf("count exact duplicates: %w", err)
	}
	if exactDuplicates > 0 {
		log.Warn("exact duplicates found",
			"count", exactDuplicates,
			"note", "same player/type/scope/rank appearing multiple times")
	}

	if dryRun {
		log.Info("[DRY RUN] would delete", "rows", duplicateRows)
		log.Info("[DRY RUN] would keep", "rows", uniqueCombos)
		estimatedSizeMB := float64(duplicateRows) * 100 / 1024 / 1024 // rough estimate
		log.Info("[DRY RUN] estimated space savings", "mb", fmt.Sprintf("%.2f MB", estimatedSizeMB))
		return nil
	}

	log.Info("starting cleanup - this may take a few minutes...")
	startTime := time.Now()

	// delete duplicates, keeping only the row with the highest rowid (most recent)
	result, err := db.Exec(`
		DELETE FROM player_rankings
		WHERE rowid NOT IN (
			SELECT MAX(rowid)
			FROM player_rankings
			GROUP BY player_id, ranking_type, ranking_scope
		)
	`)
	if err != nil {
		return fmt.Errorf("delete duplicates: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()
	duration := time.Since(startTime)
	log.Info("cleanup complete",
		"rows_deleted", rowsDeleted,
		"duration", duration.Round(time.Millisecond))

	// verify cleanup
	var newTotal int64
	if err := db.QueryRow("SELECT COUNT(*) FROM player_rankings").Scan(&newTotal); err != nil {
		return fmt.Errorf("count after cleanup: %w", err)
	}
	log.Info("verification",
		"rows_before", totalRows,
		"rows_after", newTotal,
		"rows_removed", totalRows-newTotal)

	if vacuum {
		log.Info("running VACUUM to reclaim disk space...")
		vacuumStart := time.Now()
		if _, err := db.Exec("VACUUM"); err != nil {
			log.Warn("vacuum failed", "error", err)
		} else {
			log.Info("vacuum complete", "duration", time.Since(vacuumStart).Round(time.Millisecond))
		}
	} else {
		log.Info("skipping VACUUM (use --vacuum to reclaim disk space)")
	}

	log.Info("[SUCCESS] player_rankings cleanup completed successfully!")
	return nil
}
