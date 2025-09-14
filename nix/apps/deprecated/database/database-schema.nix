{
  writers,
  python3Packages,
  ...
}: let
  schemaScript =
    writers.writePython3Bin "database-schema" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import sqlite3
      import sys
      import time

      # database path
      DB_PATH = "./web/public/database.sqlite3"

      def create_database_schema(cursor):
          """Create complete database schema for challenge mode leaderboards and player profiles"""
          print("Creating database schema...")

          # optimize database for HTTP range requests
          cursor.execute("PRAGMA page_size = 4096")
          cursor.execute("PRAGMA journal_mode = DELETE")

          # dungeons reference table
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS dungeons (
                  id INTEGER PRIMARY KEY,
                  slug TEXT UNIQUE NOT NULL,
                  name TEXT NOT NULL,
                  map_id INTEGER,
                  map_challenge_mode_id INTEGER UNIQUE
              )
          """)

          # realms reference table
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS realms (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  slug TEXT UNIQUE NOT NULL,
                  name TEXT NOT NULL,
                  region TEXT NOT NULL,
                  connected_realm_id INTEGER UNIQUE
              )
          """)

          # challenge mode runs (raw data from API)
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS challenge_runs (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  duration INTEGER NOT NULL,
                  completed_timestamp INTEGER NOT NULL,
                  keystone_level INTEGER DEFAULT 1,
                  dungeon_id INTEGER REFERENCES dungeons(id),
                  realm_id INTEGER REFERENCES realms(id),
                  period_id INTEGER,
                  period_start_timestamp INTEGER,
                  period_end_timestamp INTEGER,
                  team_signature TEXT NOT NULL,
                  UNIQUE(dungeon_id, realm_id, team_signature, completed_timestamp, duration)
              )
          """)

          # Players
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS players (
                  id INTEGER PRIMARY KEY,
                  name TEXT NOT NULL,
                  realm_id INTEGER REFERENCES realms(id),
                  UNIQUE(id, realm_id)
              )
          """)

          # Run members (links runs to players)
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS run_members (
                  run_id INTEGER REFERENCES challenge_runs(id),
                  player_id INTEGER REFERENCES players(id),
                  spec_id INTEGER,
                  faction TEXT,
                  PRIMARY KEY (run_id, player_id)
              )
          """)

          # Player aggregation tables
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_profiles (
                  player_id INTEGER PRIMARY KEY REFERENCES players(id),
                  name TEXT NOT NULL,
                  realm_id INTEGER REFERENCES realms(id),
                  main_spec_id INTEGER,
                  dungeons_completed INTEGER DEFAULT 0,
                  total_runs INTEGER DEFAULT 0,
                  combined_best_time INTEGER,
                  average_best_time REAL,
                  global_ranking INTEGER,
                  regional_ranking INTEGER,
                  realm_ranking INTEGER,
                  has_complete_coverage BOOLEAN DEFAULT FALSE,
                  last_updated INTEGER
              )
          """)

          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_best_runs (
                  player_id INTEGER REFERENCES players(id),
                  dungeon_id INTEGER REFERENCES dungeons(id),
                  run_id INTEGER REFERENCES challenge_runs(id),
                  duration INTEGER NOT NULL,
                  global_ranking INTEGER,
                  global_ranking_filtered INTEGER,
                  regional_ranking_filtered INTEGER,
                  realm_ranking_filtered INTEGER,
                  completed_timestamp INTEGER NOT NULL,
                  PRIMARY KEY (player_id, dungeon_id)
              )
          """)

          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_rankings (
                  player_id INTEGER REFERENCES players(id),
                  ranking_type TEXT NOT NULL,
                  ranking_scope TEXT NOT NULL,
                  ranking INTEGER NOT NULL,
                  combined_best_time INTEGER NOT NULL,
                  last_updated INTEGER,
                  PRIMARY KEY (player_id, ranking_type, ranking_scope)
              )
          """)

          # Extended player information from API
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_details (
                  player_id INTEGER PRIMARY KEY REFERENCES players(id),
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
              )
          """)

          # Equipment snapshots
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_equipment (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  player_id INTEGER REFERENCES players(id),
                  slot_type TEXT,
                  item_id INTEGER,
                  upgrade_id INTEGER,
                  quality TEXT,
                  item_name TEXT,
                  snapshot_timestamp INTEGER,
                  UNIQUE(player_id, slot_type, snapshot_timestamp)
              )
          """)

          # Equipment modifications (gems, enchants, tinkers)
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS player_equipment_enchantments (
                  id INTEGER PRIMARY KEY AUTOINCREMENT,
                  equipment_id INTEGER REFERENCES player_equipment(id),
                  enchantment_id INTEGER,
                  slot_id INTEGER,
                  slot_type TEXT,
                  display_string TEXT,
                  source_item_id INTEGER,
                  source_item_name TEXT,
                  spell_id INTEGER
              )
          """)

          # metadata table for tracking API fetch timestamps
          # computed rankings (separate from raw run data)
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS run_rankings (
                  run_id INTEGER REFERENCES challenge_runs(id),
                  dungeon_id INTEGER REFERENCES dungeons(id),
                  ranking_type TEXT NOT NULL,
                  ranking_scope TEXT NOT NULL,
                  ranking INTEGER NOT NULL,
                  computed_at INTEGER,
                  PRIMARY KEY (run_id, ranking_type, ranking_scope)
              )
          """)

          cursor.execute("""
              CREATE TABLE IF NOT EXISTS api_fetch_metadata (
                  id INTEGER PRIMARY KEY,
                  fetch_type TEXT UNIQUE NOT NULL,
                  last_fetch_timestamp INTEGER,
                  last_successful_fetch INTEGER,
                  runs_fetched INTEGER DEFAULT 0,
                  players_fetched INTEGER DEFAULT 0
              )
          """)

          # items database for icon lookups
          cursor.execute("""
              CREATE TABLE IF NOT EXISTS items (
                  id INTEGER PRIMARY KEY,
                  name TEXT,
                  icon TEXT,
                  quality INTEGER,
                  type INTEGER,
                  stats TEXT
              )
          """)

          # performance indexes
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_challenge_runs_dungeon ON challenge_runs(dungeon_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_challenge_runs_duration ON challenge_runs(duration)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_challenge_runs_realm ON challenge_runs(realm_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_challenge_runs_team_signature ON challenge_runs(team_signature)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_run_members_player ON run_members(player_id)")

          # covering indexes that filter by dungeon FIRST to minimize page reads
          # global filtered leaderboard by dungeon
          cursor.execute("""CREATE INDEX IF NOT EXISTS idx_global_filtered_by_dungeon
                           ON run_rankings(dungeon_id, ranking_type, ranking_scope, ranking, run_id)
                           WHERE ranking_type = 'global' AND ranking_scope = 'filtered'""")

          # global unfiltered leaderboard by dungeon
          cursor.execute("""CREATE INDEX IF NOT EXISTS idx_global_all_by_dungeon
                           ON run_rankings(dungeon_id, ranking_type, ranking_scope, ranking, run_id)
                           WHERE ranking_type = 'global' AND ranking_scope = 'all'""")

          # regional leaderboards by dungeon
          cursor.execute("""CREATE INDEX IF NOT EXISTS idx_regional_by_dungeon
                           ON run_rankings(dungeon_id, ranking_type, ranking_scope, ranking, run_id)
                           WHERE ranking_type = 'regional'""")

          # realm leaderboards by dungeon
          cursor.execute("""CREATE INDEX IF NOT EXISTS idx_realm_by_dungeon
                           ON run_rankings(dungeon_id, ranking_type, ranking_scope, ranking, run_id)
                           WHERE ranking_type = 'realm'""")

          # composite indexes for efficient JOINs
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_challenge_runs_composite ON challenge_runs(dungeon_id, duration, completed_timestamp, keystone_level, realm_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_realms_region_slug ON realms(region, slug, id, name)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_dungeons_lookup ON dungeons(id, name)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_profiles_combined_time ON player_profiles(combined_best_time)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_profiles_realm ON player_profiles(realm_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_best_runs_duration ON player_best_runs(duration)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_rankings_type_scope ON player_rankings(ranking_type, ranking_scope)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_equipment_player ON player_equipment(player_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_player_equipment_snapshot ON player_equipment(snapshot_timestamp)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_equipment_enchantments_equipment ON player_equipment_enchantments(equipment_id)")
          cursor.execute("CREATE INDEX IF NOT EXISTS idx_items_id ON items(id)")

          print("[OK] Complete database schema created with all tables and indexes")

      def main():
          print("=== Database Schema Creation Script ===")
          print(f"Database path: {os.path.abspath(DB_PATH)}")

          # create database connection
          os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              # create complete schema
              create_database_schema(cursor)
              conn.commit()

              # optimize database for HTTP range requests
              print("Optimizing database structure...")
              cursor.execute("VACUUM")
              print("Database optimized")

              print("\n[OK] Database schema setup complete!")
              print("Next steps:")
              print("  1. Run 'nix run .#getCM' to populate challenge mode data")
              print("  2. Run 'nix run .#rankingProcessor' to compute rankings")
              print("  3. Run 'nix run .#playerAggregation' to generate player leaderboards")
              print("  4. Run 'nix run .#populateItems' to add item icon data (optional)")
              print("  5. Run 'nix run .#playerProfiles' to fetch detailed player data (optional)")
              print("  6. Data is now ready for AstroDB/Turso deployment")

          except Exception as e:
              print(f"Schema creation failed: {e}")
              conn.rollback()
              sys.exit(1)
          finally:
              conn.close()

      if __name__ == "__main__":
          main()
    '';
in
  schemaScript
