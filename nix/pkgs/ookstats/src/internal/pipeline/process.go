package pipeline

// season assignment has been migrated to use cr.season_id directly instead of period_seasons lookups.
// all queries now use timestamp-based season assignment from the challenge_runs table.

import (
	"database/sql"
	"fmt"

	"github.com/charmbracelet/log"
)

// ProcessPlayersOptions contains options for player processing
type ProcessPlayersOptions struct {
	Verbose bool
}

// ProcessPlayers processes player aggregations and rankings
func ProcessPlayers(db *sql.DB, opts ProcessPlayersOptions) (profilesCreated int, qualifiedPlayers int, err error) {
	log.Info("player aggregation")

	// check if we have data
	var runCount, playerCount int
	db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)
	db.QueryRow("SELECT COUNT(*) FROM players").Scan(&playerCount)

	log.Info("found data in database", "runs", runCount, "players", playerCount)

	if runCount == 0 {
		return 0, 0, fmt.Errorf("no runs found in database - run 'fetch cm' first")
	}

	// begin transaction for all player operations
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// step 0: ensure seasons are properly configured
	log.Info("checking season configuration")
	var seasonCount int
	tx.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&seasonCount)
	if seasonCount == 0 {
		log.Warn("no seasons found in database - proceeding with legacy all-time processing")
	} else {
		log.Info("found seasons configured", "count", seasonCount)
	}

	// step 1: create player aggregations (season-aware if seasons exist)
	log.Info("creating player aggregations")
	profilesCreated, err = createPlayerAggregations(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create player aggregations: %w", err)
	}

	// step 2: compute player rankings (global, regional, realm) per season
	log.Info("computing player rankings")
	qualifiedPlayers, err = computePlayerRankings(tx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compute player rankings: %w", err)
	}

	// step 3: compute class-specific rankings per season
	log.Info("computing class-specific rankings")
	if err = computePlayerClassRankings(tx); err != nil {
		return 0, 0, fmt.Errorf("failed to compute class rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit player aggregations: %w", err)
	}

	// optimize database
	log.Info("optimizing database")
	if _, err := db.Exec("VACUUM"); err != nil {
		log.Warn("database optimization failed", "error", err)
	}

	log.Info("player aggregation complete",
		"profiles", profilesCreated,
		"qualified_players", qualifiedPlayers)

	return profilesCreated, qualifiedPlayers, nil
}

// ProcessRunRankingsOptions contains options for run ranking processing
type ProcessRunRankingsOptions struct {
	Verbose bool
}

// ProcessRunRankings computes global, regional, and realm rankings for all runs
func ProcessRunRankings(db *sql.DB, opts ProcessRunRankingsOptions) error {
	log.Info("run ranking processor")

	// check if we have data
	var runCount int
	db.QueryRow("SELECT COUNT(*) FROM challenge_runs").Scan(&runCount)

	if runCount == 0 {
		return fmt.Errorf("no runs found in database - run 'fetch cm' first")
	}

	log.Info("found runs in database", "runs", runCount)

	// begin transaction for all ranking operations
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// step 1: compute global run rankings
	log.Info("computing global run rankings")
	if err := computeGlobalRankings(tx); err != nil {
		return fmt.Errorf("failed to compute global rankings: %w", err)
	}

	// step 2: compute regional run rankings
	log.Info("computing regional run rankings")
	if err := computeRegionalRankings(tx); err != nil {
		return fmt.Errorf("failed to compute regional rankings: %w", err)
	}

	// step 3: compute realm run rankings (pool-based for connected realms)
	log.Info("computing realm run rankings (pool-based)")
	if err := computeRealmRankings(tx); err != nil {
		return fmt.Errorf("failed to compute realm rankings: %w", err)
	}

	// commit all changes
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit run rankings: %w", err)
	}

	// optimize database
	log.Info("optimizing database")
	if _, err := db.Exec("VACUUM"); err != nil {
		log.Warn("database optimization failed", "error", err)
	}

	log.Info("run ranking computation complete")
	return nil
}
