package pipeline

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"ookstats/internal/playerid"
	"ookstats/internal/wow"
)

// FingerprintOptions controls the fingerprint fetcher.
type FingerprintOptions struct {
	Verbose    bool
	BatchSize  int
	MaxPlayers int
}

// FingerprintResult summarizes processing totals.
type FingerprintResult struct {
	Processed     int
	Created       int
	Skipped       int
	MarkedInvalid int
	Duration      time.Duration
}

// BuildPlayerFingerprints fetches achievements for players missing fingerprints and stores them.
func BuildPlayerFingerprints(db *database.DatabaseService, client *blizzard.Client, opts FingerprintOptions) (*FingerprintResult, error) {
	start := time.Now()
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 25
	}

	result := &FingerprintResult{}
	logger := log.With("component", "fingerprint")
	seen := make(map[int64]bool)
	totalRemaining, err := db.CountPlayersMissingFingerprints()
	var tracker *BatchTracker
	if err != nil {
		logger.Warn("unable to count players needing fingerprints", "error", err)
		tracker = NewBatchTracker(start, -1, opts.MaxPlayers)
	} else {
		tracker = NewBatchTracker(start, totalRemaining, opts.MaxPlayers)
		logger.Info("found players awaiting fingerprints",
			"count", totalRemaining,
			"target", tracker.DescribeTarget())
	}

	logger.Info("loading existing fingerprints for collision detection")
	collisionMap, err := db.GetAllFingerprintHashes()
	if err != nil {
		return nil, fmt.Errorf("failed to load fingerprint collision map: %w", err)
	}
	logger.Info("loaded fingerprint collision map", "existing_fingerprints", len(collisionMap))

	// Mutex to protect concurrent access to collisionMap
	var collisionMapMutex sync.RWMutex

	batchNumber := 0
	for {
		if tracker.ShouldStop(result.Processed) {
			break
		}

		currentBatchSize := tracker.AdjustBatchSize(batchSize, result.Processed)
		if currentBatchSize == 0 {
			break
		}

		candidates, err := db.GetPlayersNeedingFingerprints(currentBatchSize)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			break
		}

		batchCandidates := make([]database.PlayerFingerprintCandidate, 0, len(candidates))
		scheduled := 0
		for _, cand := range candidates {
			if seen[cand.PlayerID] {
				continue
			}
			if tracker.ShouldStop(result.Processed + scheduled) {
				break
			}
			seen[cand.PlayerID] = true
			batchCandidates = append(batchCandidates, cand)
			scheduled++
		}
		if len(batchCandidates) == 0 {
			continue
		}

		logger.Info("processing fingerprint candidates", "batch", batchNumber+1, "batch_size", len(batchCandidates))
		batchNumber++

		outcomes := make(chan fingerprintOutcome, len(batchCandidates))
		var wg sync.WaitGroup
		for _, cand := range batchCandidates {
			wg.Add(1)
			go func(c database.PlayerFingerprintCandidate) {
				defer wg.Done()
				outcomes <- processFingerprintCandidate(db, client, c, collisionMap, &collisionMapMutex, logger)
			}(cand)
		}

		go func() {
			wg.Wait()
			close(outcomes)
		}()

		for oc := range outcomes {
			result.Processed++
			if oc.err != nil {
				return nil, oc.err
			}
			if oc.statusChecked > 0 {
				if err := db.UpdatePlayerStatus(oc.playerID, oc.statusValid, oc.statusChecked, oc.charID); err != nil {
					return nil, fmt.Errorf("update player status %d: %w", oc.playerID, err)
				}
			}
			if oc.fingerprint != nil {
				if err := db.UpsertPlayerFingerprint(*oc.fingerprint); err != nil {
					return nil, err
				}
				collisionMapMutex.Lock()
				collisionMap[oc.fingerprint.FingerprintHash] = oc.fingerprint.PlayerID
				collisionMapMutex.Unlock()
			}
			if oc.created {
				result.Created++
			}
			if oc.skipped {
				result.Skipped++
			}
			if oc.invalid {
				result.MarkedInvalid++
			}
		}

		logBatchProgress(tracker, logger, result.Processed, batchNumber)
	}

	result.Duration = time.Since(start)
	return result, nil
}

func deriveClassID(c database.PlayerFingerprintCandidate) (int, bool) {
	if c.DetailsClassID.Valid && c.DetailsClassID.Int64 > 0 {
		return int(c.DetailsClassID.Int64), true
	}
	if c.LatestSpecID.Valid && c.LatestSpecID.Int64 > 0 {
		return wow.GetClassIDForSpec(int(c.LatestSpecID.Int64))
	}
	return 0, false
}

func extractKeyTimestamps(resp *blizzard.CharacterAchievementsResponse) (int64, int64, int64, error) {
	var level85, level90, earliestHeroic int64

	for _, ach := range resp.Achievements {
		if ach.CompletedTimestamp == nil || !ach.Criteria.IsCompleted {
			continue
		}
		ts := *ach.CompletedTimestamp
		switch ach.ID {
		case playerid.Level85AchievementID:
			if level85 == 0 || ts < level85 {
				level85 = ts
			}
		case playerid.Level90AchievementID:
			if level90 == 0 || ts < level90 {
				level90 = ts
			}
		default:
			if isHeroicAchievement(ach.ID) {
				if earliestHeroic == 0 || ts < earliestHeroic {
					earliestHeroic = ts
				}
			}
		}
	}

	var missing []string
	if level85 == 0 {
		missing = append(missing, "level85")
	}
	if level90 == 0 {
		missing = append(missing, "level90")
	}
	if earliestHeroic == 0 {
		missing = append(missing, "heroic")
	}
	if len(missing) > 0 {
		return 0, 0, 0, fmt.Errorf("missing achievements: %s", strings.Join(missing, ", "))
	}

	return level85, level90, earliestHeroic, nil
}

func isHeroicAchievement(id int) bool {
	idx := sort.SearchInts(playerid.HeroicDungeonAchievementIDs, id)
	return idx < len(playerid.HeroicDungeonAchievementIDs) && playerid.HeroicDungeonAchievementIDs[idx] == id
}

type fingerprintOutcome struct {
	playerID      int64
	fingerprint   *database.PlayerFingerprint
	statusValid   bool
	statusChecked int64
	charID        *int
	created       bool
	skipped       bool
	invalid       bool
	err           error
}

func processFingerprintCandidate(db *database.DatabaseService, client *blizzard.Client, cand database.PlayerFingerprintCandidate, collisionMap map[string]int64, collisionMapMutex *sync.RWMutex, logger *log.Logger) fingerprintOutcome {
	out := fingerprintOutcome{playerID: cand.PlayerID}
	logger.Info("fingerprinting player",
		"player_id", cand.PlayerID,
		"name", cand.Name,
		"region", strings.ToUpper(cand.Region),
		"realm", cand.RealmSlug)
	classID, ok := deriveClassID(cand)
	if !ok {
		logger.Warn("skipping candidate - no class data, marking invalid",
			"player_id", cand.PlayerID)
		out.invalid = true
		out.statusValid = false
		out.statusChecked = nowMillis()
		return out
	}

	canonicalRealm := blizzard.NormalizeRealmSlug(cand.Region, cand.RealmSlug)
	statusResp, err := client.FetchCharacterStatus(cand.Name, canonicalRealm, cand.Region)
	if err != nil {
		if isNotFoundError(err) {
			logger.Warn("status 404, marking invalid",
				"player_id", cand.PlayerID,
				"name", cand.Name,
				"region", cand.Region,
				"realm", cand.RealmSlug)
			out.invalid = true
			out.statusValid = false
			out.statusChecked = nowMillis()
			return out
		}
		logger.Warn("status fetch error",
			"player_id", cand.PlayerID,
			"name", cand.Name,
			"region", cand.Region,
			"realm", cand.RealmSlug,
			"error", err)
		out.skipped = true
		return out
	}

	out.statusValid = statusResp.IsValid
	out.statusChecked = nowMillis()
	out.charID = extractCharacterID(statusResp)

	if !statusResp.IsValid {
		reason := statusResp.Reason
		if reason == "" {
			reason = "invalid"
		}
		logger.Warn("status invalid",
			"player_id", cand.PlayerID,
			"name", cand.Name,
			"region", cand.Region,
			"realm", cand.RealmSlug,
			"reason", reason)
		out.invalid = true
		return out
	}

	if slug := statusResp.Character.Realm.Slug; slug != "" {
		canonicalRealm = blizzard.NormalizeRealmSlug(cand.Region, slug)
	}
	if name := statusResp.Character.Name; strings.TrimSpace(name) != "" {
		cand.Name = name
	}

	resp, err := client.FetchCharacterAchievements(cand.Name, canonicalRealm, cand.Region)
	if err != nil {
		if isNotFoundError(err) {
			logger.Warn("achievements 404, marking invalid",
				"player_id", cand.PlayerID,
				"name", cand.Name,
				"region", cand.Region,
				"realm", cand.RealmSlug)
			out.invalid = true
			out.statusValid = false
			out.statusChecked = nowMillis()
			return out
		}
		logger.Warn("achievements fetch error",
			"player_id", cand.PlayerID,
			"name", cand.Name,
			"region", cand.Region,
			"realm", cand.RealmSlug,
			"error", err)
		out.skipped = true
		return out
	}

	level85, level90, heroic, err := extractKeyTimestamps(resp)
	if err != nil {
		logger.Warn("missing required achievements, marking invalid",
			"player_id", cand.PlayerID,
			"error", err)
		out.invalid = true
		out.statusValid = false
		out.statusChecked = nowMillis()
		return out
	}

	fpInput := playerid.FingerprintInput{
		ClassID:                 classID,
		Level85Timestamp:        level85,
		Level90Timestamp:        level90,
		EarliestHeroicTimestamp: heroic,
	}
	hash, err := playerid.ComputeHash(fpInput)
	if err != nil {
		logger.Warn("failed to compute fingerprint", "player_id", cand.PlayerID, "error", err)
		out.skipped = true
		return out
	}

	// Check for collision with mutex protection
	collisionMapMutex.RLock()
	existing := collisionMap[hash]
	collisionMapMutex.RUnlock()

	if existing != 0 && existing != cand.PlayerID {
		// Merge: the candidate (cand) is the newer player, existing is the old one
		// Migrate runs FROM old (existing) TO new (cand)
		runsMigrated, err := db.MigratePlayerRuns(existing, cand.PlayerID)
		if err != nil {
			out.err = fmt.Errorf("failed to migrate runs: %w", err)
			return out
		}

		// Mark the old player as invalid
		if err := db.UpdatePlayerStatus(existing, false, nowMillis(), nil); err != nil {
			logger.Warn("failed to invalidate old player",
				"player_id", existing,
				"error", err)
		}

		// Delete old player's fingerprint so the new one becomes canonical
		if err := db.DeletePlayerFingerprint(existing); err != nil {
			logger.Warn("failed to delete old fingerprint",
				"player_id", existing,
				"error", err)
		}

		// Update collision map to point to the new canonical player
		collisionMapMutex.Lock()
		collisionMap[hash] = cand.PlayerID
		collisionMapMutex.Unlock()

		logger.Info("merged player identity (old → new)",
			"old_id", existing,
			"new_id", cand.PlayerID,
			"runs_migrated", runsMigrated,
			"hash", hash[:16])

		// Continue processing to create fingerprint for the new canonical player
	}

	now := nowMillis()
	firstRun := cand.FirstRunTimestamp.Int64
	if !cand.FirstRunTimestamp.Valid || firstRun == 0 {
		firstRun = now
	}
	lastRun := cand.LastRunTimestamp.Int64
	if !cand.LastRunTimestamp.Valid || lastRun == 0 {
		lastRun = now
	}

	lastSeenName := resp.Character.Name
	if lastSeenName == "" {
		lastSeenName = cand.Name
	}
	lastSeenRealm := resp.Character.Realm.Slug
	if lastSeenRealm == "" {
		lastSeenRealm = canonicalRealm
	}

	out.fingerprint = &database.PlayerFingerprint{
		PlayerID:                cand.PlayerID,
		FingerprintHash:         hash,
		ClassID:                 classID,
		Level85Timestamp:        level85,
		Level90Timestamp:        level90,
		EarliestHeroicTimestamp: heroic,
		LastSeenName:            lastSeenName,
		LastSeenRealmSlug:       lastSeenRealm,
		LastSeenTimestamp:       lastRun,
		FirstRunTimestamp:       firstRun,
		CreatedAt:               now,
	}
	out.created = true

	logger.Info("fingerprint ready",
		"player_id", cand.PlayerID,
		"hash", hash[:12],
		"class_id", classID,
		"heroic_ts", heroic)
	return out
}
