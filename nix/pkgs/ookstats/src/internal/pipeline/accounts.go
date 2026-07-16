package pipeline

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
	"ookstats/internal/playerid"
)

type AccountGroupingOptions struct {
	MaxTupleShare   float64
	MinEdgeMatches  int
	BackfillWorkers int
	MaxBackfill     int
	ManualLinksPath string
}

func defaultedOptions(o AccountGroupingOptions) AccountGroupingOptions {
	if o.MaxTupleShare <= 0 {
		o.MaxTupleShare = 0.05
	}
	if o.MinEdgeMatches <= 0 {
		o.MinEdgeMatches = 5
	}
	if o.BackfillWorkers <= 0 {
		o.BackfillWorkers = 20
	}
	return o
}

type AccountResult struct {
	Backfilled        int
	BackfillErrors    int
	Characters        int
	Accounts          int
	MultiCharAccounts int
	Duration          time.Duration
}

func ProcessAccounts(db *database.DatabaseService, client *blizzard.Client, opts AccountGroupingOptions) (*AccountResult, error) {
	opts = defaultedOptions(opts)
	logger := log.With("component", "accounts")
	start := time.Now()

	res := &AccountResult{}

	if client != nil {
		backfilled, errs, err := backfillAccountFingerprints(db, client, opts, logger)
		if err != nil {
			return nil, fmt.Errorf("backfill: %w", err)
		}
		res.Backfilled = backfilled
		res.BackfillErrors = errs
	}

	rows, err := db.LoadAllPlayerAccountFingerprints()
	if err != nil {
		return nil, fmt.Errorf("load fingerprints: %w", err)
	}
	logger.Info("loaded fingerprints", "count", len(rows))

	forced, unresolved, err := loadForcedEdges(db, opts.ManualLinksPath, logger)
	if err != nil {
		return nil, fmt.Errorf("manual links: %w", err)
	}
	if len(forced) > 0 || unresolved > 0 {
		logger.Info("manual links applied", "edges", len(forced), "unresolved", unresolved)
	}

	groups := groupAccounts(rows, forced, opts)
	res.Characters = len(rows)
	res.Accounts = len(groups)
	for _, g := range groups {
		if len(g.PlayerIDs) > 1 {
			res.MultiCharAccounts++
		}
	}

	logger.Info("rebuilding accounts", "groups", len(groups), "multi_char", res.MultiCharAccounts)
	stamped, err := db.RebuildAccounts(groups, time.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("rebuild accounts: %w", err)
	}
	logger.Info("accounts rebuilt", "characters_stamped", stamped)

	res.Duration = time.Since(start)
	return res, nil
}

// deleted or private characters fail every fetch; wait out a cooldown instead of retrying each build
const accountBackfillRetryAfter = 7 * 24 * time.Hour

func backfillAccountFingerprints(db *database.DatabaseService, client *blizzard.Client, opts AccountGroupingOptions, logger *log.Logger) (int, int, error) {
	retryBefore := time.Now().Add(-accountBackfillRetryAfter).UnixMilli()
	candidates, err := db.PlayersMissingAccountFingerprint(retryBefore)
	if err != nil {
		return 0, 0, err
	}
	if len(candidates) == 0 {
		logger.Info("no backfill needed")
		return 0, 0, nil
	}
	if opts.MaxBackfill > 0 && len(candidates) > opts.MaxBackfill {
		candidates = candidates[:opts.MaxBackfill]
	}
	logger.Info("backfilling account fingerprints", "candidates", len(candidates), "workers", opts.BackfillWorkers)

	jobs := make(chan database.AccountBackfillCandidate, opts.BackfillWorkers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var done, success, fail int

	for i := 0; i < opts.BackfillWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				resp, err := client.FetchCharacterAchievements(c.Name, c.RealmSlug, c.Region)
				mu.Lock()
				done++
				if done%500 == 0 {
					logger.Info("backfill progress", "done", done, "total", len(candidates))
				}
				mu.Unlock()
				if err != nil {
					mu.Lock()
					fail++
					mu.Unlock()
					continue
				}
				tuples := playerid.ExtractTrustedTuples(resp)
				if len(tuples) == 0 {
					// skip rather than insert empty; the attempt stamp defers retry to the next cooldown
					continue
				}
				j, err := json.Marshal(tuples)
				if err != nil {
					mu.Lock()
					fail++
					mu.Unlock()
					continue
				}
				if err := db.UpsertPlayerAccountFingerprint(c.PlayerID, c.Region, string(j), time.Now().UnixMilli()); err != nil {
					mu.Lock()
					fail++
					mu.Unlock()
					continue
				}
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	for _, c := range candidates {
		jobs <- c
	}
	close(jobs)
	wg.Wait()

	ids := make([]int64, len(candidates))
	for i, c := range candidates {
		ids[i] = c.PlayerID
	}
	if err := db.MarkAccountFingerprintAttempts(ids, time.Now().UnixMilli()); err != nil {
		logger.Error("mark backfill attempts", "error", err)
	}

	logger.Info("backfill complete", "success", success, "errors", fail)
	return success, fail, nil
}

// region carried so manual-link-only chars still get a region in their group
type ForcedEdge struct {
	A, B   int64
	Region string
}

// stale entries log + skip; pipeline doesn't fail if file is out of date
func loadForcedEdges(db *database.DatabaseService, path string, logger *log.Logger) ([]ForcedEdge, int, error) {
	if path == "" {
		return nil, 0, nil
	}
	links, err := playerid.LoadManualLinks(path)
	if err != nil {
		return nil, 0, err
	}
	if len(links) == 0 {
		return nil, 0, nil
	}
	out := make([]ForcedEdge, 0, len(links))
	unresolved := 0
	for _, l := range links {
		ca, cb := l.A.Canonical(), l.B.Canonical()
		if ca.Region != cb.Region {
			logger.Warn("manual link spans regions; skipping", "a", l.A, "b", l.B)
			unresolved++
			continue
		}
		idA, errA := db.GetPlayerByNameRealmRegion(ca.Name, ca.Realm, ca.Region)
		if errA != nil {
			logger.Warn("manual link: cannot resolve A; skipping", "a", l.A, "err", errA)
			unresolved++
			continue
		}
		idB, errB := db.GetPlayerByNameRealmRegion(cb.Name, cb.Realm, cb.Region)
		if errB != nil {
			logger.Warn("manual link: cannot resolve B; skipping", "b", l.B, "err", errB)
			unresolved++
			continue
		}
		if idA == idB {
			continue
		}
		out = append(out, ForcedEdge{A: idA, B: idB, Region: ca.Region})
	}
	return out, unresolved, nil
}

// forcedEdges bypass the threshold filters so manual links survive even when the algorithm wouldn't have inferred them
func groupAccounts(rows []database.PlayerAccountFingerprint, forcedEdges []ForcedEdge, opts AccountGroupingOptions) []database.AccountGroup {
	type tuple struct {
		ID int   `json:"id"`
		Ts int64 `json:"ts"`
	}
	type tupleKey struct {
		Region string
		ID     int
		Ts     int64
	}

	playerRegion := make(map[int64]string, len(rows))
	playerTuples := make(map[int64][]tupleKey, len(rows))
	tupleCount := make(map[tupleKey]int)

	for _, r := range rows {
		var raw []tuple
		if err := json.Unmarshal([]byte(r.TuplesJSON), &raw); err != nil {
			continue
		}
		playerRegion[r.PlayerID] = r.Region
		ks := make([]tupleKey, 0, len(raw))
		for _, t := range raw {
			k := tupleKey{Region: r.Region, ID: t.ID, Ts: t.Ts}
			ks = append(ks, k)
			tupleCount[k]++
		}
		playerTuples[r.PlayerID] = ks
	}

	// per-region threshold so a small region's dense overlaps don't get falsely pruned
	regionPop := make(map[string]int)
	for _, region := range playerRegion {
		regionPop[region]++
	}
	dropped := make(map[tupleKey]struct{})
	for k, c := range tupleCount {
		threshold := int(float64(regionPop[k.Region]) * opts.MaxTupleShare)
		if threshold < 2 {
			threshold = 2
		}
		if c > threshold {
			dropped[k] = struct{}{}
		}
	}

	type pairKey struct{ A, B int64 }
	pairCount := make(map[pairKey]int)
	tuplePlayers := make(map[tupleKey][]int64)
	for pid, keys := range playerTuples {
		for _, k := range keys {
			if _, drop := dropped[k]; drop {
				continue
			}
			tuplePlayers[k] = append(tuplePlayers[k], pid)
		}
	}
	for _, ps := range tuplePlayers {
		if len(ps) < 2 {
			continue
		}
		for i := 0; i < len(ps); i++ {
			for j := i + 1; j < len(ps); j++ {
				a, b := ps[i], ps[j]
				if a > b {
					a, b = b, a
				}
				pairCount[pairKey{a, b}]++
			}
		}
	}

	parent := make(map[int64]int64, len(playerTuples))
	for pid := range playerTuples {
		parent[pid] = pid
	}
	var find func(int64) int64
	find = func(k int64) int64 {
		p := parent[k]
		if p == k {
			return k
		}
		r := find(p)
		parent[k] = r
		return r
	}
	union := func(a, b int64) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}
	for k, c := range pairCount {
		if c >= opts.MinEdgeMatches {
			union(k.A, k.B)
		}
	}

	// seed parent + region so manual-link-only chars without a fingerprint still land in the resulting account
	for _, e := range forcedEdges {
		if _, ok := parent[e.A]; !ok {
			parent[e.A] = e.A
		}
		if _, ok := parent[e.B]; !ok {
			parent[e.B] = e.B
		}
		if _, ok := playerRegion[e.A]; !ok && e.Region != "" {
			playerRegion[e.A] = e.Region
		}
		if _, ok := playerRegion[e.B]; !ok && e.Region != "" {
			playerRegion[e.B] = e.Region
		}
		union(e.A, e.B)
	}

	// iterate parent (not playerTuples) so manual-link-only chars get emitted
	byRoot := make(map[int64][]int64)
	for pid := range parent {
		root := find(pid)
		byRoot[root] = append(byRoot[root], pid)
	}
	out := make([]database.AccountGroup, 0, len(byRoot))
	for root, ids := range byRoot {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		out = append(out, database.AccountGroup{
			Region:    playerRegion[root],
			PlayerIDs: ids,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].PlayerIDs) != len(out[j].PlayerIDs) {
			return len(out[i].PlayerIDs) > len(out[j].PlayerIDs)
		}
		return out[i].PlayerIDs[0] < out[j].PlayerIDs[0]
	})
	return out
}
