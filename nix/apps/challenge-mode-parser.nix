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

      # The root directory where the fetcher script saves its data.
      INPUT_ROOT = "./web/public/data/challenge-mode"
      # A new directory to store the processed and ranked leaderboards.
      OUTPUT_ROOT = "./web/public/data/leaderboards"
      # The number of top runs to collect from each realm's file.
      TOP_N_PER_REALM = 50

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
                  dungeon_data[dungeon_slug] = {
                      "dungeon_name": data.get("map", {}).get("name", {}).get("en_US", dungeon_slug),
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
          # Remove duplicate runs caused by cross-realm groups appearing on multiple realm leaderboards
          seen = set()
          deduplicated = []
          
          for run in runs:
              # Create a unique identifier for each run using timestamp, duration, and sorted player IDs
              player_ids = sorted([member["profile"]["id"] for member in run["members"]])
              unique_key = (run["completed_timestamp"], run["duration"], tuple(player_ids))
              
              if unique_key not in seen:
                  seen.add(unique_key)
                  deduplicated.append(run)
              else:
                  print(f"    Removing duplicate run: {run['duration']}ms at {run['completed_timestamp']}")
          
          return deduplicated

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

              # 1. Generate and save REGIONAL leaderboards
              regional_path = os.path.join(OUTPUT_ROOT, "regional")
              for region, runs in data["runs"].items():
                  if not runs:
                      continue

                  # Deduplicate and sort regional runs
                  print(f"    Deduplicating {region.upper()} runs ({len(runs)} -> ", end="")
                  deduplicated_runs = deduplicate_runs(runs)
                  print(f"{len(deduplicated_runs)})")
                  deduplicated_runs.sort(key=sort_key)
                  
                  # Re-rank runs for regional leaderboard
                  for i, run in enumerate(deduplicated_runs):
                      run["ranking"] = i + 1
                  
                  all_regional_runs.extend(deduplicated_runs)
                  output_dir = os.path.join(regional_path, region, dungeon_slug)
                  os.makedirs(output_dir, exist_ok=True)
                  output_file = os.path.join(output_dir, "leaderboard.json")

                  with open(output_file, 'w', encoding='utf-8') as f:
                      json.dump({
                          "dungeon_name": data["dungeon_name"],
                          "dungeon_slug": dungeon_slug,
                          "region": region,
                          "leaderboard": deduplicated_runs,
                      }, f, indent=2)

              # 2. Generate and save GLOBAL leaderboard
              if not all_regional_runs:
                  continue

              # Deduplicate global runs (cross-region duplicates)
              print(f"    Deduplicating global runs ({len(all_regional_runs)} -> ", end="")
              deduplicated_global = deduplicate_runs(all_regional_runs)
              print(f"{len(deduplicated_global)})")
              deduplicated_global.sort(key=sort_key)
              
              # Re-rank runs for global leaderboard
              for i, run in enumerate(deduplicated_global):
                  run["ranking"] = i + 1
              
              global_path = os.path.join(OUTPUT_ROOT, "global", dungeon_slug)
              os.makedirs(global_path, exist_ok=True)
              global_output_file = os.path.join(global_path, "leaderboard.json")

              with open(global_output_file, 'w', encoding='utf-8') as f:
                  json.dump({
                      "dungeon_name": data["dungeon_name"],
                      "dungeon_slug": dungeon_slug,
                      "leaderboard": deduplicated_global,
                  }, f, indent=2)

      def main():
          aggregated_data = parse_and_aggregate_data()
          rank_and_save_leaderboards(aggregated_data)
          print(f"\nDone. Ranked leaderboards are available in: {os.path.abspath(OUTPUT_ROOT)}")


      if __name__ == "__main__":
          main()
    '';
in
  parserScript
