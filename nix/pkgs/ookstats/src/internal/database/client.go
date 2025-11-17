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
            return dbPathOverride
        }
        return "file:" + dbPathOverride
    }
    if v := os.Getenv("OOKSTATS_DB"); v != "" {
        if strings.HasPrefix(v, "file:") {
            return v
        }
        return "file:" + v
    }
    if v := os.Getenv("ASTRO_DATABASE_FILE"); v != "" {
        if strings.HasPrefix(v, "file:") {
            return v
        }
        return "file:" + v
    }
    return "file:local.db"
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
    // apply modest cache size for local SQLite file
    if _, err := db.Exec("PRAGMA cache_size = -64000"); err != nil {
        // ignore if not supported
    }

	// check journal mode
	var journalMode string
	err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err == nil {
		fmt.Printf("[OK] Database journal mode: %s\n", journalMode)
	}

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
			team_signature TEXT
		)`,
		
		`CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			name TEXT,
			name_lower TEXT,
			realm_id INTEGER
		)`,
		
		`CREATE TABLE IF NOT EXISTS run_members (
			run_id INTEGER,
			player_id INTEGER,
			spec_id INTEGER,
			faction TEXT
		)`,
		
		// Player aggregation and rankings (season-scoped)
		`CREATE TABLE IF NOT EXISTS player_profiles (
			player_id INTEGER,
			season_id INTEGER NOT NULL,
			name TEXT,
			realm_id INTEGER,
			main_spec_id INTEGER,
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
			player_id INTEGER,
			ranking_type TEXT,
			ranking_scope TEXT,
			ranking INTEGER,
			combined_best_time INTEGER,
			last_updated INTEGER
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
			id INTEGER PRIMARY KEY,
			season_number INTEGER UNIQUE,
			start_timestamp INTEGER,
			end_timestamp INTEGER,
			season_name TEXT,
			first_period_id INTEGER,
			last_period_id INTEGER
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
        "CREATE UNIQUE INDEX IF NOT EXISTS idx_run_members_pair ON run_members(run_id, player_id)",
        // Speed up canonical run selection: partition/order by team per dungeon
        "CREATE INDEX IF NOT EXISTS idx_runs_dungeon_team_duration ON challenge_runs(dungeon_id, team_signature, duration, completed_timestamp, id)",
        // Additional indexes for performance
        "CREATE INDEX IF NOT EXISTS idx_run_members_player_id ON run_members(player_id)",
        "CREATE INDEX IF NOT EXISTS idx_challenge_runs_dungeon_duration ON challenge_runs(dungeon_id, duration)",
        // Season-related indexes
        "CREATE INDEX IF NOT EXISTS idx_run_rankings_season ON run_rankings(season_id, ranking_type, ranking_scope, dungeon_id)",
        "CREATE INDEX IF NOT EXISTS idx_player_best_runs_season ON player_best_runs(season_id, player_id)",
        "CREATE INDEX IF NOT EXISTS idx_player_profiles_season ON player_profiles(season_id, global_ranking)",
        "CREATE INDEX IF NOT EXISTS idx_player_profiles_season_coverage ON player_profiles(season_id, has_complete_coverage, combined_best_time)",
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
    if err != nil { return err }
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
    `); err != nil { return fmt.Errorf("create realms_new: %w", err) }

    // Copy data
    if _, err := tx.Exec(`INSERT INTO realms_new (id, slug, name, region, connected_realm_id, parent_realm_slug)
                          SELECT id, slug, name, region, connected_realm_id,
                                 CASE WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='realms' AND sql LIKE '%parent_realm_slug%')
                                      THEN parent_realm_slug ELSE NULL END
                          FROM realms`); err != nil {
        return fmt.Errorf("copy realms: %w", err)
    }

    // Rename old and new
    if _, err := tx.Exec(`ALTER TABLE realms RENAME TO realms_old`); err != nil { return fmt.Errorf("rename old: %w", err) }
    if _, err := tx.Exec(`ALTER TABLE realms_new RENAME TO realms`); err != nil { return fmt.Errorf("rename new: %w", err) }

    // Drop old
    if _, err := tx.Exec(`DROP TABLE realms_old`); err != nil { return fmt.Errorf("drop old: %w", err) }

    // Ensure composite unique index
    if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_realms_region_slug ON realms(region, slug)`); err != nil {
        return fmt.Errorf("create composite unique index: %w", err)
    }

    if err := tx.Commit(); err != nil { return err }
    fmt.Printf("[OK] Realms table migrated\n")
    return nil
}
