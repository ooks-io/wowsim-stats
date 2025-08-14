{
  writers,
  python3Packages,
  ...
}: let
  parserScript =
    writers.writePython3Bin "cm-leaderboard-parser" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import json
      import glob
      from pathlib import Path
      import sys

      # the root directory where the fetcher script saves its data.
      INPUT_ROOT = "./web/public/data/challenge-mode"
      # a new directory to store the processed and ranked leaderboards.
      OUTPUT_ROOT = "./web/public/data/leaderboards"
      # the number of top runs to collect from each realm's file.
      TOP_N_PER_REALM = 50
      # the number of top runs to keep in final aggregated leaderboards.
      TOP_N_FINAL = 50

      def parse_and_aggregate_data():
          # finds all fetched leaderboard files, parses them, and aggregates the data
          # by dungeon and region
          print("Starting data aggregation...")
          dungeon_data = {}

          search_path = os.path.join(INPUT_ROOT, "**", "*.json")
          leaderboard_files = glob.glob(search_path, recursive=True)

          if not leaderboard_files:
              print(f"FATAL: No leaderboard JSON files found in {os.path.abspath(INPUT_ROOT)}", file=sys.stderr)
              print("Please run the fetcher script first.", file=sys.stderr)
              sys.exit(1)

          print(f"Found {len(leaderboard_files)} leaderboard files to process.")

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

              if dungeon_slug not in dungeon_data:
                  # extract only the English name from the multilingual data
                  map_name = data.get("map", {}).get("name", {})
                  dungeon_name = map_name.get("en_US", dungeon_slug) if isinstance(map_name, dict) else map_name

                  dungeon_data[dungeon_slug] = {
                      "dungeon_name": dungeon_name,
                      "runs": {"us": [], "eu": [], "kr": []}
                  }

              # slice the list to only get the top N runs from this file.
              top_runs = data.get("leading_groups", [])[:TOP_N_PER_REALM]

              for group in top_runs:
                  group["realm_slug"] = realm_slug
                  group["region"] = region
                  dungeon_data[dungeon_slug]["runs"][region].append(group)

          return dungeon_data


      def deduplicate_runs(runs):
          # remove duplicate runs caused by cross-realm groups appearing on multiple realm leaderboards
          seen = set()
          deduplicated = []

          for run in runs:
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
                  deduplicated.append(run)
              else:
                  print(f"    Removing duplicate run: {run['duration']}ms at {run['completed_timestamp']}")

          return deduplicated

      def optimize_run_data(run):
          # optimize individual run data to remove redundant information
          optimized_run = {
              "ranking": run["ranking"],
              "duration": run["duration"],
              "completed_timestamp": run["completed_timestamp"],
              "keystone_level": run.get("keystone_level", 1),
              "members": []
          }

          # add realm and region info if present
          if "realm_slug" in run:
              optimized_run["realm_slug"] = run["realm_slug"]
          if "region" in run:
              optimized_run["region"] = run["region"]

          # optimize member data - handle both old format and already optimized format
          for member in run.get("members", []):
              if "profile" in member:
                  # old format with nested profile data
                  optimized_member = {
                      "name": member["profile"]["name"],
                      "id": member["profile"]["id"],
                      "realm_slug": member["profile"]["realm"]["slug"],
                      "faction": member["faction"]["type"],
                      "spec_id": member["specialization"]["id"]
                  }
              else:
                  # already optimized format - just copy it
                  optimized_member = member.copy()
              optimized_run["members"].append(optimized_member)

          return optimized_run

      def rank_and_save_leaderboards(dungeon_data):
          # sorts the aggregated data to create regional and global leaderboards,
          # then saves them to new JSON files
          if not dungeon_data:
              print("No data was aggregated. Exiting.")
              return

          print("\nRanking leaderboards and generating output files...")
          os.makedirs(OUTPUT_ROOT, exist_ok=True)
          sort_key = lambda run: run["duration"]

          for dungeon_slug, data in dungeon_data.items():
              print(f"  Processing dungeon: {data['dungeon_name']}")
              all_regional_runs = []

              # generate and save REGIONAL leaderboards
              regional_path = os.path.join(OUTPUT_ROOT, "regional")
              for region, runs in data["runs"].items():
                  if not runs:
                      continue

                  # deduplicate and sort regional runs
                  print(f"    Deduplicating {region.upper()} runs ({len(runs)} -> ", end="")
                  deduplicated_runs = deduplicate_runs(runs)
                  print(f"{len(deduplicated_runs)} -> ", end="")
                  deduplicated_runs.sort(key=sort_key)

                  # limit to top N runs for regional leaderboard
                  final_runs = deduplicated_runs[:TOP_N_FINAL]
                  print(f"{len(final_runs)})")

                  # re-rank and optimize runs for regional leaderboard
                  optimized_runs = []
                  for i, run in enumerate(final_runs):
                      run["ranking"] = i + 1
                      optimized_run = optimize_run_data(run)
                      optimized_runs.append(optimized_run)

                  all_regional_runs.extend(optimized_runs)
                  output_dir = os.path.join(regional_path, region, dungeon_slug)
                  os.makedirs(output_dir, exist_ok=True)
                  output_file = os.path.join(output_dir, "leaderboard.json")

                  with open(output_file, 'w', encoding='utf-8') as f:
                      json.dump({
                          "dungeon_name": data["dungeon_name"],
                          "dungeon_slug": dungeon_slug,
                          "region": region,
                          "leaderboard": optimized_runs,
                      }, f, separators=(',', ':'))

              if not all_regional_runs:
                  continue

              # deduplicate global runs (cross-region duplicates)
              print(f"    Deduplicating global runs ({len(all_regional_runs)} -> ", end="")
              deduplicated_global = deduplicate_runs(all_regional_runs)
              print(f"{len(deduplicated_global)} -> ", end="")
              deduplicated_global.sort(key=sort_key)

              # limit to top N runs for global leaderboard
              final_global = deduplicated_global[:TOP_N_FINAL]
              print(f"{len(final_global)})")

              # re-rank and optimize runs for global leaderboard
              optimized_global_runs = []
              for i, run in enumerate(final_global):
                  run["ranking"] = i + 1
                  optimized_run = optimize_run_data(run)
                  optimized_global_runs.append(optimized_run)

              global_path = os.path.join(OUTPUT_ROOT, "global", dungeon_slug)
              os.makedirs(global_path, exist_ok=True)
              global_output_file = os.path.join(global_path, "leaderboard.json")

              with open(global_output_file, 'w', encoding='utf-8') as f:
                  json.dump({
                      "dungeon_name": data["dungeon_name"],
                      "dungeon_slug": dungeon_slug,
                      "leaderboard": optimized_global_runs,
                  }, f, separators=(',', ':'))

      def optimize_individual_files():
          print("\nOptimizing individual challenge mode files...")

          search_path = os.path.join(INPUT_ROOT, "**", "*.json")
          leaderboard_files = glob.glob(search_path, recursive=True)

          if not leaderboard_files:
              print("No individual files found to optimize.")
              return

          success_count = 0
          total_original_size = 0
          total_optimized_size = 0

          for file_path in leaderboard_files:
              try:
                  # get original file size
                  original_size = os.path.getsize(file_path)
                  total_original_size += original_size

                  with open(file_path, 'r', encoding='utf-8') as f:
                      data = json.load(f)

                  # extract and limit leading groups
                  leading_groups = data.get("leading_groups", [])
                  original_count = len(leading_groups)

                  if original_count == 0:
                      continue

                  # limit to TOP_N_PER_REALM records
                  limited_groups = leading_groups[:TOP_N_PER_REALM]

                  # optimize each run
                  optimized_groups = []
                  for i, run in enumerate(limited_groups):
                      optimized_run = optimize_run_data(run)
                      optimized_run["ranking"] = i + 1  # Re-rank after limiting
                      optimized_groups.append(optimized_run)

                  # extract dungeon name
                  map_name = data.get("map", {}).get("name", {})
                  dungeon_name = map_name.get("en_US", "Unknown") if isinstance(map_name, dict) else map_name

                  # create optimized data structure
                  optimized_data = {
                      "_links": data.get("_links", {}),
                      "map": {
                          "name": {"en_US": dungeon_name},
                          "id": data.get("map", {}).get("id", 0)
                      },
                      "period": data.get("period", 0),
                      "period_start_timestamp": data.get("period_start_timestamp", 0),
                      "period_end_timestamp": data.get("period_end_timestamp", 0),
                      "connected_realm": data.get("connected_realm", {}),
                      "map_challenge_mode_id": data.get("map_challenge_mode_id", 0),
                      "name": {"en_US": dungeon_name},
                      "leading_groups": optimized_groups
                  }

                  # write back with minified json
                  with open(file_path, 'w', encoding='utf-8') as f:
                      json.dump(optimized_data, f, separators=(',', ':'))

                  # get new file size
                  optimized_size = os.path.getsize(file_path)
                  total_optimized_size += optimized_size

                  size_reduction = original_count - len(optimized_groups)
                  if size_reduction > 0:
                      print(f"  Optimized {file_path}: {original_count} → {len(optimized_groups)} records (-{size_reduction})")
                  else:
                      print(f"  Minified {file_path}: {len(optimized_groups)} records")

                  success_count += 1

              except Exception as e:
                  print(f"  Error processing {file_path}: {e}")

          print(f"\nIndividual file optimization complete:")
          print(f"  Files processed: {success_count}/{len(leaderboard_files)}")
          print(f"  Total size reduction: {total_original_size:,} → {total_optimized_size:,} bytes")

          if total_original_size > 0:
              reduction_percent = ((total_original_size - total_optimized_size) / total_original_size) * 100
              print(f"  Size reduction: {reduction_percent:.1f}%")

      def main():
          # first optimize individual files in-place
          optimize_individual_files()

          # then create aggregated leaderboards
          aggregated_data = parse_and_aggregate_data()
          rank_and_save_leaderboards(aggregated_data)
          print(f"\nDone. Individual files optimized and ranked leaderboards available in: {os.path.abspath(OUTPUT_ROOT)}")


      if __name__ == "__main__":
          main()
    '';
in
  parserScript
