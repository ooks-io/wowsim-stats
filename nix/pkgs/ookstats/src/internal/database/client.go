package database

import (
    "database/sql"
    "fmt"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "time"

    libsql "github.com/tursodatabase/go-libsql"
    _ "github.com/tursodatabase/go-libsql"
)

// connect creates a local libSQL database connection (no Turso syncing)
var dbPathOverride string

// SetDBPath allows callers (CLI) to override the local SQLite filename (or libsql conn string)
func SetDBPath(path string) {
    dbPathOverride = path
}

// DBConnString returns the libsql connection string for the local SQLite file
func DBConnString() string {
    // Priority: explicit override → env vars → default
    if dbPathOverride != "" {
        if strings.HasPrefix(dbPathOverride, "file:") || strings.Contains(dbPathOverride, "://") {
            return dbPathOverride
        }
        return "file:" + dbPathOverride
    }
    if v := os.Getenv("OOKSTATS_DB"); v != "" {
        if strings.HasPrefix(v, "file:") || strings.Contains(v, "://") {
            return v
        }
        return "file:" + v
    }
    // Support common Turso/Astro envs for remote connections without requiring OOKSTATS_DB
    if url := strings.TrimSpace(os.Getenv("TURSO_DATABASE_URL")); url != "" {
        tok := strings.TrimSpace(os.Getenv("TURSO_AUTH_TOKEN"))
        if tok != "" && !strings.Contains(url, "authToken=") {
            sep := "?"
            if strings.Contains(url, "?") { sep = "&" }
            return url + sep + "authToken=" + tok
        }
        return url
    }
    if url := strings.TrimSpace(os.Getenv("ASTRO_DB_REMOTE_URL")); url != "" {
        tok := strings.TrimSpace(os.Getenv("ASTRO_DB_APP_TOKEN"))
        if tok != "" && !strings.Contains(url, "authToken=") {
            sep := "?"
            if strings.Contains(url, "?") { sep = "&" }
            return url + sep + "authToken=" + tok
        }
        return url
    }
    if v := os.Getenv("ASTRO_DATABASE_FILE"); v != "" {
        // Astro uses file: URIs typically
        if strings.HasPrefix(v, "file:") || strings.Contains(v, "://") {
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
    fmt.Printf("Using local libSQL database: %s\n", dsn)

    fmt.Printf("Opening database connection...\n")

    // Detect embedded replica mode: file: DSN with either vfs=libsql or sync_url param
    if strings.HasPrefix(dsn, "file:") && (strings.Contains(dsn, "vfs=libsql") || strings.Contains(dsn, "sync_url=")) {
        // Parse local path and query params
        raw := strings.TrimPrefix(dsn, "file:")
        var dbFile string
        var q string
        if i := strings.Index(raw, "?"); i >= 0 {
            dbFile = raw[:i]
            q = raw[i+1:]
        } else {
            dbFile = raw
        }
        // Resolve to absolute path for reliability
        if abs, err := filepath.Abs(dbFile); err == nil {
            dbFile = abs
        }
        vals, _ := url.ParseQuery(q)
        primaryURL := vals.Get("sync_url")
        if primaryURL == "" {
            // fallback to env vars
            primaryURL = strings.TrimSpace(os.Getenv("TURSO_DATABASE_URL"))
            if primaryURL == "" {
                primaryURL = strings.TrimSpace(os.Getenv("ASTRO_DB_REMOTE_URL"))
            }
        }
        authToken := vals.Get("authToken")
        if authToken == "" {
            // fallback to env
            authToken = strings.TrimSpace(os.Getenv("TURSO_AUTH_TOKEN"))
            if authToken == "" {
                authToken = strings.TrimSpace(os.Getenv("ASTRO_DB_APP_TOKEN"))
            }
        }
        if primaryURL == "" || authToken == "" {
            return nil, fmt.Errorf("embedded replica requires sync_url and authToken (or TURSO_DATABASE_URL/TURSO_AUTH_TOKEN)")
        }

        // Build embedded replica connector
        connector, err := libsql.NewEmbeddedReplicaConnector(dbFile, primaryURL, libsql.WithAuthToken(authToken))
        if err != nil {
            return nil, fmt.Errorf("failed to create embedded replica connector: %w", err)
        }
        // Open DB through connector
        db := sql.OpenDB(connector)

        fmt.Printf("Testing database connection...\n")
        if err := db.Ping(); err != nil {
            db.Close()
            return nil, fmt.Errorf("failed to ping embedded replica: %w", err)
        }

        // Warm schema once
        for i := 0; i < 5; i++ {
            var cnt int
            if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&cnt); err == nil {
                break
            }
            time.Sleep(200 * time.Millisecond)
        }

        fmt.Printf("✓ Embedded replica connected (file: %s)\n", dbFile)
        return db, nil
    }

    // Default: remote url or local file via registered driver
    db, err := sql.Open("libsql", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open libsql database: %w", err)
    }

    fmt.Printf("Testing database connection...\n")
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to ping local database: %w", err)
    }

    // configure database for optimal performance
    if err := configureDatabaseSettings(db, dsn); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to configure database: %w", err)
    }

    fmt.Printf("✓ Local libSQL database connected\n")
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

// configureDatabaseSettings optimizes database for performance and reliability
func configureDatabaseSettings(db *sql.DB, dsn string) error {
    // for libSQL embedded replica, we should be more conservative with PRAGMA settings
    // many settings are already optimized by libSQL itself

    // Skip unsupported PRAGMAs for remote libsql/turso URLs (non-file DSN)
    if strings.HasPrefix(dsn, "file:") || (!strings.Contains(dsn, "://")) {
        // local SQLite file: apply a modest cache size
        if _, err := db.Exec("PRAGMA cache_size = -64000"); err != nil {
            // ignore if not supported
        }
    }

	// check if we're in WAL mode (embedded replica should already be in WAL mode)
	var journalMode string
    err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err == nil {
		fmt.Printf("✓ Database journal mode: %s\n", journalMode)
	}

	fmt.Printf("✓ Database configured\n")
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
			slug TEXT UNIQUE,
			name TEXT,
			region TEXT,
			connected_realm_id INTEGER UNIQUE
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
		
		// Player aggregation and rankings
		`CREATE TABLE IF NOT EXISTS player_profiles (
			player_id INTEGER PRIMARY KEY,
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
			last_updated INTEGER
		)`,
		
		`CREATE TABLE IF NOT EXISTS player_best_runs (
			player_id INTEGER,
			dungeon_id INTEGER,
			run_id INTEGER,
			duration INTEGER,
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
			completed_timestamp INTEGER
		)`,
		
		`CREATE TABLE IF NOT EXISTS player_rankings (
			player_id INTEGER,
			ranking_type TEXT,
			ranking_scope TEXT,
			ranking INTEGER,
			combined_best_time INTEGER,
			last_updated INTEGER
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
			computed_at INTEGER
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
	}
	
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	
	fmt.Printf("✓ All tables created\n")
	
	// Create indexes
	return ensureRecommendedIndexes(db)
}

// ensureRecommendedIndexes creates indexes used by hot paths if missing
func ensureRecommendedIndexes(db *sql.DB) error {
    stmts := []string{
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
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            // don't fail the connection on index creation errors
            fmt.Printf("Warning: index creation failed: %v\n", err)
        }
    }
    
    fmt.Printf("✓ All indexes ensured\n")
    return nil
}
