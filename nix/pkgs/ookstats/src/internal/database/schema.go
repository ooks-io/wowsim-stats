package database

import (
	"database/sql"
	"fmt"
)

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
