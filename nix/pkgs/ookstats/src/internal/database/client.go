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
