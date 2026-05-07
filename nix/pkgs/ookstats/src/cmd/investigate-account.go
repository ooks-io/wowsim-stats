package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
)

var investigateAccountCmd = &cobra.Command{
	Use:   "investigate-account",
	Short: "Probe shared-account fingerprinting via account-wide achievement timestamps",
	Long: `Single pair: --region/--realm-a/--name-a/--name-b/--realm-b
Batch:       --pairs-file pairs.csv

Batch mode reads labelled pairs (same/different account), runs the probe on
each, then aggregates to suggest a trusted account-wide allowlist plus a
blocklist of globally-granted achievements.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pairsFile, _ := cmd.Flags().GetString("pairs-file")
		if pairsFile != "" {
			return runBatch(cmd, pairsFile)
		}
		return runSinglePair(cmd)
	},
}

// pairResult holds the per-pair probe outcome that the batch aggregator needs.
type pairResult struct {
	Region       string
	Label        string // human-readable, e.g. "Saiyoran/Saiyopink"
	Relationship string // "same" | "different"
	// CharA and CharB are the canonical (region, realm, name) keys for each
	// character; the aggregator union-finds across `same` pairs to dedupe
	// accounts when the same character appears in multiple pairs.
	CharA charKey
	CharB charKey
	// matches: id -> ts (only entries where both characters had a non-nil
	// completed_timestamp AND the timestamps were equal)
	Matches map[int]int64
}

// charKey identifies a character. Lowercased on construction so name
// capitalisation in the input file doesn't fragment the union-find.
type charKey struct {
	Region string
	Realm  string
	Name   string
}

func makeCharKey(region, realm, name string) charKey {
	return charKey{
		Region: strings.ToLower(region),
		Realm:  strings.ToLower(realm),
		Name:   strings.ToLower(name),
	}
}

// unionFind is a tiny path-compressed disjoint-set over charKey. Used to
// collapse `same` pairs into account-equivalence classes before aggregating.
type unionFind struct {
	parent map[charKey]charKey
}

func newUnionFind() *unionFind {
	return &unionFind{parent: make(map[charKey]charKey)}
}

func (u *unionFind) find(k charKey) charKey {
	p, ok := u.parent[k]
	if !ok {
		u.parent[k] = k
		return k
	}
	if p == k {
		return k
	}
	root := u.find(p)
	u.parent[k] = root
	return root
}

func (u *unionFind) union(a, b charKey) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

// runSinglePair preserves the original interactive probe behaviour.
func runSinglePair(cmd *cobra.Command) error {
	region, _ := cmd.Flags().GetString("region")
	realmA, _ := cmd.Flags().GetString("realm-a")
	nameA, _ := cmd.Flags().GetString("name-a")
	realmB, _ := cmd.Flags().GetString("realm-b")
	nameB, _ := cmd.Flags().GetString("name-b")
	showDefs, _ := cmd.Flags().GetBool("definitions")
	limitDefs, _ := cmd.Flags().GetInt("definitions-limit")

	if region == "" || realmA == "" || nameA == "" || nameB == "" {
		return fmt.Errorf("--region, --realm-a, --name-a, --name-b are required (or use --pairs-file)")
	}
	if realmB == "" {
		realmB = realmA
	}

	client, err := blizzard.NewClient()
	if err != nil {
		return fmt.Errorf("blizzard client: %w", err)
	}

	fmt.Printf("=== Account Fingerprint Investigation ===\n")
	fmt.Printf("Region: %s\n", region)
	fmt.Printf("A: %s on %s\n", nameA, realmA)
	fmt.Printf("B: %s on %s\n\n", nameB, realmB)

	res, aCompleted, bCompleted, mismatchCount, err := probePair(client, region, realmA, nameA, realmB, nameB)
	if err != nil {
		return err
	}

	fmt.Printf("Completed achievements - A: %d, B: %d\n", aCompleted, bCompleted)
	fmt.Printf("Achievements completed on both: %d\n", len(res.Matches)+mismatchCount)
	fmt.Printf("  -> identical timestamp: %d (likely account-wide)\n", len(res.Matches))
	fmt.Printf("  -> different timestamp: %d (character-specific)\n\n", mismatchCount)

	if len(res.Matches) == 0 {
		fmt.Println("No timestamp overlap. Either different accounts, or no overlapping account-wide achievements.")
		return nil
	}

	fmt.Printf("VERDICT: %s and %s share an account (confidence: very high)\n\n", nameA, nameB)

	matches := sortedMatchEntries(res.Matches)

	if showDefs {
		n := limitDefs
		if n <= 0 || n > len(matches) {
			n = len(matches)
		}
		fmt.Printf("=== Sample of %d/%d matching achievements ===\n", n, len(matches))
		fmt.Printf("%-8s  %-19s  %-7s  %s\n", "ID", "Completed (UTC)", "AcctWide", "Name")
		acctWideCount := 0
		fetched := 0
		cache := make(map[int]*blizzard.AchievementDetailResponse)
		for i := 0; i < n; i++ {
			m := matches[i]
			det, err := fetchAchievementCached(client, region, m.ID, cache)
			if err != nil {
				fmt.Printf("%-8d  %-19s  %-7s  <fetch error: %v>\n",
					m.ID, fmtTs(m.Ts), "?", err)
				continue
			}
			fetched++
			if det.IsAccountWide {
				acctWideCount++
			}
			fmt.Printf("%-8d  %-19s  %-7s  %s\n", det.ID, fmtTs(m.Ts), yesNo(det.IsAccountWide), det.Name)
		}
		fmt.Printf("\n  %d/%d sampled matches are flagged is_account_wide=true\n", acctWideCount, fetched)
	} else {
		fmt.Printf("=== %d matching achievements (id, completed_timestamp) ===\n", len(matches))
		fmt.Printf("(Pass --definitions to fetch names + is_account_wide flag)\n\n")
		for _, m := range matches {
			fmt.Printf("  %-8d  %s\n", m.ID, fmtTs(m.Ts))
		}
	}
	return nil
}

// runBatch reads a CSV of labelled pairs and emits an aggregate fingerprint
// analysis suitable for building the trusted set + blocklist.
func runBatch(cmd *cobra.Command, pairsFile string) error {
	outFile, _ := cmd.Flags().GetString("output")

	client, err := blizzard.NewClient()
	if err != nil {
		return fmt.Errorf("blizzard client: %w", err)
	}

	pairs, err := readPairsFile(pairsFile)
	if err != nil {
		return fmt.Errorf("read pairs file: %w", err)
	}

	fmt.Printf("=== Batch Account Fingerprint Investigation ===\n")
	fmt.Printf("Pairs: %d (same: %d, different: %d)\n\n",
		len(pairs), countRel(pairs, "same"), countRel(pairs, "different"))

	defCache := make(map[int]*blizzard.AchievementDetailResponse)
	results := make([]pairResult, 0, len(pairs))

	for i, p := range pairs {
		fmt.Printf("[%d/%d] %s (%s)... ", i+1, len(pairs), p.Label, p.Relationship)
		res, _, _, _, err := probePair(client, p.Region, p.RealmA, p.NameA, p.RealmB, p.NameB)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		res.Region = p.Region
		res.Label = p.Label
		res.Relationship = p.Relationship
		res.CharA = makeCharKey(p.Region, p.RealmA, p.NameA)
		res.CharB = makeCharKey(p.Region, p.RealmB, p.NameB)
		results = append(results, res)
		fmt.Printf("%d matches\n", len(res.Matches))
	}

	fmt.Printf("\nFetching achievement definitions for %d unique IDs...\n", uniqueMatchedIDs(results))
	for _, r := range results {
		for id := range r.Matches {
			if _, err := fetchAchievementCached(client, r.Region, id, defCache); err != nil {
				fmt.Printf("  WARN: id %d: %v\n", id, err)
			}
		}
	}

	report := analyse(results, defCache)
	printReport(report, defCache)

	if outFile != "" {
		if err := writeReportJSON(outFile, report, defCache); err != nil {
			return fmt.Errorf("write %s: %w", outFile, err)
		}
		fmt.Printf("\nWrote curated set to %s\n", outFile)
	}
	return nil
}

// pairInput is the parsed CSV row.
type pairInput struct {
	Region       string
	RealmA       string
	NameA        string
	RealmB       string
	NameB        string
	Relationship string
	Label        string
}

func readPairsFile(path string) ([]pairInput, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.Comment = '#'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	// Header is required so column order is explicit.
	header := rows[0]
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[strings.ToLower(strings.TrimSpace(col))] = i
	}
	required := []string{"region", "realm_a", "name_a", "name_b", "relationship"}
	for _, c := range required {
		if _, ok := idx[c]; !ok {
			return nil, fmt.Errorf("missing required column: %s", c)
		}
	}

	out := make([]pairInput, 0, len(rows)-1)
	for n, row := range rows[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}
		get := func(col string) string {
			i, ok := idx[col]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		p := pairInput{
			Region:       strings.ToLower(get("region")),
			RealmA:       strings.ToLower(get("realm_a")),
			NameA:        get("name_a"),
			RealmB:       strings.ToLower(get("realm_b")),
			NameB:        get("name_b"),
			Relationship: strings.ToLower(get("relationship")),
		}
		if p.RealmB == "" {
			p.RealmB = p.RealmA
		}
		if p.Region == "" || p.RealmA == "" || p.NameA == "" || p.NameB == "" {
			return nil, fmt.Errorf("row %d: missing required field", n+2)
		}
		if p.Relationship != "same" && p.Relationship != "different" {
			return nil, fmt.Errorf("row %d: relationship must be 'same' or 'different', got %q", n+2, p.Relationship)
		}
		p.Label = fmt.Sprintf("%s/%s", p.NameA, p.NameB)
		out = append(out, p)
	}
	return out, nil
}

// probePair fetches both characters' achievements and returns the matched set.
func probePair(c *blizzard.Client, region, realmA, nameA, realmB, nameB string) (pairResult, int, int, int, error) {
	res := pairResult{Matches: make(map[int]int64)}

	achA, err := c.FetchCharacterAchievements(nameA, realmA, region)
	if err != nil {
		return res, 0, 0, 0, fmt.Errorf("fetch %s: %w", nameA, err)
	}
	achB, err := c.FetchCharacterAchievements(nameB, realmB, region)
	if err != nil {
		return res, 0, 0, 0, fmt.Errorf("fetch %s: %w", nameB, err)
	}

	bByID := make(map[int]int64, len(achB.Achievements))
	bCompleted := 0
	for _, a := range achB.Achievements {
		if a.CompletedTimestamp != nil {
			bByID[a.ID] = *a.CompletedTimestamp
			bCompleted++
		}
	}
	aCompleted := 0
	mismatch := 0
	for _, a := range achA.Achievements {
		if a.CompletedTimestamp == nil {
			continue
		}
		aCompleted++
		tsB, ok := bByID[a.ID]
		if !ok {
			continue
		}
		if *a.CompletedTimestamp == tsB {
			res.Matches[a.ID] = tsB
		} else {
			mismatch++
		}
	}
	return res, aCompleted, bCompleted, mismatch, nil
}

// achievementStat aggregates one achievement's behaviour across all accounts.
// Same-pair matches are deduped via union-find: pairs that share a character
// collapse to the same account, so an achievement only earned once on that
// account is counted once regardless of how many pairs sampled it.
type achievementStat struct {
	ID               int
	IsAccountWide    bool
	SameAccounts     int   // distinct accounts that matched on this id
	SameAccountTs    []int64 // one ts per matching account (deduped per account)
	DifferentPairs   int   // count of `different`-pair matches (cross-account collisions)
	DistinctTsInSame int   // unique ts across the matched accounts
}

// classification is the suggested bucket for an achievement.
type classification string

const (
	clsTrusted        classification = "trusted"
	clsGloballyGranted classification = "globally-granted"
	clsCharSpecific   classification = "character-specific"
	clsThin           classification = "thin"
)

// reportRow is one line in the final aggregated report.
type reportRow struct {
	Stat           achievementStat
	Classification classification
	Reason         string
}

// analyse walks the per-pair results and classifies each matched achievement.
// Same-pairs that share a character collapse into one account via union-find,
// so the per-achievement counts reflect distinct accounts rather than pairs.
func analyse(results []pairResult, defCache map[int]*blizzard.AchievementDetailResponse) []reportRow {
	uf := newUnionFind()
	for _, r := range results {
		if r.Relationship == "same" {
			uf.union(r.CharA, r.CharB)
		}
	}

	// id -> account-root -> ts (one ts per account, since within an account
	// the matching ts is the same by construction)
	perID := make(map[int]map[charKey]int64)
	differentPairs := make(map[int]int)
	for _, r := range results {
		if r.Relationship != "same" {
			for id := range r.Matches {
				differentPairs[id]++
			}
			continue
		}
		acct := uf.find(r.CharA)
		for id, ts := range r.Matches {
			byAcct, ok := perID[id]
			if !ok {
				byAcct = make(map[charKey]int64)
				perID[id] = byAcct
			}
			byAcct[acct] = ts
		}
	}

	stats := make(map[int]*achievementStat)
	for id, byAcct := range perID {
		s := &achievementStat{ID: id}
		if det, hit := defCache[id]; hit {
			s.IsAccountWide = det.IsAccountWide
		}
		s.SameAccounts = len(byAcct)
		distinct := make(map[int64]struct{})
		for _, ts := range byAcct {
			s.SameAccountTs = append(s.SameAccountTs, ts)
			distinct[ts] = struct{}{}
		}
		s.DistinctTsInSame = len(distinct)
		stats[id] = s
	}
	// Cover ids only seen in different-pairs.
	for id, n := range differentPairs {
		s, ok := stats[id]
		if !ok {
			s = &achievementStat{ID: id}
			if det, hit := defCache[id]; hit {
				s.IsAccountWide = det.IsAccountWide
			}
			stats[id] = s
		}
		s.DifferentPairs = n
	}
	// Apply different-pair counts to ids that also matched in same-pairs.
	for id, n := range differentPairs {
		if s, ok := stats[id]; ok && s.DifferentPairs == 0 {
			s.DifferentPairs = n
		}
	}

	rows := make([]reportRow, 0, len(stats))
	for _, s := range stats {
		row := reportRow{Stat: *s}
		switch {
		case !s.IsAccountWide:
			row.Classification = clsCharSpecific
			row.Reason = "is_account_wide=false"
		case s.DifferentPairs > 0:
			// Same ts on a genuinely-different account - globally-granted.
			row.Classification = clsGloballyGranted
			row.Reason = fmt.Sprintf("matched in %d different-account pair(s)", s.DifferentPairs)
		case s.SameAccounts >= 2 && s.DistinctTsInSame == 1:
			// Multiple distinct accounts share one ts - globally-granted.
			row.Classification = clsGloballyGranted
			row.Reason = fmt.Sprintf("%d accounts share one ts", s.SameAccounts)
		case s.SameAccounts >= 2 && s.DistinctTsInSame == s.SameAccounts:
			row.Classification = clsTrusted
			row.Reason = fmt.Sprintf("%d accounts, all distinct ts", s.SameAccounts)
		case s.SameAccounts == 1:
			row.Classification = clsThin
			row.Reason = "only 1 account sampled"
		default:
			row.Classification = clsTrusted
			row.Reason = fmt.Sprintf("%d accounts, %d distinct ts", s.SameAccounts, s.DistinctTsInSame)
		}
		rows = append(rows, row)
	}

	// Trusted first, then thin, globally-granted, character-specific. Within a
	// class, sort by sample count desc so the most-evidenced rows surface first.
	classRank := map[classification]int{
		clsTrusted: 0, clsThin: 1, clsGloballyGranted: 2, clsCharSpecific: 3,
	}
	sort.Slice(rows, func(i, j int) bool {
		if classRank[rows[i].Classification] != classRank[rows[j].Classification] {
			return classRank[rows[i].Classification] < classRank[rows[j].Classification]
		}
		ai := rows[i].Stat.SameAccounts + rows[i].Stat.DifferentPairs
		aj := rows[j].Stat.SameAccounts + rows[j].Stat.DifferentPairs
		if ai != aj {
			return ai > aj
		}
		return rows[i].Stat.ID < rows[j].Stat.ID
	})
	return rows
}

func printReport(rows []reportRow, defCache map[int]*blizzard.AchievementDetailResponse) {
	counts := make(map[classification]int)
	for _, r := range rows {
		counts[r.Classification]++
	}

	fmt.Printf("\n=== Aggregate report (%d unique achievements) ===\n", len(rows))
	fmt.Printf("trusted: %d, thin: %d, globally-granted: %d, character-specific: %d\n\n",
		counts[clsTrusted], counts[clsThin], counts[clsGloballyGranted], counts[clsCharSpecific])

	fmt.Printf("%-9s  %-8s  %-5s  %-5s  %-5s  %-32s  %s\n",
		"CLASS", "ID", "ACCTS", "DIFFt", "DISTt", "REASON", "NAME")
	for _, r := range rows {
		name := "?"
		if det, ok := defCache[r.Stat.ID]; ok {
			name = det.Name
		}
		fmt.Printf("%-9s  %-8d  %-5d  %-5d  %-5d  %-32s  %s\n",
			r.Classification, r.Stat.ID,
			r.Stat.SameAccounts, r.Stat.DifferentPairs, r.Stat.DistinctTsInSame,
			r.Reason, name,
		)
	}
}

// jsonReport is the on-disk shape of the curated set.
type jsonReport struct {
	GeneratedAt       int64     `json:"generated_at"`
	Trusted           []jsonAch `json:"trusted"`
	Thin              []jsonAch `json:"thin"`
	GloballyGranted   []jsonAch `json:"globally_granted"`
	CharacterSpecific []jsonAch `json:"character_specific"`
}

type jsonAch struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	IsAccountWide bool   `json:"is_account_wide"`
	SameAccounts  int    `json:"same_accounts"`
	DiffPairs     int    `json:"different_pairs"`
	DistinctTs    int    `json:"distinct_ts_in_same"`
	Reason        string `json:"reason"`
}

func writeReportJSON(path string, rows []reportRow, defCache map[int]*blizzard.AchievementDetailResponse) error {
	rep := jsonReport{GeneratedAt: time.Now().UnixMilli()}
	for _, r := range rows {
		entry := jsonAch{
			ID:            r.Stat.ID,
			IsAccountWide: r.Stat.IsAccountWide,
			SameAccounts:  r.Stat.SameAccounts,
			DiffPairs:     r.Stat.DifferentPairs,
			DistinctTs:    r.Stat.DistinctTsInSame,
			Reason:        r.Reason,
		}
		if det, ok := defCache[r.Stat.ID]; ok {
			entry.Name = det.Name
		}
		switch r.Classification {
		case clsTrusted:
			rep.Trusted = append(rep.Trusted, entry)
		case clsThin:
			rep.Thin = append(rep.Thin, entry)
		case clsGloballyGranted:
			rep.GloballyGranted = append(rep.GloballyGranted, entry)
		case clsCharSpecific:
			rep.CharacterSpecific = append(rep.CharacterSpecific, entry)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

// --- helpers ---------------------------------------------------------------

type matchEntry struct {
	ID int
	Ts int64
}

func sortedMatchEntries(m map[int]int64) []matchEntry {
	out := make([]matchEntry, 0, len(m))
	for id, ts := range m {
		out = append(out, matchEntry{ID: id, Ts: ts})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts < out[j].Ts })
	return out
}

func fetchAchievementCached(c *blizzard.Client, region string, id int, cache map[int]*blizzard.AchievementDetailResponse) (*blizzard.AchievementDetailResponse, error) {
	if det, ok := cache[id]; ok {
		return det, nil
	}
	det, err := c.FetchAchievement(id, region)
	if err != nil {
		return nil, err
	}
	cache[id] = det
	return det, nil
}

func uniqueMatchedIDs(rs []pairResult) int {
	set := make(map[int]struct{})
	for _, r := range rs {
		for id := range r.Matches {
			set[id] = struct{}{}
		}
	}
	return len(set)
}

func countRel(pairs []pairInput, rel string) int {
	n := 0
	for _, p := range pairs {
		if p.Relationship == rel {
			n++
		}
	}
	return n
}

func fmtTs(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02 15:04:05")
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func init() {
	rootCmd.AddCommand(investigateAccountCmd)
	investigateAccountCmd.Flags().String("region", "", "Region (us|eu|kr|tw)")
	investigateAccountCmd.Flags().String("realm-a", "", "Realm slug for character A")
	investigateAccountCmd.Flags().String("name-a", "", "Name of character A")
	investigateAccountCmd.Flags().String("realm-b", "", "Realm slug for character B (defaults to realm-a)")
	investigateAccountCmd.Flags().String("name-b", "", "Name of character B")
	investigateAccountCmd.Flags().Bool("definitions", false, "Fetch each matching achievement's definition (single-pair mode)")
	investigateAccountCmd.Flags().Int("definitions-limit", 25, "Cap definition lookups in single-pair mode")
	investigateAccountCmd.Flags().String("pairs-file", "", "CSV of labelled pairs for batch aggregate analysis")
	investigateAccountCmd.Flags().String("output", "", "Optional path to write the curated JSON report (batch mode)")
}
