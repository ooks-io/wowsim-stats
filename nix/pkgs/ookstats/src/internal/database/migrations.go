package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// migrateRealmsCompositeSlug upgrades the realms table from slug-unique to (region,slug)-unique if necessary.
func migrateRealmsCompositeSlug(db *sql.DB) error {
	// Inspect table SQL
	var createSQL string
	_ = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='realms'`).Scan(&createSQL)
	// If table already created without UNIQUE on slug, nothing to do
	if !strings.Contains(strings.ToLower(createSQL), "slug text unique") {
		return nil
	}

	fmt.Printf("[MIGRATE] Upgrading realms table to composite (region,slug) uniqueness...\n")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new table without UNIQUE on slug
	if _, err := tx.Exec(`
        CREATE TABLE IF NOT EXISTS realms_new (
            id INTEGER PRIMARY KEY,
            slug TEXT,
            name TEXT,
            region TEXT,
            connected_realm_id INTEGER UNIQUE,
            parent_realm_slug TEXT
        )
    `); err != nil {
		return fmt.Errorf("create realms_new: %w", err)
	}

	// Copy data
	if _, err := tx.Exec(`INSERT INTO realms_new (id, slug, name, region, connected_realm_id, parent_realm_slug)
                          SELECT id, slug, name, region, connected_realm_id,
                                 CASE WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='realms' AND sql LIKE '%parent_realm_slug%')
                                      THEN parent_realm_slug ELSE NULL END
                          FROM realms`); err != nil {
		return fmt.Errorf("copy realms: %w", err)
	}

	// Rename old and new
	if _, err := tx.Exec(`ALTER TABLE realms RENAME TO realms_old`); err != nil {
		return fmt.Errorf("rename old: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE realms_new RENAME TO realms`); err != nil {
		return fmt.Errorf("rename new: %w", err)
	}

	// Drop old
	if _, err := tx.Exec(`DROP TABLE realms_old`); err != nil {
		return fmt.Errorf("drop old: %w", err)
	}

	// Ensure composite unique index
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_realms_region_slug ON realms(region, slug)`); err != nil {
		return fmt.Errorf("create composite unique index: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("[OK] Realms table migrated\n")
	return nil
}

// migrateSeasonsAddRegion adds region column to seasons table if it doesn't exist
func migrateSeasonsAddRegion(db *sql.DB) error {
	// Check if region column already exists
	var createSQL string
	_ = db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='seasons'`).Scan(&createSQL)

	// If table already has region column, nothing to do
	if strings.Contains(strings.ToLower(createSQL), "region text") {
		return nil
	}

	fmt.Printf("[MIGRATE] Adding region column to seasons table...\n")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create new table with region column
	if _, err := tx.Exec(`
        CREATE TABLE IF NOT EXISTS seasons_new (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            season_number INTEGER NOT NULL,
            region TEXT NOT NULL,
            start_timestamp INTEGER,
            end_timestamp INTEGER,
            season_name TEXT,
            first_period_id INTEGER,
            last_period_id INTEGER,
            UNIQUE(season_number, region)
        )
    `); err != nil {
		return fmt.Errorf("create seasons_new: %w", err)
	}

	// Copy existing data, defaulting region to 'us' for backward compatibility
	if _, err := tx.Exec(`
        INSERT INTO seasons_new (id, season_number, region, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id)
        SELECT id, season_number, 'us', start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
        FROM seasons
    `); err != nil {
		return fmt.Errorf("copy seasons: %w", err)
	}

	// Rename old and new
	if _, err := tx.Exec(`ALTER TABLE seasons RENAME TO seasons_old`); err != nil {
		return fmt.Errorf("rename old: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE seasons_new RENAME TO seasons`); err != nil {
		return fmt.Errorf("rename new: %w", err)
	}

	// Drop old
	if _, err := tx.Exec(`DROP TABLE seasons_old`); err != nil {
		return fmt.Errorf("drop old: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("[OK] Seasons table migrated - added region column\n")
	return nil
}

// migratePlayersIdentityColumns ensures identity + status metadata columns exist on players.
func migratePlayersIdentityColumns(db *sql.DB) error {
	hasBlizzardID, err := columnExists(db, "players", "blizzard_character_id")
	if err != nil {
		return err
	}
	if !hasBlizzardID {
		if _, err := db.Exec(`ALTER TABLE players ADD COLUMN blizzard_character_id INTEGER`); err != nil {
			return fmt.Errorf("add blizzard_character_id: %w", err)
		}
		if _, err := db.Exec(`UPDATE players SET blizzard_character_id = id WHERE blizzard_character_id IS NULL`); err != nil {
			return fmt.Errorf("backfill blizzard_character_id: %w", err)
		}
	}

	hasIsValid, err := columnExists(db, "players", "is_valid")
	if err != nil {
		return err
	}
	if !hasIsValid {
		if _, err := db.Exec(`ALTER TABLE players ADD COLUMN is_valid INTEGER DEFAULT 1`); err != nil {
			return fmt.Errorf("add is_valid: %w", err)
		}
		if _, err := db.Exec(`UPDATE players SET is_valid = 1 WHERE is_valid IS NULL`); err != nil {
			return fmt.Errorf("backfill is_valid: %w", err)
		}
	}

	hasStatusChecked, err := columnExists(db, "players", "status_checked_at")
	if err != nil {
		return err
	}
	if !hasStatusChecked {
		if _, err := db.Exec(`ALTER TABLE players ADD COLUMN status_checked_at INTEGER`); err != nil {
			return fmt.Errorf("add status_checked_at: %w", err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notnull    int
			dfltValue  any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &primaryKey); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

// migratePlayerRankingsPrimaryKey adds PRIMARY KEY constraint to player_rankings table if missing
func migratePlayerRankingsPrimaryKey(db *sql.DB) error {
	// Check if table already has PRIMARY KEY by inspecting schema
	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='player_rankings'`).Scan(&createSQL)
	if err != nil {
		// Table doesn't exist yet, skip migration
		return nil
	}

	// If PRIMARY KEY already exists, skip migration
	if strings.Contains(strings.ToUpper(createSQL), "PRIMARY KEY") {
		return nil
	}

	fmt.Printf("Migrating player_rankings table to add PRIMARY KEY constraint...\n")

	// Begin transaction for safe migration
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create new table with PRIMARY KEY
	if _, err := tx.Exec(`
		CREATE TABLE player_rankings_new (
			player_id INTEGER NOT NULL,
			ranking_type TEXT NOT NULL,
			ranking_scope TEXT NOT NULL,
			ranking INTEGER,
			combined_best_time INTEGER,
			last_updated INTEGER,
			PRIMARY KEY (player_id, ranking_type, ranking_scope)
		)
	`); err != nil {
		return fmt.Errorf("create new table: %w", err)
	}

	// Copy data, keeping only the most recent entry per (player_id, ranking_type, ranking_scope)
	if _, err := tx.Exec(`
		INSERT INTO player_rankings_new
		SELECT
			player_id,
			ranking_type,
			ranking_scope,
			ranking,
			combined_best_time,
			last_updated
		FROM player_rankings
		WHERE rowid IN (
			SELECT MAX(rowid)
			FROM player_rankings
			GROUP BY player_id, ranking_type, ranking_scope
		)
	`); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	// Drop old table
	if _, err := tx.Exec(`DROP TABLE player_rankings`); err != nil {
		return fmt.Errorf("drop old table: %w", err)
	}

	// Rename new table
	if _, err := tx.Exec(`ALTER TABLE player_rankings_new RENAME TO player_rankings`); err != nil {
		return fmt.Errorf("rename new table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("[OK] player_rankings table migrated - added PRIMARY KEY and removed duplicates\n")
	return nil
}
