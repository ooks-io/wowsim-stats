package cmd

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"

    "github.com/spf13/cobra"
    "ookstats/internal/blizzard"
)

// analyzeCmd performs a quick multi-period sweep to summarize latest runs per realm
var analyzeCmd = &cobra.Command{
    Use:   "analyze",
    Short: "Analyze CM endpoints and output latest runs per realm",
    Long:  `Fetches leaderboards across a set of periods and summarizes the latest recorded run timestamp per realm. Optionally writes a JSON file for the website.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        outPath, _ := cmd.Flags().GetString("out")
        statusDir, _ := cmd.Flags().GetString("status-dir")
        regionsCSV, _ := cmd.Flags().GetString("regions")
        periodsCSV, _ := cmd.Flags().GetString("periods")
        rng, _ := cmd.Flags().GetString("range")
        concurrency, _ := cmd.Flags().GetInt("concurrency")

        client, err := blizzard.NewClient()
        if err != nil {
            return fmt.Errorf("blizzard client: %w", err)
        }
        if concurrency > 0 {
            client.SetConcurrency(concurrency)
        }

        // Dungeons from constants
        _, dungeons := blizzard.GetHardcodedPeriodAndDungeons()

        // Realms from constants
        realms := blizzard.GetAllRealms()
        if strings.TrimSpace(regionsCSV) != "" {
            allowed := map[string]bool{}
            for _, r := range strings.Split(regionsCSV, ",") {
                r = strings.TrimSpace(strings.ToLower(r))
                if r != "" { allowed[r] = true }
            }
            for slug, info := range realms {
                if !allowed[strings.ToLower(info.Region)] {
                    delete(realms, slug)
                }
            }
        }

        // Periods
        var periods []string
        if strings.TrimSpace(periodsCSV) != "" {
            for _, p := range strings.Split(periodsCSV, ",") {
                if v := strings.TrimSpace(p); v != "" { periods = append(periods, v) }
            }
        } else if strings.TrimSpace(rng) != "" {
            parts := strings.Split(strings.TrimSpace(rng), "-")
            if len(parts) != 2 { return errors.New("invalid --range format (expected start-end)") }
            a, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
            b, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
            if err1 != nil || err2 != nil || a <= 0 || b <= 0 || b < a { return errors.New("invalid --range values") }
            for i := a; i <= b; i++ { periods = append(periods, fmt.Sprintf("%d", i)) }
        } else {
            periods = blizzard.GetGlobalPeriods()
        }

        fmt.Printf("Analyze: %d realms, %d dungeons, periods=%v, concurrency=%d\n", len(realms), len(dungeons), periods, concurrency)

        type latest struct{
            Region        string `json:"region"`
            RealmSlug     string `json:"realm_slug"`
            RealmName     string `json:"realm_name"`
            RealmID       int    `json:"realm_id"`
            MostRecent    int64  `json:"most_recent"`
            MostRecentISO string `json:"most_recent_iso"`
            PeriodID      string `json:"period_id"`
            DungeonSlug   string `json:"dungeon_slug"`
            DungeonName   string `json:"dungeon_name"`
            RunCount      int    `json:"run_count"`
            HasRuns       bool   `json:"has_runs"`
            LatestRun     struct{
                CompletedTimestamp int64  `json:"completed_timestamp"`
                Duration           int    `json:"duration_ms"`
                KeystoneLevel      int    `json:"keystone_level"`
                Members            []struct{
                    Name      string `json:"name"`
                    SpecID    int    `json:"spec_id"`
                    RealmSlug string `json:"realm_slug"`
                    Region    string `json:"region"`
                } `json:"members"`
            } `json:"latest_run"`
        }

        latestByRealm := map[string]latest{} // key: region|realm_slug
        // Track latest run per realm+dungeon for per-realm status files
        perRealmDungeonLatest := map[string]latest{} // key: region|realm_slug|dungeon_slug
        // coverage[realmKey][dungeonSlug][period] = {latest ts, run_count}
        type agg struct{ ts int64; runs int }
        coverage := map[string]map[string]map[string]agg{}
        total := 0
        success := 0
        failed := 0
        start := time.Now()

        // Iterate periods and fetch concurrently using the client helper
        for _, period := range periods {
            fmt.Printf("\n--- Period %s ---\n", period)
            expected := len(realms) * len(dungeons)
            fmt.Printf("Expecting %d requests (%d realms × %d dungeons)\n", expected, len(realms), len(dungeons))
            processed := 0

            ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
            results := client.FetchAllRealmsConcurrent(ctx, realms, dungeons, period)
            for res := range results {
                processed++
                total++
                if res.Error != nil {
                    failed++
                    // Print compact progress line every 50 items, or when a non-404 error occurs
                    if !strings.Contains(strings.ToLower(res.Error.Error()), "404") || processed%50 == 0 {
                        fmt.Printf("  [%4d/%4d] ERR %-3s %-18s %-24s: %v\n",
                            processed, expected, strings.ToUpper(res.RealmInfo.Region), res.RealmInfo.Slug, res.Dungeon.Slug, res.Error)
                    }
                    continue
                }
                success++
                // compute latest run timestamp, run count, and capture latest run details
                lb := res.Leaderboard
                maxTs := int64(0)
                var latestRun blizzard.ChallengeRun
                if lb != nil {
                    for _, run := range lb.LeadingGroups {
                        if run.CompletedTimestamp > maxTs {
                            maxTs = run.CompletedTimestamp
                            latestRun = run
                        }
                    }
                }
                key := res.RealmInfo.Region + "|" + res.RealmInfo.Slug
                cur := latestByRealm[key]
                if maxTs > cur.MostRecent {
                    entry := latest{
                        Region: res.RealmInfo.Region,
                        RealmSlug: res.RealmInfo.Slug,
                        RealmName: res.RealmInfo.Name,
                        RealmID: res.RealmInfo.ID,
                        MostRecent: maxTs,
                        MostRecentISO: time.UnixMilli(maxTs).UTC().Format("2006-01-02 15:04:05 UTC"),
                        PeriodID: period,
                        DungeonSlug: res.Dungeon.Slug,
                        DungeonName: res.Dungeon.Name,
                        RunCount: len(lb.LeadingGroups),
                        HasRuns: len(lb.LeadingGroups) > 0,
                    }
                    // enrich latest run members/specs for status page
                    entry.LatestRun.CompletedTimestamp = latestRun.CompletedTimestamp
                    entry.LatestRun.Duration = latestRun.Duration
                    entry.LatestRun.KeystoneLevel = latestRun.KeystoneLevel
                    entry.LatestRun.Members = make([]struct{
                        Name string "json:\"name\""
                        SpecID int "json:\"spec_id\""
                        RealmSlug string "json:\"realm_slug\""
                        Region string "json:\"region\""
                    }, 0, len(latestRun.Members))
                    for _, m := range latestRun.Members {
                        nm := ""; if v, ok := m.GetPlayerName(); ok { nm = v }
                        sid := 0; if v, ok := m.GetSpecID(); ok { sid = v }
                        rslug := ""; if v, ok := m.GetRealmSlug(); ok { rslug = v }
                        entry.LatestRun.Members = append(entry.LatestRun.Members, struct{
                            Name string "json:\"name\""
                            SpecID int "json:\"spec_id\""
                            RealmSlug string "json:\"realm_slug\""
                            Region string "json:\"region\""
                        }{ Name: nm, SpecID: sid, RealmSlug: rslug, Region: res.RealmInfo.Region })
                    }
                    latestByRealm[key] = entry
                }
                // Track latest per realm+dungeon
                rdKey := res.RealmInfo.Region + "|" + res.RealmInfo.Slug + "|" + res.Dungeon.Slug
                curRD := perRealmDungeonLatest[rdKey]
                if maxTs > curRD.MostRecent {
                    entry := latest{
                        Region: res.RealmInfo.Region,
                        RealmSlug: res.RealmInfo.Slug,
                        RealmName: res.RealmInfo.Name,
                        RealmID: res.RealmInfo.ID,
                        MostRecent: maxTs,
                        MostRecentISO: time.UnixMilli(maxTs).UTC().Format("2006-01-02 15:04:05 UTC"),
                        PeriodID: period,
                        DungeonSlug: res.Dungeon.Slug,
                        DungeonName: res.Dungeon.Name,
                        RunCount: len(lb.LeadingGroups),
                        HasRuns: len(lb.LeadingGroups) > 0,
                    }
                    entry.LatestRun.CompletedTimestamp = latestRun.CompletedTimestamp
                    entry.LatestRun.Duration = latestRun.Duration
                    entry.LatestRun.KeystoneLevel = latestRun.KeystoneLevel
                    entry.LatestRun.Members = make([]struct{
                        Name string "json:\"name\""
                        SpecID int "json:\"spec_id\""
                        RealmSlug string "json:\"realm_slug\""
                        Region string "json:\"region\""
                    }, 0, len(latestRun.Members))
                    for _, m := range latestRun.Members {
                        nm := ""; if v, ok := m.GetPlayerName(); ok { nm = v }
                        sid := 0; if v, ok := m.GetSpecID(); ok { sid = v }
                        rslug := ""; if v, ok := m.GetRealmSlug(); ok { rslug = v }
                        entry.LatestRun.Members = append(entry.LatestRun.Members, struct{
                            Name string "json:\"name\""
                            SpecID int "json:\"spec_id\""
                            RealmSlug string "json:\"realm_slug\""
                            Region string "json:\"region\""
                        }{ Name: nm, SpecID: sid, RealmSlug: rslug, Region: res.RealmInfo.Region })
                    }
                    perRealmDungeonLatest[rdKey] = entry
                }
                if processed%50 == 0 {
                    elapsed := time.Since(start)
                    fmt.Printf("  [%4d/%4d] OK  %-3s %-18s %-24s  ts=%d  (elapsed %s)\n",
                        processed, expected, strings.ToUpper(res.RealmInfo.Region), res.RealmInfo.Slug, res.Dungeon.Slug, maxTs, elapsed.Truncate(time.Second))
                }

                // Record coverage
                key = res.RealmInfo.Region + "|" + res.RealmInfo.Slug
                if _, ok := coverage[key]; !ok { coverage[key] = map[string]map[string]agg{} }
                if _, ok := coverage[key][res.Dungeon.Slug]; !ok { coverage[key][res.Dungeon.Slug] = map[string]agg{} }
                prev := coverage[key][res.Dungeon.Slug][period]
                if maxTs > prev.ts {
                    coverage[key][res.Dungeon.Slug][period] = agg{ ts: maxTs, runs: len(lb.LeadingGroups) }
                } else if prev.ts == 0 {
                    coverage[key][res.Dungeon.Slug][period] = agg{ ts: maxTs, runs: len(lb.LeadingGroups) }
                }
            }
            cancel()
        }

        // Prepare sorted slice
        items := make([]latest, 0, len(latestByRealm))
        for _, v := range latestByRealm { items = append(items, v) }
        sort.Slice(items, func(i,j int) bool { return items[i].MostRecent > items[j].MostRecent })

        // Print summary
        elapsed := time.Since(start)
        fmt.Printf("\nAnalyze complete in %v: total=%d success=%d failed=%d realms_with_data=%d\n", elapsed, total, success, failed, len(items))
        fmt.Println("\nLatest recorded run per realm:")
        for _, e := range items {
            fmt.Printf("  %-20s [%s] (%s) -> %s  | %s  period=%s  runs=%d\n", e.RealmName, strings.ToUpper(e.Region), e.RealmSlug, e.MostRecentISO, e.DungeonName, e.PeriodID, e.RunCount)
        }

        // Build realm_status structure for JSON
        type periodCoverage struct{
            PeriodID   string `json:"period_id"`
            HasRuns    bool   `json:"has_runs"`
            LatestTs   int64  `json:"latest_ts"`
            LatestISO  string `json:"latest_iso"`
            RunCount   int    `json:"run_count"`
        }
        type dungeonStatus struct{
            DungeonSlug string `json:"dungeon_slug"`
            DungeonName string `json:"dungeon_name"`
            LatestTs    int64  `json:"latest_ts"`
            LatestISO   string `json:"latest_iso"`
            LatestPeriod string `json:"latest_period"`
            Periods      []periodCoverage `json:"periods"`
            MissingPeriods []string `json:"missing_periods"`
        }
        type realmStatus struct{
            Region    string `json:"region"`
            RealmSlug string `json:"realm_slug"`
            RealmName string `json:"realm_name"`
            RealmID   int    `json:"realm_id"`
            Dungeons  []dungeonStatus `json:"dungeons"`
        }

        // Map for realm info lookup
        realmInfoByKey := map[string]blizzard.RealmInfo{}
        for _, ri := range realms {
            realmInfoByKey[ri.Region+"|"+ri.Slug] = ri
        }
        // Build realmStatus slice
        realmStatuses := make([]realmStatus, 0, len(coverage))
        for key, byDungeon := range coverage {
            ri := realmInfoByKey[key]
            rs := realmStatus{ Region: ri.Region, RealmSlug: ri.Slug, RealmName: ri.Name, RealmID: ri.ID }
            // For stable order, sort dungeon slugs by name
            type dn struct{ slug, name string }
            dns := make([]dn, 0, len(byDungeon))
            // get dungeon names from constants dungeons list
            dName := func(slug string) string {
                for _, d := range dungeons { if d.Slug == slug { return d.Name } }
                return slug
            }
            for slug := range byDungeon { dns = append(dns, dn{slug: slug, name: dName(slug)}) }
            sort.Slice(dns, func(i,j int) bool { return dns[i].name < dns[j].name })

            for _, d := range dns {
                perMap := byDungeon[d.slug]
                ds := dungeonStatus{ DungeonSlug: d.slug, DungeonName: d.name }
                // collect coverages per tested period order
                latestTs := int64(0)
                latestPer := ""
                periodsCover := make([]periodCoverage, 0, len(periods))
                missing := []string{}
                for _, p := range periods {
                    if ag, ok := perMap[p]; ok && ag.ts > 0 {
                        if ag.ts > latestTs { latestTs = ag.ts; latestPer = p }
                        periodsCover = append(periodsCover, periodCoverage{ PeriodID: p, HasRuns: ag.runs > 0, LatestTs: ag.ts, LatestISO: time.UnixMilli(ag.ts).UTC().Format("2006-01-02 15:04:05 UTC"), RunCount: ag.runs })
                    } else {
                        periodsCover = append(periodsCover, periodCoverage{ PeriodID: p, HasRuns: false, LatestTs: 0, LatestISO: "", RunCount: 0 })
                        missing = append(missing, p)
                    }
                }
                ds.LatestTs = latestTs
                if latestTs > 0 { ds.LatestISO = time.UnixMilli(latestTs).UTC().Format("2006-01-02 15:04:05 UTC") }
                ds.LatestPeriod = latestPer
                ds.Periods = periodsCover
                ds.MissingPeriods = missing
                rs.Dungeons = append(rs.Dungeons, ds)
            }
            realmStatuses = append(realmStatuses, rs)
        }
        sort.Slice(realmStatuses, func(i,j int) bool { if realmStatuses[i].Region == realmStatuses[j].Region { return realmStatuses[i].RealmName < realmStatuses[j].RealmName }; return realmStatuses[i].Region < realmStatuses[j].Region })

        // If a statusDir is provided (default), prefer writing latest-runs under it when --out not set
        if strings.TrimSpace(outPath) == "" && strings.TrimSpace(statusDir) != "" {
            outPath = filepath.Join(statusDir, "latest-runs.json")
        }

        // Write single combined JSON if requested or derived
        if strings.TrimSpace(outPath) != "" {
            payload := map[string]any{
                "generated_at": time.Now().UnixMilli(),
                "periods": periods,
                "summary": map[string]any{
                    "endpoints_tested": total,
                    "success": success,
                    "failed": failed,
                },
                "latest_runs": items,
                "realm_status": realmStatuses,
            }
            if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
                return fmt.Errorf("mkdir out: %w", err)
            }
            f, err := os.Create(outPath)
            if err != nil { return fmt.Errorf("create out: %w", err) }
            enc := json.NewEncoder(f)
            enc.SetIndent("", "  ")
            if err := enc.Encode(payload); err != nil { f.Close(); return fmt.Errorf("encode json: %w", err) }
            if err := f.Close(); err != nil { return err }
            fmt.Printf("\nWrote analysis JSON to %s\n", outPath)
        }

        // Write per-realm status files if requested
        if strings.TrimSpace(statusDir) != "" {
            type latestRunOut struct{
                CompletedTimestamp int64  `json:"completed_timestamp"`
                Duration           int    `json:"duration_ms"`
                KeystoneLevel      int    `json:"keystone_level"`
                Members            []struct{
                    Name      string `json:"name"`
                    SpecID    int    `json:"spec_id"`
                    RealmSlug string `json:"realm_slug"`
                    Region    string `json:"region"`
                } `json:"members"`
            }
            type dungeonOut struct{
                DungeonSlug string `json:"dungeon_slug"`
                DungeonName string `json:"dungeon_name"`
                LatestTs    int64  `json:"latest_ts"`
                LatestISO   string `json:"latest_iso"`
                LatestPeriod string `json:"latest_period"`
                Periods      []periodCoverage `json:"periods"`
                MissingPeriods []string `json:"missing_periods"`
                LatestRun    latestRunOut `json:"latest_run"`
            }
            type realmOut struct{
                GeneratedAt int64  `json:"generated_at"`
                Region    string `json:"region"`
                RealmSlug string `json:"realm_slug"`
                RealmName string `json:"realm_name"`
                RealmID   int    `json:"realm_id"`
                Periods   []string `json:"periods"`
                Dungeons  []dungeonOut `json:"dungeons"`
            }

            // For each realm in realmStatuses, construct file
            for _, rs := range realmStatuses {
                // compile dungeons with latest runs
                dout := make([]dungeonOut, 0, len(rs.Dungeons))
                for _, d := range rs.Dungeons {
                    rdKey := rs.Region + "|" + rs.RealmSlug + "|" + d.DungeonSlug
                    lr := perRealmDungeonLatest[rdKey]
                    od := dungeonOut{
                        DungeonSlug: d.DungeonSlug,
                        DungeonName: d.DungeonName,
                        LatestTs:    d.LatestTs,
                        LatestISO:   d.LatestISO,
                        LatestPeriod: d.LatestPeriod,
                        Periods:     d.Periods,
                        MissingPeriods: d.MissingPeriods,
                    }
                    if lr.MostRecent > 0 {
                        od.LatestRun.CompletedTimestamp = lr.LatestRun.CompletedTimestamp
                        od.LatestRun.Duration = lr.LatestRun.Duration
                        od.LatestRun.KeystoneLevel = lr.LatestRun.KeystoneLevel
                        od.LatestRun.Members = lr.LatestRun.Members
                    }
                    dout = append(dout, od)
                }
                payload := realmOut{
                    GeneratedAt: time.Now().UnixMilli(),
                    Region: rs.Region,
                    RealmSlug: rs.RealmSlug,
                    RealmName: rs.RealmName,
                    RealmID: rs.RealmID,
                    Periods: periods,
                    Dungeons: dout,
                }
                // path: statusDir/{region}/{realmSlug}.json
                dir := filepath.Join(statusDir, rs.Region)
                if err := os.MkdirAll(dir, 0o755); err != nil {
                    return fmt.Errorf("mkdir status %s: %w", dir, err)
                }
                fp := filepath.Join(dir, rs.RealmSlug+".json")
                f, err := os.Create(fp)
                if err != nil { return fmt.Errorf("create status file: %w", err) }
                enc := json.NewEncoder(f)
                enc.SetIndent("", "  ")
                if err := enc.Encode(payload); err != nil { f.Close(); return fmt.Errorf("encode status: %w", err) }
                if err := f.Close(); err != nil { return err }
            }
            fmt.Printf("Wrote per-realm status files to %s\n", statusDir)
        }

        return nil
    },
}

func init() {
    rootCmd.AddCommand(analyzeCmd)
    analyzeCmd.Flags().String("out", "", "Optional path to write latest-runs JSON (default: {status-dir}/latest-runs.json)")
    analyzeCmd.Flags().String("status-dir", "web/public/api/status", "Base dir to write status JSON files (latest-runs and per-realm)")
    analyzeCmd.Flags().String("regions", "", "Comma-separated regions to include (us,eu,kr,tw)")
    analyzeCmd.Flags().String("periods", "", "Comma-separated period IDs to test (default: global periods)")
    analyzeCmd.Flags().String("range", "", "Period range to test (e.g., 1026-1030)")
    analyzeCmd.Flags().Int("concurrency", 20, "Max concurrent API requests")
}
