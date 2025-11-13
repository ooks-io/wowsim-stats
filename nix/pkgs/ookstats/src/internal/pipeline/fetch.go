package pipeline

import (
	"context"
	"fmt"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"strings"
	"time"
)

// FetchCMOptions contains options for fetching challenge mode leaderboards
type FetchCMOptions struct {
	Verbose       bool
	Regions       []string
	Realms        []string
	Dungeons      []string
	Periods       []string
	Concurrency   int
	Timeout       time.Duration
}

// FetchCMResult contains statistics from the fetch operation
type FetchCMResult struct {
	TotalRuns    int
	TotalPlayers int
	Duration     time.Duration
}

// FetchChallengeMode fetches challenge mode leaderboard data for specified realms/dungeons/periods
func FetchChallengeMode(db *database.DatabaseService, client *blizzard.Client, opts FetchCMOptions) (*FetchCMResult, error) {
	// Get dungeons and realms from hardcoded lists
	_, dungeons := blizzard.GetHardcodedPeriodAndDungeons()
	allRealms := blizzard.GetAllRealms()

	fmt.Printf("Dungeons: %d, Realms: %d\n", len(dungeons), len(allRealms))

	// Apply region filter
	if len(opts.Regions) > 0 {
		allowed := make(map[string]bool)
		for _, r := range opts.Regions {
			allowed[strings.TrimSpace(r)] = true
		}
		for slug, info := range allRealms {
			if !allowed[info.Region] {
				delete(allRealms, slug)
			}
		}
	}

	// Apply realm filter
	if len(opts.Realms) > 0 {
		allowed := make(map[string]bool)
		for _, s := range opts.Realms {
			s = strings.TrimSpace(s)
			if s != "" {
				allowed[s] = true
			}
		}
		filtered := make(map[string]blizzard.RealmInfo)
		for key, info := range allRealms {
			if allowed[key] || allowed[info.Slug] {
				filtered[key] = info
			}
		}
		allRealms = filtered
	}

	// Apply dungeon filter
	if len(opts.Dungeons) > 0 {
		allowed := make(map[string]bool)
		for _, s := range opts.Dungeons {
			allowed[strings.TrimSpace(s)] = true
		}
		filtered := make([]blizzard.DungeonInfo, 0, len(dungeons))
		for _, d := range dungeons {
			idStr := fmt.Sprintf("%d", d.ID)
			if allowed[idStr] || allowed[d.Slug] {
				filtered = append(filtered, d)
			}
		}
		if len(filtered) > 0 {
			dungeons = filtered
		}
	}

	// Pre-populate reference data
	fmt.Printf("Pre-populating reference data...\n")
	fmt.Printf("  - Ensuring dungeons (%d)\n", len(dungeons))
	if err := db.EnsureDungeonsOnce(dungeons); err != nil {
		return nil, fmt.Errorf("failed to ensure dungeons: %w", err)
	}
	fmt.Printf("  [OK] Dungeons ensured\n")
	fmt.Printf("  - Ensuring realms (%d)\n", len(allRealms))
	if err := db.EnsureRealmsBatch(allRealms); err != nil {
		return nil, fmt.Errorf("failed to ensure realms: %w", err)
	}
	fmt.Printf("  [OK] Realms ensured\n")
	fmt.Printf("Reference data populated for %d realms and %d dungeons\n", len(allRealms), len(dungeons))

	// Group realms by region
	realmsByRegion := make(map[string]map[string]blizzard.RealmInfo)
	for slug, info := range allRealms {
		if realmsByRegion[info.Region] == nil {
			realmsByRegion[info.Region] = make(map[string]blizzard.RealmInfo)
		}
		realmsByRegion[info.Region][slug] = info
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	totalRuns := 0
	totalPlayers := 0
	sweepStart := time.Now()

	// Process each region independently
	for region, regionRealms := range realmsByRegion {
		fmt.Printf("\n========== Region: %s (%d realms) ==========\n", strings.ToUpper(region), len(regionRealms))

		// Determine periods for this region
		var periods []string
		var err error

		if len(opts.Periods) > 0 {
			// User-specified periods
			periods = opts.Periods
			fmt.Printf("Using user-specified periods: %v (%d periods)\n", periods, len(periods))
		} else {
			// Fetch periods dynamically from Blizzard API for this region
			fmt.Printf("Fetching period list dynamically from Blizzard API for %s...\n", strings.ToUpper(region))
			periods, err = client.GetDynamicPeriodList(region)
			if err != nil {
				fmt.Printf("Failed to fetch period list for %s: %v - skipping region\n", strings.ToUpper(region), err)
				continue
			}
		}

		if len(periods) == 0 {
			fmt.Printf("No periods to process for %s - skipping region\n", strings.ToUpper(region))
			continue
		}

		// Period sweep for this region
		fmt.Printf("Starting period sweep for %s: %d periods\n", strings.ToUpper(region), len(periods))
		for _, period := range periods {
			fmt.Printf("\n--- %s Period %s ---\n", strings.ToUpper(region), period)
			res := client.FetchAllRealmsConcurrent(ctx, regionRealms, dungeons, period)
			runs, players, berr := db.BatchProcessFetchResults(ctx, res)
			if berr != nil {
				fmt.Printf("Batch errors in %s period %s: %v\n", strings.ToUpper(region), period, berr)
			}
			fmt.Printf("%s Period %s -> inserted runs: %d, new players: %d\n", strings.ToUpper(region), period, runs, players)
			totalRuns += runs
			totalPlayers += players
		}
	}

	duration := time.Since(sweepStart)
	fmt.Printf("\n========== Sweep complete in %v ==========\n", duration)

	// Update fetch metadata
	if err := db.UpdateFetchMetadata("challenge_mode_leaderboard", totalRuns, totalPlayers); err != nil {
		return nil, fmt.Errorf("failed to update fetch metadata: %w", err)
	}

	return &FetchCMResult{
		TotalRuns:    totalRuns,
		TotalPlayers: totalPlayers,
		Duration:     duration,
	}, nil
}

// FetchProfilesOptions contains options for fetching player profiles
type FetchProfilesOptions struct {
	Verbose    bool
	BatchSize  int
	MaxPlayers int
}

// FetchProfilesResult contains statistics from the profile fetch operation
type FetchProfilesResult struct {
	TotalProfiles  int
	TotalEquipment int
	ProcessedCount int
	Duration       time.Duration
}

// FetchPlayerProfiles fetches detailed player profile data including equipment
func FetchPlayerProfiles(db *database.DatabaseService, client *blizzard.Client, opts FetchProfilesOptions) (*FetchProfilesResult, error) {
	// Get eligible players (9/9 completion)
	fmt.Println("Finding eligible players with complete coverage (9/9 dungeons)...")
	players, err := db.GetEligiblePlayersForProfileFetch()
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible players: %w", err)
	}

	if len(players) == 0 {
		fmt.Println("No eligible players found. Run 'ookstats process players' first to generate player profiles.")
		return &FetchProfilesResult{}, nil
	}

	fmt.Printf("Found %d eligible players with 9/9 completion\n", len(players))

	// Apply max players limit
	if opts.MaxPlayers > 0 && len(players) > opts.MaxPlayers {
		players = players[:opts.MaxPlayers]
		fmt.Printf("Limited to first %d players due to max-players limit\n", opts.MaxPlayers)
	}

	// Default batch size
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 20
	}

	fmt.Printf("Processing %d players in batches of %d with 20 concurrent requests\n", len(players), batchSize)
	fmt.Printf("\nStarting player profile fetching...\n")

	startTime := time.Now()
	totalProfiles := 0
	totalEquipment := 0
	processedCount := 0

	// Process in batches to avoid overwhelming the API
	for i := 0; i < len(players); i += batchSize {
		end := i + batchSize
		if end > len(players) {
			end = len(players)
		}
		batch := players[i:end]

		batchNumber := (i / batchSize) + 1
		totalBatches := (len(players) + batchSize - 1) / batchSize

		fmt.Printf("\n--- Batch %d/%d (%d players) ---\n", batchNumber, totalBatches, len(batch))

		// Fetch profiles concurrently for this batch with fallback attempts
		sem := make(chan struct{}, 20)
		type out struct {
			profiles  int
			equipment int
			err       error
		}
		outCh := make(chan out, len(batch))
		timestamp := time.Now().UnixMilli()

		for _, p := range batch {
			sem <- struct{}{}
			go func(p blizzard.PlayerInfo) {
				defer func() { <-sem }()

				// Build candidate list
				region := p.Region
				realm := blizzard.NormalizeRealmSlug(region, p.RealmSlug)
				name := p.Name
				tried := map[string]bool{}
				candidates := [][2]string{{realm, name}}

				// Connected-realm sweep
				if slugs, err := db.GetConnectedRealmSlugs(region, realm); err == nil {
					for _, s := range slugs {
						s = blizzard.NormalizeRealmSlug(region, s)
						if s != realm {
							candidates = append(candidates, [2]string{s, name})
						}
					}
				}

				// Last-run realm heuristic
				if lrRegion, lrSlug, _, err := db.GetLastRunRealmForPlayer(p.ID); err == nil && lrRegion != "" && lrSlug != "" {
					if lrRegion == region {
						lrSlug = blizzard.NormalizeRealmSlug(region, lrSlug)
						candidates = append(candidates, [2]string{lrSlug, name})
					}
				}

				// Attempt candidates
				var profs, items int
				var finalErr error
				for _, c := range candidates {
					key := c[0] + "|" + c[1]
					if tried[key] {
						continue
					}
					tried[key] = true

					// Fetch summary first; if it 404s, skip to next candidate
					sum, err := client.FetchCharacterSummary(c[1], c[0], region)
					if err != nil {
						// try next candidate
						finalErr = err
						continue
					}
					eq, err2 := client.FetchCharacterEquipment(c[1], c[0], region)
					if err2 != nil {
						finalErr = err2
					}
					med, err3 := client.FetchCharacterMedia(c[1], c[0], region)
					if err3 != nil {
						finalErr = err3
					}

					// Insert profile data
					res := blizzard.PlayerProfileResult{
						PlayerID:   p.ID,
						PlayerName: c[1],
						RealmSlug:  c[0],
						Region:     region,
						Summary:    sum,
						Equipment:  eq,
						Media:      med,
					}
					pr, eqc, derr := db.InsertPlayerProfileData(res, timestamp)
					if derr != nil {
						finalErr = derr
						continue
					}
					profs += pr
					items += eqc
					// Update players table to resolved identity for this build
					_ = db.UpdatePlayerIdentity(p.ID, c[1], region, c[0])
					break
				}
				outCh <- out{profiles: profs, equipment: items, err: finalErr}
			}(p)
		}

		// Wait for batch: collect exactly len(batch) results
		batchProfiles := 0
		batchEquipment := 0
		for k := 0; k < len(batch); k++ {
			r := <-outCh
			processedCount++
			if r.err != nil && r.profiles == 0 && r.equipment == 0 {
				// only log errors if nothing was inserted
				if opts.Verbose {
					fmt.Printf("  [ERROR] profile fetch failed: %v\n", r.err)
				}
			}
			batchProfiles += r.profiles
			batchEquipment += r.equipment
		}
		close(outCh)

		totalProfiles += batchProfiles
		totalEquipment += batchEquipment

		elapsed := time.Since(startTime)
		fmt.Printf("  -> Batch %d complete: %d profiles, %d items (Total: %d/%d players, %.1f players/min)\n",
			batchNumber, batchProfiles, batchEquipment, processedCount, len(players),
			float64(processedCount)/elapsed.Minutes())

		// Small delay between batches to be respectful to the API
		if i+batchSize < len(players) {
			fmt.Printf("  [INFO] Waiting 1 second before next batch...\n")
			time.Sleep(1 * time.Second)
		}
	}

	elapsed := time.Since(startTime)
	return &FetchProfilesResult{
		TotalProfiles:  totalProfiles,
		TotalEquipment: totalEquipment,
		ProcessedCount: processedCount,
		Duration:       elapsed,
	}, nil
}
