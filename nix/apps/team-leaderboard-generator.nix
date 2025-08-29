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
          # merge teams that share most of their best times
          remaining_teams = teams.copy()
          deduplicated = []

          while remaining_teams:
              current_team = remaining_teams.pop(0)
              current_best_times = current_team["best_runs_per_dungeon"]

              # Find teams that share most best times with current team
              similar_teams = [current_team]
              non_similar = []

              for other_team in remaining_teams:
                  other_best_times = other_team["best_runs_per_dungeon"]

                  # Count how many dungeons have identical best times
                  matching_dungeons = 0
                  total_dungeons = min(len(current_best_times), len(other_best_times))

                  for dungeon_slug in current_best_times:
                      if dungeon_slug in other_best_times:
                          if current_best_times[dungeon_slug]["duration"] == other_best_times[dungeon_slug]["duration"]:
                              matching_dungeons += 1

                  # If they share 6+ identical times out of 9 dungeons, consider them same team
                  match_percentage = matching_dungeons / total_dungeons if total_dungeons > 0 else 0
                  if matching_dungeons >= 6 and match_percentage >= 0.6:
                      similar_teams.append(other_team)
                  else:
                      non_similar.append(other_team)

              remaining_teams = non_similar

              if len(similar_teams) == 1:
                  # No similar teams found
                  deduplicated.append(current_team)
              else:
                  # Merge similar teams (same underlying team with different 3-player core perspectives)
                  # Collect all members and data
                  all_core_members = set()
                  all_extended_members = set()
                  all_runs = []
                  regions_played = set()
                  total_runs = 0
                  best_combined_time = float('inf')
                  best_team_runs = None

                  for team in similar_teams:
                      # Collect core members
                      core_ids = list(map(int, team["team_signature"].split("-")))
                      all_core_members.update(core_ids)

                      # Collect extended roster
                      for member in team["extended_roster"]:
                          all_extended_members.add(member["name"] + "@" + member["realm_slug"])

                      # Collect runs and other data
                      all_runs.extend(team["all_runs"])
                      regions_played.update(team["regions_played"])
                      total_runs += team["total_runs"]

                      # Use the best performance among similar teams
                      if team["combined_best_time"] < best_combined_time:
                          best_combined_time = team["combined_best_time"]
                          best_team_runs = team["best_runs_per_dungeon"]

                  # Create merged team - limit core to 3 most consistent players across best runs
                  # Count participation in best runs across all similar teams
                  player_best_participation = defaultdict(int)
                  for team in similar_teams:
                      for run_data in team["best_runs_per_dungeon"].values():
                          # Find actual run to count participation
                          for dungeon_runs in [team["all_runs"]]:  # Use all_runs from team
                              for run in dungeon_runs:
                                  if run["duration"] == run_data["duration"] and run["completed_timestamp"] == run_data["completed_timestamp"]:
                                      for member_id in run["member_ids"]:
                                          player_best_participation[member_id] += 1
                                      break

                  # Select top 3 most consistent players as merged core
                  top_players = sorted(player_best_participation.items(), key=lambda x: x[1], reverse=True)[:3]
                  merged_core_ids = sorted([player_id for player_id, count in top_players])
                  merged_core_info = get_team_info(merged_core_ids, all_members_data)

                  # Create extended roster from players who appear in the merged team's best runs
                  merged_extended_roster_ids = set()
                  for run_data in best_team_runs.values():
                      # Find players who participated in this best run across all similar teams
                      for team in similar_teams:
                          for run in team["all_runs"]:
                              if run["duration"] == run_data["duration"] and run["completed_timestamp"] == run_data["completed_timestamp"]:
                                  merged_extended_roster_ids.update(run["member_ids"])
                                  break

                  merged_extended_roster = []
                  for member_id in sorted(merged_extended_roster_ids):
                      if member_id in all_members_data:
                          member_data = all_members_data[member_id]
                          roster_member = {
                              "name": member_data["name"],
                              "realm_slug": member_data["realm_slug"]
                          }
                          # Add spec info if available (use first spec if multiple)
                          if member_data.get("specs"):
                              roster_member["spec_id"] = member_data["specs"][0]
                          merged_extended_roster.append(roster_member)

                  merged_sig = create_team_signature(merged_core_ids)

                  merged_team = {
                      "team_signature": merged_sig,
                      "core_members": merged_core_info,
                      "extended_roster": merged_extended_roster,
                      "dungeons_completed": similar_teams[0]["dungeons_completed"],
                      "total_runs": total_runs // len(similar_teams),
                      "combined_best_time": best_combined_time,
                      "average_best_time": best_combined_time / similar_teams[0]["dungeons_completed"],
                      "regions_played": list(regions_played),
                      "best_runs_per_dungeon": best_team_runs,
                      "all_runs": sorted(all_runs, key=lambda x: x["duration"])
                  }

                  # Validate that merged core players appear in most of their best runs
                  core_participation_count = 0
                  total_best_runs = len(best_team_runs)
                  failed_runs = []
                  
                  for dungeon_slug, run_data in best_team_runs.items():
                      # Find actual run to check core participation
                      run_found = False
                      for team in similar_teams:
                          for run in team["all_runs"]:
                              if run["duration"] == run_data["duration"] and run["completed_timestamp"] == run_data["completed_timestamp"]:
                                  # Count how many core members participated in this run
                                  core_in_run = sum(1 for core_id in merged_core_ids if core_id in run["member_ids"])
                                  if core_in_run >= 2:  # At least 2 of 3 core members
                                      core_participation_count += 1
                                  else:
                                      failed_runs.append(f"{dungeon_slug}: {core_in_run}/3 core members")
                                  run_found = True
                                  break
                          if run_found:
                              break
                  
                  core_participation_rate = core_participation_count / total_best_runs if total_best_runs > 0 else 0
                  if core_participation_rate >= 0.85 and len(failed_runs) <= 1:  # Allow max 1 failed run
                      deduplicated.append(merged_team)
                  else:
                      print(f"Warning: Rejecting merged team {merged_sig} - core players only appear in {core_participation_rate:.1%} of best runs. Failed runs: {failed_runs}")
                      # Add individual teams instead of the problematic merged team
                      deduplicated.extend(similar_teams)

          return deduplicated

      def analyze_teams():
          # analyze all challenge mode data to identify unique teams and their optimal 3-player cores
          print("Starting team analysis...")

          print("Loading global rankings...")
          global_rankings = load_global_rankings()

          # first pass: collect all runs and group by extended rosters
          roster_runs = defaultdict(lambda: defaultdict(list))
          available_dungeons = set()
          all_members_data = {}

          search_path = os.path.join(INPUT_ROOT, "**", "*.json")
          leaderboard_files = glob.glob(search_path, recursive=True)

          if not leaderboard_files:
              print(f"FATAL: No leaderboard JSON files found in {os.path.abspath(INPUT_ROOT)}", file=sys.stderr)
              print("Please run the challenge mode parser first.", file=sys.stderr)
              sys.exit(1)

          print(f"Found {len(leaderboard_files)} leaderboard files to analyze.")

          # First pass: collect all runs and identify unique extended rosters
          extended_rosters = {}  # roster_sig -> set of all player_ids who have run together

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

                  # Track extended rosters - players who run together consistently
                  # Use stricter criteria to prevent artificial team merging
                  found_roster = None
                  current_players = set(member_ids)

                  # Check if this run matches any existing extended roster
                  for roster_sig, roster_players in extended_rosters.items():
                      overlap = len(current_players.intersection(roster_players))
                      overlap_percentage = overlap / len(current_players.union(roster_players))
                      
                      # More balanced criteria: Allow extended rosters but prevent mega-merging
                      # Require 3+ overlap but with minimum 35% similarity to prevent distant connections
                      if overlap >= 3 and overlap_percentage >= 0.35:
                          found_roster = roster_sig
                          # Add new players to the extended roster
                          extended_rosters[roster_sig].update(current_players)
                          break

                  # If no matching roster found, create new one
                  if found_roster is None:
                      roster_sig = create_team_signature(sorted(member_ids))
                      extended_rosters[roster_sig] = current_players.copy()
                      found_roster = roster_sig

                  # Store run for this extended roster
                  roster_runs[found_roster][dungeon_slug].append(run_data)

          print(f"Identified {len(extended_rosters)} unique extended rosters.")
          print(f"Available dungeons: {sorted(available_dungeons)}")

          # Second pass: For each extended roster, identify best 3-player core and performance
          print("Analyzing extended rosters to identify consistent 3-player cores...")
          qualified_teams = []
          rosters_analyzed = 0
          rosters_with_complete_coverage = 0

          for roster_sig, dungeon_data in roster_runs.items():
              rosters_analyzed += 1
              dungeons_completed = set(dungeon_data.keys())

              # REQUIREMENT: Roster must have runs in ALL available dungeons
              if dungeons_completed != available_dungeons:
                  continue

              rosters_with_complete_coverage += 1

              # Find best run for each dungeon and track player participation in best runs
              best_runs_per_dungeon = {}
              player_participation_in_best = defaultdict(int)

              for dungeon_slug, runs in dungeon_data.items():
                  # Deduplicate runs within this dungeon (same run across different realms)
                  unique_runs = []
                  seen_dungeon_runs = set()
                  for run in runs:
                      member_names_sorted = tuple(sorted(run["member_names"]))
                      run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                      if run_id not in seen_dungeon_runs:
                          seen_dungeon_runs.add(run_id)
                          unique_runs.append(run)

                  # Sort unique runs by duration to get best time for this roster in this dungeon
                  sorted_runs = sorted(unique_runs, key=lambda x: x["duration"])
                  best_run = sorted_runs[0]

                  # Look up global ranking for this run
                  global_ranking = lookup_global_ranking(best_run, global_rankings)

                  best_runs_per_dungeon[dungeon_slug] = {
                      "duration": best_run["duration"],
                      "dungeon_name": best_run["dungeon_name"],
                      "ranking": global_ranking,  # Use global ranking instead of realm ranking
                      "completed_timestamp": best_run["completed_timestamp"],
                      "region": best_run["region"],
                      "realm_slug": best_run["realm_slug"],
                      "members": best_run["member_names"]
                  }

                  # Track which players appear in the best runs (for core identification)
                  for player_id in best_run["member_ids"]:
                      player_participation_in_best[player_id] += 1

              # Identify the 3-player core: players who appear in the most best runs
              total_dungeons = len(dungeons_completed)
              participation_sorted = sorted(player_participation_in_best.items(), key=lambda x: x[1], reverse=True)

              # Take top 3 players who appear in the most best runs as the core
              if len(participation_sorted) >= 3:
                  core_ids = sorted([pid for pid, count in participation_sorted[:3]])
              else:
                  continue  # Not enough consistent players

              # Get extended roster (only players who appear in the best runs per dungeon)
              extended_roster_ids = set()
              for run_data in best_runs_per_dungeon.values():
                  # Find the actual run to get member_ids
                  for dungeon_slug, runs in dungeon_data.items():
                      for run in runs:
                          if run["duration"] == run_data["duration"] and run["completed_timestamp"] == run_data["completed_timestamp"]:
                              extended_roster_ids.update(run["member_ids"])
                              break

              # REQUIREMENT: Roster must have minimum number of total runs
              total_runs = sum(len(runs) for runs in dungeon_data.values())
              if total_runs < MIN_TEAM_RUNS:
                  continue

              # Create core team info
              core_info = get_team_info(core_ids, all_members_data)

              # Create extended roster info (only players who contributed to best times)
              extended_roster = []
              for player_id in sorted(extended_roster_ids):
                  if player_id in all_members_data:
                      member_data = all_members_data[player_id]
                      roster_member = {
                          "name": member_data["name"],
                          "realm_slug": member_data["realm_slug"]
                      }
                      # Add spec info if available (use first spec if multiple)
                      if member_data.get("specs"):
                          roster_member["spec_id"] = member_data["specs"][0]
                      extended_roster.append(roster_member)

              # Calculate combined best time across ALL dungeons
              combined_best_time = sum(run["duration"] for run in best_runs_per_dungeon.values())

              # Calculate team statistics
              regions_played = set()
              all_team_runs = []
              seen_runs = set()  # Track unique runs by (duration, timestamp)
              for runs in dungeon_data.values():
                  for run in runs:
                      regions_played.add(run["region"])
                      # Create unique identifier for run deduplication (include member names for cross-realm duplicates)
                      member_names_sorted = tuple(sorted(run["member_names"]))
                      run_id = (run["duration"], run["completed_timestamp"], member_names_sorted)
                      if run_id not in seen_runs:
                          seen_runs.add(run_id)
                          all_team_runs.append(run)

              # Use core signature for uniqueness
              core_sig = create_team_signature(core_ids)

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
          print(f"  Extended rosters analyzed: {rosters_analyzed}")
          print(f"  Rosters with complete coverage: {rosters_with_complete_coverage}")
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
