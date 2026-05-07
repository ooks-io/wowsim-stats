package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
)

// analyzeAccountsCmd is the prototype account-grouping pass. Reads the
// curated trusted-achievement set from disk, fetches achievements for every
// in-scope 9/9 character, and runs connected-components grouping using
// matching (id, ts) tuples as edges. Prints distribution stats; no DB writes.
var analyzeAccountsCmd = &cobra.Command{
	Use:   "analyze-accounts",
	Short: "Prototype: group 9/9 characters into accounts via account-wide achievement tuples",
	RunE: func(cmd *cobra.Command, args []string) error {
		region, _ := cmd.Flags().GetString("region")
		realm, _ := cmd.Flags().GetString("realm")
		trustedPath, _ := cmd.Flags().GetString("trusted-set")
		workers, _ := cmd.Flags().GetInt("workers")
		topN, _ := cmd.Flags().GetInt("top")
		maxTupleShare, _ := cmd.Flags().GetFloat64("max-tuple-share")
		diagnoseTuples, _ := cmd.Flags().GetBool("diagnose-tuples")
		minEdgeMatches, _ := cmd.Flags().GetInt("min-edge-matches")

		trustedIDs, err := readTrustedSet(trustedPath)
		if err != nil {
			return fmt.Errorf("read trusted set: %w", err)
		}
		fmt.Printf("Trusted achievement IDs: %d\n", len(trustedIDs))

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("db: %w", err)
		}
		defer db.Close()

		players, err := loadInScopePlayers(db, region, realm)
		if err != nil {
			return fmt.Errorf("load players: %w", err)
		}
		fmt.Printf("In-scope 9/9 characters: %d\n", len(players))
		if len(players) == 0 {
			return nil
		}

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("blizzard client: %w", err)
		}

		fmt.Printf("Fetching achievements with %d workers...\n", workers)
		start := time.Now()
		fingerprints := fetchTrustedFingerprints(client, players, trustedIDs, workers)
		fmt.Printf("Fetched in %s\n\n", time.Since(start).Round(time.Second))

		dropped := pruneSharedTuples(fingerprints, maxTupleShare, diagnoseTuples)
		if dropped > 0 {
			fmt.Printf("Dropped %d (id, ts) tuples that exceeded share threshold %.1f%%\n\n",
				dropped, maxTupleShare*100)
		}

		groups := groupByTupleOverlap(fingerprints, minEdgeMatches)
		printAccountStats(groups, fingerprints, players, topN)
		return nil
	},
}

// pruneSharedTuples drops (id, ts) tuples that appear in more than `share`
// fraction of the population - those are almost certainly globally-granted
// (or otherwise non-account-discriminating) and bridge unrelated accounts in
// the union-find. Returns the count of (id, ts) pairs pruned.
func pruneSharedTuples(fps map[int64]*playerFingerprint, share float64, diagnose bool) int {
	if share <= 0 || share >= 1 {
		return 0
	}
	count := make(map[trustedTuple]int)
	for _, fp := range fps {
		for _, t := range fp.Tuples {
			count[t]++
		}
	}

	threshold := int(float64(len(fps)) * share)
	if threshold < 2 {
		threshold = 2
	}

	if diagnose {
		// Show the top frequencies so we can see which tuples are bridging.
		type entry struct {
			Tuple trustedTuple
			Count int
		}
		entries := make([]entry, 0, len(count))
		for t, c := range count {
			entries = append(entries, entry{t, c})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })
		fmt.Printf("Tuple-frequency diagnostic (top 20 of %d unique tuples; threshold %d):\n",
			len(entries), threshold)
		fmt.Printf("  %-8s  %-19s  %s\n", "ID", "TS (UTC)", "PLAYERS")
		for i := 0; i < 20 && i < len(entries); i++ {
			e := entries[i]
			fmt.Printf("  %-8d  %-19s  %d\n",
				e.Tuple.ID, time.UnixMilli(e.Tuple.Ts).UTC().Format("2006-01-02 15:04:05"), e.Count)
		}
		fmt.Println()
	}

	dropSet := make(map[trustedTuple]struct{})
	for t, c := range count {
		if c > threshold {
			dropSet[t] = struct{}{}
		}
	}
	if len(dropSet) == 0 {
		return 0
	}
	for _, fp := range fps {
		kept := fp.Tuples[:0]
		for _, t := range fp.Tuples {
			if _, drop := dropSet[t]; !drop {
				kept = append(kept, t)
			}
		}
		fp.Tuples = kept
	}
	return len(dropSet)
}

// playerScope is one row from the in-scope query: enough to fetch + report.
type playerScope struct {
	PlayerID  int64
	Name      string
	RealmSlug string
	RealmName string
	Region    string
}

func loadInScopePlayers(db *sql.DB, region, realm string) ([]playerScope, error) {
	q := `
		SELECT p.id, p.name, r.slug, r.name, r.region
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		JOIN player_profiles pp ON p.id = pp.player_id
		WHERE pp.has_complete_coverage = 1
	`
	args := []any{}
	if region != "" {
		q += " AND r.region = ?"
		args = append(args, region)
	}
	if realm != "" {
		q += " AND r.slug = ?"
		args = append(args, realm)
	}
	q += " GROUP BY p.id ORDER BY p.id"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []playerScope
	for rows.Next() {
		var p playerScope
		if err := rows.Scan(&p.PlayerID, &p.Name, &p.RealmSlug, &p.RealmName, &p.Region); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// trustedTuple is a (achievement id, ts) pair from the trusted set.
type trustedTuple struct {
	ID int
	Ts int64
}

// playerFingerprint is the per-player extracted set of trusted tuples.
type playerFingerprint struct {
	PlayerID int64
	Tuples   []trustedTuple // sorted by ID for deterministic comparison
	Err      error          // non-nil if fetch failed; player is treated as ungrouped
}

// fetchTrustedFingerprints runs a worker pool to fetch each player's
// achievements and extract their trusted-tuple set.
func fetchTrustedFingerprints(client *blizzard.Client, players []playerScope, trusted map[int]struct{}, workers int) map[int64]*playerFingerprint {
	if workers <= 0 {
		workers = 10
	}

	out := make(map[int64]*playerFingerprint, len(players))
	var mu sync.Mutex

	jobs := make(chan playerScope, workers*2)
	var wg sync.WaitGroup
	var done int64
	var doneMu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				fp := &playerFingerprint{PlayerID: p.PlayerID}
				resp, err := client.FetchCharacterAchievements(p.Name, p.RealmSlug, p.Region)
				if err != nil {
					fp.Err = err
				} else {
					for _, a := range resp.Achievements {
						if a.CompletedTimestamp == nil {
							continue
						}
						if _, ok := trusted[a.ID]; !ok {
							continue
						}
						fp.Tuples = append(fp.Tuples, trustedTuple{ID: a.ID, Ts: *a.CompletedTimestamp})
					}
					sort.Slice(fp.Tuples, func(i, j int) bool { return fp.Tuples[i].ID < fp.Tuples[j].ID })
				}
				mu.Lock()
				out[p.PlayerID] = fp
				mu.Unlock()

				doneMu.Lock()
				done++
				if done%100 == 0 {
					fmt.Printf("  ... %d/%d\n", done, len(players))
				}
				doneMu.Unlock()
			}
		}()
	}
	for _, p := range players {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	return out
}

// groupByTupleOverlap runs union-find on player_ids: any two players that
// share at least minMatches tuples from the trusted set get unioned. minMatches
// >= 2 is much more robust against single-tuple raid-team-style false bridges.
func groupByTupleOverlap(fps map[int64]*playerFingerprint, minMatches int) map[int64][]int64 {
	if minMatches < 1 {
		minMatches = 1
	}

	// pair counts: for each pair of players that share a tuple, increment.
	type pairKey struct{ A, B int64 }
	pairCount := make(map[pairKey]int)

	tuplePlayers := make(map[trustedTuple][]int64)
	for pid, fp := range fps {
		if fp.Err != nil {
			continue
		}
		for _, t := range fp.Tuples {
			tuplePlayers[t] = append(tuplePlayers[t], pid)
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

	uf := newPlayerUF()
	for pid := range fps {
		uf.parent[pid] = pid
	}
	for k, c := range pairCount {
		if c >= minMatches {
			uf.union(k.A, k.B)
		}
	}

	groups := make(map[int64][]int64)
	for pid := range fps {
		root := uf.find(pid)
		groups[root] = append(groups[root], pid)
	}
	return groups
}

// playerUF is a tiny union-find over player_ids.
type playerUF struct {
	parent map[int64]int64
}

func newPlayerUF() *playerUF { return &playerUF{parent: make(map[int64]int64)} }

func (u *playerUF) find(k int64) int64 {
	p, ok := u.parent[k]
	if !ok {
		u.parent[k] = k
		return k
	}
	if p == k {
		return k
	}
	r := u.find(p)
	u.parent[k] = r
	return r
}

func (u *playerUF) union(a, b int64) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

// printAccountStats summarizes the grouping: histogram + top-N largest.
func printAccountStats(groups map[int64][]int64, fps map[int64]*playerFingerprint, players []playerScope, topN int) {
	playerByID := make(map[int64]playerScope, len(players))
	for _, p := range players {
		playerByID[p.PlayerID] = p
	}

	// Categorize each character.
	noTuples := 0
	fetchErr := 0
	for _, fp := range fps {
		if fp.Err != nil {
			fetchErr++
		} else if len(fp.Tuples) == 0 {
			noTuples++
		}
	}

	// Group sizes.
	hist := make(map[int]int)
	var multiCharGroups []int64
	for root, members := range groups {
		hist[len(members)]++
		if len(members) > 1 {
			multiCharGroups = append(multiCharGroups, root)
		}
	}

	fmt.Printf("=== Account grouping report ===\n")
	fmt.Printf("Characters: %d\n", len(fps))
	fmt.Printf("  with trusted tuples: %d\n", len(fps)-noTuples-fetchErr)
	fmt.Printf("  no trusted tuples (ungrouped singletons): %d\n", noTuples)
	fmt.Printf("  fetch errors: %d\n", fetchErr)
	fmt.Printf("Accounts (= connected components): %d\n", len(groups))
	fmt.Printf("Multi-character accounts: %d\n\n", len(multiCharGroups))

	// Histogram of account sizes.
	sizes := make([]int, 0, len(hist))
	for s := range hist {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	fmt.Printf("Account-size histogram:\n")
	fmt.Printf("  %-6s %s\n", "SIZE", "ACCOUNTS")
	for _, s := range sizes {
		fmt.Printf("  %-6d %d\n", s, hist[s])
	}

	// Top-N largest accounts.
	sort.Slice(multiCharGroups, func(i, j int) bool {
		return len(groups[multiCharGroups[i]]) > len(groups[multiCharGroups[j]])
	})
	if topN <= 0 || topN > len(multiCharGroups) {
		topN = len(multiCharGroups)
	}
	if topN == 0 {
		return
	}
	fmt.Printf("\nTop %d largest accounts:\n", topN)
	for i := 0; i < topN; i++ {
		root := multiCharGroups[i]
		members := groups[root]
		fmt.Printf("  account #%d (%d chars):\n", i+1, len(members))
		// Sort members by realm/name for stable output.
		sort.Slice(members, func(a, b int) bool {
			pa, pb := playerByID[members[a]], playerByID[members[b]]
			if pa.Region != pb.Region {
				return pa.Region < pb.Region
			}
			if pa.RealmSlug != pb.RealmSlug {
				return pa.RealmSlug < pb.RealmSlug
			}
			return pa.Name < pb.Name
		})
		for _, m := range members {
			p := playerByID[m]
			fp := fps[m]
			tupleCount := 0
			if fp != nil {
				tupleCount = len(fp.Tuples)
			}
			fmt.Printf("    %-3s/%-20s %-25s (%d trusted tuples)\n",
				p.Region, p.RealmSlug, p.Name, tupleCount)
		}
	}
}

// readTrustedSet pulls the curated trusted achievement IDs out of the report
// JSON written by `investigate-account --output ...`.
func readTrustedSet(path string) (map[int]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rep struct {
		Trusted []struct {
			ID int `json:"id"`
		} `json:"trusted"`
	}
	if err := json.NewDecoder(f).Decode(&rep); err != nil {
		return nil, err
	}
	out := make(map[int]struct{}, len(rep.Trusted))
	for _, e := range rep.Trusted {
		out[e.ID] = struct{}{}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("trusted set is empty in %s", path)
	}
	return out, nil
}

func init() {
	rootCmd.AddCommand(analyzeAccountsCmd)
	analyzeAccountsCmd.Flags().String("region", "", "Filter to one region (us|eu|kr|tw); blank = all")
	analyzeAccountsCmd.Flags().String("realm", "", "Filter to one realm slug; blank = all")
	analyzeAccountsCmd.Flags().String("trusted-set", "account-fingerprint-trusted.json", "Path to the curated trusted-set JSON")
	analyzeAccountsCmd.Flags().Int("workers", 10, "Concurrent achievement fetches")
	analyzeAccountsCmd.Flags().Int("top", 20, "Show this many largest accounts (0 = all multi-char accounts)")
	analyzeAccountsCmd.Flags().Float64("max-tuple-share", 0.05, "Drop (id, ts) tuples seen in more than this fraction of the population (0 = no prune)")
	analyzeAccountsCmd.Flags().Bool("diagnose-tuples", false, "Print tuple-frequency diagnostic before grouping")
	analyzeAccountsCmd.Flags().Int("min-edge-matches", 2, "Require N matching tuples between two characters before unioning them")
}
