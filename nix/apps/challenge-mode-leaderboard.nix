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

      ALL_REALMS = ${builtins.toJSON api.realm}

      # TODO: find repo root
      OUTPUT_ROOT = "./web/public/data"
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


      def merge_leaderboard_data(existing_path, new_data):
        #FIX ME: merge new leaderboard data with existing data, preserving historical entries (https://github.com/ClassicWoWCommunity/mop-classic-bugs/issues/2208)
          if not os.path.exists(existing_path):
              return new_data

          try:
              with open(existing_path, 'r', encoding='utf-8') as f:
                  existing_data = json.load(f)

              # get existing and new leading groups
              existing_groups = existing_data.get('leading_groups', [])
              new_groups = new_data.get('leading_groups', [])

              # combine all groups for deduplication
              all_groups = existing_groups + new_groups

              # deduplicate runs using the same logic as the parser
              seen = set()
              deduplicated_groups = []

              for run in all_groups:
                  # create a unique identifier for each run using timestamp, duration, and sorted player IDs
                  # support both old format (member["profile"]["id"]) and optimized format (member["id"])
                  player_ids = []
                  for member in run["members"]:
                      member_id = member.get("id") or member.get("profile", {}).get("id", 0)
                      player_ids.append(member_id)
                  player_ids = sorted(player_ids)
                  unique_key = (run["completed_timestamp"], run["duration"], tuple(player_ids))

                  if unique_key not in seen:
                      seen.add(unique_key)
                      deduplicated_groups.append(run)

              # sort by duration and re-rank (same logic as parser)
              sort_key = lambda run: run["duration"]
              deduplicated_groups.sort(key=sort_key)

              # re-rank runs from 1 to N based on sorted order
              for i, run in enumerate(deduplicated_groups):
                  run["ranking"] = i + 1

              # use new data as base (has current metadata) but with properly sorted and ranked groups
              merged_data = new_data.copy()
              merged_data['leading_groups'] = deduplicated_groups

              print(f"    Merged {len(existing_groups)} existing + {len(new_groups)} new = {len(deduplicated_groups)} sorted and ranked entries")

              return merged_data

          except (json.JSONDecodeError, KeyError) as e:
              print(f"    ERROR reading existing file, using new data only: {e}")
              return new_data


      def main():
          if not API_TOKEN:
              print(
                  "FATAL: BLIZZARD_API_TOKEN environment variable not set.",
                  file=sys.stderr
              )
              sys.exit(1)

          print("Starting leaderboard fetch...")
          print(f"Outputting data to: {os.path.abspath(OUTPUT_ROOT)}")

          session = requests.Session()
          session.headers.update({"Authorization": f"Bearer {API_TOKEN}"})

          for realm_slug, realm_info in ALL_REALMS.items():
              print(
                  f"\nProcessing Realm: {realm_info['name']} "
                  f"({realm_info['region'].upper()})"
              )

              period_id, dungeons = get_hardcoded_period_and_dungeons()

              for dungeon in dungeons:
                  print(f"  - Fetching dungeon: {dungeon['name']}")
                  leaderboard = get_leaderboard_data(
                      realm_info, dungeon, period_id, session
                  )
                  if not leaderboard:
                      continue

                  output_path = os.path.join(
                      OUTPUT_ROOT, "challenge-mode", realm_info['region'],
                      realm_slug, dungeon['slug'],
                      f"{realm_slug}-{dungeon['slug']}-leaderboard.json"
                  )
                  os.makedirs(os.path.dirname(output_path), exist_ok=True)

                  # merge with existing data to preserve historical records for all regions
                  final_data = merge_leaderboard_data(output_path, leaderboard)

                  with open(output_path, 'w', encoding='utf-8') as f:
                      json.dump(final_data, f, indent=2)

              print("\nDone.")


      if __name__ == "__main__":
          main()
    '';
in
  fetcherScript
