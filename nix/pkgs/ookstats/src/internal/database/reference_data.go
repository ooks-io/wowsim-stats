package database

import (
	"database/sql"
	"fmt"
	"ookstats/internal/blizzard"
	"sort"
	"strings"
)

// EnsureReferenceData ensures that realm and dungeon reference data exists in the database
func (ds *DatabaseService) EnsureReferenceData(realmInfo blizzard.RealmInfo, dungeons []blizzard.DungeonInfo) error {
	// insert realm data
	realmQuery := `
		INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := ds.db.Exec(realmQuery, realmInfo.Slug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
	if err != nil {
		return fmt.Errorf("failed to insert realm data: %w", err)
	}

	// insert dungeon data
	dungeonQuery := `
		INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id)
		VALUES (?, ?, ?, ?)
	`

	for _, dungeon := range dungeons {
		_, err := ds.db.Exec(dungeonQuery, dungeon.ID, dungeon.Slug, dungeon.Name, dungeon.ID)
		if err != nil {
			return fmt.Errorf("failed to insert dungeon data for %s: %w", dungeon.Name, err)
		}
	}

	return nil
}

// EnsureDungeonsOnce inserts all known dungeons once (idempotent). Optimized for remote DBs.
func (ds *DatabaseService) EnsureDungeonsOnce(dungeons []blizzard.DungeonInfo) error {
	if len(dungeons) == 0 {
		return nil
	}
	// Build a single INSERT OR IGNORE with multi-row VALUES to reduce round trips
	var b strings.Builder
	args := make([]any, 0, len(dungeons)*4)
	b.WriteString("INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id) VALUES ")
	for i, d := range dungeons {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("(?, ?, ?, ?)")
		args = append(args, d.ID, d.Slug, d.Name, d.ID)
	}
	_, err := ds.db.Exec(b.String(), args...)
	if err != nil {
		return fmt.Errorf("failed to ensure dungeons: %w", err)
	}
	return nil
}

// EnsureRealmsBatch inserts/updates all known realms in a single transaction with a prepared statement
func (ds *DatabaseService) EnsureRealmsBatch(realms map[string]blizzard.RealmInfo) error {
	if len(realms) == 0 {
		return nil
	}
	tx, err := ds.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	keys := make([]string, 0, len(realms))
	for k := range realms {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	total := len(keys)
	for i, slug := range keys {
		ri := realms[slug]
		if _, err := stmt.Exec(ri.Slug, ri.Name, ri.Region, ri.ID, ri.ParentRealmSlug); err != nil {
			return fmt.Errorf("failed to insert realm %s: %w", ri.Slug, err)
		}
		if (i+1)%10 == 0 || i+1 == total {
			fmt.Printf("    - Ensured %d/%d realms\n", i+1, total)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ensureReferenceDataTx ensures reference data within a transaction
func (ds *DatabaseService) ensureReferenceDataTx(tx *sql.Tx, realmInfo blizzard.RealmInfo, dungeons []blizzard.DungeonInfo) error {
	// insert realm data
	realmQuery := `INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug) VALUES (?, ?, ?, ?, ?)`
	_, err := tx.Exec(realmQuery, realmInfo.Slug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
	if err != nil {
		return fmt.Errorf("failed to insert realm data: %w", err)
	}

	// insert dungeon data
	dungeonQuery := `INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id) VALUES (?, ?, ?, ?)`
	for _, dungeon := range dungeons {
		_, err := tx.Exec(dungeonQuery, dungeon.ID, dungeon.Slug, dungeon.Name, dungeon.ID)
		if err != nil {
			return fmt.Errorf("failed to insert dungeon data: %w", err)
		}
	}

	return nil
}
