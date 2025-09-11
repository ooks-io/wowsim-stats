{
  writers,
  python3Packages,
  ...
}: let
  periodAnalyzerScript =
    writers.writePython3Bin "period-analyzer" {
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

      # Test with one realm - EU Argent Dawn (connected realm 4477)
      TEST_REALM = {
          "id": 4476,
          "name": "Gehennas",
          "region": "eu"
      }

      # Test single dungeon for detailed analysis
      TEST_DUNGEON = {"id": 2, "name": "Gate"}  # Change this to test different dungeons

      API_TOKEN = os.getenv("BLIZZARD_API_TOKEN")
      PERIOD_RANGE = range(1000, 1029)  # Test periods 1-1027
      MAX_CONCURRENT = 75  # 75 concurrent requests per batch
      BATCH_SIZE = 75  # 75 requests per batch

      async def test_period_dungeon(session, semaphore, period_id, dungeon_id):
          """Test a specific period/dungeon combination asynchronously"""
          async with semaphore:  # Limit concurrent requests
              region = TEST_REALM["region"]
              realm_id = TEST_REALM["id"]
              namespace = f"dynamic-classic-{region}"

              url = (
                  f"https://{region}.api.blizzard.com/data/wow/connected-realm/"
                  f"{realm_id}/mythic-leaderboard/{dungeon_id}/period/"
                  f"{period_id}?namespace={namespace}"
              )

              try:
                  async with session.get(url, timeout=10) as response:
                      if response.status == 404:
                          return None  # No data for this period/dungeon

                      response.raise_for_status()
                      data = await response.json()

                      leading_groups = data.get('leading_groups', [])
                      if not leading_groups:
                          return None

                      # Extract stats
                      stats = {
                          'period_id': period_id,
                          'dungeon_id': dungeon_id,
                          'period_start': data.get('period_start_timestamp'),
                          'period_end': data.get('period_end_timestamp'),
                          'run_count': len(leading_groups),
                          'best_time': min(run.get('duration', float('inf')) for run in leading_groups),
                          'most_recent': max(run.get('completed_timestamp', 0) for run in leading_groups),
                          'data_available': True
                      }
                      return stats

              except Exception as e:
                  print(f"    Error testing period {period_id}, dungeon {dungeon_id}: {e}")
                  return None

      def format_duration(ms):
          """Format duration from milliseconds to readable format"""
          if ms == float('inf'):
              return "N/A"
          minutes = ms // 60000
          seconds = (ms % 60000) // 1000
          return f"{minutes}:{seconds:02d}"

      def format_timestamp(ts):
          """Format timestamp to readable date"""
          if ts == 0:
              return "N/A"
          return time.strftime('%Y-%m-%d %H:%M:%S UTC', time.gmtime(ts / 1000))

      async def main():
          if not API_TOKEN:
              print("FATAL: BLIZZARD_API_TOKEN environment variable not set.")
              sys.exit(1)

          print("=== WoW Challenge Mode Period Analyzer (Single Dungeon) ===")
          print(f"Testing realm: {TEST_REALM['name']} ({TEST_REALM['region'].upper()}) - ID {TEST_REALM['id']}")
          print(f"Testing dungeon: {TEST_DUNGEON['name']} (ID {TEST_DUNGEON['id']})")
          print(f"Testing periods: {PERIOD_RANGE.start} to {PERIOD_RANGE.stop - 1}")
          print(f"Max concurrent requests: {MAX_CONCURRENT}")
          print()

          # Store results
          results = defaultdict(dict)  # results[period_id][dungeon_id] = stats

          total_tests = len(PERIOD_RANGE)  # Only testing one dungeon now
          print(f"Total requests to make: {total_tests}")

          # Create semaphore to limit concurrent requests
          semaphore = asyncio.Semaphore(MAX_CONCURRENT)

          # Setup async HTTP session
          timeout = aiohttp.ClientTimeout(total=10)
          headers = {"Authorization": f"Bearer {API_TOKEN}"}

          async with aiohttp.ClientSession(timeout=timeout, headers=headers) as session:
              print("Starting batched async requests...")
              start_time = time.time()

              # Create all request parameters (single dungeon)
              all_requests = []
              for period_id in PERIOD_RANGE:
                  all_requests.append((period_id, TEST_DUNGEON["id"]))

              # Process in batches
              completed_count = 0
              for batch_idx in range(0, len(all_requests), BATCH_SIZE):
                  batch = all_requests[batch_idx:batch_idx + BATCH_SIZE]
                  batch_num = batch_idx // BATCH_SIZE + 1
                  total_batches = (len(all_requests) + BATCH_SIZE - 1) // BATCH_SIZE

                  print(f"Processing batch {batch_num}/{total_batches} ({len(batch)} requests)")

                  # Create tasks for this batch
                  tasks = []
                  for period_id, dungeon_id in batch:
                      task = test_period_dungeon(session, semaphore, period_id, dungeon_id)
                      tasks.append(task)

                  # Execute batch
                  batch_results = await asyncio.gather(*tasks, return_exceptions=True)

                  # Process results
                  for result in batch_results:
                      completed_count += 1
                      if result and not isinstance(result, Exception):
                          period_id = result['period_id']
                          dungeon_id = result['dungeon_id']
                          results[period_id][dungeon_id] = result
                      elif isinstance(result, Exception):
                          print(f"    Error in batch: {result}")

                  # Progress update
                  elapsed = time.time() - start_time
                  progress = (completed_count / total_tests) * 100
                  rate = completed_count / elapsed if elapsed > 0 else 0
                  print(f"  Batch complete: {progress:.1f}% total ({completed_count}/{total_tests}) - {rate:.1f} req/s avg")

                  # Sleep 1 second between batches (except for last batch)
                  if batch_idx + BATCH_SIZE < len(all_requests):
                      await asyncio.sleep(1.0)  # 1 second delay between batches of 75

              elapsed = time.time() - start_time
              print(f"\\nCompleted all {total_tests} requests in {elapsed:.1f}s ({total_tests/elapsed:.1f} req/s avg)")

          print(f"\n=== ANALYSIS RESULTS FOR {TEST_DUNGEON['name'].upper()} ===")

          if not results:
              print("No data found in any period!")
              return

          # Find periods with data
          periods_with_data = sorted(results.keys())
          print(f"Periods with data: {len(periods_with_data)} out of {len(PERIOD_RANGE)} tested")
          print(f"Range: {min(periods_with_data)} to {max(periods_with_data)}")

          # Find best time and most recent across all periods
          best_time = float('inf')
          best_time_period = None
          most_recent_time = 0
          most_recent_period = None

          period_details = []

          for period_id in sorted(periods_with_data):
              dungeon_id = TEST_DUNGEON["id"]
              if dungeon_id in results[period_id]:
                  stats = results[period_id][dungeon_id]
                  period_details.append({
                      'period': period_id,
                      'run_count': stats['run_count'],
                      'best_time': stats['best_time'],
                      'most_recent': stats['most_recent'],
                      'period_start': stats.get('period_start'),
                      'period_end': stats.get('period_end')
                  })

                  if stats['best_time'] < best_time:
                      best_time = stats['best_time']
                      best_time_period = period_id

                  if stats['most_recent'] > most_recent_time:
                      most_recent_time = stats['most_recent']
                      most_recent_period = period_id

          # Find oldest completion time
          oldest_time = float('inf')
          oldest_periods = []

          for detail in period_details:
              if detail['most_recent'] < oldest_time:
                  oldest_time = detail['most_recent']
                  oldest_periods = [detail['period']]
              elif detail['most_recent'] == oldest_time:
                  oldest_periods.append(detail['period'])

          # Periods with no data
          all_periods = set(PERIOD_RANGE)
          periods_with_data_set = set(periods_with_data)
          periods_with_no_data = sorted(all_periods - periods_with_data_set)

          print(f"\n--- SUMMARY ---")
          print(f"Period with best time: {best_time_period} ({format_duration(best_time)})")
          print(f"Period with latest time: {most_recent_period}")
          print(f"Period(s) with oldest time: {oldest_periods}")
          no_data_preview = periods_with_no_data[:10]
          ellipsis = "..." if len(periods_with_no_data) > 10 else ""
          print(f"Periods with no times: {no_data_preview}{ellipsis} ({len(periods_with_no_data)} total)")

      if __name__ == "__main__":
          asyncio.run(main())
    '';
in
  periodAnalyzerScript
