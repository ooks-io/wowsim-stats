package database

import (
    "context"
    "database/sql"
    "fmt"
    "sort"
    "strings"
    "time"
    "ookstats/internal/blizzard"
    "ookstats/internal/utils"
    "sync"
)

// EnsureReferenceData ensures that realm and dungeon reference data exists in the database
func (ds *DatabaseService) EnsureReferenceData(realmInfo blizzard.RealmInfo, dungeons []blizzard.DungeonInfo) error {
	// insert realm data
	realmQuery := `
		INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := ds.db.Exec(realmQuery, realmInfo.Slug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
	if err != nil {
		return fmt.Errorf("failed to insert realm data: %w", err)
	}

	// insert dungeon data
	dungeonQuery := `
		INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id)
		VALUES (?, ?, ?, ?)
	`

	for _, dungeon := range dungeons {
		_, err := ds.db.Exec(dungeonQuery, dungeon.ID, dungeon.Slug, dungeon.Name, dungeon.ID)
		if err != nil {
			return fmt.Errorf("failed to insert dungeon data for %s: %w", dungeon.Name, err)
		}
	}

	return nil
}

// EnsureDungeonsOnce inserts all known dungeons once (idempotent). Optimized for remote DBs.
func (ds *DatabaseService) EnsureDungeonsOnce(dungeons []blizzard.DungeonInfo) error {
    if len(dungeons) == 0 {
        return nil
    }
    // Build a single INSERT OR IGNORE with multi-row VALUES to reduce round trips
    // INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id) VALUES (..),(..)...
    var b strings.Builder
    args := make([]any, 0, len(dungeons)*4)
    b.WriteString("INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id) VALUES ")
    for i, d := range dungeons {
        if i > 0 {
            b.WriteString(",")
        }
        b.WriteString("(?, ?, ?, ?)")
        args = append(args, d.ID, d.Slug, d.Name, d.ID)
    }
    _, err := ds.db.Exec(b.String(), args...)
    if err != nil {
        return fmt.Errorf("failed to ensure dungeons: %w", err)
    }
    return nil
}

// EnsureRealmsBatch inserts/updates all known realms in a single transaction with a prepared statement
func (ds *DatabaseService) EnsureRealmsBatch(realms map[string]blizzard.RealmInfo) error {
    if len(realms) == 0 {
        return nil
    }
    tx, err := ds.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
        VALUES (?, ?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    // Insert each known realm once
    // Note: caller should have set realmInfo.Slug
    // This minimizes network round trips for remote DBs
    // and is idempotent thanks to OR IGNORE
    // Also allows partial success without failing entire batch
    keys := make([]string, 0, len(realms))
    for k := range realms {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    total := len(keys)
    for i, slug := range keys {
        ri := realms[slug]
        if _, err := stmt.Exec(ri.Slug, ri.Name, ri.Region, ri.ID, ri.ParentRealmSlug); err != nil {
            return fmt.Errorf("failed to insert realm %s: %w", ri.Slug, err)
        }
        // light progress every 10 items
        if (i+1)%10 == 0 || i+1 == total {
            fmt.Printf("    - Ensured %d/%d realms\n", i+1, total)
        }
    }

    if err := tx.Commit(); err != nil {
        return err
    }
    return nil
}

// InsertLeaderboardData inserts leaderboard data and returns the number of runs and players inserted
func (ds *DatabaseService) InsertLeaderboardData(leaderboard *blizzard.LeaderboardResponse, realmInfo blizzard.RealmInfo, dungeon blizzard.DungeonInfo) (int, int, error) {
	if leaderboard == nil || len(leaderboard.LeadingGroups) == 0 {
		return 0, 0, nil
	}

    // Get realm and dungeon IDs
    realmID, err := ds.GetRealmIDByRegionAndSlug(realmInfo.Region, realmInfo.Slug)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get realm ID: %w", err)
	}
	if realmID == 0 {
		return 0, 0, fmt.Errorf("realm not found: %s", realmInfo.Slug)
	}

	dungeonID, err := ds.GetDungeonID(dungeon.Slug)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get dungeon ID: %w", err)
	}
	if dungeonID == 0 {
		return 0, 0, fmt.Errorf("dungeon not found: %s", dungeon.Slug)
	}

	runsInserted := 0
	playersInserted := 0
	allRealms := blizzard.GetAllRealms()

	// begin transaction
	tx, err := ds.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, run := range leaderboard.LeadingGroups {
		// extract player IDs for team signature
		var playerIDs []int
		for _, member := range run.Members {
			if id, ok := member.GetPlayerID(); ok {
				playerIDs = append(playerIDs, id)
			}
		}

		if len(playerIDs) == 0 {
			continue // skip runs with no valid player IDs
		}

		teamSignature := utils.ComputeTeamSignature(playerIDs)

		// insert run with team signature to prevent duplicates
		runQuery := `
			INSERT OR IGNORE INTO challenge_runs
			(duration, completed_timestamp, keystone_level, dungeon_id, realm_id, period_id, period_start_timestamp, period_end_timestamp, team_signature)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := tx.Exec(runQuery,
			run.Duration,
			run.CompletedTimestamp,
			run.KeystoneLevel,
			dungeonID,
			realmID,
			leaderboard.Period,
			leaderboard.PeriodStartTimestamp,
			leaderboard.PeriodEndTimestamp,
			teamSignature,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert run: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to check rows affected: %w", err)
		}

		if rowsAffected == 0 {
			// run already exists (due to OR IGNORE), skip member insertion
			continue
		}

		runID, err := result.LastInsertId()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get run ID: %w", err)
		}

		runsInserted++

		// insert team members
		for _, member := range run.Members {
			playerID, hasPlayerID := member.GetPlayerID()
			playerName, _ := member.GetPlayerName()
			playerRealmSlug, hasRealmSlug := member.GetRealmSlug()

			if !hasPlayerID {
				continue // skip members without player ID
			}

			// get or create player realm
			var playerRealmID int
			if hasRealmSlug {
				playerRealmID, err = ds.getOrCreateRealmByRegion(tx, playerRealmSlug, realmInfo.Region, allRealms)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to get/create player realm: %w", err)
				}
			} else {
				playerRealmID = realmID // Fallback to run realm
			}

			// insert or ignore player
			playerQuery := `
				INSERT INTO players (id, name, name_lower, realm_id)
				VALUES (?, ?, lower(?), ?)
				ON CONFLICT(id) DO UPDATE SET
				  name = excluded.name,
				  name_lower = lower(excluded.name),
				  realm_id = excluded.realm_id
				WHERE excluded.name != name OR excluded.realm_id != realm_id
			`

			playerResult, err := tx.Exec(playerQuery, playerID, playerName, playerName, playerRealmID)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to insert player: %w", err)
			}

			// check if player was actually inserted (new player)
			rowsAffected, err := playerResult.RowsAffected()
			if err != nil {
				return 0, 0, fmt.Errorf("failed to get rows affected: %w", err)
			}
			if rowsAffected > 0 {
				playersInserted++
			}

			// get spec ID and faction
			specID, _ := member.GetSpecID()
			faction, _ := member.GetFaction()

			// link player to run
			memberQuery := `
				INSERT INTO run_members (run_id, player_id, spec_id, faction)
				VALUES (?, ?, ?, ?)
			`

			var specPtr *int
			var factionPtr *string
			if specID > 0 {
				specPtr = &specID
			}
			if faction != "" {
				factionPtr = &faction
			}

			_, err = tx.Exec(memberQuery, runID, playerID, specPtr, factionPtr)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to insert run member: %w", err)
			}
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return runsInserted, playersInserted, nil
}

// getOrCreateRealm gets an existing realm ID or creates a new realm record
func (ds *DatabaseService) getOrCreateRealm(tx *sql.Tx, realmSlug string, allRealms map[string]blizzard.RealmInfo) (int, error) {
	// try to get existing realm
	var realmID int
	err := tx.QueryRow("SELECT id FROM realms WHERE slug = ?", realmSlug).Scan(&realmID)
	if err == nil {
		return realmID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// realm doesn't exist, create it
	if realmInfo, exists := allRealms[realmSlug]; exists {
		// insert known realm
		result, err := tx.Exec(`
			INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
			VALUES (?, ?, ?, ?, ?)
		`, realmSlug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
		if err != nil {
			return 0, err
		}

		id, err := result.LastInsertId()
		return int(id), err
	} else {
		// insert unknown realm as placeholder
		result, err := tx.Exec(`
			INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
			VALUES (?, ?, ?, NULL, ?)
		`, realmSlug, utils.Slugify(realmSlug), "unknown", "")
		if err != nil {
			return 0, err
		}

		id, err := result.LastInsertId()
		return int(id), err
	}
}

// BatchProcessFetchResults processes fetch results concurrently with transaction batching
func (ds *DatabaseService) BatchProcessFetchResults(ctx context.Context, results <-chan blizzard.FetchResult) (int, int, error) {
	totalRuns := 0
	totalPlayers := 0
	processedCount := 0
	errorCount := 0

	// channel to collect results for batching
	batchChan := make(chan blizzard.FetchResult, 50) // Buffer for batching

	// batch processor goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		fmt.Printf("[INFO] Batch processor goroutine started\n")

		batch := make([]blizzard.FetchResult, 0, 10) // Process in batches of 10
		batchNumber := 0

		for {
			select {
			case result, ok := <-batchChan:
				if !ok {
					fmt.Printf("[INFO] Batch channel closed, processing final batch...\n")
					// channel closed, process remaining batch
					if len(batch) > 0 {
						batchNumber++
						fmt.Printf("[INFO] Processing final batch %d with %d items...\n", batchNumber, len(batch))
						runs, players, err := ds.processBatch(batch)
						if err != nil {
							fmt.Printf("[ERROR] Final batch %d failed: %v\n", batchNumber, err)
						} else {
							totalRuns += runs
							totalPlayers += players
							if runs > 0 || players > 0 {
								fmt.Printf("[OK] Final batch %d: +%d runs, +%d players (total: %d runs, %d players)\n",
									batchNumber, runs, players, totalRuns, totalPlayers)
							}
						}
					}
					fmt.Printf("[INFO] Batch processor goroutine ending\n")
					return
				}

				batch = append(batch, result)

				// process batch when full
				if len(batch) >= 10 {
					batchNumber++
					runs, players, err := ds.processBatch(batch)
					if err != nil {
						fmt.Printf("[ERROR] Batch %d failed: %v\n", batchNumber, err)
					} else {
						totalRuns += runs
						totalPlayers += players
						if runs > 0 || players > 0 {
							fmt.Printf("[INFO] Batch %d: +%d runs, +%d players (total: %d runs, %d players)\n",
								batchNumber, runs, players, totalRuns, totalPlayers)
						}
					}
					batch = batch[:0] // Clear batch
				}

			case <-ctx.Done():
				fmt.Printf("[WARN] Batch processor context cancelled\n")
				return
			}
		}
	}()

	// forward results to batch processor with progress tracking
	fmt.Printf("[INFO] Starting to process API results...\n")
	for result := range results {
		processedCount++

        if result.Error != nil {
            errorCount++
            // Suppress noisy 404s unless verbose is enabled
            if verbose {
                fmt.Printf("[ERROR] API error [%d] %s/%s: %v\n", processedCount, result.RealmInfo.Name, result.Dungeon.Name, result.Error)
            } else if !strings.Contains(strings.ToLower(result.Error.Error()), "404") {
                fmt.Printf("[ERROR] API error [%d] %s/%s: %v\n", processedCount, result.RealmInfo.Name, result.Dungeon.Name, result.Error)
            }
            continue
        }

		// show periodic progress
		if processedCount%10 == 0 {
			fmt.Printf("[INFO] Progress: %d requests processed, %d errors, queued for batch processing...\n", processedCount, errorCount)
		}

		select {
		case batchChan <- result:
		case <-ctx.Done():
			fmt.Printf("[WARN] Context cancelled, stopping processing\n")
			close(batchChan)
			wg.Wait()
			fmt.Printf("\n[INFO] Final stats: %d requests processed, %d errors, %d runs, %d players\n",
				processedCount, errorCount, totalRuns, totalPlayers)
			return totalRuns, totalPlayers, ctx.Err()
		}
	}

	fmt.Printf("[INFO] Finished processing all API results, closing batch channel...\n")

	close(batchChan)
	wg.Wait()

	fmt.Printf("\n[INFO] Final stats: %d requests processed, %d errors, %d runs, %d players\n",
		processedCount, errorCount, totalRuns, totalPlayers)

	return totalRuns, totalPlayers, nil
}

// processBatch processes a batch of fetch results in a single transaction
func (ds *DatabaseService) processBatch(batch []blizzard.FetchResult) (int, int, error) {
    if len(batch) == 0 {
        return 0, 0, nil
    }

    // Pre-scan batch to decide which items actually need writes (local diff -> insert-new)
    type batchItem struct {
        idx         int
        r           blizzard.RealmInfo
        d           blizzard.DungeonInfo
        board       *blizzard.LeaderboardResponse
        realmID     int
        dungeonID   int
        existingMax int64
        marker      int64
        maxCT       int64
        needsWrite  bool
    }

    items := make([]batchItem, 0, len(batch))
    for i, res := range batch {
        if res.Leaderboard == nil || len(res.Leaderboard.LeadingGroups) == 0 {
            continue
        }
        // resolve IDs (read-only)
        realmID, err := ds.GetRealmIDByRegionAndSlug(res.RealmInfo.Region, res.RealmInfo.Slug)
        if err != nil {
            return 0, 0, fmt.Errorf("failed to resolve realm id: %w", err)
        }
        dungeonID, err := ds.GetDungeonID(res.Dungeon.Slug)
        if err != nil {
            return 0, 0, fmt.Errorf("failed to resolve dungeon id: %w", err)
        }
        // compute payload maxCT
        maxCT := int64(0)
        for _, run := range res.Leaderboard.LeadingGroups {
            if run.CompletedTimestamp > maxCT {
                maxCT = run.CompletedTimestamp
            }
        }
        // read DB high-water and marker (read-only; outside TX)
        var existingMax sql.NullInt64
        _ = ds.db.QueryRow(`SELECT MAX(completed_timestamp) FROM challenge_runs WHERE realm_id = ? AND dungeon_id = ?`, realmID, dungeonID).Scan(&existingMax)
        var marker int64
        _ = ds.db.QueryRow(`SELECT last_completed_ts FROM api_fetch_markers WHERE realm_slug = ? AND dungeon_id = ? AND period_id = ?`, res.RealmInfo.Slug, dungeonID, res.Leaderboard.Period).Scan(&marker)
        minCT := marker
        if existingMax.Valid && existingMax.Int64 > minCT {
            minCT = existingMax.Int64
        }
        needsWrite := maxCT > 0 && maxCT > minCT

        if !needsWrite {
            fmt.Printf("    [SKIP] %s/%s: up-to-date (minCT=%d, maxCT=%d)\n", res.RealmInfo.Name, res.Dungeon.Name, minCT, maxCT)
        }
        items = append(items, batchItem{
            idx:         i + 1,
            r:           res.RealmInfo,
            d:           res.Dungeon,
            board:       res.Leaderboard,
            realmID:     realmID,
            dungeonID:   dungeonID,
            existingMax: existingMax.Int64,
            marker:      marker,
            maxCT:       maxCT,
            needsWrite:  needsWrite,
        })
    }

    // filter to items needing writes
    toWrite := make([]batchItem, 0, len(items))
    for _, it := range items {
        if it.needsWrite {
            toWrite = append(toWrite, it)
        }
    }

    if len(toWrite) == 0 {
        // nothing to write; avoid starting a transaction
        return 0, 0, nil
    }

    // begin transaction for write set only
    startBatch := time.Now()
    tx, err := ds.db.Begin()
    if err != nil {
        return 0, 0, fmt.Errorf("failed to begin batch transaction: %w", err)
    }
    defer tx.Rollback()

    totalRuns := 0
    totalPlayers := 0

    for _, it := range toWrite {
        itemStart := time.Now()
        runs, players, err := ds.insertLeaderboardDataTx(tx, it.board, it.r, it.d)
        if err != nil {
            return 0, 0, fmt.Errorf("failed to insert leaderboard data: %w", err)
        }
        totalRuns += runs
        totalPlayers += players
        fmt.Printf("    - Batch item %d: %s/%s -> +%d runs, +%d players in %dms\n",
            it.idx, it.r.Name, it.d.Name, runs, players, time.Since(itemStart).Milliseconds())
    }

    if err := tx.Commit(); err != nil {
        return 0, 0, fmt.Errorf("failed to commit batch transaction: %w", err)
    }
    fmt.Printf("    [OK] Batch committed in %dms (total +%d runs, +%d players)\n",
        time.Since(startBatch).Milliseconds(), totalRuns, totalPlayers)
    return totalRuns, totalPlayers, nil
}

// ensureReferenceDataTx ensures reference data within a transaction
func (ds *DatabaseService) ensureReferenceDataTx(tx *sql.Tx, realmInfo blizzard.RealmInfo, dungeons []blizzard.DungeonInfo) error {
	// insert realm data
	realmQuery := `INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug) VALUES (?, ?, ?, ?, ?)`
	_, err := tx.Exec(realmQuery, realmInfo.Slug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
	if err != nil {
		return fmt.Errorf("failed to insert realm data: %w", err)
	}

	// insert dungeon data
	dungeonQuery := `INSERT OR IGNORE INTO dungeons (id, slug, name, map_challenge_mode_id) VALUES (?, ?, ?, ?)`
	for _, dungeon := range dungeons {
		_, err := tx.Exec(dungeonQuery, dungeon.ID, dungeon.Slug, dungeon.Name, dungeon.ID)
		if err != nil {
			return fmt.Errorf("failed to insert dungeon data: %w", err)
		}
	}

	return nil
}

// insertLeaderboardDataTx inserts leaderboard data within a transaction
func (ds *DatabaseService) insertLeaderboardDataTx(tx *sql.Tx, leaderboard *blizzard.LeaderboardResponse, realmInfo blizzard.RealmInfo, dungeon blizzard.DungeonInfo) (int, int, error) {
    // Resolve realm and dungeon IDs (read-only; realms/dungeons are pre-populated)
    realmID, err := ds.getRealmIDTx(tx, realmInfo.Slug, realmInfo.Region)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to get realm ID: %w", err)
    }
    dungeonID, err := ds.getDungeonIDTx(tx, dungeon.Slug)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to get dungeon ID: %w", err)
    }

    // Compute newest completed_timestamp in this leaderboard
    maxCT := int64(0)
    for _, run := range leaderboard.LeadingGroups {
        if run.CompletedTimestamp > maxCT {
            maxCT = run.CompletedTimestamp
        }
    }
    // Quick skip using existing DB max completed_timestamp for this realm+dungeon
    var existingMax sql.NullInt64
    if err := tx.QueryRow(`SELECT MAX(completed_timestamp) FROM challenge_runs WHERE realm_id = ? AND dungeon_id = ?`, realmID, dungeonID).Scan(&existingMax); err == nil {
        if existingMax.Valid && maxCT > 0 && existingMax.Int64 >= maxCT {
            fmt.Printf("    [SKIP] %s/%s: existingMax=%d >= maxCT=%d (DB)\n", realmInfo.Name, dungeon.Name, existingMax.Int64, maxCT)
            return 0, 0, nil
        }
    }
    var marker int64
    if err := tx.QueryRow(`SELECT last_completed_ts FROM api_fetch_markers WHERE realm_slug = ? AND dungeon_id = ? AND period_id = ?`, realmInfo.Slug, dungeonID, leaderboard.Period).Scan(&marker); err != nil && err != sql.ErrNoRows {
        return 0, 0, fmt.Errorf("failed to read fetch marker: %w", err)
    }
    if marker >= maxCT && maxCT > 0 {
        fmt.Printf("    [SKIP] %s/%s: up-to-date (marker=%d, maxCT=%d)\n", realmInfo.Name, dungeon.Name, marker, maxCT)
        return 0, 0, nil
    }

    runsInserted := 0
    playersInserted := 0

    for _, run := range leaderboard.LeadingGroups {
        // fast-skip duplicates using DB max and marker
        if (existingMax.Valid && run.CompletedTimestamp <= existingMax.Int64) || (marker > 0 && run.CompletedTimestamp <= marker) {
            continue
        }
        // extract player IDs for team signature
        var playerIDs []int
        for _, member := range run.Members {
            if id, ok := member.GetPlayerID(); ok {
                playerIDs = append(playerIDs, id)
            }
        }

		if len(playerIDs) == 0 {
			continue
		}

		teamSignature := utils.ComputeTeamSignature(playerIDs)

		// insert run
		runQuery := `
			INSERT OR IGNORE INTO challenge_runs
			(duration, completed_timestamp, keystone_level, dungeon_id, realm_id, period_id, period_start_timestamp, period_end_timestamp, team_signature)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := tx.Exec(runQuery,
			run.Duration,
			run.CompletedTimestamp,
			run.KeystoneLevel,
			dungeonID,
			realmID,
			leaderboard.Period,
			leaderboard.PeriodStartTimestamp,
			leaderboard.PeriodEndTimestamp,
			teamSignature,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert run: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to check rows affected: %w", err)
		}

		if rowsAffected == 0 {
			// run already exists (due to OR IGNORE), skip member insertion
			continue
		}

		runID, err := result.LastInsertId()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get run ID: %w", err)
		}

		runsInserted++

		// insert team members
		for _, member := range run.Members {
			playerID, hasPlayerID := member.GetPlayerID()
			playerName, _ := member.GetPlayerName()
			playerRealmSlug, hasRealmSlug := member.GetRealmSlug()

			if !hasPlayerID {
				continue
			}

			// get or create player realm
			var playerRealmID int
            if hasRealmSlug {
                // resolve player realm id (realms are pre-populated; create only if truly unknown)
                playerRealmID, err = ds.getRealmIDTx(tx, playerRealmSlug, realmInfo.Region)
                if err != nil {
                    return 0, 0, fmt.Errorf("failed to resolve player realm: %w", err)
                }
                if playerRealmID == 0 {
                    if _, err := tx.Exec(`INSERT OR IGNORE INTO realms (slug, name, region, connected_realm_id, parent_realm_slug) VALUES (?, ?, ?, NULL, ?)`,
                        playerRealmSlug, playerRealmSlug, realmInfo.Region, ""); err != nil {
                        return 0, 0, fmt.Errorf("failed to create placeholder realm: %w", err)
                    }
                    id2, err := ds.getRealmIDTx(tx, playerRealmSlug, realmInfo.Region)
                    if err != nil || id2 == 0 {
                        return 0, 0, fmt.Errorf("failed to resolve placeholder realm id: %w", err)
                    }
                    playerRealmID = id2
                }
            } else {
                playerRealmID = realmID
            }

			// insert player
			playerQuery := `
				INSERT INTO players (id, name, name_lower, realm_id)
				VALUES (?, ?, lower(?), ?)
				ON CONFLICT(id) DO UPDATE SET
				  name = excluded.name,
				  name_lower = lower(excluded.name),
				  realm_id = excluded.realm_id
				WHERE excluded.name != name OR excluded.realm_id != realm_id
			`
			playerResult, err := tx.Exec(playerQuery, playerID, playerName, playerName, playerRealmID)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to insert player: %w", err)
			}

			rowsAffected, err := playerResult.RowsAffected()
			if err != nil {
				return 0, 0, fmt.Errorf("failed to get rows affected: %w", err)
			}
			if rowsAffected > 0 {
				playersInserted++
			}

			// get spec ID and faction
			specID, _ := member.GetSpecID()
			faction, _ := member.GetFaction()

			// link player to run
			memberQuery := `INSERT INTO run_members (run_id, player_id, spec_id, faction) VALUES (?, ?, ?, ?)`

			var specPtr *int
			var factionPtr *string
			if specID > 0 {
				specPtr = &specID
			}
			if faction != "" {
				factionPtr = &faction
			}

			_, err = tx.Exec(memberQuery, runID, playerID, specPtr, factionPtr)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to insert run member: %w", err)
			}
		}
	}

    // update marker inside the transaction if we progressed
    if maxCT > marker && maxCT > 0 {
        if _, err := tx.Exec(`INSERT INTO api_fetch_markers (realm_slug, dungeon_id, period_id, last_completed_ts)
                               VALUES (?, ?, ?, ?)
                               ON CONFLICT(realm_slug, dungeon_id, period_id)
                               DO UPDATE SET last_completed_ts = MAX(last_completed_ts, excluded.last_completed_ts)`,
            realmInfo.Slug, dungeonID, leaderboard.Period, maxCT); err != nil {
            return 0, 0, fmt.Errorf("failed to update fetch marker: %w", err)
        }
    }

    return runsInserted, playersInserted, nil
}

// helper functions for transaction-based operations
func (ds *DatabaseService) getRealmIDTx(tx *sql.Tx, slug string, region string) (int, error) {
    var realmID int
    err := tx.QueryRow("SELECT id FROM realms WHERE slug = ? AND region = ?", slug, region).Scan(&realmID)
    if err == sql.ErrNoRows {
        // Not found is not an error for callers that may insert placeholders/aliases
        return 0, nil
    }
    return realmID, err
}

func (ds *DatabaseService) getDungeonIDTx(tx *sql.Tx, slug string) (int, error) {
	var dungeonID int
	err := tx.QueryRow("SELECT id FROM dungeons WHERE slug = ?", slug).Scan(&dungeonID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("dungeon not found: %s", slug)
	}
	return dungeonID, err
}

func (ds *DatabaseService) getOrCreateRealmTx(tx *sql.Tx, realmSlug string, allRealms map[string]blizzard.RealmInfo) (int, error) {
	// try to get existing realm
	var realmID int
	err := tx.QueryRow("SELECT id FROM realms WHERE slug = ?", realmSlug).Scan(&realmID)
	if err == nil {
		return realmID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// realm doesn't exist, create it
	if realmInfo, exists := allRealms[realmSlug]; exists {
		result, err := tx.Exec(`
			INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
			VALUES (?, ?, ?, ?, ?)
		`, realmSlug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
		if err != nil {
			return 0, err
		}

		id, err := result.LastInsertId()
		return int(id), err
	} else {
		result, err := tx.Exec(`
			INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
			VALUES (?, ?, ?, NULL, ?)
		`, realmSlug, utils.Slugify(realmSlug), "unknown", "")
		if err != nil {
			return 0, err
		}

		id, err := result.LastInsertId()
		return int(id), err
	}
}

// getOrCreateRealmByRegion resolves a realm by region+slug or creates a placeholder/known realm
func (ds *DatabaseService) getOrCreateRealmByRegion(tx *sql.Tx, realmSlug string, region string, allRealms map[string]blizzard.RealmInfo) (int, error) {
    // try to get existing realm by composite key
    var realmID int
    err := tx.QueryRow("SELECT id FROM realms WHERE slug = ? AND region = ?", realmSlug, region).Scan(&realmID)
    if err == nil {
        return realmID, nil
    }
    if err != sql.ErrNoRows {
        return 0, err
    }

    // If we know this realm (by slug) from constants, use its canonical region/name/id
    if realmInfo, ok := allRealms[realmSlug]; ok {
        result, err := tx.Exec(`
            INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
            VALUES (?, ?, ?, ?, ?)
        `, realmSlug, realmInfo.Name, realmInfo.Region, realmInfo.ID, realmInfo.ParentRealmSlug)
        if err != nil {
            return 0, err
        }
        newID, err := result.LastInsertId()
        return int(newID), err
    }

    // otherwise create a placeholder scoped to the provided region
    result, err := tx.Exec(`
        INSERT INTO realms (slug, name, region, connected_realm_id, parent_realm_slug)
        VALUES (?, ?, ?, NULL, ?)
    `, realmSlug, utils.Slugify(realmSlug), region, "")
    if err != nil {
        return 0, err
    }
    newID, err := result.LastInsertId()
    return int(newID), err
}

// player profile database operations

// GetEligiblePlayersForProfileFetch returns players with complete coverage (9/9 dungeons)
func (ds *DatabaseService) GetEligiblePlayersForProfileFetch() ([]blizzard.PlayerInfo, error) {
	query := `
		SELECT p.id, p.name, r.slug as realm_slug, r.region
		FROM players p
		JOIN player_profiles pp ON p.id = pp.player_id
		JOIN realms r ON p.realm_id = r.id
		WHERE pp.has_complete_coverage = 1
		ORDER BY pp.global_ranking
	`

	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible players: %w", err)
	}
	defer rows.Close()

	var players []blizzard.PlayerInfo
	for rows.Next() {
		var player blizzard.PlayerInfo
		err := rows.Scan(&player.ID, &player.Name, &player.RealmSlug, &player.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player row: %w", err)
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating player rows: %w", err)
	}

	return players, nil
}

// InsertPlayerProfileData inserts player profile data and returns counts
func (ds *DatabaseService) InsertPlayerProfileData(result blizzard.PlayerProfileResult, timestamp int64) (int, int, error) {
	if result.Error != nil {
		return 0, 0, result.Error
	}

	profilesUpdated := 0
	equipmentUpdated := 0

	// begin transaction
	tx, err := ds.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// insert player summary/details
	if result.Summary != nil {
		err := ds.insertPlayerDetailsTx(tx, result.PlayerID, result.Summary, result.Media, timestamp)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert player details: %w", err)
		}
		profilesUpdated++
	}

	// insert player equipment
	if result.Equipment != nil {
		itemCount, err := ds.insertPlayerEquipmentTx(tx, result.PlayerID, result.Equipment, timestamp)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert player equipment: %w", err)
		}
		equipmentUpdated += itemCount
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return profilesUpdated, equipmentUpdated, nil
}

// insertPlayerDetailsTx inserts player summary data within a transaction
func (ds *DatabaseService) insertPlayerDetailsTx(tx *sql.Tx, playerID int, summary *blizzard.CharacterSummaryResponse, media *blizzard.CharacterMediaResponse, timestamp int64) error {
	// extract avatar URL from media data
	var avatarURL *string
	if media != nil {
		for _, asset := range media.Assets {
			if asset.Key == "avatar" {
				avatarURL = &asset.Value
				break
			}
		}
	}

	// extract guild name
	var guildName *string
	if summary.Guild != nil {
		guildName = &summary.Guild.Name
	}

    // Upsert without touching last_login_timestamp to avoid noisy writes.
    // Only update when a value actually changes (WHERE guard).
    _, err := tx.Exec(`
        INSERT INTO player_details (
            player_id, race_id, race_name, gender, class_id, class_name,
            active_spec_id, active_spec_name, guild_name, level,
            average_item_level, equipped_item_level, avatar_url, last_updated
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(player_id) DO UPDATE SET
            race_id = excluded.race_id,
            race_name = excluded.race_name,
            gender = excluded.gender,
            class_id = excluded.class_id,
            class_name = excluded.class_name,
            active_spec_id = excluded.active_spec_id,
            active_spec_name = excluded.active_spec_name,
            guild_name = excluded.guild_name,
            level = excluded.level,
            average_item_level = excluded.average_item_level,
            equipped_item_level = excluded.equipped_item_level,
            avatar_url = excluded.avatar_url,
            last_updated = excluded.last_updated
        WHERE
            player_details.race_id               IS NOT excluded.race_id OR
            player_details.race_name            IS NOT excluded.race_name OR
            player_details.gender               IS NOT excluded.gender OR
            player_details.class_id             IS NOT excluded.class_id OR
            player_details.class_name           IS NOT excluded.class_name OR
            player_details.active_spec_id       IS NOT excluded.active_spec_id OR
            player_details.active_spec_name     IS NOT excluded.active_spec_name OR
            player_details.guild_name           IS NOT excluded.guild_name OR
            player_details.level                IS NOT excluded.level OR
            player_details.average_item_level   IS NOT excluded.average_item_level OR
            player_details.equipped_item_level  IS NOT excluded.equipped_item_level OR
            player_details.avatar_url           IS NOT excluded.avatar_url
    `,
        playerID,
        summary.Race.ID,
        summary.Race.Name,
        summary.Gender.Type,
        summary.CharacterClass.ID,
        summary.CharacterClass.Name,
        summary.ActiveSpec.ID,
        summary.ActiveSpec.Name,
        guildName,
        summary.Level,
        summary.AverageItemLevel,
        summary.EquippedItemLevel,
        avatarURL,
        timestamp,
    )

    return err
}

// insertPlayerEquipmentTx inserts player equipment data within a transaction
func (ds *DatabaseService) insertPlayerEquipmentTx(tx *sql.Tx, playerID int, equipment *blizzard.CharacterEquipmentResponse, timestamp int64) (int, error) {
	if equipment == nil || len(equipment.EquippedItems) == 0 {
		return 0, nil
	}

	equipmentCount := 0

    for _, item := range equipment.EquippedItems {
        // Check latest snapshot for this slot; skip writing if unchanged
        var prevID sql.NullInt64
        var prevItemID sql.NullInt64
        var prevUpgradeID sql.NullInt64
        var prevQuality, prevName sql.NullString
        if err := tx.QueryRow(
            `SELECT id, item_id, upgrade_id, quality, item_name
             FROM player_equipment
             WHERE player_id = ? AND slot_type = ?
             ORDER BY snapshot_timestamp DESC
             LIMIT 1`,
            playerID, item.Slot.Type,
        ).Scan(&prevID, &prevItemID, &prevUpgradeID, &prevQuality, &prevName); err != nil && err != sql.ErrNoRows {
            return 0, fmt.Errorf("failed to query latest equipment: %w", err)
        }

        unchanged := false
        if prevID.Valid {
            // Basics comparison
            prevUpg := 0
            if prevUpgradeID.Valid {
                prevUpg = int(prevUpgradeID.Int64)
            }
            curUpg := 0
            if item.UpgradeID != nil {
                curUpg = *item.UpgradeID
            }
            sameBasics := prevItemID.Valid && int(prevItemID.Int64) == item.Item.ID && prevQuality.Valid && prevQuality.String == item.Quality.Type && prevName.Valid && prevName.String == item.Name && prevUpg == curUpg

            if sameBasics {
                // Compare enchantments as a canonical sorted signature
                dbRows, qerr := tx.Query(
                    `SELECT
                        COALESCE(enchantment_id, -1) as eid,
                        COALESCE(source_item_id, -1) as sid,
                        COALESCE(slot_id, -1) as slotId,
                        COALESCE(slot_type, '') as slotType,
                        COALESCE(spell_id, -1) as spellId,
                        COALESCE(display_string, '') as disp
                     FROM player_equipment_enchantments
                     WHERE equipment_id = ?`, prevID.Int64)
                if qerr != nil {
                    return 0, fmt.Errorf("failed to load existing enchantments: %w", qerr)
                }
                var dbSigs []string
                for dbRows.Next() {
                    var eid, sid, slotId, spellId int
                    var slotType, disp string
                    if err := dbRows.Scan(&eid, &sid, &slotId, &slotType, &spellId, &disp); err != nil {
                        dbRows.Close()
                        return 0, fmt.Errorf("failed to scan enchantment: %w", err)
                    }
                    dbSigs = append(dbSigs, fmt.Sprintf("%d|%d|%d|%s|%d|%s", eid, sid, slotId, slotType, spellId, disp))
                }
                dbRows.Close()
                sort.Strings(dbSigs)

                var curSigs []string
                for _, ench := range item.Enchantments {
                    eid := -1
                    if ench.EnchantmentID != nil {
                        eid = *ench.EnchantmentID
                    }
                    sid := -1
                    if ench.SourceItem != nil {
                        sid = ench.SourceItem.ID
                    }
                    slotId := -1
                    var slotType string
                    if ench.EnchantmentSlot != nil {
                        slotId = ench.EnchantmentSlot.ID
                        slotType = ench.EnchantmentSlot.Type
                    }
                    spellId := -1
                    if ench.Spell != nil {
                        spellId = ench.Spell.Spell.ID
                    }
                    disp := ench.DisplayString
                    curSigs = append(curSigs, fmt.Sprintf("%d|%d|%d|%s|%d|%s", eid, sid, slotId, slotType, spellId, disp))
                }
                sort.Strings(curSigs)

                if strings.Join(dbSigs, ";") == strings.Join(curSigs, ";") {
                    unchanged = true
                }
            }
        }

        if unchanged {
            continue // skip inserting a new snapshot for this slot
        }

        // insert equipment item
        result, err := tx.Exec(`
            INSERT INTO player_equipment (
                player_id, slot_type, item_id, upgrade_id, quality, item_name, snapshot_timestamp
            ) VALUES (?, ?, ?, ?, ?, ?, ?)
        `,
            playerID,
            item.Slot.Type,
            item.Item.ID,
            item.UpgradeID,
            item.Quality.Type,
            item.Name,
            timestamp,
        )

		if err != nil {
			return 0, fmt.Errorf("failed to insert equipment item: %w", err)
		}

		equipmentID, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get equipment ID: %w", err)
		}

		equipmentCount++

		// insert enchantments (gems, enchants, tinkers)
		for _, enchant := range item.Enchantments {
			var sourceItemID *int
			var sourceItemName *string
			if enchant.SourceItem != nil {
				sourceItemID = &enchant.SourceItem.ID
				sourceItemName = &enchant.SourceItem.Name
			}

			var spellID *int
			if enchant.Spell != nil {
				spellID = &enchant.Spell.Spell.ID
			}

			var slotID *int
			var slotType *string
			if enchant.EnchantmentSlot != nil {
				slotID = &enchant.EnchantmentSlot.ID
				slotType = &enchant.EnchantmentSlot.Type
			}

			_, err := tx.Exec(`
				INSERT INTO player_equipment_enchantments (
					equipment_id, enchantment_id, slot_id, slot_type,
					display_string, source_item_id, source_item_name, spell_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`,
				equipmentID,
				enchant.EnchantmentID,
				slotID,
				slotType,
				enchant.DisplayString,
				sourceItemID,
				sourceItemName,
				spellID,
			)

			if err != nil {
				return 0, fmt.Errorf("failed to insert enchantment: %w", err)
			}
		}
	}

	return equipmentCount, nil
}

// Season management operations

// UpsertSeason inserts or updates a season record and returns the auto-increment ID
func (ds *DatabaseService) UpsertSeason(seasonID int, region string, seasonName string, startTimestamp int64) (int, error) {
	query := `
		INSERT INTO seasons (season_number, region, season_name, start_timestamp)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(season_number, region) DO UPDATE SET
			season_name = excluded.season_name,
			start_timestamp = excluded.start_timestamp
		RETURNING id
	`
	var id int
	err := ds.db.QueryRow(query, seasonID, region, seasonName, startTimestamp).Scan(&id)
	return id, err
}

// UpdateSeasonPeriodRange updates the first_period_id and last_period_id for a season
func (ds *DatabaseService) UpdateSeasonPeriodRange(seasonID, firstPeriodID, lastPeriodID int) error {
	query := `
		UPDATE seasons
		SET first_period_id = ?, last_period_id = ?
		WHERE id = ?
	`
	_, err := ds.db.Exec(query, firstPeriodID, lastPeriodID, seasonID)
	return err
}

// UpdateSeasonEndTimestamp updates the end_timestamp for a season
func (ds *DatabaseService) UpdateSeasonEndTimestamp(seasonID int, endTimestamp int64) error {
	query := `UPDATE seasons SET end_timestamp = ? WHERE id = ?`
	_, err := ds.db.Exec(query, endTimestamp, seasonID)
	return err
}

// LinkPeriodToSeason creates a mapping between a period and season
func (ds *DatabaseService) LinkPeriodToSeason(periodID, seasonID int) error {
	query := `INSERT OR IGNORE INTO period_seasons (period_id, season_id) VALUES (?, ?)`
	_, err := ds.db.Exec(query, periodID, seasonID)
	return err
}

// GetSeasonByID retrieves a season by its ID
func (ds *DatabaseService) GetSeasonByID(seasonID int) (*Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		WHERE id = ?
	`
	var season Season
	var endTimestamp sql.NullInt64
	var firstPeriod, lastPeriod sql.NullInt64
	err := ds.db.QueryRow(query, seasonID).Scan(
		&season.ID,
		&season.SeasonNumber,
		&season.StartTimestamp,
		&endTimestamp,
		&season.SeasonName,
		&firstPeriod,
		&lastPeriod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if endTimestamp.Valid {
		season.EndTimestamp = &endTimestamp.Int64
	}
	if firstPeriod.Valid {
		fp := int(firstPeriod.Int64)
		season.FirstPeriodID = &fp
	}
	if lastPeriod.Valid {
		lp := int(lastPeriod.Int64)
		season.LastPeriodID = &lp
	}
	return &season, nil
}

// GetAllSeasons retrieves all seasons ordered by start timestamp
func (ds *DatabaseService) GetAllSeasons() ([]Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		ORDER BY start_timestamp DESC
	`
	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []Season
	for rows.Next() {
		var season Season
		var endTimestamp sql.NullInt64
		var firstPeriod, lastPeriod sql.NullInt64
		err := rows.Scan(
			&season.ID,
			&season.SeasonNumber,
			&season.StartTimestamp,
			&endTimestamp,
			&season.SeasonName,
			&firstPeriod,
			&lastPeriod,
		)
		if err != nil {
			return nil, err
		}
		if endTimestamp.Valid {
			season.EndTimestamp = &endTimestamp.Int64
		}
		if firstPeriod.Valid {
			fp := int(firstPeriod.Int64)
			season.FirstPeriodID = &fp
		}
		if lastPeriod.Valid {
			lp := int(lastPeriod.Int64)
			season.LastPeriodID = &lp
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

// GetCurrentSeason retrieves the current active season (most recent without end timestamp)
func (ds *DatabaseService) GetCurrentSeason() (*Season, error) {
	query := `
		SELECT id, season_number, start_timestamp, end_timestamp, season_name, first_period_id, last_period_id
		FROM seasons
		WHERE end_timestamp IS NULL
		ORDER BY start_timestamp DESC
		LIMIT 1
	`
	var season Season
	var endTimestamp sql.NullInt64
	var firstPeriod, lastPeriod sql.NullInt64
	err := ds.db.QueryRow(query).Scan(
		&season.ID,
		&season.SeasonNumber,
		&season.StartTimestamp,
		&endTimestamp,
		&season.SeasonName,
		&firstPeriod,
		&lastPeriod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if endTimestamp.Valid {
		season.EndTimestamp = &endTimestamp.Int64
	}
	if firstPeriod.Valid {
		fp := int(firstPeriod.Int64)
		season.FirstPeriodID = &fp
	}
	if lastPeriod.Valid {
		lp := int(lastPeriod.Int64)
		season.LastPeriodID = &lp
	}
	return &season, nil
}

// GetSeasonForPeriod retrieves the season ID for a given period
func (ds *DatabaseService) GetSeasonForPeriod(periodID int) (int, error) {
	query := `SELECT season_id FROM period_seasons WHERE period_id = ? LIMIT 1`
	var seasonID int
	err := ds.db.QueryRow(query, periodID).Scan(&seasonID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return seasonID, err
}

// GetPeriodsForSeason retrieves all period IDs for a given season
func (ds *DatabaseService) GetPeriodsForSeason(seasonID int) ([]int, error) {
	query := `SELECT period_id FROM period_seasons WHERE season_id = ? ORDER BY period_id`
	rows, err := ds.db.Query(query, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []int
	for rows.Next() {
		var periodID int
		if err := rows.Scan(&periodID); err != nil {
			return nil, err
		}
		periods = append(periods, periodID)
	}
	return periods, rows.Err()
}

// GetPeriodsForRegion retrieves all period IDs for all seasons in a given region
func (ds *DatabaseService) GetPeriodsForRegion(region string) ([]int, error) {
	query := `
		SELECT DISTINCT ps.period_id
		FROM period_seasons ps
		JOIN seasons s ON ps.season_id = s.id
		WHERE s.region = ?
		ORDER BY ps.period_id DESC
	`
	rows, err := ds.db.Query(query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []int
	for rows.Next() {
		var periodID int
		if err := rows.Scan(&periodID); err != nil {
			return nil, err
		}
		periods = append(periods, periodID)
	}
	return periods, rows.Err()
}

// GetRealmPoolIDs returns all realm IDs in a realm pool (parent + all children)
// For a child realm, it returns the parent and all siblings
// For a parent realm, it returns itself and all children
// For an independent realm, it returns just itself
func (ds *DatabaseService) GetRealmPoolIDs(region, slug string) ([]int, error) {
	// First, get the realm's parent_realm_slug
	var parentSlug sql.NullString
	err := ds.db.QueryRow(`
		SELECT parent_realm_slug
		FROM realms
		WHERE region = ? AND slug = ?
	`, region, slug).Scan(&parentSlug)
	if err != nil {
		if err == sql.ErrNoRows {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to query realm: %w", err)
	}

	// Determine the pool leader slug
	poolLeaderSlug := slug
	if parentSlug.Valid && parentSlug.String != "" {
		// This is a child realm, use the parent as pool leader
		poolLeaderSlug = parentSlug.String
	}

	// Get all realms in the pool: the pool leader + all realms that have it as parent
	query := `
		SELECT id FROM realms
		WHERE region = ? AND (
			slug = ? OR parent_realm_slug = ?
		)
		ORDER BY id
	`
	rows, err := ds.db.Query(query, region, poolLeaderSlug, poolLeaderSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query realm pool: %w", err)
	}
	defer rows.Close()

	var poolIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		poolIDs = append(poolIDs, id)
	}
	return poolIDs, rows.Err()
}
