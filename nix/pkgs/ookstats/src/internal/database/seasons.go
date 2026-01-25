package database

import (
	"database/sql"
	"fmt"
)

// UpsertSeason inserts or updates a season record and returns the auto-increment ID
func (ds *DatabaseService) UpsertSeason(seasonID int, region string, seasonName string, startTimestamp int64) (int, error) {
	query := `
		INSERT INTO seasons (season_number, region, season_name, start_timestamp)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(season_number, region) DO UPDATE SET
			season_name = excluded.season_name,
			start_timestamp = excluded.start_timestamp
		RETURNING id
	`
	var id int
	err := ds.db.QueryRow(query, seasonID, region, seasonName, startTimestamp).Scan(&id)
	return id, err
}

// UpdateSeasonPeriodRange updates the first_period_id and last_period_id for a season
func (ds *DatabaseService) UpdateSeasonPeriodRange(seasonID, firstPeriodID, lastPeriodID int) error {
	query := `
		UPDATE seasons
		SET first_period_id = ?, last_period_id = ?
		WHERE id = ?
	`
	_, err := ds.db.Exec(query, firstPeriodID, lastPeriodID, seasonID)
	return err
}

// UpdateSeasonEndTimestamp updates the end_timestamp for a season
func (ds *DatabaseService) UpdateSeasonEndTimestamp(seasonID int, endTimestamp int64) error {
	query := `UPDATE seasons SET end_timestamp = ? WHERE id = ?`
	_, err := ds.db.Exec(query, endTimestamp, seasonID)
	return err
}

// LinkPeriodToSeason creates a mapping between a period and season
func (ds *DatabaseService) LinkPeriodToSeason(periodID, seasonID int) error {
	query := `INSERT OR IGNORE INTO period_seasons (period_id, season_id) VALUES (?, ?)`
	_, err := ds.db.Exec(query, periodID, seasonID)
	return err
}

// GetSeasonByID retrieves a season by its ID
func (ds *DatabaseService) GetSeasonByID(seasonID int) (*Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		WHERE id = ?
	`
	var season Season
	var endTimestamp sql.NullInt64
	var firstPeriod, lastPeriod sql.NullInt64
	err := ds.db.QueryRow(query, seasonID).Scan(
		&season.ID,
		&season.SeasonNumber,
		&season.StartTimestamp,
		&endTimestamp,
		&season.SeasonName,
		&firstPeriod,
		&lastPeriod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if endTimestamp.Valid {
		season.EndTimestamp = &endTimestamp.Int64
	}
	if firstPeriod.Valid {
		fp := int(firstPeriod.Int64)
		season.FirstPeriodID = &fp
	}
	if lastPeriod.Valid {
		lp := int(lastPeriod.Int64)
		season.LastPeriodID = &lp
	}
	return &season, nil
}

// GetAllSeasons retrieves all seasons ordered by start timestamp
func (ds *DatabaseService) GetAllSeasons() ([]Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		ORDER BY start_timestamp DESC
	`
	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []Season
	for rows.Next() {
		var season Season
		var endTimestamp sql.NullInt64
		var firstPeriod, lastPeriod sql.NullInt64
		err := rows.Scan(
			&season.ID,
			&season.SeasonNumber,
			&season.StartTimestamp,
			&endTimestamp,
			&season.SeasonName,
			&firstPeriod,
			&lastPeriod,
		)
		if err != nil {
			return nil, err
		}
		if endTimestamp.Valid {
			season.EndTimestamp = &endTimestamp.Int64
		}
		if firstPeriod.Valid {
			fp := int(firstPeriod.Int64)
			season.FirstPeriodID = &fp
		}
		if lastPeriod.Valid {
			lp := int(lastPeriod.Int64)
			season.LastPeriodID = &lp
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

// GetCurrentSeason retrieves the current active season (most recent without end timestamp)
func (ds *DatabaseService) GetCurrentSeason() (*Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		WHERE end_timestamp IS NULL
		ORDER BY start_timestamp DESC
		LIMIT 1
	`
	var season Season
	var endTimestamp sql.NullInt64
	var firstPeriod, lastPeriod sql.NullInt64
	err := ds.db.QueryRow(query).Scan(
		&season.ID,
		&season.SeasonNumber,
		&season.StartTimestamp,
		&endTimestamp,
		&season.SeasonName,
		&firstPeriod,
		&lastPeriod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if endTimestamp.Valid {
		season.EndTimestamp = &endTimestamp.Int64
	}
	if firstPeriod.Valid {
		fp := int(firstPeriod.Int64)
		season.FirstPeriodID = &fp
	}
	if lastPeriod.Valid {
		lp := int(lastPeriod.Int64)
		season.LastPeriodID = &lp
	}
	return &season, nil
}

// GetSeasonForPeriod retrieves the season ID for a given period
func (ds *DatabaseService) GetSeasonForPeriod(periodID int) (int, error) {
	query := `SELECT season_id FROM period_seasons WHERE period_id = ? LIMIT 1`
	var seasonID int
	err := ds.db.QueryRow(query, periodID).Scan(&seasonID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return seasonID, err
}

// GetPeriodsForSeason retrieves all period IDs for a given season
func (ds *DatabaseService) GetPeriodsForSeason(seasonID int) ([]int, error) {
	query := `SELECT period_id FROM period_seasons WHERE season_id = ? ORDER BY period_id`
	rows, err := ds.db.Query(query, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []int
	for rows.Next() {
		var periodID int
		if err := rows.Scan(&periodID); err != nil {
			return nil, err
		}
		periods = append(periods, periodID)
	}
	return periods, rows.Err()
}

// GetPeriodsForRegion retrieves all period IDs for all seasons in a given region
func (ds *DatabaseService) GetPeriodsForRegion(region string) ([]int, error) {
	query := `
		SELECT DISTINCT ps.period_id
		FROM period_seasons ps
		JOIN seasons s ON ps.season_id = s.id
		WHERE s.region = ?
		ORDER BY ps.period_id DESC
	`
	rows, err := ds.db.Query(query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []int
	for rows.Next() {
		var periodID int
		if err := rows.Scan(&periodID); err != nil {
			return nil, err
		}
		periods = append(periods, periodID)
	}
	return periods, rows.Err()
}

// GetLatestPeriodsPerRegion retrieves only the latest 2 periods from the current season for a given region
func (ds *DatabaseService) GetLatestPeriodsPerRegion(region string) ([]int, error) {
	query := `
		SELECT DISTINCT ps.period_id
		FROM period_seasons ps
		JOIN seasons s ON ps.season_id = s.id
		WHERE s.region = ?
		  AND s.end_timestamp IS NULL
		ORDER BY ps.period_id DESC
		LIMIT 2
	`
	rows, err := ds.db.Query(query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []int
	for rows.Next() {
		var periodID int
		if err := rows.Scan(&periodID); err != nil {
			return nil, err
		}
		periods = append(periods, periodID)
	}
	return periods, rows.Err()
}

// determineSeasonForRunTx determines which season a run belongs to based on timestamp and region
func (ds *DatabaseService) determineSeasonForRunTx(tx *sql.Tx, region string, completedTimestamp int64) (int, error) {
	query := `
		SELECT season_number
		FROM seasons
		WHERE region = ?
		  AND start_timestamp <= ?
		  AND (end_timestamp IS NULL OR end_timestamp > ?)
		ORDER BY start_timestamp DESC
		LIMIT 1
	`
	var seasonNumber int
	err := tx.QueryRow(query, region, completedTimestamp, completedTimestamp).Scan(&seasonNumber)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return seasonNumber, err
}

// AssignRunsToSeasons assigns season_id to all challenge_runs based on completed_timestamp
func (ds *DatabaseService) AssignRunsToSeasons() error {
	regions := []string{"us", "eu", "kr", "tw"}

	for _, region := range regions {
		fmt.Printf("Assigning runs to seasons for region: %s\n", region)

		query := `
			UPDATE challenge_runs
			SET season_id = (
				SELECT s.season_number
				FROM seasons s
				JOIN realms r ON r.region = s.region
				WHERE r.id = challenge_runs.realm_id
				  AND s.start_timestamp <= challenge_runs.completed_timestamp
				  AND (s.end_timestamp IS NULL OR s.end_timestamp > challenge_runs.completed_timestamp)
				ORDER BY s.start_timestamp DESC
				LIMIT 1
			)
			WHERE realm_id IN (
				SELECT id FROM realms WHERE region = ?
			)
		`

		result, err := ds.db.Exec(query, region)
		if err != nil {
			return fmt.Errorf("failed to assign seasons for region %s: %w", region, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected for region %s: %w", region, err)
		}

		fmt.Printf("  Updated %d runs for region %s\n", rowsAffected, region)
	}

	var orphanedRuns int
	err := ds.db.QueryRow("SELECT COUNT(*) FROM challenge_runs WHERE season_id IS NULL").Scan(&orphanedRuns)
	if err != nil {
		return fmt.Errorf("failed to count orphaned runs: %w", err)
	}

	if orphanedRuns > 0 {
		fmt.Printf("Warning: %d runs could not be assigned to any season\n", orphanedRuns)
	}

	return nil
}
