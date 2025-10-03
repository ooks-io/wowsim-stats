package database

import (
    "database/sql"
    "strings"
    "time"
)

// ChallengeRun represents a database challenge run record
type ChallengeRun struct {
	ID                   int64
	Duration             int
	CompletedTimestamp   int64
	KeystoneLevel        int
	DungeonID            int
	RealmID              int
	PeriodID             *int
	PeriodStartTimestamp *int64
	PeriodEndTimestamp   *int64
	TeamSignature        string
}

// Player represents a database player record
type Player struct {
	ID      int64
	Name    string
	RealmID int
}

// RunMember represents a database run member record
type RunMember struct {
	RunID    int64
	PlayerID int64
	SpecID   *int
	Faction  *string
}

// Realm represents a database realm record
type Realm struct {
	ID               int
	Slug             string
	Name             string
	Region           string
	ConnectedRealmID *int
}

// Dungeon represents a database dungeon record
type Dungeon struct {
	ID                 int
	Slug               string
	Name               string
	MapID              *int
	MapChallengeModeID *int
}

// APIFetchMetadata represents fetch tracking data
type APIFetchMetadata struct {
	ID                  int
	FetchType           string
	LastFetchTimestamp  *int64
	LastSuccessfulFetch *int64
	RunsFetched         int
	PlayersFetched      int
}

// DatabaseService handles database operations for challenge mode data
type DatabaseService struct {
    db *sql.DB
}

// Verbose logging toggle for database batch processing
var verbose bool

// SetVerbose controls internal logging verbosity (e.g., 404 noise suppression)
func SetVerbose(v bool) { verbose = v }

// NewDatabaseService creates a new database service instance
func NewDatabaseService(db *sql.DB) *DatabaseService {
	return &DatabaseService{db: db}
}

// GetLastFetchInfo retrieves information about the last successful fetch
func (ds *DatabaseService) GetLastFetchInfo(fetchType string) (*int64, int, int, error) {
    query := `
        SELECT last_successful_fetch, runs_fetched, players_fetched
        FROM api_fetch_metadata
        WHERE fetch_type = ?
    `

    var lastFetch *int64
    var runsFetched, playersFetched int

    err := ds.db.QueryRow(query, fetchType).Scan(&lastFetch, &runsFetched, &playersFetched)
    if err == sql.ErrNoRows {
        return nil, 0, 0, nil // First time running
    }
    if err != nil {
        // If using embedded replica and schema isn't warmed yet, try to warm schema and retry once
        if strings.Contains(err.Error(), "no such table") {
            var _tmp int
            _ = ds.db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&_tmp)
            err2 := ds.db.QueryRow(query, fetchType).Scan(&lastFetch, &runsFetched, &playersFetched)
            if err2 == nil {
                return lastFetch, runsFetched, playersFetched, nil
            }
            if err2 == sql.ErrNoRows {
                return nil, 0, 0, nil
            }
            return nil, 0, 0, err2
        }
        return nil, 0, 0, err
    }

    return lastFetch, runsFetched, playersFetched, nil
}

// UpdateFetchMetadata updates the fetch metadata with current statistics
func (ds *DatabaseService) UpdateFetchMetadata(fetchType string, runsFetched, playersFetched int) error {
	currentTimestamp := time.Now().UnixMilli()

	query := `
		INSERT OR REPLACE INTO api_fetch_metadata
		(fetch_type, last_fetch_timestamp, last_successful_fetch, runs_fetched, players_fetched)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := ds.db.Exec(query, fetchType, currentTimestamp, currentTimestamp, runsFetched, playersFetched)
	return err
}

// GetRealmID retrieves realm ID by slug, returns 0 if not found
func (ds *DatabaseService) GetRealmID(slug string) (int, error) {
    // Deprecated: prefer GetRealmIDByRegionAndSlug
    query := `SELECT id FROM realms WHERE slug = ?`

	var realmID int
	err := ds.db.QueryRow(query, slug).Scan(&realmID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return realmID, nil
}

// GetRealmIDByRegionAndSlug retrieves realm ID by composite (region, slug), returns 0 if not found
func (ds *DatabaseService) GetRealmIDByRegionAndSlug(region, slug string) (int, error) {
    query := `SELECT id FROM realms WHERE region = ? AND slug = ?`

    var realmID int
    err := ds.db.QueryRow(query, region, slug).Scan(&realmID)
    if err == sql.ErrNoRows {
        return 0, nil
    }
    if err != nil {
        return 0, err
    }
    return realmID, nil
}

// GetDungeonID retrieves dungeon ID by slug, returns 0 if not found
func (ds *DatabaseService) GetDungeonID(slug string) (int, error) {
	query := `SELECT id FROM dungeons WHERE slug = ?`

	var dungeonID int
	err := ds.db.QueryRow(query, slug).Scan(&dungeonID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return dungeonID, nil
}

// GetPlayerCurrentIdentity returns region, realm_slug, and name for a player from players/realms tables
func (ds *DatabaseService) GetPlayerCurrentIdentity(playerID int) (string, string, string, error) {
    const q = `
        SELECT r.region, r.slug, p.name
        FROM players p
        JOIN realms r ON p.realm_id = r.id
        WHERE p.id = ?
    `
    var region, slug, name string
    err := ds.db.QueryRow(q, playerID).Scan(&region, &slug, &name)
    if err == sql.ErrNoRows {
        return "", "", "", nil
    }
    return region, slug, name, err
}

// GetConnectedRealmSlugs returns all slugs in the same connected_realm_id as (region, realmSlug)
func (ds *DatabaseService) GetConnectedRealmSlugs(region, realmSlug string) ([]string, error) {
    var connID sql.NullInt64
    if err := ds.db.QueryRow(`SELECT connected_realm_id FROM realms WHERE region = ? AND slug = ?`, region, realmSlug).Scan(&connID); err != nil {
        if err == sql.ErrNoRows { return []string{}, nil }
        return nil, err
    }
    if !connID.Valid || connID.Int64 == 0 {
        return []string{}, nil
    }
    rows, err := ds.db.Query(`SELECT slug FROM realms WHERE region = ? AND connected_realm_id = ? ORDER BY slug`, region, connID.Int64)
    if err != nil { return nil, err }
    defer rows.Close()
    var slugs []string
    for rows.Next() {
        var s string
        if err := rows.Scan(&s); err != nil { return nil, err }
        slugs = append(slugs, s)
    }
    return slugs, nil
}

// GetLastRunRealmForPlayer returns the most recent run's region/realm_slug for a player
func (ds *DatabaseService) GetLastRunRealmForPlayer(playerID int) (string, string, int64, error) {
    const q = `
        SELECT rr.region, rr.slug, cr.completed_timestamp
        FROM run_members rm
        JOIN challenge_runs cr ON cr.id = rm.run_id
        JOIN realms rr ON rr.id = cr.realm_id
        WHERE rm.player_id = ?
        ORDER BY cr.completed_timestamp DESC
        LIMIT 1
    `
    var region, slug string
    var ts int64
    err := ds.db.QueryRow(q, playerID).Scan(&region, &slug, &ts)
    if err == sql.ErrNoRows { return "", "", 0, nil }
    return region, slug, ts, err
}

// UpdatePlayerIdentity updates players.name and players.realm_id to the given (region, slug)
func (ds *DatabaseService) UpdatePlayerIdentity(playerID int, name, region, realmSlug string) error {
    realmID, err := ds.GetRealmIDByRegionAndSlug(region, realmSlug)
    if err != nil { return err }
    if realmID == 0 { return sql.ErrNoRows }
    _, err = ds.db.Exec(`UPDATE players SET name = ?, name_lower = lower(?), realm_id = ? WHERE id = ?`, name, name, realmID, playerID)
    return err
}
