{
  writers,
  python3Packages,
  ...
}: let
  playerProfileScript =
    writers.writePython3Bin "player-profiles" {
      libraries = [python3Packages.requests python3Packages.aiohttp];
      doCheck = false;
    }
    ''
      import os
      import aiohttp
      import asyncio
      import json
      import sqlite3
      import time
      import sys

      DB_PATH = "./web/public/database.sqlite3"
      API_TOKEN = os.getenv("BLIZZARD_API_TOKEN")


      def get_eligible_players(cursor):
          """Get players with complete coverage (9/9 dungeons)"""
          cursor.execute("""
              SELECT p.id, p.name, r.slug as realm_slug, r.region
              FROM players p
              JOIN player_profiles pp ON p.id = pp.player_id
              JOIN realms r ON p.realm_id = r.id
              WHERE pp.has_complete_coverage = 1
              ORDER BY pp.global_ranking
          """)

          players = cursor.fetchall()
          print(f"Found {len(players)} eligible players with 9/9 completion")
          return players

      async def fetch_player_data(session, player_id, player_name, realm_slug, region):
          """Fetch player summary, equipment, and avatar in parallel"""
          base_url = f"https://{region}.api.blizzard.com/profile/wow/character/{realm_slug}/{player_name.lower()}"
          namespace = f"profile-classic-{region}"

          headers = {"Authorization": f"Bearer {API_TOKEN}"}

          summary_url = f"{base_url}?namespace={namespace}&locale=en_US"
          equipment_url = f"{base_url}/equipment?namespace={namespace}&locale=en_US"
          media_url = f"{base_url}/character-media?namespace={namespace}&locale=en_US"

          try:
              async with session.get(summary_url, headers=headers, timeout=15) as summary_response, \
                         session.get(equipment_url, headers=headers, timeout=15) as equipment_response, \
                         session.get(media_url, headers=headers, timeout=15) as media_response:

                  summary_data = None
                  equipment_data = None
                  media_data = None

                  if summary_response.status == 200:
                      summary_data = await summary_response.json()
                  elif summary_response.status != 404:
                      print(f"    WARNING: Summary fetch failed for {player_name}: HTTP {summary_response.status}")

                  if equipment_response.status == 200:
                      equipment_data = await equipment_response.json()
                  elif equipment_response.status != 404:
                      print(f"    WARNING: Equipment fetch failed for {player_name}: HTTP {equipment_response.status}")

                  if media_response.status == 200:
                      media_data = await media_response.json()
                  elif media_response.status != 404:
                      print(f"    WARNING: Media fetch failed for {player_name}: HTTP {media_response.status}")

                  return player_id, summary_data, equipment_data, media_data

          except Exception as e:
              print(f"    ERROR fetching data for {player_name}: {e}")
              return player_id, None, None, None

      async def fetch_players_batch(players_batch, semaphore):
          """Fetch player data in parallel with concurrency control"""
          async with aiohttp.ClientSession(
              connector=aiohttp.TCPConnector(limit=50),
              timeout=aiohttp.ClientTimeout(total=30)
          ) as session:
              tasks = []
              for player_id, player_name, realm_slug, region in players_batch:
                  async with semaphore:  # Control concurrency
                      task = fetch_player_data(session, player_id, player_name, realm_slug, region)
                      tasks.append(task)

              return await asyncio.gather(*tasks, return_exceptions=True)

      def insert_player_summary(cursor, player_id, summary_data, media_data, timestamp):
          """Insert player summary data with avatar URL"""
          if not summary_data:
              return

          # Extract avatar URL from media data
          avatar_url = None
          if media_data and 'assets' in media_data:
              for asset in media_data['assets']:
                  if asset.get('key') == 'avatar':
                      avatar_url = asset.get('value')
                      break

          cursor.execute("""
              INSERT OR REPLACE INTO player_details (
                  player_id, race_id, race_name, gender, class_id, class_name,
                  active_spec_id, active_spec_name, guild_name, level,
                  average_item_level, equipped_item_level, avatar_url, last_login_timestamp, last_updated
              ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
          """, (
              player_id,
              summary_data.get('race', {}).get('id'),
              summary_data.get('race', {}).get('name'),
              summary_data.get('gender', {}).get('type'),
              summary_data.get('character_class', {}).get('id'),
              summary_data.get('character_class', {}).get('name'),
              summary_data.get('active_spec', {}).get('id'),
              summary_data.get('active_spec', {}).get('name'),
              summary_data.get('guild', {}).get('name'),
              summary_data.get('level'),
              summary_data.get('average_item_level'),
              summary_data.get('equipped_item_level'),
              avatar_url,
              summary_data.get('last_login_timestamp'),
              timestamp
          ))

      def insert_player_equipment(cursor, player_id, equipment_data, timestamp):
          """Insert player equipment data"""
          if not equipment_data or 'equipped_items' not in equipment_data:
              return

          equipment_ids = []

          for item in equipment_data['equipped_items']:
              # Insert equipment item
              cursor.execute("""
                  INSERT INTO player_equipment (
                      player_id, slot_type, item_id, upgrade_id, quality, item_name, snapshot_timestamp
                  ) VALUES (?, ?, ?, ?, ?, ?, ?)
              """, (
                  player_id,
                  item.get('slot', {}).get('type'),
                  item.get('item', {}).get('id'),
                  item.get('upgrade_id'),
                  item.get('quality', {}).get('type'),
                  item.get('name'),
                  timestamp
              ))

              equipment_id = cursor.lastrowid
              equipment_ids.append(equipment_id)

              # Insert enchantments (gems, enchants, tinkers)
              for enchant in item.get('enchantments', []):
                  cursor.execute("""
                      INSERT INTO player_equipment_enchantments (
                          equipment_id, enchantment_id, slot_id, slot_type,
                          display_string, source_item_id, source_item_name, spell_id
                      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                  """, (
                      equipment_id,
                      enchant.get('enchantment_id'),
                      enchant.get('enchantment_slot', {}).get('id'),
                      enchant.get('enchantment_slot', {}).get('type'),
                      enchant.get('display_string'),
                      enchant.get('source_item', {}).get('id'),
                      enchant.get('source_item', {}).get('name'),
                      enchant.get('spell', {}).get('spell', {}).get('id')
                  ))

          return len(equipment_ids)

      async def process_players_async(players):
          """Process players with parallel API calls"""
          timestamp = int(time.time() * 1000)

          # Concurrency control - 20 concurrent requests (well under 100/sec limit)
          semaphore = asyncio.Semaphore(20)
          batch_size = 33  # Process 33 players at a time

          profiles_updated = 0
          equipment_updated = 0

          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              for i in range(0, len(players), batch_size):
                  batch = players[i:i + batch_size]
                  print(f"\nProcessing batch {i//batch_size + 1}/{(len(players) + batch_size - 1)//batch_size} ({len(batch)} players)")

                  # Fetch all players in batch concurrently
                  results = await fetch_players_batch(batch, semaphore)

                  # Process results and update database
                  for result in results:
                      if isinstance(result, Exception):
                          print(f"    ERROR in batch: {result}")
                          continue

                      player_id, summary_data, equipment_data, media_data = result
                      player_name = next(p[1] for p in batch if p[0] == player_id)

                      if summary_data:
                          insert_player_summary(cursor, player_id, summary_data, media_data, timestamp)
                          profiles_updated += 1
                          avatar_status = " (with avatar)" if media_data else ""
                          print(f"  [OK] {player_name}: Summary updated{avatar_status}")

                      if equipment_data:
                          items_count = insert_player_equipment(cursor, player_id, equipment_data, timestamp)
                          equipment_updated += items_count
                          print(f"  [OK] {player_name}: Equipment updated ({items_count} items)")

                  # Commit after each batch
                  conn.commit()
                  print(f"  -> Batch saved ({profiles_updated}/{len(players)} profiles, {equipment_updated} items)")

                  # Small delay between batches to be respectful
                  await asyncio.sleep(0.5)

              print(f"\n[OK] Player profile fetching complete!")
              print(f"  Updated {profiles_updated} player profiles")
              print(f"  Updated {equipment_updated} equipment items")
              return profiles_updated, equipment_updated

          finally:
              conn.close()

      def main():
          if not API_TOKEN:
              print("FATAL: BLIZZARD_API_TOKEN environment variable not set.", file=sys.stderr)
              sys.exit(1)

          if not os.path.exists(DB_PATH):
              print(f"FATAL: Database not found at {DB_PATH}")
              print("Please run the challenge-mode-leaderboard and player-aggregation scripts first.")
              sys.exit(1)

          print("=== Player Profile Fetcher Script (Parallel) ===")
          print(f"Database: {os.path.abspath(DB_PATH)}")

          conn = sqlite3.connect(DB_PATH)
          cursor = conn.cursor()

          try:
              # verify database schema exists
              cursor.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='player_details'")
              if not cursor.fetchone():
                  print("FATAL: Database schema not found.")
                  print("Please run 'nix run .#databaseSchema' first to create the database schema.")
                  sys.exit(1)

              # Get eligible players (9/9 completion)
              players = get_eligible_players(cursor)

              if not players:
                  print("No eligible players found. Run player-aggregation first.")
                  sys.exit(1)

              print(f"Processing {len(players)} players with parallel API calls (20 concurrent)")

              # Run async processing
              profiles_updated, equipment_updated = asyncio.run(process_players_async(players))

              print("Next steps:")
              print("  Run 'nix run .#updateDatabaseChunks' to regenerate chunks with new player data")

          except Exception as e:
              print(f"Player profile fetching failed: {e}")
              import traceback
              traceback.print_exc()
              sys.exit(1)
          finally:
              conn.close()

      if __name__ == "__main__":
          main()
    '';
in
  playerProfileScript
