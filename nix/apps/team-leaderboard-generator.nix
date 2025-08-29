{
  writers,
  python3Packages,
  ...
}: let
  teamLeaderboardScript =
    writers.writePython3Bin "team-leaderboard-generator" {
      libraries = [python3Packages.requests];
      doCheck = false;
    }
    ''
      import os
      import json
      import glob
      from pathlib import Path
      from itertools import combinations
      from collections import defaultdict
      import sys

      INPUT_ROOT = "./web/public/data/challenge-mode"
      GLOBAL_LEADERBOARD_ROOT = "./web/public/data/leaderboards/global"
      OUTPUT_ROOT = "./web/public/data/team-leaderboards"
      MIN_TEAM_RUNS = 2
      TOP_N_TEAMS = 50

      def get_player_id(member):
          # extract player ID from member data
          return member.get("id") or member.get("profile", {}).get("id", 0)

      def get_player_name(member):
          # extract player name from member data
          return member.get("name") or member.get("profile", {}).get("name", "Unknown")

      def get_player_realm(member):
          # extract player realm from member data
          return member.get("realm_slug") or member.get("profile", {}).get("realm", {}).get("slug", "unknown")

      def generate_team_cores(members):
          # generate all possible 3-player core combinations from a 5-player team
          player_ids = [get_player_id(member) for member in members]
          core_combinations = list(combinations(sorted(player_ids), 3))
          return core_combinations

      def create_team_signature(core_ids):
          # create a unique signature for a 3-player core team
          return "-".join(map(str, sorted(core_ids)))

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

      def get_team_info(core_ids, all_members_data):
          # get team information for a core team
          team_info = []
          for player_id in core_ids:
              if player_id in all_members_data:
                  member_data = all_members_data[player_id]
                  team_member = {
                      "id": player_id,
                      "name": member_data["name"],
                      "realm_slug": member_data["realm_slug"]
                  }
                  # add spec info if available
                  if member_data.get("specs"):
                      team_member["spec_id"] = member_data["specs"][0]
                  team_info.append(team_member)
              else:
                  team_info.append({
                      "id": player_id,
                      "name": "Unknown",
                      "realm_slug": "unknown"
                  })
          return team_info

      def deduplicate_overlapping_teams(teams, all_members_data):
          # Simple rule: No two teams can share ANY runs - if they do, they're the same team
          remaining_teams = teams.copy()
          deduplicated = []

          while remaining_teams:
              current_team = remaining_teams.pop(0)
              current_run_ids = set()
              
              # Create unique identifiers for all runs in current team
              for run in current_team["all_runs"]:
                  member_names_sorted = tuple(sorted(run["member_names"]))
                  run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                  current_run_ids.add(run_id)

              # Find teams that share ANY runs with current team
              same_teams = [current_team]
              non_overlapping = []

              for other_team in remaining_teams:
                  other_run_ids = set()
                  
                  # Create unique identifiers for all runs in other team
                  for run in other_team["all_runs"]:
                      member_names_sorted = tuple(sorted(run["member_names"]))
                      run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                      other_run_ids.add(run_id)
                  
                  # If they share ANY runs, they're the same underlying team
                  if current_run_ids.intersection(other_run_ids):
                      same_teams.append(other_team)
                  else:
                      non_overlapping.append(other_team)

              remaining_teams = non_overlapping

              if len(same_teams) == 1:
                  # No overlapping teams found, keep as is
                  deduplicated.append(current_team)
              else:
                  # Multiple teams share runs - merge them into one team
                  # Use the team with the best combined time as the representative
                  best_team = min(same_teams, key=lambda t: t["combined_best_time"])
                  
                  # Merge all data from overlapping teams
                  # BUT only include players who appear in the BEST runs
                  all_runs = []
                  regions_played = set()
                  total_runs = 0
                  seen_runs = set()

                  for team in same_teams:
                      # Collect all unique runs
                      regions_played.update(team["regions_played"])
                      total_runs += team["total_runs"]
                      
                      for run in team["all_runs"]:
                          member_names_sorted = tuple(sorted(run["member_names"]))
                          run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                          if run_id not in seen_runs:
                              seen_runs.add(run_id)
                              all_runs.append(run)

                  # Build extended roster ONLY from players in the merged team's best runs
                  all_extended_roster_ids = set()
                  for dungeon_slug, run_data in best_team["best_runs_per_dungeon"].items():
                      # Find players who participated in this best run
                      for run in all_runs:
                          if (run["duration"] == run_data["duration"] and 
                              run["completed_timestamp"] == run_data["completed_timestamp"]):
                              all_extended_roster_ids.update(run["member_ids"])
                              break

                  # Create merged extended roster
                  merged_extended_roster = []
                  for player_id in sorted(all_extended_roster_ids):
                      if player_id in all_members_data:
                          member_data = all_members_data[player_id]
                          roster_member = {
                              "name": member_data["name"],
                              "realm_slug": member_data["realm_slug"]
                          }
                          if member_data.get("specs"):
                              roster_member["spec_id"] = member_data["specs"][0]
                          merged_extended_roster.append(roster_member)

                  # Use best performing team as base and update with merged data
                  merged_team = best_team.copy()
                  merged_team["extended_roster"] = merged_extended_roster
                  merged_team["total_runs"] = len(all_runs)  # Use actual total unique runs
                  merged_team["regions_played"] = list(regions_played)
                  merged_team["all_runs"] = sorted(all_runs, key=lambda x: x["duration"])

                  deduplicated.append(merged_team)

          return deduplicated

      def analyze_teams():
          # analyze all challenge mode data to identify unique teams and their optimal 3-player cores
          print("Starting team analysis...")

          print("Loading global rankings...")
          global_rankings = load_global_rankings()

          # Collect all runs and track player data
          all_runs = []
          available_dungeons = set()
          all_members_data = {}

          search_path = os.path.join(INPUT_ROOT, "**", "*.json")
          leaderboard_files = glob.glob(search_path, recursive=True)

          if not leaderboard_files:
              print(f"FATAL: No leaderboard JSON files found in {os.path.abspath(INPUT_ROOT)}", file=sys.stderr)
              print("Please run the challenge mode parser first.", file=sys.stderr)
              sys.exit(1)

          print(f"Found {len(leaderboard_files)} leaderboard files to analyze.")

          # Collect all runs
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

              # Extract dungeon name and track available dungeons
              map_name = data.get("map", {}).get("name", {})
              dungeon_name = map_name.get("en_US", dungeon_slug) if isinstance(map_name, dict) else map_name
              available_dungeons.add(dungeon_slug)

              runs = data.get("leading_groups", [])

              for run in runs:
                  members = run.get("members", [])
                  if len(members) != 5:
                      continue

                  # Store member data for later lookup (track all specs they've played)
                  for member in members:
                      player_id = get_player_id(member)
                      if player_id not in all_members_data:
                          all_members_data[player_id] = {
                              "name": get_player_name(member),
                              "realm_slug": get_player_realm(member),
                              "specs": []
                          }
                      # Add spec_id if it exists
                      spec_id = member.get("spec_id")
                      if spec_id and spec_id not in all_members_data[player_id]["specs"]:
                          all_members_data[player_id]["specs"].append(spec_id)

                  # Create run data
                  member_ids = [get_player_id(m) for m in members]
                  run_data = {
                      "duration": run["duration"],
                      "completed_timestamp": run["completed_timestamp"],
                      "ranking": run.get("ranking", 0),
                      "region": region,
                      "realm_slug": realm_slug,
                      "dungeon_name": dungeon_name,
                      "dungeon_slug": dungeon_slug,
                      "members": members,
                      "member_ids": member_ids,
                      "member_names": [get_player_name(m) for m in members]
                  }

                  all_runs.append(run_data)

          print(f"Collected {len(all_runs)} total runs across {len(available_dungeons)} dungeons")
          print(f"Available dungeons: {sorted(available_dungeons)}")

          # Generate all possible 3-player cores from runs and track their dungeon coverage
          print("Analyzing 3-player cores for complete dungeon coverage...")
          core_runs = defaultdict(lambda: defaultdict(list))  # core_sig -> dungeon -> [runs]
          
          for run_data in all_runs:
              member_ids = run_data["member_ids"]
              dungeon_slug = run_data["dungeon_slug"]
              
              # Generate all possible 3-player combinations from this 5-player run
              core_combinations = generate_team_cores(run_data["members"])
              
              for core_combo in core_combinations:
                  core_sig = create_team_signature(core_combo)
                  core_runs[core_sig][dungeon_slug].append(run_data)

          print(f"Generated {len(core_runs)} unique 3-player cores from all runs")

          # Filter cores that have complete dungeon coverage
          qualified_teams = []
          cores_with_complete_coverage = 0

          for core_sig, dungeon_data in core_runs.items():
              dungeons_completed = set(dungeon_data.keys())
              
              # REQUIREMENT: Core must have runs in ALL available dungeons
              if dungeons_completed != available_dungeons:
                  continue
                  
              cores_with_complete_coverage += 1
              
              # Get core player IDs
              core_ids = sorted([int(x) for x in core_sig.split("-")])

              # Find best run for each dungeon containing this core
              best_runs_per_dungeon = {}
              total_runs = 0
              
              for dungeon_slug, runs in dungeon_data.items():
                  # Deduplicate runs within this dungeon
                  unique_runs = []
                  seen_dungeon_runs = set()
                  for run in runs:
                      member_names_sorted = tuple(sorted(run["member_names"]))
                      run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                      if run_id not in seen_dungeon_runs:
                          seen_dungeon_runs.add(run_id)
                          unique_runs.append(run)

                  total_runs += len(unique_runs)
                  
                  # Sort by duration to get best time
                  sorted_runs = sorted(unique_runs, key=lambda x: x["duration"])
                  best_run = sorted_runs[0]

                  # Look up global ranking
                  global_ranking = lookup_global_ranking(best_run, global_rankings)

                  best_runs_per_dungeon[dungeon_slug] = {
                      "duration": best_run["duration"],
                      "dungeon_name": best_run["dungeon_name"],
                      "ranking": global_ranking,
                      "completed_timestamp": best_run["completed_timestamp"],
                      "region": best_run["region"],
                      "realm_slug": best_run["realm_slug"],
                      "members": best_run["member_names"]
                  }

              # REQUIREMENT: Must have minimum number of total runs
              if total_runs < MIN_TEAM_RUNS:
                  continue

              # Build extended roster ONLY from players who appear in best runs
              extended_roster_ids = set()
              all_team_runs = []
              seen_runs = set()
              regions_played = set()
              
              # First, collect ONLY the players from best runs per dungeon
              for dungeon_slug, run_data in best_runs_per_dungeon.items():
                  # Find the actual best run to get member_ids
                  for run in dungeon_data[dungeon_slug]:
                      if run["duration"] == run_data["duration"] and run["completed_timestamp"] == run_data["completed_timestamp"]:
                          extended_roster_ids.update(run["member_ids"])
                          break
              
              # Then collect all runs for this core (for statistics, not for roster)
              for run in core_runs[core_sig].values():
                  for single_run in run:
                      regions_played.add(single_run["region"])
                      member_names_sorted = tuple(sorted(single_run["member_names"]))
                      run_id = (single_run["duration"], single_run["completed_timestamp"], member_names_sorted)
                      if run_id not in seen_runs:
                          seen_runs.add(run_id)
                          all_team_runs.append(single_run)

              # Create core team info
              core_info = get_team_info(core_ids, all_members_data)

              # Create extended roster
              extended_roster = []
              for player_id in sorted(extended_roster_ids):
                  if player_id in all_members_data:
                      member_data = all_members_data[player_id]
                      roster_member = {
                          "name": member_data["name"],
                          "realm_slug": member_data["realm_slug"]
                      }
                      if member_data.get("specs"):
                          roster_member["spec_id"] = member_data["specs"][0]
                      extended_roster.append(roster_member)

              # Calculate combined best time
              combined_best_time = sum(run["duration"] for run in best_runs_per_dungeon.values())

              qualified_teams.append({
                  "team_signature": core_sig,
                  "core_members": core_info,
                  "extended_roster": extended_roster,
                  "dungeons_completed": len(dungeons_completed),
                  "total_runs": total_runs,
                  "combined_best_time": combined_best_time,
                  "average_best_time": combined_best_time / len(dungeons_completed),
                  "regions_played": list(regions_played),
                  "best_runs_per_dungeon": best_runs_per_dungeon,
                  "all_runs": sorted(all_team_runs, key=lambda x: x["duration"])
              })

          print(f"Analysis results:")
          print(f"  3-player cores analyzed: {len(core_runs)}")
          print(f"  Cores with complete coverage: {cores_with_complete_coverage}")
          print(f"  Final qualifying teams: {len(qualified_teams)}")

          return qualified_teams, all_members_data

      def generate_team_leaderboard(teams):
          """Generate team leaderboard file sorted by combined best time"""
          if not teams:
              print("No qualifying teams found.")
              return

          print(f"Generating team leaderboard...")
          os.makedirs(OUTPUT_ROOT, exist_ok=True)

          # Sort by combined best times across ALL dungeons (primary ranking)
          teams_by_combined = sorted(teams, key=lambda x: x["combined_best_time"])[:TOP_N_TEAMS]

          # Add rankings
          for i, team in enumerate(teams_by_combined):
              team["ranking"] = i + 1

          output_file = os.path.join(OUTPUT_ROOT, "best-overall.json")
          with open(output_file, 'w', encoding='utf-8') as f:
              json.dump({
                  "title": "Best Teams Overall",
                  "description": "Teams ranked by their combined best times across all 9 dungeons (complete coverage required)",
                  "generated_timestamp": int(__import__('time').time() * 1000),
                  "total_teams": len(teams),
                  "min_runs_required": MIN_TEAM_RUNS,
                  "leaderboard": teams_by_combined
              }, f, separators=(',', ':'))

          print(f"  Generated best-overall.json with {len(teams_by_combined)} teams")

          # Generate summary statistics
          summary_file = os.path.join(OUTPUT_ROOT, "summary.json")
          with open(summary_file, 'w', encoding='utf-8') as f:
              json.dump({
                  "total_teams_analyzed": len(teams),
                  "teams_with_complete_coverage": len(teams),  # All teams in the list have complete coverage
                  "total_runs_processed": sum(t["total_runs"] for t in teams) // 10,  # Divide by 10 since each run creates 10 team combinations
                  "average_runs_per_team": sum(t["total_runs"] for t in teams) / len(teams) if teams else 0,
                  "most_active_team_runs": max(t["total_runs"] for t in teams) if teams else 0,
                  "generated_timestamp": int(__import__('time').time() * 1000)
              }, f, separators=(',', ':'))

      def main():
          print("=== WoW Challenge Mode Team Leaderboard Generator ===")
          print(f"Minimum runs required per team: {MIN_TEAM_RUNS}")
          print(f"Top teams per leaderboard: {TOP_N_TEAMS}")
          print()

          # Analyze teams from challenge mode data
          teams, all_members_data = analyze_teams()

          # Deduplicate overlapping teams (same underlying team with different core perspectives)
          teams = deduplicate_overlapping_teams(teams, all_members_data)
          print(f"After deduplication: {len(teams)} unique teams")

          # Generate leaderboard file
          generate_team_leaderboard(teams)

          print(f"\nTeam leaderboard generated in: {os.path.abspath(OUTPUT_ROOT)}")
          print("Available files:")
          print("  - best-overall.json: Teams by combined best time across all dungeons")
          print("  - summary.json: Analysis statistics")

      if __name__ == "__main__":
          main()
    '';
in
  teamLeaderboardScript
