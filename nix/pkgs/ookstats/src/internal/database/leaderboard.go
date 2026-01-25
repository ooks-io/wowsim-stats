package database

import (
	"context"
	"database/sql"
	"fmt"
	"ookstats/internal/blizzard"
	"ookstats/internal/utils"
	"strings"
	"time"
)

// InsertLeaderboardData inserts leaderboard data and returns the number of runs and players inserted
func (ds *DatabaseService) InsertLeaderboardData(leaderboard *blizzard.LeaderboardResponse, realmInfo blizzard.RealmInfo, dungeon blizzard.DungeonInfo) (int, int, error) {
	if leaderboard == nil || len(leaderboard.LeadingGroups) == 0 {
		return 0, 0, nil
	}

	tx, err := ds.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	runs, players, err := ds.insertLeaderboardDataTx(tx, leaderboard, realmInfo, dungeon)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return runs, players, nil
}

// BatchProcessFetchResults processes fetch results concurrently with transaction batching
func (ds *DatabaseService) BatchProcessFetchResults(ctx context.Context, results <-chan blizzard.FetchResult) (int, int, error) {
	totalRuns := 0
	totalPlayers := 0
	processedCount := 0
	errorCount := 0

	batch := make([]blizzard.FetchResult, 0, 10)
	batchNumber := 0

	fmt.Printf("[INFO] Starting to process API results...\n")
	for result := range results {
		processedCount++
		ds.recordFetchStatusResult(result)

		if result.Error != nil {
			errorCount++
			if verbose {
				fmt.Printf("[ERROR] API error [%d] %s/%s: %v\n", processedCount, result.RealmInfo.Name, result.Dungeon.Name, result.Error)
			} else if !strings.Contains(strings.ToLower(result.Error.Error()), "404") {
				fmt.Printf("[ERROR] API error [%d] %s/%s: %v\n", processedCount, result.RealmInfo.Name, result.Dungeon.Name, result.Error)
			}
			continue
		}

		batch = append(batch, result)

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
			batch = batch[:0]
		}

		if processedCount%10 == 0 {
			fmt.Printf("[INFO] Progress: %d requests processed, %d errors\n", processedCount, errorCount)
		}

		select {
		case <-ctx.Done():
			fmt.Printf("[WARN] Context cancelled, stopping processing\n")
			return totalRuns, totalPlayers, ctx.Err()
		default:
		}
	}

	if len(batch) > 0 {
		batchNumber++
		runs, players, err := ds.processBatch(batch)
		if err != nil {
			fmt.Printf("[ERROR] Final batch %d failed: %v\n", batchNumber, err)
		} else {
			totalRuns += runs
			totalPlayers += players
			if runs > 0 || players > 0 {
				fmt.Printf("[INFO] Final batch %d: +%d runs, +%d players\n", batchNumber, runs, players)
			}
		}
	}

	fmt.Printf("\n[INFO] Final stats: %d requests processed, %d errors, %d runs, %d players\n",
		processedCount, errorCount, totalRuns, totalPlayers)

	return totalRuns, totalPlayers, nil
}

// processBatch processes a batch of fetch results in a single transaction
func (ds *DatabaseService) processBatch(batch []blizzard.FetchResult) (int, int, error) {
	if len(batch) == 0 {
		return 0, 0, nil
	}

	type batchItem struct {
		idx   int
		r     blizzard.RealmInfo
		d     blizzard.DungeonInfo
		board *blizzard.LeaderboardResponse
	}

	items := make([]batchItem, 0, len(batch))
	for i, res := range batch {
		if res.Leaderboard == nil || len(res.Leaderboard.LeadingGroups) == 0 {
			continue
		}
		items = append(items, batchItem{
			idx:   i + 1,
			r:     res.RealmInfo,
			d:     res.Dungeon,
			board: res.Leaderboard,
		})
	}

	if len(items) == 0 {
		return 0, 0, nil
	}

	startBatch := time.Now()
	tx, err := ds.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin batch transaction: %w", err)
	}
	defer tx.Rollback()

	totalRuns := 0
	totalPlayers := 0

	for _, it := range items {
		itemStart := time.Now()
		runs, players, err := ds.insertLeaderboardDataTx(tx, it.board, it.r, it.d)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert leaderboard data: %w", err)
		}
		totalRuns += runs
		totalPlayers += players
		if runs > 0 || players > 0 {
			fmt.Printf("    - Batch item %d: %s/%s -> +%d runs, +%d players in %dms\n",
				it.idx, it.r.Name, it.d.Name, runs, players, time.Since(itemStart).Milliseconds())
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit batch transaction: %w", err)
	}
	fmt.Printf("    [OK] Batch committed in %dms (total +%d runs, +%d players)\n",
		time.Since(startBatch).Milliseconds(), totalRuns, totalPlayers)
	return totalRuns, totalPlayers, nil
}

// insertLeaderboardDataTx inserts leaderboard data within a transaction
func (ds *DatabaseService) insertLeaderboardDataTx(tx *sql.Tx, leaderboard *blizzard.LeaderboardResponse, realmInfo blizzard.RealmInfo, dungeon blizzard.DungeonInfo) (int, int, error) {
	realmID, err := ds.getRealmIDTx(tx, realmInfo.Slug, realmInfo.Region)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get realm ID: %w", err)
	}
	dungeonID, err := ds.getDungeonIDTx(tx, dungeon.Slug)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get dungeon ID: %w", err)
	}

	// compute newest completed_timestamp in this leaderboard (for marker update)
	maxCT := int64(0)
	for _, run := range leaderboard.LeadingGroups {
		if run.CompletedTimestamp > maxCT {
			maxCT = run.CompletedTimestamp
		}
	}

	runsInserted := 0
	playersInserted := 0

	for _, run := range leaderboard.LeadingGroups {
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

		var runSeasonID *int
		if run.CompletedTimestamp > 0 {
			sid, err := ds.determineSeasonForRunTx(tx, realmInfo.Region, run.CompletedTimestamp)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to determine season for run: %w", err)
			}
			if sid > 0 {
				runSeasonID = &sid
			}
		}

		runQuery := `
			INSERT OR IGNORE INTO challenge_runs
			(duration, completed_timestamp, keystone_level, dungeon_id, realm_id, period_id, period_start_timestamp, period_end_timestamp, team_signature, season_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			runSeasonID,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert run: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to check rows affected: %w", err)
		}

		if rowsAffected == 0 {
			continue
		}

		runID, err := result.LastInsertId()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get run ID: %w", err)
		}

		runsInserted++

		for _, member := range run.Members {
			playerID, hasPlayerID := member.GetPlayerID()
			playerName, _ := member.GetPlayerName()
			playerRealmSlug, hasRealmSlug := member.GetRealmSlug()

			if !hasPlayerID {
				continue
			}

			var playerRealmID int
			if hasRealmSlug {
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

			// check if player with same name+realm already exists (handles faction transfers where Blizzard ID changes)
			existingPlayerID, err := getExistingPlayerIDTx(tx, playerName, playerRealmID)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to check existing player: %w", err)
			}

			effectivePlayerID := playerID

			if existingPlayerID != 0 && existingPlayerID != int64(playerID) {
				// player exists with different ID (likely faction transfer)
				// migrate old runs to the new player ID and mark old player invalid
				_, err := tx.Exec(`UPDATE run_members SET player_id = ? WHERE player_id = ?`, playerID, existingPlayerID)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to migrate runs to new player ID: %w", err)
				}
				_, err = tx.Exec(`UPDATE players SET is_valid = 0 WHERE id = ?`, existingPlayerID)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to invalidate old player: %w", err)
				}
			}

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

			specID, _ := member.GetSpecID()
			faction, _ := member.GetFaction()

			memberQuery := `INSERT INTO run_members (run_id, player_id, spec_id, faction) VALUES (?, ?, ?, ?)`

			var specPtr *int
			var factionPtr *string
			if specID > 0 {
				specPtr = &specID
			}
			if faction != "" {
				factionPtr = &faction
			}

			_, err = tx.Exec(memberQuery, runID, effectivePlayerID, specPtr, factionPtr)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to insert run member: %w", err)
			}
		}
	}

	// update marker inside the transaction
	if maxCT > 0 {
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
