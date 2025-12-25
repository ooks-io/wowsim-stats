package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/tursodatabase/go-libsql"
)

// connect creates a local libSQL database connection
var dbPathOverride string

// SetDBPath allows callers (CLI) to override the local SQLite filename
func SetDBPath(path string) {
	dbPathOverride = path
}

// DBConnString returns the libsql connection string for the local SQLite file
func DBConnString() string {
	// Priority: explicit override -> env vars -> default
	if dbPathOverride != "" {
		if strings.HasPrefix(dbPathOverride, "file:") {
			return ensureDSNParams(dbPathOverride)
		}
		return ensureDSNParams("file:" + dbPathOverride)
	}
	if v := os.Getenv("OOKSTATS_DB"); v != "" {
		if strings.HasPrefix(v, "file:") {
			return ensureDSNParams(v)
		}
		return ensureDSNParams("file:" + v)
	}
	if v := os.Getenv("ASTRO_DATABASE_FILE"); v != "" {
		if strings.HasPrefix(v, "file:") {
			return ensureDSNParams(v)
		}
		return ensureDSNParams("file:" + v)
	}
	return ensureDSNParams("file:local.db")
}

func ensureDSNParams(base string) string {
	if !strings.HasPrefix(base, "file:") {
		return base
	}
	if strings.Contains(base, "?") {
		return base
	}
	return base + "?" +
		"_pragma=journal_mode(WAL)&" +
		"_pragma=synchronous=NORMAL&" +
		"_pragma=busy_timeout=5000&" +
		"_pragma=cache_size=-64000"
}

// DBFilePath returns the plain filesystem path for the local DB (without file: prefix)
func DBFilePath() string {
	conn := DBConnString()
	if strings.HasPrefix(conn, "file:") {
		return strings.TrimPrefix(conn, "file:")
	}
	return conn
}

func Connect() (*sql.DB, error) {
	dsn := DBConnString()
	fmt.Printf("Using local SQLite database: %s\n", dsn)
	fmt.Printf("Opening database connection...\n")

	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	fmt.Printf("Testing database connection...\n")
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// configure database for optimal performance
	if err := configureDatabaseSettings(db, dsn); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	fmt.Printf("[OK] Local SQLite database connected\n")
	return db, nil
}

func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintf(os.Stderr, "Error: %s environment variable is required\n", key)
		os.Exit(1)
	}
	return value
}

// ExecuteSQL executes SQL with error handling
func ExecuteSQL(db *sql.DB, query string, args ...any) error {
	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("SQL execution failed: %w\nQuery: %s", err, query)
	}
	return nil
}

// QuerySQL executes query and returns rows
func QuerySQL(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("SQL query failed: %w\nQuery: %s", err, query)
	}
	return rows, nil
}

// configureDatabaseSettings optimizes database for performance
func configureDatabaseSettings(db *sql.DB, dsn string) error {
	fmt.Printf("[OK] Database configured\n")
	return nil
}

// EnsureCompleteSchema creates all required tables and indexes for the ookstats database
func EnsureCompleteSchema(db *sql.DB) error {
	fmt.Printf("Ensuring complete database schema...\n")

	// Create all tables first
	tables := []string{
		// Reference tables
		`CREATE TABLE IF NOT EXISTS dungeons (
			id INTEGER PRIMARY KEY,
			slug TEXT UNIQUE,
			name TEXT,
			map_id INTEGER,
			map_challenge_mode_id INTEGER UNIQUE
		)`,

		`CREATE TABLE IF NOT EXISTS realms (
            id INTEGER PRIMARY KEY,
            slug TEXT,
            name TEXT,
            region TEXT,
            connected_realm_id INTEGER UNIQUE,
            parent_realm_slug TEXT
        )`,

		// Core leaderboard data
		`CREATE TABLE IF NOT EXISTS challenge_runs (
			id INTEGER PRIMARY KEY,
			duration INTEGER,
			completed_timestamp INTEGER,
			keystone_level INTEGER DEFAULT 1,
			dungeon_id INTEGER,
			realm_id INTEGER,
			period_id INTEGER,
			period_start_timestamp INTEGER,
			period_end_timestamp INTEGER,
			team_signature TEXT,
			season_id INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			blizzard_character_id INTEGER,
			name TEXT,
			name_lower TEXT,
			realm_id INTEGER,
			is_valid INTEGER DEFAULT 1,
			status_checked_at INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS run_members (
			run_id INTEGER,
			player_id INTEGER,
			spec_id INTEGER,
			faction TEXT
		)`,

		`CREATE TABLE IF NOT EXISTS player_fingerprints (
			player_id INTEGER PRIMARY KEY REFERENCES players(id),
			fingerprint_hash TEXT UNIQUE,
			class_id INTEGER NOT NULL,
			level85_timestamp INTEGER NOT NULL,
			level90_timestamp INTEGER NOT NULL,
			earliest_heroic_timestamp INTEGER NOT NULL,
			last_seen_name TEXT,
			last_seen_realm_slug TEXT,
			last_seen_timestamp INTEGER,
			first_run_timestamp INTEGER,
			created_at INTEGER
		)`,

		// Player aggregation and rankings (season-scoped)
		`CREATE TABLE IF NOT EXISTS player_profiles (
			player_id INTEGER,
			season_id INTEGER NOT NULL,
			name TEXT,
			realm_id INTEGER,
			main_spec_id INTEGER,
			class_name TEXT,
			dungeons_completed INTEGER DEFAULT 0,
			total_runs INTEGER DEFAULT 0,
			combined_best_time INTEGER,
			average_best_time INTEGER,
			global_ranking INTEGER,
			regional_ranking INTEGER,
			realm_ranking INTEGER,
			global_ranking_bracket TEXT,
			regional_ranking_bracket TEXT,
			realm_ranking_bracket TEXT,
			global_class_rank INTEGER,
			region_class_rank INTEGER,
			realm_class_rank INTEGER,
			global_class_bracket TEXT,
			region_class_bracket TEXT,
			realm_class_bracket TEXT,
			has_complete_coverage INTEGER DEFAULT 0,
			last_updated INTEGER,
			PRIMARY KEY (player_id, season_id)
		)`,

		`CREATE TABLE IF NOT EXISTS player_best_runs (
			player_id INTEGER,
			dungeon_id INTEGER,
			run_id INTEGER,
			duration INTEGER,
			season_id INTEGER NOT NULL,
			global_ranking INTEGER,
			global_ranking_filtered INTEGER,
			regional_ranking INTEGER,
			realm_ranking INTEGER,
			regional_ranking_filtered INTEGER,
			realm_ranking_filtered INTEGER,
			percentile_bracket TEXT,
			global_percentile_bracket TEXT,
			regional_percentile_bracket TEXT,
			realm_percentile_bracket TEXT,
			completed_timestamp INTEGER,
			PRIMARY KEY (player_id, dungeon_id, season_id)
		)`,

		`CREATE TABLE IF NOT EXISTS player_rankings (
			player_id INTEGER NOT NULL,
			ranking_type TEXT NOT NULL,
			ranking_scope TEXT NOT NULL,
			ranking INTEGER,
			combined_best_time INTEGER,
			last_updated INTEGER,
			PRIMARY KEY (player_id, ranking_type, ranking_scope)
		)`,

		`CREATE TABLE IF NOT EXISTS player_seasonal_rankings (
			player_id INTEGER,
			season_id INTEGER NOT NULL,
			dungeons_completed INTEGER DEFAULT 0,
			combined_best_time INTEGER,
			global_ranking INTEGER,
			regional_ranking INTEGER,
			realm_ranking INTEGER,
			global_ranking_bracket TEXT,
			regional_ranking_bracket TEXT,
			realm_ranking_bracket TEXT,
			last_updated INTEGER,
			PRIMARY KEY (player_id, season_id)
		)`,

		// Extended player information
		`CREATE TABLE IF NOT EXISTS player_details (
			player_id INTEGER PRIMARY KEY,
			race_id INTEGER,
			race_name TEXT,
			gender TEXT,
			class_id INTEGER,
			class_name TEXT,
			active_spec_id INTEGER,
			active_spec_name TEXT,
			guild_name TEXT,
			level INTEGER,
			average_item_level INTEGER,
			equipped_item_level INTEGER,
			avatar_url TEXT,
			last_login_timestamp INTEGER,
			last_updated INTEGER
		)`,

		// Equipment system
		`CREATE TABLE IF NOT EXISTS player_equipment (
			id INTEGER PRIMARY KEY,
			player_id INTEGER,
			slot_type TEXT,
			item_id INTEGER,
			upgrade_id INTEGER,
			quality TEXT,
			item_name TEXT,
			snapshot_timestamp INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS player_equipment_enchantments (
			id INTEGER PRIMARY KEY,
			equipment_id INTEGER,
			enchantment_id INTEGER,
			slot_id INTEGER,
			slot_type TEXT,
			display_string TEXT,
			source_item_id INTEGER,
			source_item_name TEXT,
			spell_id INTEGER
		)`,

		// Computed rankings
		`CREATE TABLE IF NOT EXISTS run_rankings (
			run_id INTEGER,
			dungeon_id INTEGER,
			ranking_type TEXT,
			ranking_scope TEXT,
			ranking INTEGER,
			percentile_bracket TEXT,
			season_id INTEGER NOT NULL,
			computed_at INTEGER,
			PRIMARY KEY (run_id, ranking_type, ranking_scope, season_id)
		)`,

		// Metadata and items
		`CREATE TABLE IF NOT EXISTS api_fetch_metadata (
			id INTEGER PRIMARY KEY,
			fetch_type TEXT UNIQUE,
			last_fetch_timestamp INTEGER,
			last_successful_fetch INTEGER,
			runs_fetched INTEGER DEFAULT 0,
			players_fetched INTEGER DEFAULT 0
		)`,
		// Incremental markers used by fetchers/batch processing
		`CREATE TABLE IF NOT EXISTS api_fetch_markers (
			realm_slug TEXT NOT NULL,
			dungeon_id INTEGER NOT NULL,
			period_id INTEGER NOT NULL,
			last_completed_ts INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (realm_slug, dungeon_id, period_id)
		)`,

		`CREATE TABLE IF NOT EXISTS fetch_status (
			region TEXT NOT NULL,
			realm_slug TEXT NOT NULL,
			dungeon_id INTEGER NOT NULL,
			period_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			http_status INTEGER,
			checked_at INTEGER NOT NULL,
			message TEXT,
			PRIMARY KEY (region, realm_slug, dungeon_id, period_id)
		)`,

		`CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY,
			name TEXT,
			icon TEXT,
			quality INTEGER,
			type INTEGER,
			stats TEXT
		)`,

		// Season tables
		`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			season_number INTEGER NOT NULL,
			region TEXT NOT NULL,
			start_timestamp INTEGER,
			end_timestamp INTEGER,
			season_name TEXT,
			first_period_id INTEGER,
			last_period_id INTEGER,
			UNIQUE(season_number, region)
		)`,

		`CREATE TABLE IF NOT EXISTS period_seasons (
			period_id INTEGER,
			season_id INTEGER,
			PRIMARY KEY (period_id, season_id)
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	fmt.Printf("[OK] All tables created\n")

	// Migrate realms schema if needed (ensure (region, slug) composite uniqueness)
	if err := migrateRealmsCompositeSlug(db); err != nil {
		return err
	}

	// Migrate seasons schema if needed (add region column)
	if err := migrateSeasonsAddRegion(db); err != nil {
		return err
	}

	// Ensure players table has identity/status columns
	if err := migratePlayersIdentityColumns(db); err != nil {
		return err
	}

	// Migrate player_rankings to add PRIMARY KEY constraint
	if err := migratePlayerRankingsPrimaryKey(db); err != nil {
		return err
	}

	// Create indexes
	return ensureRecommendedIndexes(db)
}

// ensureRecommendedIndexes creates indexes used by hot paths if missing
func ensureRecommendedIndexes(db *sql.DB) error {
	stmts := []string{
		// Ensure composite uniqueness for realms
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_realms_region_slug ON realms(region, slug)",
		// Fast path for high-water checks
		"CREATE INDEX IF NOT EXISTS idx_runs_realm_dungeon_ct ON challenge_runs(realm_id, dungeon_id, completed_timestamp)",
		// Uniqueness key to avoid duplicates
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_unique ON challenge_runs(completed_timestamp, dungeon_id, duration, realm_id, team_signature)",
		// Lookups used elsewhere
		"CREATE INDEX IF NOT EXISTS idx_players_name_lower ON players(name_lower)",
		"CREATE INDEX IF NOT EXISTS idx_players_blizzard_id ON players(blizzard_character_id)",
		"CREATE INDEX IF NOT EXISTS idx_players_status_checked ON players(status_checked_at)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_run_members_pair ON run_members(run_id, player_id)",
		"CREATE INDEX IF NOT EXISTS idx_fetch_status_realm ON fetch_status(region, realm_slug)",
		"CREATE INDEX IF NOT EXISTS idx_fetch_status_dungeon ON fetch_status(dungeon_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_player_fingerprints_hash ON player_fingerprints(fingerprint_hash)",
		// Speed up canonical run selection: partition/order by team per dungeon
		"CREATE INDEX IF NOT EXISTS idx_runs_dungeon_team_duration ON challenge_runs(dungeon_id, team_signature, duration, completed_timestamp, id)",
		// Additional indexes for performance
		"CREATE INDEX IF NOT EXISTS idx_run_members_player_id ON run_members(player_id)",
		"CREATE INDEX IF NOT EXISTS idx_challenge_runs_dungeon_duration ON challenge_runs(dungeon_id, duration)",
		// Season-related indexes
		"CREATE INDEX IF NOT EXISTS idx_challenge_runs_season ON challenge_runs(season_id, dungeon_id, completed_timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_run_rankings_season ON run_rankings(season_id, ranking_type, ranking_scope, dungeon_id)",
		"CREATE INDEX IF NOT EXISTS idx_player_best_runs_season ON player_best_runs(season_id, player_id)",
		"CREATE INDEX IF NOT EXISTS idx_player_profiles_season ON player_profiles(season_id, global_ranking)",
		"CREATE INDEX IF NOT EXISTS idx_player_profiles_season_coverage ON player_profiles(season_id, has_complete_coverage, combined_best_time)",
		// Player rankings indexes
		"CREATE INDEX IF NOT EXISTS idx_player_rankings_scope ON player_rankings(ranking_type, ranking_scope)",
		"CREATE INDEX IF NOT EXISTS idx_player_rankings_player ON player_rankings(player_id)",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			// don't fail the connection on index creation errors
			fmt.Printf("Warning: index creation failed: %v\n", err)
		}
	}

	fmt.Printf("[OK] All indexes ensured\n")
	return nil
}

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
