{
  writers,
  api,
  python3Packages,
  ...
}: let
  fetcherScript =
    writers.writePython3Bin "cm-leaderboard-fetcher" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import requests
      import json
      import re
      import sys
      import time
      import sqlite3

      ALL_REALMS = ${builtins.toJSON api.realm}

      # database path
      DB_PATH = "./web/public/database.sqlite3"
      # TODO: fix me
      API_TOKEN = os.getenv("BLIZZARD_API_TOKEN")

      def slugify(text):
          # converts a string to a url friendly slug
          text = text.lower()
          text = re.sub(r'[\s\'\W]+', '-', text)
          return text.strip('-')


      def get_hardcoded_period_and_dungeons():
          # hardcoded values since the index endpoint is broken
          # all records are now in period 1025
          period_id = "1025"

          dungeons = [
              {"id": 2, "name": "Temple of the Jade Serpent", "slug": "temple-of-the-jade-serpent"},
              {"id": 56, "name": "Stormstout Brewery", "slug": "stormstout-brewery"},
              {"id": 57, "name": "Gate of the Setting Sun", "slug": "gate-of-the-setting-sun"},
              {"id": 58, "name": "Shado-Pan Monastery", "slug": "shado-pan-monastery"},
              {"id": 59, "name": "Siege of Niuzao Temple", "slug": "siege-of-niuzao-temple"},
              {"id": 60, "name": "Mogu'shan Palace", "slug": "mogu-shan-palace"},
              {"id": 76, "name": "Scholomance", "slug": "scholomance"},
              {"id": 77, "name": "Scarlet Halls", "slug": "scarlet-halls"},
              {"id": 78, "name": "Scarlet Monastery", "slug": "scarlet-monastery"}
          ]

          print(f"  Using hardcoded Period: {period_id}, Dungeons: {len(dungeons)}")
          return period_id, dungeons


      def get_leaderboard_data(realm_info, dungeon, period_id, session):
          # fetch a specific leaderboard.
          region, realm_id = realm_info['region'], realm_info['id']
          namespace = f"dynamic-classic-{region}"
          url = (
              f"https://{region}.api.blizzard.com/data/wow/connected-realm/"
              f"{realm_id}/mythic-leaderboard/{dungeon['id']}/period/"
              f"{period_id}?namespace={namespace}"
          )

          try:
              time.sleep(0.05)
              response = session.get(url, timeout=15)
              response.raise_for_status()
              return response.json()
          except requests.exceptions.RequestException as e:
              print(f"    ERROR fetching for {dungeon['name']}: {e}", file=sys.stderr)
              return None


      def ensure_reference_data(cursor, realm_info, dungeons):
          # insert realm and dungeon reference data"""
          # insert realm
          cursor.execute("""
              INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id)
              VALUES (?, ?, ?, ?)
          """, (realm_info['slug'], realm_info['name'], realm_info['region'], realm_info['id']))

          # insert dungeons
          for dungeon in dungeons:
              cursor.execute("""
                  INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id)
                  VALUES (?, ?, ?, ?)
              """, (dungeon['id'], dungeon['slug'], dungeon['name'], dungeon['id']))

      def compute_team_signature(members_list):
          # compute team signature from member list (sorted player IDs)
          player_ids = []
          for member in members_list:
              if "profile" in member:
                  player_id = member["profile"]["id"]
              else:
                  player_id = member.get("id")
              if player_id:
                  player_ids.append(str(player_id))
          return ",".join(sorted(player_ids))

      def get_primary_realm_id(cursor, members_list):
          # get the primary realm ID (lowest realm_id among team members)
          realm_ids = []
          for member in members_list:
              if "profile" in member:
                  realm_slug = member["profile"]["realm"]["slug"]
              else:
                  realm_slug = member.get("realm_slug")

              if realm_slug:
                  cursor.execute("SELECT id FROM realms WHERE slug = ?", (realm_slug,))
                  result = cursor.fetchone()
                  if result:
                      realm_ids.append(result[0])

          return min(realm_ids) if realm_ids else None

      def insert_leaderboard_data(cursor, leaderboard_data, realm_info, dungeon):
          # insert leaderboard data directly into database
          if not leaderboard_data:
              return 0, 0

          # get realm and dungeon ids
          cursor.execute("SELECT id FROM realms WHERE slug = ?", (realm_info['slug'],))
          realm_id = cursor.fetchone()[0]

          cursor.execute("SELECT id FROM dungeons WHERE slug = ?", (dungeon['slug'],))
          dungeon_id = cursor.fetchone()[0]

          runs = leaderboard_data.get('leading_groups', [])
          period_id = leaderboard_data.get('period')
          period_start = leaderboard_data.get('period_start_timestamp')
          period_end = leaderboard_data.get('period_end_timestamp')

          runs_inserted = 0
          players_inserted = 0
          runs_skipped = 0

          for run in runs:
              members_list = run.get('members', [])
              team_signature = compute_team_signature(members_list)

              if not team_signature:
                  print(f"      WARNING: Skipping run with no valid team signature")
                  continue

              # insert run with team signature (prevent duplicates)
              cursor.execute("""
                  INSERT OR IGNORE INTO challenge_runs
                  (duration, completed_timestamp, keystone_level, dungeon_id, realm_id, period_id, period_start_timestamp, period_end_timestamp, team_signature)
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
              """, (
                  run.get('duration'),
                  run.get('completed_timestamp'),
                  run.get('keystone_level', 1),
                  dungeon_id,
                  realm_id,
                  period_id,
                  period_start,
                  period_end,
                  team_signature
              ))

              run_id = cursor.lastrowid
              runs_inserted += 1

              # insert team members only for new runs
              members_list = run.get('members', [])

              for i, member in enumerate(members_list):
                  # handle both old and new member format
                  if "profile" in member:
                      # old format with nested profile data
                      player_id = member["profile"]["id"]
                      player_name = member["profile"]["name"]
                      player_realm_slug = member["profile"]["realm"]["slug"]
                      spec_id = member.get("specialization", {}).get("id")
                      faction = member.get("faction", {}).get("type")
                  else:
                      # new optimized format
                      player_id = member.get("id")
                      player_name = member.get("name")
                      player_realm_slug = member.get("realm_slug")
                      spec_id = member.get("spec_id")
                      faction = member.get("faction")

                  if not player_id:
                      print(f"        WARNING: Skipping member {i+1} - missing player ID")
                      continue

                  # get player's realm ID, insert if missing (for cross-realm players)
                  cursor.execute("SELECT id FROM realms WHERE slug = ?", (player_realm_slug,))
                  player_realm_result = cursor.fetchone()
                  if not player_realm_result:
                      # find realm info from ALL_REALMS or create placeholder
                      player_realm_info = None
                      for slug, info in ALL_REALMS.items():
                          if slug == player_realm_slug:
                              player_realm_info = info
                              break

                      if player_realm_info:
                          # insert the missing realm
                          cursor.execute("""
                              INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id)
                              VALUES (?, ?, ?, ?)
                          """, (player_realm_slug, player_realm_info['name'], player_realm_info['region'], player_realm_info['id']))
                          player_realm_id = cursor.lastrowid
                          if player_realm_id == 0:  # Already existed
                              cursor.execute("SELECT id FROM realms WHERE slug = ?", (player_realm_slug,))
                              player_realm_id = cursor.fetchone()[0]
                      else:
                          # unknown realm - create placeholder
                          cursor.execute("""
                              INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id)
                              VALUES (?, ?, ?, ?)
                          """, (player_realm_slug, player_realm_slug.title(), 'unknown', 0))
                          player_realm_id = cursor.lastrowid
                          if player_realm_id == 0:  # Already existed
                              cursor.execute("SELECT id FROM realms WHERE slug = ?", (player_realm_slug,))
                              player_realm_id = cursor.fetchone()[0]
                  else:
                      player_realm_id = player_realm_result[0]

                  # insert player
                  cursor.execute("""
                      INSERT OR IGNORE INTO players (id, name, realm_id)
                      VALUES (?, ?, ?)
                  """, (player_id, player_name, player_realm_id))

                  if cursor.rowcount > 0:
                      players_inserted += 1

                  # link player to run
                  cursor.execute("""
                      INSERT INTO run_members (run_id, player_id, spec_id, faction)
                      VALUES (?, ?, ?, ?)
                  """, (run_id, player_id, spec_id, faction))

          print(f"    Inserted {runs_inserted} runs and {players_inserted} new players")

          return runs_inserted, players_inserted



      def update_fetch_metadata(cursor, fetch_type, runs_fetched, players_fetched):
          # update api fetch metadata with current run statistics
          current_timestamp = int(time.time() * 1000)
          cursor.execute("""
              INSERT OR REPLACE INTO api_fetch_metadata
              (fetch_type, last_fetch_timestamp, last_successful_fetch, runs_fetched, players_fetched)
              VALUES (?, ?, ?, ?, ?)
          """, (fetch_type, current_timestamp, current_timestamp, runs_fetched, players_fetched))

      def get_last_fetch_info(cursor, fetch_type):
          # get information about the last successful fetch"""
          cursor.execute("""
              SELECT last_successful_fetch, runs_fetched, players_fetched
              FROM api_fetch_metadata
              WHERE fetch_type = ?
          """, (fetch_type,))
          result = cursor.fetchone()
          return result if result else (None, 0, 0)

      def main():
          if not API_TOKEN:
              print(
                  "FATAL: BLIZZARD_API_TOKEN environment variable not set.",
                  file=sys.stderr
              )
              sys.exit(1)

          print("Starting leaderboard fetch to SQLite database...")
          print(f"Database path: {os.path.abspath(DB_PATH)}")

          # create database connection
          os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)
          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              # verify database schema exists
              cursor.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='challenge_runs'")
              if not cursor.fetchone():
                  print("FATAL: Database schema not found.")
                  print("Please run 'nix run .#databaseSchema' first to create the database schema.")
                  sys.exit(1)

              # check last fetch info
              last_fetch, prev_runs, prev_players = get_last_fetch_info(cursor, "challenge_mode_leaderboard")
              if last_fetch:
                  print(f"Last fetch: {time.strftime('%Y-%m-%d %H:%M:%S UTC', time.gmtime(last_fetch/1000))}")
                  print(f"Previous fetch: {prev_runs} runs, {prev_players} players")
              else:
                  print("First time running - full database population")

              # get dungeons for reference data
              period_id, dungeons = get_hardcoded_period_and_dungeons()

              # setup API session
              session = requests.Session()
              session.headers.update({"Authorization": f"Bearer {API_TOKEN}"})

              total_runs = 0
              total_players = 0

              for realm_slug, realm_info in ALL_REALMS.items():
                  print(
                      f"\nProcessing Realm: {realm_info['name']} "
                      f"({realm_info['region'].upper()})"
                  )

                  # ensure reference data exists
                  realm_info['slug'] = realm_slug  # add slug to realm_info
                  ensure_reference_data(cursor, realm_info, dungeons)

                  realm_runs = 0
                  realm_players = 0

                  for dungeon in dungeons:
                      print(f"  - Fetching dungeon: {dungeon['name']}")
                      leaderboard = get_leaderboard_data(
                          realm_info, dungeon, period_id, session
                      )
                      if not leaderboard:
                          continue

                      # insert directly into database with improved tracking
                      runs_inserted, players_inserted = insert_leaderboard_data(cursor, leaderboard, realm_info, dungeon)
                      realm_runs += runs_inserted
                      realm_players += players_inserted

                  total_runs += realm_runs
                  total_players += realm_players

                  if realm_runs > 0:
                      print(f"  ✓ Realm totals: {realm_runs} new runs, {realm_players} new players")
                  else:
                      print(f"  → No new data for this realm")

              # update fetch metadata
              update_fetch_metadata(cursor, "challenge_mode_leaderboard", total_runs, total_players)

              conn.commit()

              # optimize database for HTTP range requests
              print("Optimizing database structure...")
              cursor.execute("VACUUM")
              print("Database optimized")

              print(f"\nSuccessfully inserted {total_runs} runs and {total_players} new players into database")
              print("Next steps:")
              print("  1. Run 'nix run .#rankingProcessor' to compute rankings and deduplicate")
              print("  2. Run 'nix run .#playerAggregation' to compute player leaderboards")
              print("  3. Run 'nix run .#playerProcessor' to fetch player details (optional)")
              print("  4. Run 'nix run .#updateDatabaseChunks' to regenerate chunks for frontend")

          except Exception as e:
              print(f"Database operation failed: {e}")
              conn.rollback()
              sys.exit(1)
          finally:
              conn.close()


      if __name__ == "__main__":
          main()
    '';
in
  fetcherScript
