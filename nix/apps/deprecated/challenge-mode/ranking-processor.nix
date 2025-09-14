{
  writers,
  python3Packages,
  ...
}: let
  rankingProcessorScript =
    writers.writePython3Bin "ranking-processor" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import sqlite3
      import time
      import sys

      DB_PATH = "./web/public/database.sqlite3"

      def deduplicate_runs(cursor):
          # remove duplicate runs, keeping the one with earliest completed_timestamp
          print("Deduplicating runs by team signature...")

          # find duplicates
          cursor.execute("""
              SELECT team_signature, dungeon_id, duration, COUNT(*) as count
              FROM challenge_runs
              GROUP BY team_signature, dungeon_id, duration
              HAVING COUNT(*) > 1
          """)

          duplicates = cursor.fetchall()
          print(f"Found {len(duplicates)} duplicate team+dungeon+duration combinations")

          total_removed = 0
          for team_sig, dungeon_id, duration, count in duplicates:
              # keep the run with earliest completed_timestamp, delete others
              cursor.execute("""
                  DELETE FROM challenge_runs
                  WHERE team_signature = ? AND dungeon_id = ? AND duration = ?
                  AND id NOT IN (
                      SELECT id FROM challenge_runs
                      WHERE team_signature = ? AND dungeon_id = ? AND duration = ?
                      ORDER BY completed_timestamp ASC
                      LIMIT 1
                  )
              """, (team_sig, dungeon_id, duration, team_sig, dungeon_id, duration))

              removed = cursor.rowcount
              total_removed += removed
              if removed > 0:
                  print(f"  Removed {removed} duplicate runs for team {team_sig[:20]}... in dungeon {dungeon_id}")

          print(f"Removed {total_removed} duplicate runs")
          return total_removed

      def compute_global_rankings(cursor):
          # compute global rankings for all runs"""
          print("Computing global rankings...")

          # clear existing global rankings
          cursor.execute("DELETE FROM run_rankings WHERE ranking_type = 'global'")

          # unfiltered global rankings
          cursor.execute("""
              INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
              SELECT
                  id as run_id,
                  dungeon_id,
                  'global' as ranking_type,
                  'all' as ranking_scope,
                  ROW_NUMBER() OVER (PARTITION BY dungeon_id ORDER BY duration ASC, completed_timestamp ASC) as ranking,
                  ? as computed_at
              FROM challenge_runs
          """, (int(time.time() * 1000),))

          # filtered global rankings (best time per team)
          cursor.execute("SELECT id FROM dungeons")
          dungeon_ids = [row[0] for row in cursor.fetchall()]

          for dungeon_id in dungeon_ids:
              cursor.execute("""
                  WITH best_team_runs AS (
                      SELECT
                          team_signature,
                          MIN(duration) as best_duration
                      FROM challenge_runs
                      WHERE dungeon_id = ?
                      GROUP BY team_signature
                  ),
                  filtered_runs AS (
                      SELECT
                          cr.id as run_id,
                          cr.duration,
                          cr.completed_timestamp,
                          ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
                      FROM challenge_runs cr
                      INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
                                                   AND cr.duration = btr.best_duration
                      WHERE cr.dungeon_id = ?
                      GROUP BY cr.team_signature
                      HAVING cr.id = MIN(cr.id)
                  )
                  INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
                  SELECT
                      run_id,
                      ? as dungeon_id,
                      'global' as ranking_type,
                      'filtered' as ranking_scope,
                      filtered_rank as ranking,
                      ? as computed_at
                  FROM filtered_runs
              """, (dungeon_id, dungeon_id, dungeon_id, int(time.time() * 1000)))

          print("[OK] Computed global rankings (all and filtered)")

      def compute_regional_rankings(cursor):
          # compute regional rankings for all runs
          print("Computing regional rankings...")

          # clear existing regional rankings
          cursor.execute("DELETE FROM run_rankings WHERE ranking_type = 'regional'")

          cursor.execute("SELECT DISTINCT region FROM realms")
          regions = [row[0] for row in cursor.fetchall()]

          for region in regions:
              # unfiltered regional rankings
              cursor.execute("""
                  INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
                  SELECT
                      cr.id as run_id,
                      cr.dungeon_id,
                      'regional' as ranking_type,
                      ? as ranking_scope,
                      ROW_NUMBER() OVER (PARTITION BY cr.dungeon_id ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as ranking,
                      ? as computed_at
                  FROM challenge_runs cr
                  INNER JOIN realms r ON cr.realm_id = r.id
                  WHERE r.region = ?
              """, (region, int(time.time() * 1000), region))

              # filtered regional rankings
              cursor.execute("SELECT id FROM dungeons")
              dungeon_ids = [row[0] for row in cursor.fetchall()]

              for dungeon_id in dungeon_ids:
                  cursor.execute("""
                      WITH best_team_runs AS (
                          SELECT
                              cr.team_signature,
                              MIN(cr.duration) as best_duration
                          FROM challenge_runs cr
                          INNER JOIN realms r ON cr.realm_id = r.id
                          WHERE cr.dungeon_id = ? AND r.region = ?
                          GROUP BY cr.team_signature
                      ),
                      filtered_runs AS (
                          SELECT
                              cr.id as run_id,
                              cr.duration,
                              cr.completed_timestamp,
                              ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
                          FROM challenge_runs cr
                          INNER JOIN realms r ON cr.realm_id = r.id
                          INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
                                                       AND cr.duration = btr.best_duration
                          WHERE cr.dungeon_id = ? AND r.region = ?
                          GROUP BY cr.team_signature
                          HAVING cr.id = MIN(cr.id)
                      )
                      INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
                      SELECT
                          run_id,
                          ? as dungeon_id,
                          'regional' as ranking_type,
                          ? as ranking_scope,
                          filtered_rank as ranking,
                          ? as computed_at
                      FROM filtered_runs
                  """, (dungeon_id, region, dungeon_id, region, dungeon_id, f"{region}_filtered", int(time.time() * 1000)))

          print(f"[OK] Computed regional rankings for {len(regions)} regions")

      def compute_realm_rankings(cursor):
          # compute realm rankings for all runs
          print("Computing realm rankings...")

          # clear existing realm rankings
          cursor.execute("DELETE FROM run_rankings WHERE ranking_type = 'realm'")

          cursor.execute("SELECT id FROM realms")
          realm_ids = [row[0] for row in cursor.fetchall()]

          for realm_id in realm_ids:
              # unfiltered realm rankings
              cursor.execute("""
                  INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
                  SELECT
                      id as run_id,
                      dungeon_id,
                      'realm' as ranking_type,
                      ? as ranking_scope,
                      ROW_NUMBER() OVER (PARTITION BY dungeon_id ORDER BY duration ASC, completed_timestamp ASC) as ranking,
                      ? as computed_at
                  FROM challenge_runs
                  WHERE realm_id = ?
              """, (str(realm_id), int(time.time() * 1000), realm_id))

              # filtered realm rankings
              cursor.execute("SELECT id FROM dungeons")
              dungeon_ids = [row[0] for row in cursor.fetchall()]

              for dungeon_id in dungeon_ids:
                  cursor.execute("""
                      WITH best_team_runs AS (
                          SELECT
                              team_signature,
                              MIN(duration) as best_duration
                          FROM challenge_runs
                          WHERE dungeon_id = ? AND realm_id = ?
                          GROUP BY team_signature
                      ),
                      filtered_runs AS (
                          SELECT
                              cr.id as run_id,
                              cr.duration,
                              cr.completed_timestamp,
                              ROW_NUMBER() OVER (ORDER BY cr.duration ASC, cr.completed_timestamp ASC) as filtered_rank
                          FROM challenge_runs cr
                          INNER JOIN best_team_runs btr ON cr.team_signature = btr.team_signature
                                                       AND cr.duration = btr.best_duration
                          WHERE cr.dungeon_id = ? AND cr.realm_id = ?
                          GROUP BY cr.team_signature
                          HAVING cr.id = MIN(cr.id)
                      )
                      INSERT INTO run_rankings (run_id, dungeon_id, ranking_type, ranking_scope, ranking, computed_at)
                      SELECT
                          run_id,
                          ? as dungeon_id,
                          'realm' as ranking_type,
                          ? as ranking_scope,
                          filtered_rank as ranking,
                          ? as computed_at
                      FROM filtered_runs
                  """, (dungeon_id, realm_id, dungeon_id, realm_id, dungeon_id, f"{realm_id}_filtered", int(time.time() * 1000)))

          print(f"Computed realm rankings for {len(realm_ids)} realms")

      def main():
          if not os.path.exists(DB_PATH):
              print(f"FATAL: Database not found at {DB_PATH}")
              print("Please run the challenge-mode-leaderboard script first.")
              sys.exit(1)

          print("=== Ranking Processor Script ===")
          print(f"Database: {os.path.abspath(DB_PATH)}")

          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              # check if we have data
              cursor.execute("SELECT COUNT(*) FROM challenge_runs")
              run_count = cursor.fetchone()[0]

              print(f"Found {run_count} runs in database")

              if run_count == 0:
                  print("No runs found in database. Please run challenge-mode-leaderboard first.")
                  sys.exit(1)

              # step 1: deduplicate runs
              removed_count = deduplicate_runs(cursor)

              # step 2: compute all rankings
              compute_global_rankings(cursor)
              compute_regional_rankings(cursor)
              compute_realm_rankings(cursor)

              # check final counts
              cursor.execute("SELECT COUNT(*) FROM challenge_runs")
              final_runs = cursor.fetchone()[0]
              cursor.execute("SELECT COUNT(*) FROM run_rankings")
              ranking_count = cursor.fetchone()[0]

              conn.commit()

              # optimize database after ranking computation
              print("Optimizing database structure...")
              cursor.execute("VACUUM")
              print("Database optimized")

              print(f"\nRanking processing complete!")
              print(f"  Final runs: {final_runs} (removed {removed_count} duplicates)")
              print(f"  Rankings computed: {ranking_count}")
              print("Next steps:")
              print("  1. Run 'nix run .#playerAggregation' to compute player leaderboards")
              print("  2. Run 'nix run .#updateDatabaseChunks' to regenerate chunks for frontend")

          except Exception as e:
              print(f"Ranking processing failed: {e}")
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
  rankingProcessorScript
