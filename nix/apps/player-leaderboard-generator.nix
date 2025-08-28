{
  writers,
  python3Packages,
  ...
}: let
  playerLeaderboardScript =
    writers.writePython3Bin "player-leaderboard-generator" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import json
      import glob
      from pathlib import Path
      from collections import defaultdict
      import sys

      INPUT_ROOT = "./web/public/data/challenge-mode"
      GLOBAL_LEADERBOARD_ROOT = "./web/public/data/leaderboards/global"
      OUTPUT_ROOT = "./web/public/data/player-leaderboards"
      TOP_N_PLAYERS = 250

      def get_player_id(member):
          # extract player ID from member data
          return member.get("id") or member.get("profile", {}).get("id", 0)

      def get_player_name(member):
          # extract player name from member data
          return member.get("name") or member.get("profile", {}).get("name", "Unknown")

      def get_player_realm(member):
          # extract player realm from member data
          return member.get("realm_slug") or member.get("profile", {}).get("realm", {}).get("slug", "unknown")

      def load_global_rankings():
          # load global rankings for all dungeons
          global_rankings = {}

          for dungeon_slug in ['gate-of-the-setting-sun', 'mogu-shan-palace', 'scarlet-halls',
                               'scarlet-monastery', 'scholomance', 'shado-pan-monastery',
                               'siege-of-niuzao-temple', 'stormstout-brewery', 'temple-of-the-jade-serpent']:
              global_file = os.path.join(GLOBAL_LEADERBOARD_ROOT, dungeon_slug, 'leaderboard.json')
              if os.path.exists(global_file):
                  try:
                      with open(global_file, 'r', encoding='utf-8') as f:
                          data = json.load(f)
                          global_rankings[dungeon_slug] = {}

                          for run in data.get('leaderboard', []):
                              # create unique identifier for this run
                              duration = run['duration']
                              timestamp = run['completed_timestamp']
                              member_names = tuple(sorted([m['name'] for m in run['members']]))
                              run_key = (duration, timestamp, member_names)
                              global_rankings[dungeon_slug][run_key] = run.get('ranking', 0)
                  except (json.JSONDecodeError, IOError) as e:
                      print(f"Warning: Could not load global rankings for {dungeon_slug}: {e}")

          return global_rankings

      def lookup_global_ranking(run_data, global_rankings):
          # look up the global ranking for a run
          dungeon_slug = run_data['dungeon_slug']
          if dungeon_slug not in global_rankings:
              return "~"

          duration = run_data['duration']
          timestamp = run_data['completed_timestamp']
          member_names = tuple(sorted(run_data['member_names']))
          run_key = (duration, timestamp, member_names)

          # return global ranking if found otherwise ~
          return global_rankings[dungeon_slug].get(run_key, "~")

      def analyze_players():
          # analyze all challenge mode data to identify individual player performance
          print("Starting player analysis...")

          print("Loading global rankings...")
          global_rankings = load_global_rankings()

          # track player runs: player_id -> dungeon_slug -> list of runs
          player_runs = defaultdict(lambda: defaultdict(list))
          available_dungeons = set()
          all_players_data = {}

          search_path = os.path.join(INPUT_ROOT, "**", "*.json")
          leaderboard_files = glob.glob(search_path, recursive=True)

          if not leaderboard_files:
              print(f"FATAL: No leaderboard JSON files found in {os.path.abspath(INPUT_ROOT)}", file=sys.stderr)
              print("Please run the challenge mode parser first.", file=sys.stderr)
              sys.exit(1)

          print(f"Found {len(leaderboard_files)} leaderboard files to analyze.")

          # first pass: collect all runs per player
          for file_path in leaderboard_files:
              path = Path(file_path)
              parts = path.parts
              try:
                  region = parts[-4]
                  realm_slug = parts[-3]
                  dungeon_slug = parts[-2]
              except IndexError:
                  print(f"Warning: Could not parse path structure for {file_path}. Skipping.")
                  continue

              try:
                  with open(file_path, 'r', encoding='utf-8') as f:
                      data = json.load(f)
              except (json.JSONDecodeError, IOError) as e:
                  print(f"Warning: Could not read or parse {file_path}. Skipping. Error: {e}")
                  continue

              # extract dungeon name and track available dungeons
              map_name = data.get("map", {}).get("name", {})
              dungeon_name = map_name.get("en_US", dungeon_slug) if isinstance(map_name, dict) else map_name
              available_dungeons.add(dungeon_slug)

              runs = data.get("leading_groups", [])

              for run in runs:
                  members = run.get("members", [])
                  if len(members) != 5:
                      continue

                  # process each player in this run
                  for member in members:
                      player_id = get_player_id(member)
                      if player_id == 0:
                          continue

                      # store player data
                      if player_id not in all_players_data:
                          all_players_data[player_id] = {
                              "name": get_player_name(member),
                              "realm_slug": get_player_realm(member),
                              "specs": []
                          }
                      # add spec_id if it exists
                      spec_id = member.get("spec_id")
                      if spec_id and spec_id not in all_players_data[player_id]["specs"]:
                          all_players_data[player_id]["specs"].append(spec_id)

                      # create run data for this player
                      run_data = {
                          "duration": run["duration"],
                          "completed_timestamp": run["completed_timestamp"],
                          "ranking": run.get("ranking", 0),
                          "region": region,
                          "realm_slug": realm_slug,
                          "dungeon_name": dungeon_name,
                          "dungeon_slug": dungeon_slug,
                          "members": members,
                          "member_names": [get_player_name(m) for m in members]
                      }

                      # store this run for the player
                      player_runs[player_id][dungeon_slug].append(run_data)

          print(f"Identified {len(player_runs)} unique players.")
          print(f"Available dungeons: {sorted(available_dungeons)}")

          # second pass: for each player find their best time per dungeon
          print("Analyzing player performance across all dungeons...")
          qualified_players = []
          players_analyzed = 0
          players_with_complete_coverage = 0

          for player_id, dungeon_data in player_runs.items():
              players_analyzed += 1
              dungeons_completed = set(dungeon_data.keys())

              # requirement: player must have runs in all available dungeons
              if dungeons_completed != available_dungeons:
                  continue

              players_with_complete_coverage += 1

              # find best run for each dungeon
              best_runs_per_dungeon = {}

              for dungeon_slug, runs in dungeon_data.items():
                  # deduplicate runs within this dungeon
                  unique_runs = []
                  seen_dungeon_runs = set()
                  for run in runs:
                      member_names_sorted = tuple(sorted(run["member_names"]))
                      run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                      if run_id not in seen_dungeon_runs:
                          seen_dungeon_runs.add(run_id)
                          unique_runs.append(run)

                  # sort unique runs by duration to get best time for this player
                  sorted_runs = sorted(unique_runs, key=lambda x: x["duration"])
                  best_run = sorted_runs[0]

                  # look up global ranking for this run
                  global_ranking = lookup_global_ranking(best_run, global_rankings)

                  # store full member data with spec information
                  all_members_data = []
                  for member in best_run["members"]:
                      member_name = get_player_name(member)
                      all_members_data.append({
                          "name": member_name,
                          "realm_slug": get_player_realm(member),
                          "spec_id": member.get("spec_id"),
                          "id": get_player_id(member)
                      })

                  best_runs_per_dungeon[dungeon_slug] = {
                      "duration": best_run["duration"],
                      "dungeon_name": best_run["dungeon_name"],
                      "ranking": global_ranking,
                      "completed_timestamp": best_run["completed_timestamp"],
                      "region": best_run["region"],
                      "realm_slug": best_run["realm_slug"],
                      "team_members": [name for name in best_run["member_names"] if name != all_players_data[player_id]["name"]],
                      "all_members": all_members_data
                  }

              # calculate combined best time across all dungeons
              combined_best_time = sum(run["duration"] for run in best_runs_per_dungeon.values())

              # calculate player statistics
              regions_played = set()
              total_runs = 0
              spec_frequency = {}

              for runs in dungeon_data.values():
                  for run in runs:
                      regions_played.add(run["region"])
                      total_runs += 1

              player_info = all_players_data[player_id]

              # count spec frequency based on best runs per dungeon only
              for dungeon_slug, best_run_data in best_runs_per_dungeon.items():
                  # find this players spec in the all_members data for this best run
                  for member in best_run_data.get("all_members", []):
                      if member["name"] == player_info["name"]:
                          spec_id = member.get("spec_id")
                          if spec_id:
                              spec_frequency[spec_id] = spec_frequency.get(spec_id, 0) + 1
                          break

              # determine most played spec based on best runs only
              most_played_spec = max(spec_frequency.items(), key=lambda x: x[1])[0] if spec_frequency else None

              qualified_players.append({
                  "player_id": player_id,
                  "name": player_info["name"],
                  "realm_slug": player_info["realm_slug"],
                  "main_spec_id": most_played_spec,
                  "dungeons_completed": len(dungeons_completed),
                  "total_runs": total_runs,
                  "combined_best_time": combined_best_time,
                  "average_best_time": combined_best_time / len(dungeons_completed),
                  "regions_played": list(regions_played),
                  "best_runs_per_dungeon": best_runs_per_dungeon
              })

          print(f"Analysis results:")
          print(f"  Players analyzed: {players_analyzed}")
          print(f"  Players with complete coverage: {players_with_complete_coverage}")
          print(f"  Final qualifying players: {len(qualified_players)}")

          return qualified_players

      def generate_player_leaderboard(players):
          # generate player leaderboard file sorted by combined best time
          if not players:
              print("No qualifying players found.")
              return

          print(f"Generating player leaderboard...")
          os.makedirs(OUTPUT_ROOT, exist_ok=True)

          # sort by combined best times across all dungeons
          players_by_combined = sorted(players, key=lambda x: x["combined_best_time"])[:TOP_N_PLAYERS]

          # add rankings
          for i, player in enumerate(players_by_combined):
              player["ranking"] = i + 1

          output_file = os.path.join(OUTPUT_ROOT, "best-overall.json")
          with open(output_file, 'w', encoding='utf-8') as f:
              json.dump({
                  "title": "Best Players Overall",
                  "description": "Individual players ranked by their combined best times across all 9 dungeons (complete coverage required)",
                  "generated_timestamp": int(__import__('time').time() * 1000),
                  "total_players": len(players),
                  "leaderboard": players_by_combined
              }, f, separators=(',', ':'))

          print(f"  Generated best-overall.json with {len(players_by_combined)} players")

          # generate summary statistics
          summary_file = os.path.join(OUTPUT_ROOT, "summary.json")
          with open(summary_file, 'w', encoding='utf-8') as f:
              json.dump({
                  "total_players_analyzed": len(players),
                  "players_with_complete_coverage": len(players),
                  "total_runs_processed": sum(p["total_runs"] for p in players),
                  "average_runs_per_player": sum(p["total_runs"] for p in players) / len(players) if players else 0,
                  "most_active_player_runs": max(p["total_runs"] for p in players) if players else 0,
                  "generated_timestamp": int(__import__('time').time() * 1000)
              }, f, separators=(',', ':'))

      def main():
          print("=== WoW Challenge Mode Player Leaderboard Generator ===")
          print(f"Top players per leaderboard: {TOP_N_PLAYERS}")
          print()

          # analyze players from challenge mode data
          players = analyze_players()

          # generate leaderboard file
          generate_player_leaderboard(players)

          print(f"\nPlayer leaderboard generated in: {os.path.abspath(OUTPUT_ROOT)}")
          print("Available leaderboard:")
          print("  - best-overall.json: Players by combined best time across all dungeons")
          print("  - summary.json: Analysis statistics")

      if __name__ == "__main__":
          main()
    '';
in
  playerLeaderboardScript
