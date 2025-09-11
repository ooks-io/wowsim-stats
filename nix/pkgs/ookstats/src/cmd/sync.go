package cmd

import (
    "bufio"
    "bytes"
    "database/sql"
    "errors"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"
    _ "github.com/tursodatabase/go-libsql"
    "ookstats/internal/database"
)

var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Sync local database to Turso",
    Long:  `Push the local SQLite database to Turso.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        fmt.Println("Syncing local database to Turso...")

        // check if local db exists
        localPath := database.DBFilePath()
        if _, err := os.Stat(localPath); os.IsNotExist(err) {
            return fmt.Errorf("%s not found - run 'fetch cm' first", localPath)
        }

        // read flags
        bulk, _ := cmd.Flags().GetBool("bulk")
        noClear, _ := cmd.Flags().GetBool("no-clear")
        tursoDBFlag, _ := cmd.Flags().GetString("turso-db-name")
        tursoURLFlag, _ := cmd.Flags().GetString("turso-url")
        tablesFlag, _ := cmd.Flags().GetString("tables")
        pragmaDeferFK, _ := cmd.Flags().GetBool("defer-foreign-keys")
        timeoutSecs, _ := cmd.Flags().GetInt("timeout-seconds")

        if bulk {
            return syncBulk(tursoDBFlag, tursoURLFlag, noClear, tablesFlag, pragmaDeferFK, time.Duration(timeoutSecs)*time.Second)
        }

        // Default: driver-based copy (existing behavior)
        url := os.Getenv("TURSO_DATABASE_URL")
        if url == "" {
            return fmt.Errorf("TURSO_DATABASE_URL environment variable required")
        }
        authToken := os.Getenv("TURSO_AUTH_TOKEN")
        if authToken == "" {
            return fmt.Errorf("TURSO_AUTH_TOKEN environment variable required")
        }

        fmt.Printf("Connecting to remote database: %s\n", url)

        start := time.Now()

        // open local database
        localDB, err := sql.Open("libsql", database.DBConnString())
        if err != nil {
            return fmt.Errorf("failed to open local database: %w", err)
        }
        defer localDB.Close()

        // connect directly to remote database
        remoteURL := url + "?authToken=" + authToken
        remoteDB, err := sql.Open("libsql", remoteURL)
        if err != nil {
            return fmt.Errorf("failed to open remote database: %w", err)
        }
        defer remoteDB.Close()

        fmt.Println("Starting data transfer (driver mode)...")
        if err := syncLocalToRemote(localDB, remoteDB); err != nil {
            return fmt.Errorf("failed to sync data: %w", err)
        }

        elapsed := time.Since(start)
        fmt.Printf("Successfully synced %s to Turso in %v\n", localPath, elapsed)

        return nil
    },
}

// syncLocalToRemote transfers data from local database to remote database
func syncLocalToRemote(localDB, remoteDB *sql.DB) error {
	// tables to sync in order (respecting foreign key dependencies)
	tables := []string{
		"realms",
		"dungeons",
		"players",
		"challenge_runs",
		"run_members",
		"api_fetch_metadata",
	}

	fmt.Printf("Clearing remote database tables...\n")
	// clear remote tables in reverse order
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		_, err := remoteDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	// copy data from local to remote
	for _, table := range tables {
		fmt.Printf("Syncing table: %s...\n", table)
		if err := syncTable(localDB, remoteDB, table); err != nil {
			return fmt.Errorf("failed to sync table %s: %w", table, err)
		}
	}

	return nil
}

// syncTable copies all data from a local table to remote table
func syncTable(localDB, remoteDB *sql.DB, tableName string) error {
	// get all rows from local table
	rows, err := localDB.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to query local table: %w", err)
	}
	defer rows.Close()

	// get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// build column names and placeholders
	quotedColumns := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = "`" + col + "`"
		placeholders[i] = "?"
	}

	columnNames := strings.Join(quotedColumns, ", ")
	placeholderStr := strings.Join(placeholders, ", ")

	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columnNames, placeholderStr)

	stmt, err := remoteDB.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	rowCount := 0
	for rows.Next() {
		// create slice to hold row values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}

		rowCount++
	}

	fmt.Printf("  → Synced %d rows\n", rowCount)
	return nil
}

func init() {
    rootCmd.AddCommand(syncCmd)

    // Bulk mode flags
    syncCmd.Flags().Bool("bulk", false, "Use fast bulk import via sqlite3 dump and turso CLI")
    syncCmd.Flags().Bool("no-clear", false, "Do not clear tables before import (bulk mode)")
    syncCmd.Flags().String("turso-db-name", "", "Turso DB name for bulk import (falls back to TURSO_DB env)")
    syncCmd.Flags().String("tables", "", "Comma-separated list of tables to import (bulk mode). Defaults to recommended tables")
    syncCmd.Flags().String("turso-url", "", "Turso replica URL (e.g. libsql://host) for bulk import; falls back to TURSO_DATABASE_URL or ASTRO_DB_REMOTE_URL")
    syncCmd.Flags().Bool("defer-foreign-keys", true, "Set PRAGMA defer_foreign_keys=ON during bulk import")
    syncCmd.Flags().Int("timeout-seconds", 900, "Timeout for bulk import (seconds)")
}

// -------------------- Bulk sync implementation --------------------

func syncBulk(tursoDBFlag, tursoURLFlag string, noClear bool, tablesFlag string, pragmaDeferFK bool, timeout time.Duration) error {
    start := time.Now()

    // Resolve Turso DB name
    tursoDB := strings.TrimSpace(tursoDBFlag)
    if tursoDB == "" {
        tursoDB = strings.TrimSpace(os.Getenv("TURSO_DB"))
    }

    authToken := strings.TrimSpace(os.Getenv("TURSO_AUTH_TOKEN"))
    if authToken == "" {
        return errors.New("TURSO_AUTH_TOKEN environment variable required for bulk import")
    }

    // Resolve Turso replica URL (preferred) for non-interactive auth
    tursoURL := strings.TrimSpace(tursoURLFlag)
    if tursoURL == "" {
        tursoURL = strings.TrimSpace(os.Getenv("TURSO_DATABASE_URL"))
    }
    if tursoURL == "" {
        // fallback to Astro env if present
        tursoURL = strings.TrimSpace(os.Getenv("ASTRO_DB_REMOTE_URL"))
    }

    // Ensure required binaries are present
    if _, err := exec.LookPath("sqlite3"); err != nil {
        return fmt.Errorf("sqlite3 executable not found in PATH: %w", err)
    }
    if _, err := exec.LookPath("turso"); err != nil {
        return fmt.Errorf("turso CLI not found in PATH: %w", err)
    }

    // Determine table list
    tables := defaultBulkTables()
    if strings.TrimSpace(tablesFlag) != "" {
        parts := strings.Split(tablesFlag, ",")
        var cleaned []string
        for _, p := range parts {
            t := strings.TrimSpace(p)
            if t != "" {
                cleaned = append(cleaned, t)
            }
        }
        if len(cleaned) > 0 {
            tables = cleaned
        }
    }

    // Create temp dir
    tmpDir, err := os.MkdirTemp("", "ookstats-sync-")
    if err != nil {
        return fmt.Errorf("failed to create temp dir: %w", err)
    }
    defer os.RemoveAll(tmpDir)

    dumpPath := filepath.Join(tmpDir, "dump.sql")
    compressedPath := filepath.Join(tmpDir, "dump_compressed.sql")
    wrappedPath := filepath.Join(tmpDir, "dump_wrapped.sql")

    localPath := database.DBFilePath()
    fmt.Printf("Generating data-only SQL dump from %s...\n", localPath)
    if err := sqliteDump(localPath, tables, dumpPath); err != nil {
        return err
    }

    fmt.Println("Compressing INSERTs into multi-row batches...")
    if err := compressInserts(dumpPath, compressedPath, 500, noClear == true /* insertIgnore when merging */); err != nil {
        return err
    }

    fmt.Println("Wrapping dump in single transaction and clearing tables...")
    if err := wrapDump(compressedPath, wrappedPath, tables, !noClear, pragmaDeferFK); err != nil {
        return err
    }

    // Determine shell target: prefer URL + token; otherwise fallback to DB name (requires prior CLI login)
    shellTarget := tursoDB
    if tursoURL != "" {
        shellTarget = addAuthTokenToURL(tursoURL, authToken)
    } else if shellTarget == "" {
        return errors.New("missing Turso URL or DB name: set --turso-url or TURSO_DATABASE_URL (or ASTRO_DB_REMOTE_URL), or provide --turso-db-name/TURSO_DB after logging in with 'turso auth login'")
    }

    fmt.Printf("Importing into Turso via turso CLI target: %s\n", redactTarget(shellTarget))
    if err := tursoImport(shellTarget, wrappedPath, timeout); err != nil {
        return err
    }

    elapsed := time.Since(start)
    fmt.Printf("Bulk sync complete in %v\n", elapsed)
    return nil
}

func defaultBulkTables() []string {
    return []string{
        "player_equipment_enchantments",
        "player_equipment",
        "player_details",
        "player_best_runs",
        "run_members",
        "challenge_runs",
        "players",
        "realms",
        "dungeons",
        "items",
    }
}

func sqliteDump(dbPath string, tables []string, outPath string) error {
    // Build a script for sqlite3 stdin to avoid argv dot-command parsing issues
    var script strings.Builder
    script.WriteString(".mode insert\n")
    script.WriteString(".headers off\n")
    script.WriteString(".dump")
    for _, t := range tables {
        script.WriteString(" ")
        script.WriteString(t)
    }
    script.WriteString("\n")

    cmd := exec.Command("sqlite3", dbPath)
    cmd.Stdin = strings.NewReader(script.String())
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("sqlite3 dump failed: %v\n%s", err, stderr.String())
    }

    // Write stdout to outPath
    if err := os.WriteFile(outPath, stdout.Bytes(), 0o644); err != nil {
        return fmt.Errorf("write dump file failed: %w", err)
    }
    return nil
}

// compressInserts reads a dump with one INSERT per line and rewrites it to fewer
// INSERT statements using multi-row VALUES lists. This significantly reduces parse overhead.
func compressInserts(inPath, outPath string, batchSize int, insertIgnore bool) error {
    in, err := os.Open(inPath)
    if err != nil {
        return fmt.Errorf("open dump for compression: %w", err)
    }
    defer in.Close()

    out, err := os.Create(outPath)
    if err != nil {
        return fmt.Errorf("create compressed dump: %w", err)
    }
    defer out.Close()

    reader := bufio.NewScanner(in)
    writer := bufio.NewWriter(out)
    defer writer.Flush()

    var currentTable string
    var valuesBatch []string

    flush := func() error {
        if currentTable == "" || len(valuesBatch) == 0 {
            return nil
        }
        // Write: INSERT INTO "table" VALUES (...),(...);
        insertKW := "INSERT INTO "
        if insertIgnore {
            insertKW = "INSERT OR IGNORE INTO "
        }
        if _, err := writer.WriteString(insertKW + "\"" + currentTable + "\" VALUES "); err != nil {
            return err
        }
        for i, v := range valuesBatch {
            if i > 0 {
                if _, err := writer.WriteString(","); err != nil {
                    return err
                }
            }
            if _, err := writer.WriteString(v); err != nil {
                return err
            }
        }
        if _, err := writer.WriteString(";\n"); err != nil {
            return err
        }
        valuesBatch = valuesBatch[:0]
        return nil
    }

    for reader.Scan() {
        line := strings.TrimSpace(reader.Text())
        if line == "" {
            continue
        }
        // Only keep INSERT statements; drop BEGIN/COMMIT/PRAGMA/CREATE etc. to avoid nested transactions
        if strings.HasPrefix(strings.ToUpper(line), "INSERT INTO ") {
            // Extract table name and tuple part
            // Find first 'VALUES'
            upper := strings.ToUpper(line)
            idx := strings.Index(upper, " VALUES")
            if idx <= 0 {
                // write-through unexpected lines
                if err := flush(); err != nil {
                    return err
                }
                if _, err := writer.WriteString(line + "\n"); err != nil {
                    return err
                }
                currentTable = ""
                continue
            }
            // Table name between INSERT INTO and VALUES
            tblPart := strings.TrimSpace(line[len("INSERT INTO "):idx])
            // tblPart is quoted like "table"; strip quotes
            tbl := strings.Trim(tblPart, "\"")
            // Tuple between VALUES and trailing ';'
            tuple := strings.TrimSpace(line[idx+len(" VALUES"):])
            // Ensure tuple ends without trailing semicolon
            if strings.HasSuffix(tuple, ";") {
                tuple = strings.TrimSuffix(tuple, ";")
            }

            if currentTable != tbl {
                if err := flush(); err != nil {
                    return err
                }
                currentTable = tbl
            }
            valuesBatch = append(valuesBatch, tuple)
            if len(valuesBatch) >= batchSize {
                if err := flush(); err != nil {
                    return err
                }
            }
            continue
        }

        // Non-INSERT line: ignore (do not write through)
        if err := flush(); err != nil { return err }
        currentTable = ""
    }
    if err := reader.Err(); err != nil {
        return fmt.Errorf("read dump: %w", err)
    }
    if err := flush(); err != nil {
        return err
    }
    return nil
}

func wrapDump(dumpPath, wrappedPath string, tables []string, clearTables bool, deferFK bool) error {
    in, err := os.Open(dumpPath)
    if err != nil {
        return fmt.Errorf("open dump: %w", err)
    }
    defer in.Close()

    out, err := os.Create(wrappedPath)
    if err != nil {
        return fmt.Errorf("create wrapped dump: %w", err)
    }
    defer out.Close()

    w := bufio.NewWriter(out)
    _, _ = w.WriteString("BEGIN;\n")
    if deferFK {
        _, _ = w.WriteString("PRAGMA defer_foreign_keys=ON;\n")
    }
    if clearTables {
        // Disable foreign key checks entirely for speed if present (safe when replacing all data)
        _, _ = w.WriteString("PRAGMA foreign_keys=OFF;\n")
        // Drop known heavy indexes before bulk insert (recreated later)
        _, _ = w.WriteString("DROP INDEX IF EXISTS challenge_runs_completed_timestamp_dungeon_id_duration_realm_id_team_signature_idx;\n")
    }
    if clearTables {
        // Clear in reverse FK dependency order
        for _, tbl := range tables {
            _, _ = w.WriteString("DELETE FROM ")
            _, _ = w.WriteString(tbl)
            _, _ = w.WriteString(";\n")
        }
    }

    // Append original dump
    if _, err := io.Copy(w, in); err != nil {
        return fmt.Errorf("write wrapped dump: %w", err)
    }
    if clearTables {
        // Recreate dropped indexes inside the same transaction
        _, _ = w.WriteString("\nCREATE UNIQUE INDEX IF NOT EXISTS \"challenge_runs_completed_timestamp_dungeon_id_duration_realm_id_team_signature_idx\" ON \"challenge_runs\" (\"completed_timestamp\", \"dungeon_id\", \"duration\", \"realm_id\", \"team_signature\");\n")
    }
    _, _ = w.WriteString("\nCOMMIT;\n")
    if err := w.Flush(); err != nil {
        return fmt.Errorf("flush wrapped dump: %w", err)
    }
    return nil
}

func tursoImport(shellTarget, sqlPath string, timeout time.Duration) error {
    f, err := os.Open(sqlPath)
    if err != nil {
        return fmt.Errorf("open sql file: %w", err)
    }
    defer f.Close()

    // Add timeout so we don't hang forever
    cmd := exec.Command("turso", "db", "shell", shellTarget)
    cmd.Stdin = f
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    done := make(chan error, 1)
    go func() { done <- cmd.Run() }()

    var runErr error
    select {
    case runErr = <-done:
        // finished
    case <-time.After(timeout):
        _ = cmd.Process.Kill()
        return fmt.Errorf("turso import timed out after %v", timeout)
    }

    if runErr != nil {
        return fmt.Errorf("turso import failed: %v\nStdout: %s\nStderr: %s", runErr, stdout.String(), stderr.String())
    }
    if s := stderr.String(); strings.TrimSpace(s) != "" {
        // Turso CLI may print warnings; surface them
        fmt.Printf("turso CLI: %s\n", s)
    }
    return nil
}

func addAuthTokenToURL(url, token string) string {
    if token == "" {
        return url
    }
    sep := "?"
    if strings.Contains(url, "?") {
        sep = "&"
    }
    return url + sep + "authToken=" + token
}

func redactTarget(target string) string {
    // Hide auth tokens in logs
    if target == "" {
        return target
    }
    // redacts authToken parameter
    if strings.Contains(target, "authToken=") {
        // split on authToken and mask value
        parts := strings.Split(target, "authToken=")
        // trim any following params after token
        rest := parts[1]
        if i := strings.IndexAny(rest, "&# "); i >= 0 {
            rest = rest[i:]
        } else {
            rest = ""
        }
        return parts[0] + "authToken=***REDACTED***" + rest
    }
    // If it's a URL, try to hide query entirely
    if i := strings.Index(target, "?"); i >= 0 {
        return target[:i] + "?***REDACTED***"
    }
    return target
}
