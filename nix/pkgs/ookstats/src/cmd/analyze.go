package cmd

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
)

// analyzeCmd summarizes CM fetch coverage to power the status API.
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Summarize CM fetch coverage and write status JSON",
	Long:  `Reads previously recorded fetch results and outputs per-realm/dungeon coverage data for the status page.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath, _ := cmd.Flags().GetString("out")
		statusDir, _ := cmd.Flags().GetString("status-dir")
		regionsCSV, _ := cmd.Flags().GetString("regions")
		periodsCSV, _ := cmd.Flags().GetString("periods")
		rng, _ := cmd.Flags().GetString("range")
		concurrency, _ := cmd.Flags().GetInt("concurrency") // retained for compatibility
		_ = concurrency

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("db connect: %w", err)
		}
		defer db.Close()

		_, dungeons := blizzard.GetHardcodedPeriodAndDungeons()

		realms := blizzard.GetAllRealms()
		if strings.TrimSpace(regionsCSV) != "" {
			allowed := map[string]bool{}
			for _, r := range strings.Split(regionsCSV, ",") {
				r = strings.ToLower(strings.TrimSpace(r))
				if r != "" {
					allowed[r] = true
				}
			}
			for slug, info := range realms {
				if !allowed[strings.ToLower(info.Region)] {
					delete(realms, slug)
				}
			}
		}

		var periodsSpec string
		if strings.TrimSpace(periodsCSV) != "" {
			periodsSpec = periodsCSV
		} else if strings.TrimSpace(rng) != "" {
			parts := strings.Split(strings.TrimSpace(rng), "-")
			if len(parts) != 2 {
				return errors.New("invalid --range format (expected start-end)")
			}
			a, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			b, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil || a <= 0 || b <= 0 || b < a {
				return errors.New("invalid --range values")
			}
			var list []string
			for i := a; i <= b; i++ {
				list = append(list, fmt.Sprintf("%d", i))
			}
			periodsSpec = strings.Join(list, ",")
		}

		return runAnalyze(db, realms, dungeons, periodsSpec, outPath, statusDir)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().String("out", "", "Optional path to write status JSON (default: {status-dir}/latest-runs.json)")
	analyzeCmd.Flags().String("status-dir", "web/public/api/status", "Base directory for status JSON files")
	analyzeCmd.Flags().String("regions", "", "Comma-separated regions to include (us,eu,kr,tw)")
	analyzeCmd.Flags().String("periods", "", "Comma-separated period IDs to include")
	analyzeCmd.Flags().String("range", "", "Period range to include (e.g., 1026-1030)")
	analyzeCmd.Flags().Int("concurrency", 0, "Deprecated (no longer used)")
}

type statusDungeonEntry struct {
	DungeonID    int    `json:"dungeon_id"`
	DungeonSlug  string `json:"dungeon_slug"`
	DungeonName  string `json:"dungeon_name"`
	Status       string `json:"status"`
	Periods      []int  `json:"periods"`
	Missing      []int  `json:"missing_periods"`
	ErrorPeriods []int  `json:"error_periods"`
}

type statusRealmEntry struct {
	Region         string               `json:"region"`
	RealmSlug      string               `json:"realm_slug"`
	RealmName      string               `json:"realm_name"`
	Health         string               `json:"health"`
	TotalPeriods   int                  `json:"total_periods"`
	MissingPeriods int                  `json:"missing_periods"`
	ErrorPeriods   int                  `json:"error_periods"`
	Dungeons       []statusDungeonEntry `json:"dungeons"`
}

type statusPayload struct {
	GeneratedAt string             `json:"generated_at"`
	Realms      []statusRealmEntry `json:"realms"`
}

func runAnalyze(db *sql.DB, realms map[string]blizzard.RealmInfo, dungeons []blizzard.DungeonInfo, periodsSpec string, outPath, statusDir string) error {
	log.Info("building status coverage", "realms", len(realms), "dungeons", len(dungeons))

	allowedPeriods := make(map[int]bool)
	if strings.TrimSpace(periodsSpec) != "" {
		parsed, err := blizzard.ParsePeriods(periodsSpec)
		if err != nil {
			return fmt.Errorf("parse periods: %w", err)
		}
		for _, p := range parsed {
			if pid, err := strconv.Atoi(p); err == nil {
				allowedPeriods[pid] = true
			}
		}
	}
	filterPeriods := len(allowedPeriods) > 0

	realmFilter := make(map[string]blizzard.RealmInfo)
	for _, info := range realms {
		realmFilter[realmMapKey(info.Region, info.Slug)] = info
	}

	dungeonLookup := make(map[int]blizzard.DungeonInfo)
	for _, d := range dungeons {
		dungeonLookup[d.ID] = d
	}

	type dungeonAgg struct {
		info    blizzard.DungeonInfo
		periods []int
		missing []int
		errors  []int
	}
	type realmAgg struct {
		info     blizzard.RealmInfo
		dungeons map[int]*dungeonAgg
	}

	agg := make(map[string]*realmAgg)

	rows, err := db.Query(`SELECT region, realm_slug, dungeon_id, period_id, status FROM fetch_status`)
	if err != nil {
		return fmt.Errorf("query fetch_status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var region, realmSlug string
		var dungeonID, periodID int
		var status string
		if err := rows.Scan(&region, &realmSlug, &dungeonID, &periodID, &status); err != nil {
			return fmt.Errorf("scan fetch_status: %w", err)
		}
		if filterPeriods && !allowedPeriods[periodID] {
			continue
		}
		key := realmMapKey(region, realmSlug)
		info, ok := realmFilter[key]
		if !ok {
			continue // skip realms outside requested filters
		}
		ra := agg[key]
		if ra == nil {
			ra = &realmAgg{info: info, dungeons: make(map[int]*dungeonAgg)}
			agg[key] = ra
		}
		dagg := ra.dungeons[dungeonID]
		if dagg == nil {
			dInfo, ok := dungeonLookup[dungeonID]
			if !ok {
				dInfo = blizzard.DungeonInfo{
					ID:   dungeonID,
					Slug: fmt.Sprintf("dungeon-%d", dungeonID),
					Name: fmt.Sprintf("Dungeon %d", dungeonID),
				}
			}
			dagg = &dungeonAgg{info: dInfo}
			ra.dungeons[dungeonID] = dagg
		}
		switch strings.ToLower(status) {
		case "ok":
			dagg.periods = append(dagg.periods, periodID)
		case "missing":
			dagg.missing = append(dagg.missing, periodID)
		case "error":
			dagg.errors = append(dagg.errors, periodID)
		default:
			// treat unknown statuses conservatively as errors
			dagg.errors = append(dagg.errors, periodID)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate fetch_status: %w", err)
	}

	realmKeys := make([]string, 0, len(agg))
	for key := range agg {
		realmKeys = append(realmKeys, key)
	}
	sort.Slice(realmKeys, func(i, j int) bool {
		ri := agg[realmKeys[i]].info
		rj := agg[realmKeys[j]].info
		if ri.Region == rj.Region {
			return ri.Slug < rj.Slug
		}
		return ri.Region < rj.Region
	})

	realmsOut := make([]statusRealmEntry, 0, len(realmKeys))
	for _, key := range realmKeys {
		ra := agg[key]
		realmEntry := statusRealmEntry{
			Region:    ra.info.Region,
			RealmSlug: ra.info.Slug,
			RealmName: ra.info.Name,
		}
		var realmMissing, realmErrors int

		dungeonIDs := make([]int, 0, len(ra.dungeons))
		for id := range ra.dungeons {
			dungeonIDs = append(dungeonIDs, id)
		}
		sort.Ints(dungeonIDs)

		for _, id := range dungeonIDs {
			dagg := ra.dungeons[id]
			sort.Ints(dagg.periods)
			sort.Ints(dagg.missing)
			sort.Ints(dagg.errors)

			status := coverageStatus(len(dagg.periods), len(dagg.missing), len(dagg.errors))
			entry := statusDungeonEntry{
				DungeonID:    dagg.info.ID,
				DungeonSlug:  dagg.info.Slug,
				DungeonName:  dagg.info.Name,
				Status:       status,
				Periods:      append([]int(nil), dagg.periods...),
				Missing:      append([]int(nil), dagg.missing...),
				ErrorPeriods: append([]int(nil), dagg.errors...),
			}
			realmEntry.Dungeons = append(realmEntry.Dungeons, entry)
			realmEntry.TotalPeriods += len(dagg.periods)
			realmMissing += len(dagg.missing)
			realmErrors += len(dagg.errors)
		}

		realmEntry.MissingPeriods = realmMissing
		realmEntry.ErrorPeriods = realmErrors
		realmEntry.Health = coverageStatus(realmEntry.TotalPeriods, realmMissing, realmErrors)
		realmsOut = append(realmsOut, realmEntry)
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)
	payload := statusPayload{
		GeneratedAt: generatedAt,
		Realms:      realmsOut,
	}

	if strings.TrimSpace(statusDir) == "" && strings.TrimSpace(outPath) == "" {
		return errors.New("status output path not specified")
	}

	if strings.TrimSpace(outPath) == "" {
		if err := os.MkdirAll(statusDir, 0o755); err != nil {
			return fmt.Errorf("mkdir status dir: %w", err)
		}
		outPath = filepath.Join(statusDir, "latest-runs.json")
	} else if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}

	if err := writeStatusJSON(outPath, payload); err != nil {
		return err
	}

	if strings.TrimSpace(statusDir) != "" {
		for _, realm := range realmsOut {
			dir := filepath.Join(statusDir, realm.Region)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("mkdir realm dir: %w", err)
			}
			realmPath := filepath.Join(dir, fmt.Sprintf("%s.json", realm.RealmSlug))
			realmPayload := struct {
				GeneratedAt string           `json:"generated_at"`
				Realm       statusRealmEntry `json:"realm"`
			}{
				GeneratedAt: generatedAt,
				Realm:       realm,
			}
			if err := writeStatusJSON(realmPath, realmPayload); err != nil {
				return err
			}
		}
		log.Info("wrote per-realm status files", "dir", statusDir, "count", len(realmsOut))
	}

	log.Info("status coverage generated", "realms", len(realmsOut))
	return nil
}

func writeStatusJSON(path string, payload any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func realmMapKey(region, slug string) string {
	return fmt.Sprintf("%s|%s", strings.ToLower(region), strings.ToLower(slug))
}

func coverageStatus(total, missing, errors int) string {
	if total == 0 {
		return "no_data"
	}
	deficit := missing + errors
	if deficit == 0 {
		return "ok"
	}
	if deficit >= total {
		return "no_data"
	}
	return "some_missing"
}
