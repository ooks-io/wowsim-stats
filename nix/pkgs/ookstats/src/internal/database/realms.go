package database

import (
	"database/sql"
	"fmt"
)

// getRealmIDTx gets a realm ID within a transaction
func (ds *DatabaseService) getRealmIDTx(tx *sql.Tx, slug string, region string) (int, error) {
	var realmID int
	err := tx.QueryRow("SELECT id FROM realms WHERE slug = ? AND region = ?", slug, region).Scan(&realmID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return realmID, err
}

// getDungeonIDTx gets a dungeon ID within a transaction
func (ds *DatabaseService) getDungeonIDTx(tx *sql.Tx, slug string) (int, error) {
	var dungeonID int
	err := tx.QueryRow("SELECT id FROM dungeons WHERE slug = ?", slug).Scan(&dungeonID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("dungeon not found: %s", slug)
	}
	return dungeonID, err
}

// GetRealmPoolIDs returns all realm IDs in a realm pool (parent + all children)
func (ds *DatabaseService) GetRealmPoolIDs(region, slug string) ([]int, error) {
	// First, get the realm's parent_realm_slug
	var parentSlug sql.NullString
	err := ds.db.QueryRow(`
		SELECT parent_realm_slug
		FROM realms
		WHERE region = ? AND slug = ?
	`, region, slug).Scan(&parentSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to query realm: %w", err)
	}

	// Determine the pool leader slug
	poolLeaderSlug := slug
	if parentSlug.Valid && parentSlug.String != "" {
		poolLeaderSlug = parentSlug.String
	}

	// Get all realms in the pool: the pool leader + all realms that have it as parent
	query := `
		SELECT id FROM realms
		WHERE region = ? AND (
			slug = ? OR parent_realm_slug = ?
		)
		ORDER BY id
	`
	rows, err := ds.db.Query(query, region, poolLeaderSlug, poolLeaderSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query realm pool: %w", err)
	}
	defer rows.Close()

	var poolIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		poolIDs = append(poolIDs, id)
	}
	return poolIDs, rows.Err()
}
