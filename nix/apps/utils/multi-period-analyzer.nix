{
  writers,
  python3Packages,
  api,
  ...
}: let
  inherit (api) realm;

  multiPeriodAnalyzerScript =
    writers.writePython3Bin "multi-period-analyzer" {
      libraries = [python3Packages.aiohttp];
      doCheck = false;
    }
    ''
      import os
      import aiohttp
      import asyncio
      import json
      import sys
      import time
      from collections import defaultdict
      import csv

      # All Challenge Mode dungeons (from constants.go)
      DUNGEONS = [
          {"id": 2, "name": "Temple of the Jade Serpent", "slug": "temple-of-the-jade-serpent"},
          {"id": 56, "name": "Stormstout Brewery", "slug": "stormstout-brewery"},
          {"id": 57, "name": "Gate of the Setting Sun", "slug": "gate-of-the-setting-sun"},
          {"id": 58, "name": "Shado-Pan Monastery", "slug": "shado-pan-monastery"},
          {"id": 59, "name": "Siege of Niuzao Temple", "slug": "siege-of-niuzao-temple"},
          {"id": 60, "name": "Mogu'shan Palace", "slug": "mogu-shan-palace"},
          {"id": 76, "name": "Scholomance", "slug": "scholomance"},
          {"id": 77, "name": "Scarlet Halls", "slug": "scarlet-halls"},
          {"id": 78, "name": "Scarlet Monastery", "slug": "scarlet-monastery"},
      ]

      # All realms (from constants.go and nix/api/realm.nix)
      REALMS = [
          # US Realms
          {"id": 4372, "region": "us", "name": "Atiesh", "slug": "atiesh"},
          {"id": 4373, "region": "us", "name": "Myzrael", "slug": "myzrael"},
          {"id": 4374, "region": "us", "name": "Old Blanchy", "slug": "old-blanchy"},
          {"id": 4376, "region": "us", "name": "Azuresong", "slug": "azuresong"},
          {"id": 4384, "region": "us", "name": "Mankrik", "slug": "mankrik"},
          {"id": 4385, "region": "us", "name": "Pagle", "slug": "pagle"},
          {"id": 4387, "region": "us", "name": "Ashkandi", "slug": "ashkandi"},
          {"id": 4388, "region": "us", "name": "Westfall", "slug": "westfall"},
          {"id": 4395, "region": "us", "name": "Whitemane", "slug": "whitemane"},
          {"id": 4408, "region": "us", "name": "Faerlina", "slug": "faerlina"},
          {"id": 4647, "region": "us", "name": "Grobbulus", "slug": "grobbulus"},
          {"id": 4648, "region": "us", "name": "Bloodsail Buccaneers", "slug": "bloodsail-buccaneers"},
          {"id": 4667, "region": "us", "name": "Remulos", "slug": "remulos"},
          {"id": 4669, "region": "us", "name": "Arugal", "slug": "arugal"},
          {"id": 4670, "region": "us", "name": "Yojamba", "slug": "yojamba"},
          {"id": 4725, "region": "us", "name": "Skyfury", "slug": "skyfury"},
          {"id": 4726, "region": "us", "name": "Sulfuras", "slug": "sulfuras"},
          {"id": 4727, "region": "us", "name": "Windseeker", "slug": "windseeker"},
          {"id": 4728, "region": "us", "name": "Benediction", "slug": "benediction"},
          {"id": 4731, "region": "us", "name": "Earthfury", "slug": "earthfury"},
          {"id": 4738, "region": "us", "name": "Maladath", "slug": "maladath"},
          {"id": 4795, "region": "us", "name": "Angerforge", "slug": "angerforge"},
          {"id": 4800, "region": "us", "name": "Eranikus", "slug": "eranikus"},

          # EU Realms
          {"id": 4440, "region": "eu", "name": "Everlook", "slug": "everlook"},
          {"id": 4441, "region": "eu", "name": "Auberdine", "slug": "auberdine"},
          {"id": 4442, "region": "eu", "name": "Lakeshire", "slug": "lakeshire"},
          {"id": 4452, "region": "eu", "name": "Chromie", "slug": "chromie"},
          {"id": 4453, "region": "eu", "name": "Pyrewood Village", "slug": "pyrewood-village"},
          {"id": 4454, "region": "eu", "name": "Mirage Raceway", "slug": "mirage-raceway"},
          {"id": 4455, "region": "eu", "name": "Razorfen", "slug": "razorfen"},
          {"id": 4456, "region": "eu", "name": "Nethergarde Keep", "slug": "nethergarde-keep"},
          {"id": 4464, "region": "eu", "name": "Sulfuron", "slug": "sulfuron"},
          {"id": 4465, "region": "eu", "name": "Golemagg", "slug": "golemagg"},
          {"id": 4466, "region": "eu", "name": "Patchwerk", "slug": "patchwerk"},
          {"id": 4467, "region": "eu", "name": "Firemaw", "slug": "firemaw"},
          {"id": 4474, "region": "eu", "name": "Flamegor", "slug": "flamegor"},
          {"id": 4476, "region": "eu", "name": "Gehennas", "slug": "gehennas"},
          {"id": 4477, "region": "eu", "name": "Venoxis", "slug": "venoxis"},
          {"id": 4678, "region": "eu", "name": "Hydraxian Waterlords", "slug": "hydraxian-waterlords"},
          {"id": 4701, "region": "eu", "name": "Mograine", "slug": "mograine"},
          {"id": 4703, "region": "eu", "name": "Amnennar", "slug": "amnennar"},
          {"id": 4742, "region": "eu", "name": "Ashbringer", "slug": "ashbringer"},
          {"id": 4745, "region": "eu", "name": "Transcendence", "slug": "transcendence"},
          {"id": 4749, "region": "eu", "name": "Earthshaker", "slug": "earthshaker"},
          {"id": 4811, "region": "eu", "name": "Giantstalker", "slug": "giantstalker"},
          {"id": 4813, "region": "eu", "name": "Mandokir", "slug": "mandokir"},
          {"id": 4815, "region": "eu", "name": "Thekal", "slug": "thekal"},
          {"id": 4816, "region": "eu", "name": "Jin'do", "slug": "jindo"},

          # KR Realms
          {"id": 4417, "region": "kr", "name": "Shimmering Flats", "slug": "shimmering-flats"},
          {"id": 4419, "region": "kr", "name": "Lokholar", "slug": "lokholar"},
          {"id": 4420, "region": "kr", "name": "Iceblood", "slug": "iceblood"},
          {"id": 4421, "region": "kr", "name": "Ragnaros", "slug": "ragnaros"},
          {"id": 4840, "region": "kr", "name": "Frostmourne", "slug": "frostmourne"},
      ]

      API_TOKEN = os.getenv("BLIZZARD_API_TOKEN")
      PERIOD_RANGE = range(1026, 1030)  # Test periods 1026-1028
      MAX_CONCURRENT = 50  # Reduced concurrency for comprehensive testing
      BATCH_SIZE = 50

      async def test_realm_dungeon_period(session, semaphore, realm, dungeon, period_id):
          """Test a specific realm/dungeon/period combination asynchronously"""
          async with semaphore:
              region = realm["region"]
              realm_id = realm["id"]
              dungeon_id = dungeon["id"]
              namespace = f"dynamic-classic-{region}"

              url = (
                  f"https://{region}.api.blizzard.com/data/wow/connected-realm/"
                  f"{realm_id}/mythic-leaderboard/{dungeon_id}/period/"
                  f"{period_id}?namespace={namespace}"
              )

              try:
                  async with session.get(url, timeout=10) as response:
                      result = {
                          "realm_name": realm["name"],
                          "realm_slug": realm["slug"],
                          "realm_id": realm_id,
                          "region": region,
                          "dungeon_name": dungeon["name"],
                          "dungeon_slug": dungeon["slug"],
                          "dungeon_id": dungeon_id,
                          "period_id": period_id,
                          "status_code": response.status,
                          "url": url
                      }

                      if response.status == 404:
                          result["success"] = False
                          result["error"] = "No data (404)"
                          return result

                      if response.status != 200:
                          result["success"] = False
                          result["error"] = f"HTTP {response.status}"
                          return result

                      data = await response.json()
                      leading_groups = data.get("leading_groups", [])

                      result["success"] = True
                      result["run_count"] = len(leading_groups)
                      result["has_runs"] = len(leading_groups) > 0

                      if leading_groups:
                          result["best_time"] = min(run.get("duration", float("inf")) for run in leading_groups)
                          result["most_recent"] = max(run.get("completed_timestamp", 0) for run in leading_groups)

                      return result

              except Exception as e:
                  return {
                      "realm_name": realm["name"],
                      "realm_slug": realm["slug"],
                      "realm_id": realm_id,
                      "region": region,
                      "dungeon_name": dungeon["name"],
                      "dungeon_slug": dungeon["slug"],
                      "dungeon_id": dungeon_id,
                      "period_id": period_id,
                      "status_code": None,
                      "url": url,
                      "success": False,
                      "error": str(e)
                  }

      async def main():
          if not API_TOKEN:
              print("FATAL: BLIZZARD_API_TOKEN environment variable not set.")
              sys.exit(1)

          print("=== WoW Challenge Mode Multi-Period Analyzer ===")
          print(f"Testing {len(REALMS)} realms across {len(DUNGEONS)} dungeons")
          print(f"Testing periods: {PERIOD_RANGE.start} to {PERIOD_RANGE.stop - 1}")
          print(f"Max concurrent requests: {MAX_CONCURRENT}")
          print()

          # Create all request combinations
          all_requests = []
          for realm in REALMS:
              for dungeon in DUNGEONS:
                  for period_id in PERIOD_RANGE:
                      all_requests.append((realm, dungeon, period_id))

          total_tests = len(all_requests)
          print(f"Total API endpoint tests: {total_tests}")
          print(f"({len(REALMS)} realms × {len(DUNGEONS)} dungeons × {len(PERIOD_RANGE)} periods)")
          print()

          # Store all results for analysis
          all_results = []
          failed_results = []
          successful_results = []

          # Create semaphore to limit concurrent requests
          semaphore = asyncio.Semaphore(MAX_CONCURRENT)

          # Setup async HTTP session
          timeout = aiohttp.ClientTimeout(total=15)
          headers = {"Authorization": f"Bearer {API_TOKEN}"}

          async with aiohttp.ClientSession(timeout=timeout, headers=headers) as session:
              print("Starting comprehensive API endpoint testing...")
              start_time = time.time()

              # Process in batches
              completed_count = 0
              for batch_idx in range(0, len(all_requests), BATCH_SIZE):
                  batch = all_requests[batch_idx:batch_idx + BATCH_SIZE]
                  batch_num = batch_idx // BATCH_SIZE + 1
                  total_batches = (len(all_requests) + BATCH_SIZE - 1) // BATCH_SIZE

                  print(f"Processing batch {batch_num}/{total_batches} ({len(batch)} requests)")

                  # Create tasks for this batch
                  tasks = []
                  for realm, dungeon, period_id in batch:
                      task = test_realm_dungeon_period(session, semaphore, realm, dungeon, period_id)
                      tasks.append(task)

                  # Execute batch
                  batch_results = await asyncio.gather(*tasks, return_exceptions=True)

                  # Process results
                  for result in batch_results:
                      completed_count += 1
                      if result and not isinstance(result, Exception):
                          all_results.append(result)
                          if result["success"]:
                              successful_results.append(result)
                          else:
                              failed_results.append(result)
                      elif isinstance(result, Exception):
                          print(f"    Unexpected error in batch: {result}")

                  # Progress update
                  elapsed = time.time() - start_time
                  progress = (completed_count / total_tests) * 100
                  rate = completed_count / elapsed if elapsed > 0 else 0
                  print(f"  Batch complete: {progress:.1f}% total ({completed_count}/{total_tests}) - {rate:.1f} req/s avg")
                  print(f"  Success: {len(successful_results)}, Failed: {len(failed_results)}")

                  # Sleep between batches to be API-friendly
                  if batch_idx + BATCH_SIZE < len(all_requests):
                      await asyncio.sleep(2.0)

              elapsed = time.time() - start_time
              print(f"\\nCompleted all {total_tests} requests in {elapsed:.1f}s ({total_tests/elapsed:.1f} req/s avg)")

          # Generate comprehensive report
          print(f"\n=== COMPREHENSIVE API ENDPOINT ANALYSIS ===")
          print(f"Total endpoints tested: {len(all_results)}")
          print(f"Successful endpoints: {len(successful_results)} ({len(successful_results)/len(all_results)*100:.1f}%)")
          print(f"Failed endpoints: {len(failed_results)} ({len(failed_results)/len(all_results)*100:.1f}%)")

          # Analyze failure patterns
          if failed_results:
              print(f"\n=== BROKEN ENDPOINT ANALYSIS ===")

              # Group by error type
              error_types = defaultdict(int)
              for result in failed_results:
                  error_types[result["error"]] += 1

              print("Failure breakdown by error type:")
              for error, count in sorted(error_types.items(), key=lambda x: x[1], reverse=True):
                  print(f"  {error}: {count} endpoints")

              # Group by realm
              realm_failures = defaultdict(int)
              for result in failed_results:
                  realm_failures[f"{result['realm_name']} ({result['region'].upper()})"] += 1

              print(f"\nTop 10 realms with most failures:")
              for realm, count in sorted(realm_failures.items(), key=lambda x: x[1], reverse=True)[:10]:
                  print(f"  {realm}: {count} failed endpoints")

              # Group by dungeon
              dungeon_failures = defaultdict(int)
              for result in failed_results:
                  dungeon_failures[result["dungeon_name"]] += 1

              print(f"\nDungeons with most failures:")
              for dungeon, count in sorted(dungeon_failures.items(), key=lambda x: x[1], reverse=True):
                  print(f"  {dungeon}: {count} failed endpoints")

              # Group by period
              period_failures = defaultdict(int)
              for result in failed_results:
                  period_failures[result["period_id"]] += 1

              print(f"\nPeriods with most failures:")
              for period, count in sorted(period_failures.items(), key=lambda x: x[1], reverse=True):
                  print(f"  Period {period}: {count} failed endpoints")

              # Focus on realm/dungeon combinations with no data across all periods
              print(f"\n=== REALM/DUNGEON COMBINATIONS WITH NO DATA ===")

              # Group failed results by realm+dungeon combination
              realm_dungeon_failures = defaultdict(set)  # key: (realm_slug, dungeon_slug), value: set of failed periods

              for result in failed_results:
                  key = (result["realm_slug"], result["dungeon_slug"])
                  realm_dungeon_failures[key].add(result["period_id"])

              # Find combinations that failed across ALL tested periods
              all_periods = set(PERIOD_RANGE)
              completely_broken = []
              partially_broken = []

              for (realm_slug, dungeon_slug), failed_periods in realm_dungeon_failures.items():
                  if failed_periods == all_periods:
                      # Failed in ALL periods
                      completely_broken.append((realm_slug, dungeon_slug))
                  else:
                      # Failed in some periods
                      partially_broken.append((realm_slug, dungeon_slug, failed_periods))

              print(f"\nCompletely broken realm/dungeon combinations (no data in ANY period {min(PERIOD_RANGE)}-{max(PERIOD_RANGE)}):")
              if completely_broken:
                  for i, (realm_slug, dungeon_slug) in enumerate(sorted(completely_broken), 1):
                      # Find region for this realm
                      realm_info = next((r for r in REALMS if r["slug"] == realm_slug), None)
                      region = realm_info["region"].upper() if realm_info else "??"
                      print(f"  {i:2d}. {realm_slug} ({region}) + {dungeon_slug}")
              else:
                  print("  None! All realm/dungeon combinations have data in at least one period.")

              print(f"\nPartially broken realm/dungeon combinations:")
              if partially_broken:
                  for i, (realm_slug, dungeon_slug, failed_periods) in enumerate(sorted(partially_broken), 1):
                      # Find region for this realm
                      realm_info = next((r for r in REALMS if r["slug"] == realm_slug), None)
                      region = realm_info["region"].upper() if realm_info else "??"
                      working_periods = all_periods - failed_periods
                      print(f"  {i:2d}. {realm_slug} ({region}) + {dungeon_slug}")
                      print(f"      Missing data in periods: {sorted(failed_periods)}")
                      print(f"      Has data in periods: {sorted(working_periods)}")
              else:
                  print("  None! All realm/dungeon combinations work in all periods.")

          else:
              print("\\nAll endpoints are working! 🎉")

          # Summary of known problematic cases
          print(f"\n=== SPECIFIC PROBLEMATIC CASES ===")
          gehennas_mogushan = [r for r in failed_results if r["realm_slug"] == "gehennas" and r["dungeon_slug"] == "mogu-shan-palace"]
          if gehennas_mogushan:
              print(f"Gehennas + Mogu'shan Palace failures: {len(gehennas_mogushan)}")
              for result in gehennas_mogushan:
                  print(f"  Period {result['period_id']}: {result['error']}")
          else:
              print("Gehennas + Mogu'shan Palace: All working!")

      if __name__ == "__main__":
          asyncio.run(main())
    '';
in
  multiPeriodAnalyzerScript

