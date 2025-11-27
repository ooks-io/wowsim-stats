package pipeline

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
)

type StatusOptions struct {
	Verbose     bool
	BatchSize   int
	MaxPlayers  int
	StaleAfter  time.Duration
	Concurrency int
	MaxRPS      float64
}

type StatusResult struct {
	Processed int
	Valid     int
	Invalid   int
	Errors    int
	Duration  time.Duration
}

// RefreshPlayerStatuses fetches the status endpoint for players whose cached status is stale.

func RefreshPlayerStatuses(db *database.DatabaseService, client *blizzard.Client, opts StatusOptions) (*StatusResult, error) {
	start := time.Now()
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 200
	}

	staleCutoff := int64(0)
	if opts.StaleAfter > 0 {
		staleCutoff = time.Now().Add(-opts.StaleAfter).UnixMilli()
	}

	logger := log.With("component", "status")
	res := &StatusResult{}

	totalRemaining, err := db.CountPlayersNeedingStatusCheck(staleCutoff)
	var tracker *BatchTracker
	if err != nil {
		logger.Warn("unable to count stale players", "error", err)
		tracker = NewBatchTracker(start, -1, opts.MaxPlayers)
	} else {
		tracker = NewBatchTracker(start, totalRemaining, opts.MaxPlayers)
		logger.Info("found players needing status checks",
			"count", totalRemaining,
			"target", tracker.DescribeTarget())
	}

	batchNumber := 0
	for {
		if tracker.ShouldStop(res.Processed) {
			break
		}

		currentBatchSize := tracker.AdjustBatchSize(batchSize, res.Processed)
		if currentBatchSize == 0 {
			break
		}

		candidates, err := db.GetPlayersNeedingStatusCheck(currentBatchSize, staleCutoff)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			break
		}

		logger.Info("checking player statuses", "batch", batchNumber+1, "batch_size", len(candidates))
		batchNumber++

		var wg sync.WaitGroup
		mu := sync.Mutex{}

		for _, cand := range candidates {
			if tracker.ShouldStop(res.Processed) {
				break
			}
			wg.Add(1)

			go func(cand database.PlayerStatusCandidate) {
				defer wg.Done()

				realm := blizzard.NormalizeRealmSlug(cand.Region, cand.RealmSlug)
				resp, err := client.FetchCharacterStatus(cand.Name, realm, cand.Region)

				mu.Lock()
				defer mu.Unlock()

				res.Processed++
				if opts.Verbose {
					logger.Debug("requesting status",
						"player_id", cand.PlayerID,
						"name", cand.Name,
						"region", cand.Region,
						"realm", cand.RealmSlug)
				}

				if err != nil {
					if isNotFoundError(err) {
						res.Invalid++
						_ = db.UpdatePlayerStatus(cand.PlayerID, false, nowMillis(), nil)
						logger.Warn("player missing from API, marked invalid",
							"player_id", cand.PlayerID,
							"name", cand.Name,
							"region", cand.Region,
							"realm", cand.RealmSlug)
					} else {
						res.Errors++
						logger.Error("status fetch failed",
							"player_id", cand.PlayerID,
							"name", cand.Name,
							"region", cand.Region,
							"realm", cand.RealmSlug,
							"error", err)
					}
					return
				}

				charID := extractCharacterID(resp)
				if err := db.UpdatePlayerStatus(cand.PlayerID, resp.IsValid, nowMillis(), charID); err != nil {
					logger.Error("update player status failed",
						"player_id", cand.PlayerID,
						"error", err)
					return
				}

				if resp.IsValid {
					res.Valid++
					if resp.Character.Realm.Slug != "" || resp.Character.Name != "" {
						realmSlug := resp.Character.Realm.Slug
						if realmSlug == "" {
							realmSlug = realm
						}
						name := resp.Character.Name
						if name == "" {
							name = cand.Name
						}
						_ = db.UpdatePlayerIdentity(int(cand.PlayerID), name, cand.Region, realmSlug)
					}
				} else {
					res.Invalid++
					reason := resp.Reason
					if reason == "" {
						reason = "unknown"
					}
					logger.Warn("status invalid",
						"player_id", cand.PlayerID,
						"name", cand.Name,
						"region", cand.Region,
						"realm", cand.RealmSlug,
						"reason", reason)
				}
			}(cand)
		}

		wg.Wait()

		logBatchProgress(tracker, logger, res.Processed, batchNumber)
	}

	res.Duration = time.Since(start)
	logger.Info("status refresh finished",
		"processed", res.Processed,
		"valid", res.Valid,
		"invalid", res.Invalid,
		"errors", res.Errors,
		"duration", res.Duration.Truncate(time.Second))
	return res, nil
}
