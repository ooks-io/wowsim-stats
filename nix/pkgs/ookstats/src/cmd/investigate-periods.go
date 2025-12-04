package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
)

// investigatePeriodsCmd analyzes period overlap and leaderboard size limits
var investigatePeriodsCmd = &cobra.Command{
	Use:   "investigate-periods",
	Short: "Investigate period overlap and API pagination behavior",
	Long:  `Fetches leaderboards across multiple periods for a specific realm/dungeon to analyze duplication and run limits.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		realmSlug, _ := cmd.Flags().GetString("realm")
		dungeonSlug, _ := cmd.Flags().GetString("dungeon")
		periodRange, _ := cmd.Flags().GetString("period-range")

		if realmSlug == "" || dungeonSlug == "" || periodRange == "" {
			return fmt.Errorf("--realm, --dungeon, and --period-range are required")
		}

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("blizzard client: %w", err)
		}

		// Parse period range (e.g., "995-1036")
		var startPeriod, endPeriod int
		if _, err := fmt.Sscanf(periodRange, "%d-%d", &startPeriod, &endPeriod); err != nil {
			return fmt.Errorf("invalid period range format (expected: START-END): %w", err)
		}

		// Find realm and dungeon info
		allRealms := blizzard.GetAllRealms()
		var realmInfo blizzard.RealmInfo
		var found bool
		for _, r := range allRealms {
			if r.Slug == realmSlug {
				realmInfo = r
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("realm not found: %s", realmSlug)
		}

		_, dungeons := blizzard.GetHardcodedPeriodAndDungeons()
		var dungeonInfo blizzard.DungeonInfo
		found = false
		for _, d := range dungeons {
			if d.Slug == dungeonSlug {
				dungeonInfo = d
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("dungeon not found: %s", dungeonSlug)
		}

		fmt.Printf("=== Period Overlap Investigation ===\n")
		fmt.Printf("Realm: %s (%s, region: %s)\n", realmInfo.Name, realmInfo.Slug, realmInfo.Region)
		fmt.Printf("Dungeon: %s (%s)\n", dungeonInfo.Name, dungeonInfo.Slug)
		fmt.Printf("Period Range: %d-%d (%d periods)\n\n", startPeriod, endPeriod, endPeriod-startPeriod+1)

		// Track unique runs by completed timestamp + team members
		type runSignature struct {
			timestamp     int64
			duration      int
			level         int
			teamSignature string // sorted player IDs to identify unique teams
		}
		allRuns := make(map[runSignature][]int) // signature -> list of periods it appeared in
		periodRunCounts := make(map[int]int)    // period -> run count

		// Fetch each period
		for period := startPeriod; period <= endPeriod; period++ {
			periodStr := fmt.Sprintf("%d", period)
			lb, err := client.FetchLeaderboardData(realmInfo, dungeonInfo, periodStr)
			if err != nil {
				fmt.Printf("Period %d: ERROR - %v\n", period, err)
				continue
			}

			runCount := len(lb.LeadingGroups)
			periodRunCounts[period] = runCount
			fmt.Printf("Period %d: %d runs (period start: %s, period end: %s)\n",
				period,
				runCount,
				time.UnixMilli(lb.PeriodStartTimestamp).UTC().Format("2006-01-02 15:04:05"),
				time.UnixMilli(lb.PeriodEndTimestamp).UTC().Format("2006-01-02 15:04:05"),
			)

			// Track each run
			for _, run := range lb.LeadingGroups {
				// Build team signature from player IDs
				playerIDs := make([]int, 0, len(run.Members))
				for _, m := range run.Members {
					if id, ok := m.GetPlayerID(); ok {
						playerIDs = append(playerIDs, id)
					}
				}
				// Sort player IDs to create consistent team signature
				// Simple bubble sort since we only have 5 players
				for i := 0; i < len(playerIDs)-1; i++ {
					for j := i + 1; j < len(playerIDs); j++ {
						if playerIDs[i] > playerIDs[j] {
							playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
						}
					}
				}
				teamSig := fmt.Sprintf("%v", playerIDs)

				sig := runSignature{
					timestamp:     run.CompletedTimestamp,
					duration:      run.Duration,
					level:         run.KeystoneLevel,
					teamSignature: teamSig,
				}
				allRuns[sig] = append(allRuns[sig], period)
			}

			time.Sleep(100 * time.Millisecond) // rate limit
		}

		// Analysis
		fmt.Printf("\n=== Analysis ===\n")
		fmt.Printf("Total unique runs across all periods: %d\n", len(allRuns))

		// Count duplicates
		duplicates := 0
		for _, periods := range allRuns {
			if len(periods) > 1 {
				duplicates++
			}
		}
		fmt.Printf("Runs appearing in multiple periods: %d (%.1f%%)\n",
			duplicates,
			100.0*float64(duplicates)/float64(len(allRuns)),
		)

		// Show per-period stats
		fmt.Printf("\n=== Per-Period Stats ===\n")
		for period := startPeriod; period <= endPeriod; period++ {
			count, ok := periodRunCounts[period]
			if ok {
				fmt.Printf("Period %d: %d runs\n", period, count)
			} else {
				fmt.Printf("Period %d: NO DATA\n", period)
			}
		}

		// Show examples of duplicate runs
		fmt.Printf("\n=== Example Duplicate Runs (first 5) ===\n")
		shown := 0
		for sig, periods := range allRuns {
			if len(periods) > 1 && shown < 5 {
				fmt.Printf("Run at %s (duration: %dms, level: +%d) appears in periods: %v\n",
					time.UnixMilli(sig.timestamp).UTC().Format("2006-01-02 15:04:05"),
					sig.duration,
					sig.level,
					periods,
				)
				shown++
			}
		}

		// Check if any period has 500 runs (the suspected cap)
		fmt.Printf("\n=== 500-Run Cap Analysis ===\n")
		for period := startPeriod; period <= endPeriod; period++ {
			count, ok := periodRunCounts[period]
			if ok && count >= 500 {
				fmt.Printf("⚠️  Period %d has %d runs (AT OR ABOVE 500 LIMIT)\n", period, count)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(investigatePeriodsCmd)
	investigatePeriodsCmd.Flags().String("realm", "", "Realm slug (e.g., arugal-au)")
	investigatePeriodsCmd.Flags().String("dungeon", "", "Dungeon slug (e.g., scarlet-halls)")
	investigatePeriodsCmd.Flags().String("period-range", "", "Period range to test (e.g., 995-1036)")
}
