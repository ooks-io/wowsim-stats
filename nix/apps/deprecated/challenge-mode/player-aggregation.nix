{
  writers,
  python3Packages,
  ...
}: let
  playerAggregationScript =
    writers.writePython3Bin "player-aggregation" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import sqlite3
      import time
      import sys

      DB_PATH = "./web/public/database.sqlite3"

      def create_player_aggregations(cursor):
          # efficiently compute player aggregations using SQL
          print("Computing player aggregations...")

          # clear existing aggregation data
          cursor.execute("DELETE FROM player_profiles")
          cursor.execute("DELETE FROM player_best_runs")
          cursor.execute("DELETE FROM player_rankings")
          print("Cleared existing player aggregation data")

          current_timestamp = int(time.time() * 1000)

          # step 1: find best run per player per dungeon
          print("Step 1: Computing best runs per player per dungeon...")
          cursor.execute("""
              INSERT INTO player_best_runs (player_id, dungeon_id, run_id, duration, completed_timestamp)
              SELECT
                  rm.player_id,
                  cr.dungeon_id,
                  cr.id as run_id,
                  cr.duration,
                  cr.completed_timestamp
              FROM run_members rm
              INNER JOIN challenge_runs cr ON rm.run_id = cr.id
              INNER JOIN (
                  SELECT
                      rm2.player_id,
                      cr2.dungeon_id,
                      MIN(cr2.duration) as best_duration
                  FROM run_members rm2
                  INNER JOIN challenge_runs cr2 ON rm2.run_id = cr2.id
                  GROUP BY rm2.player_id, cr2.dungeon_id
              ) best_times ON rm.player_id = best_times.player_id
                         AND cr.dungeon_id = best_times.dungeon_id
                         AND cr.duration = best_times.best_duration
              GROUP BY rm.player_id, cr.dungeon_id
              HAVING cr.id = MIN(cr.id)
          """)

          best_runs_count = cursor.rowcount
          print(f"Computed {best_runs_count} best runs")

          # step 2: copy rankings from run_rankings table
          print("Step 2: Copying rankings from run_rankings table...")
          cursor.execute("""
              UPDATE player_best_runs
              SET global_ranking = (
                  SELECT rr.ranking
                  FROM run_rankings rr
                  WHERE rr.run_id = player_best_runs.run_id
                  AND rr.ranking_type = 'global'
                  AND rr.ranking_scope = 'all'
              ),
              global_ranking_filtered = (
                  SELECT rr.ranking
                  FROM run_rankings rr
                  WHERE rr.run_id = player_best_runs.run_id
                  AND rr.ranking_type = 'global'
                  AND rr.ranking_scope = 'filtered'
              )
          """)
          print("Updated rankings from run_rankings table")

          # step 4: create player profiles with aggregated data
          print("Step 4: Creating player profiles...")
          cursor.execute("""
              INSERT INTO player_profiles (
                  player_id, name, realm_id, dungeons_completed, total_runs,
                  combined_best_time, average_best_time, has_complete_coverage, last_updated
              )
              SELECT
                  p.id as player_id,
                  p.name,
                  p.realm_id,
                  COUNT(pbr.dungeon_id) as dungeons_completed,
                  total_runs.run_count as total_runs,
                  COALESCE(SUM(pbr.duration), 0) as combined_best_time,
                  CASE
                      WHEN COUNT(pbr.dungeon_id) > 0
                      THEN CAST(SUM(pbr.duration) AS REAL) / COUNT(pbr.dungeon_id)
                      ELSE 0
                  END as average_best_time,
                  CASE WHEN COUNT(pbr.dungeon_id) = (SELECT COUNT(*) FROM dungeons) THEN 1 ELSE 0 END as has_complete_coverage,
                  ? as last_updated
              FROM players p
              LEFT JOIN player_best_runs pbr ON p.id = pbr.player_id
              INNER JOIN (
                  SELECT rm.player_id, COUNT(*) as run_count
                  FROM run_members rm
                  GROUP BY rm.player_id
              ) total_runs ON p.id = total_runs.player_id
              GROUP BY p.id, p.name, p.realm_id, total_runs.run_count
          """, (current_timestamp,))

          profiles_count = cursor.rowcount
          print(f"[OK] Created {profiles_count} player profiles")

          # step 4: determine main spec for each player based on best runs
          print("Step 4: Computing main specs...")
          cursor.execute("""
              UPDATE player_profiles
              SET main_spec_id = (
                  SELECT spec_counts.spec_id
                  FROM (
                      SELECT
                          rm.player_id,
                          rm.spec_id,
                          COUNT(*) as spec_count,
                          ROW_NUMBER() OVER (PARTITION BY rm.player_id ORDER BY COUNT(*) DESC, rm.spec_id ASC) as rank
                      FROM run_members rm
                      INNER JOIN player_best_runs pbr ON rm.run_id = pbr.run_id AND rm.player_id = pbr.player_id
                      WHERE rm.spec_id IS NOT NULL
                      GROUP BY rm.player_id, rm.spec_id
                  ) spec_counts
                  WHERE spec_counts.player_id = player_profiles.player_id AND spec_counts.rank = 1
              )
          """)
          print("updated main specs")

          return profiles_count


      def compute_player_rankings(cursor):
          # compute rankings for players with complete coverage"""
          print("Computing player rankings...")

          current_timestamp = int(time.time() * 1000)

          # get qualified players count
          cursor.execute("SELECT COUNT(*) FROM player_profiles WHERE has_complete_coverage = 1")
          qualified_count = cursor.fetchone()[0]
          print(f"Found {qualified_count} players with complete coverage")

          if qualified_count == 0:
              print("No qualified players found, skipping rankings")
              return 0

          # step 1: global rankings
          print("Computing global rankings...")
          cursor.execute("""
              UPDATE player_profiles
              SET global_ranking = (
                  SELECT ranking FROM (
                      SELECT
                          player_id,
                          ROW_NUMBER() OVER (ORDER BY combined_best_time ASC) as ranking
                      FROM player_profiles
                      WHERE has_complete_coverage = 1
                  ) global_ranks
                  WHERE global_ranks.player_id = player_profiles.player_id
              )
              WHERE has_complete_coverage = 1
          """)

          cursor.execute("""
              INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
              SELECT
                  player_id, 'best_overall', 'global', global_ranking, combined_best_time, ?
              FROM player_profiles
              WHERE has_complete_coverage = 1 AND global_ranking IS NOT NULL
          """, (current_timestamp,))

          # step 2: regional rankings
          print("Computing regional rankings...")
          cursor.execute("""
              UPDATE player_profiles
              SET regional_ranking = (
                  SELECT ranking FROM (
                      SELECT
                          pp.player_id,
                          ROW_NUMBER() OVER (PARTITION BY r.region ORDER BY pp.combined_best_time ASC) as ranking
                      FROM player_profiles pp
                      INNER JOIN realms r ON pp.realm_id = r.id
                      WHERE pp.has_complete_coverage = 1
                  ) regional_ranks
                  WHERE regional_ranks.player_id = player_profiles.player_id
              )
              WHERE has_complete_coverage = 1
          """)

          cursor.execute("""
              INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
              SELECT
                  pp.player_id, 'best_overall', r.region, pp.regional_ranking, pp.combined_best_time, ?
              FROM player_profiles pp
              INNER JOIN realms r ON pp.realm_id = r.id
              WHERE pp.has_complete_coverage = 1 AND pp.regional_ranking IS NOT NULL
          """, (current_timestamp,))

          # step 3: realm rankings
          print("Computing realm rankings...")
          cursor.execute("""
              UPDATE player_profiles
              SET realm_ranking = (
                  SELECT ranking FROM (
                      SELECT
                          player_id,
                          ROW_NUMBER() OVER (PARTITION BY realm_id ORDER BY combined_best_time ASC) as ranking
                      FROM player_profiles
                      WHERE has_complete_coverage = 1
                  ) realm_ranks
                  WHERE realm_ranks.player_id = player_profiles.player_id
              )
              WHERE has_complete_coverage = 1
          """)

          cursor.execute("""
              INSERT INTO player_rankings (player_id, ranking_type, ranking_scope, ranking, combined_best_time, last_updated)
              SELECT
                  player_id, 'best_overall', CAST(realm_id AS TEXT), realm_ranking, combined_best_time, ?
              FROM player_profiles
              WHERE has_complete_coverage = 1 AND realm_ranking IS NOT NULL
          """, (current_timestamp,))

          print(f"[OK] Computed rankings for {qualified_count} qualified players")
          return qualified_count

      def main():
          if not os.path.exists(DB_PATH):
              print(f"FATAL: Database not found at {DB_PATH}")
              print("Please run the challenge-mode-leaderboard script first.")
              sys.exit(1)

          print("=== Player Aggregation Script ===")
          print(f"Database: {os.path.abspath(DB_PATH)}")

          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              # check if we have data
              cursor.execute("SELECT COUNT(*) FROM challenge_runs")
              run_count = cursor.fetchone()[0]
              cursor.execute("SELECT COUNT(*) FROM players")
              player_count = cursor.fetchone()[0]

              print(f"Found {run_count} runs and {player_count} players in database")

              if run_count == 0:
                  print("No runs found in database. Please run challenge-mode-leaderboard first.")
                  sys.exit(1)

              # check if rankings have been computed
              cursor.execute("SELECT COUNT(*) FROM run_rankings")
              ranking_count = cursor.fetchone()[0]

              if ranking_count == 0:
                  print("No rankings found. Please run ranking-processor first.")
                  sys.exit(1)

              print(f"Found {ranking_count} pre-computed rankings")

              # compute aggregations
              profiles_created = create_player_aggregations(cursor)
              qualified_players = compute_player_rankings(cursor)

              conn.commit()

              # optimize database after aggregation
              print("Optimizing database structure...")
              cursor.execute("VACUUM")
              print("Database optimized")

              print(f"\nPlayer aggregation complete!")
              print(f"  Created {profiles_created} player profiles")
              print(f"  Computed rankings for {qualified_players} qualified players")
              print("Next steps:")
              print("  1. Run 'nix run .#playerProcessor' to fetch player details (requires BLIZZARD_API_TOKEN)")
              print("  2. Run 'nix run .#updateDatabaseChunks' to regenerate chunks for frontend")

          except Exception as e:
              print(f"Player aggregation failed: {e}")
              import traceback
              traceback.print_exc()
              conn.rollback()
              sys.exit(1)
          finally:
              conn.close()

      if __name__ == "__main__":
          main()
    '';
in
  playerAggregationScript
