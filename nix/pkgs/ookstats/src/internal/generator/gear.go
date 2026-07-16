package generator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"ookstats/internal/wow"
	"ookstats/internal/writer"
)

type GearJSON struct {
	GeneratedAt int64 `json:"generated_at"`
	// outer key: season ("season_1" | "season_2"); inner key: spec_id as string
	Scopes map[string]map[string]GearSpecBucket `json:"scopes"`
}

type GearSpecBucket struct {
	TotalPlayers int                       `json:"total_players"`
	Slots        map[string]GearSlotBucket `json:"slots"`
}

type GearSlotBucket struct {
	// denominator for share %; paired slots can contribute up to 2 per player
	// so the entry-count ceiling is PlayersWithSlot * 2
	PlayersWithSlot int             `json:"players_with_slot"`
	Items           []GearItemEntry `json:"items"`
}

// aggregation key is `name`, not `item_id`: gear has multiple item ids for the
// same piece at different ilvls (LFR/normal/heroic/upgraded) and CM players
// treat them as the same gear; ItemID is the most-equipped variant for the
// wowhead link and icon
type GearItemEntry struct {
	ItemID  int    `json:"item_id"`
	Name    string `json:"name"`
	Icon    string `json:"icon,omitempty"`
	Quality int    `json:"quality"`
	Count   int    `json:"count"`
}

const gearTopPlayers = 100

// cosmetic slots (SHIRT, TABARD) excluded; FINGER_1/_2 and TRINKET_1/_2 get
// merged in the output to a single "FINGER"/"TRINKET" bucket
var gearSlots = []string{
	"HEAD", "NECK", "SHOULDER", "BACK", "CHEST", "WRIST",
	"HANDS", "WAIST", "LEGS", "FEET",
	"FINGER_1", "FINGER_2",
	"TRINKET_1", "TRINKET_2",
	"MAIN_HAND", "OFF_HAND",
}

func gearSlotOutputName(rawSlot string) string {
	switch rawSlot {
	case "FINGER_1", "FINGER_2":
		return "FINGER"
	case "TRINKET_1", "TRINKET_2":
		return "TRINKET"
	}
	return rawSlot
}

// stat IDs in items.stats JSON, derived from inspecting known items
const (
	statKeyStr = "0"
	statKeyAgi = "1"
	statKeyInt = "3"
)

// `primary` is "str"/"agi"/"int"/"" (empty for secondary-stat-only items)
type itemStatProfile struct {
	primary wow.PrimaryStat
	name    string
	icon    string
	quality int
}

// GenerateGear writes outDir/api/gear.json with per-spec popular gear data.
func GenerateGear(db *sql.DB, outDir string) error {
	out := GearJSON{
		GeneratedAt: time.Now().UnixMilli(),
		Scopes:      make(map[string]map[string]GearSpecBucket),
	}

	// load all item stats once: ~12k items touched across the spec loop, so
	// a single scan beats per-spec joins
	itemProfiles, err := loadItemProfiles(db)
	if err != nil {
		return fmt.Errorf("load item profiles: %w", err)
	}

	seasons, err := loadGearSeasons(db)
	if err != nil {
		return fmt.Errorf("load gear seasons: %w", err)
	}

	for _, season := range seasons {
		seasonKey := fmt.Sprintf("season_%d", season)
		bySpec := make(map[string]GearSpecBucket)

		for specID := range gatherSpecIDs() {
			expected, ok := wow.GetPrimaryStat(specID)
			if !ok {
				continue
			}
			bucket, err := buildSpecGear(db, season, specID, expected, itemProfiles)
			if err != nil {
				return fmt.Errorf("season %d spec %d: %w", season, specID, err)
			}
			if bucket.TotalPlayers == 0 {
				continue
			}
			bySpec[fmt.Sprintf("%d", specID)] = bucket
		}
		out.Scopes[seasonKey] = bySpec
	}

	outPath := filepath.Join(outDir, "api", "gear.json")
	if err := writer.WriteJSONFileCompact(outPath, out); err != nil {
		return fmt.Errorf("write gear.json: %w", err)
	}
	return nil
}

// gatherSpecIDs returns every spec id we have a primary-stat mapping for. Using
// a yield-style helper keeps the caller readable without exporting the map.
func loadGearSeasons(db *sql.DB) ([]int, error) {
	rows, err := db.Query(`SELECT DISTINCT season_number FROM seasons ORDER BY season_number ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func gatherSpecIDs() map[int]struct{} {
	out := make(map[int]struct{}, 33)
	for _, sid := range []int{
		250, 251, 252, // DK
		102, 103, 104, 105, // Druid
		253, 254, 255, // Hunter
		62, 63, 64, // Mage
		268, 269, 270, // Monk
		65, 66, 70, // Paladin
		256, 257, 258, // Priest
		259, 260, 261, // Rogue
		262, 263, 264, // Shaman
		265, 266, 267, // Warlock
		71, 72, 73, // Warrior
	} {
		out[sid] = struct{}{}
	}
	return out
}

// every item, not per-spec: same item often appears across many specs
func loadItemProfiles(db *sql.DB) (map[int]itemStatProfile, error) {
	rows, err := db.Query(`SELECT id, name, icon, quality, stats FROM items`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int]itemStatProfile)
	for rows.Next() {
		var id int
		var name, icon, statsRaw sql.NullString
		var quality sql.NullInt64
		if err := rows.Scan(&id, &name, &icon, &quality, &statsRaw); err != nil {
			return nil, err
		}
		profile := itemStatProfile{
			name:    name.String,
			icon:    icon.String,
			quality: int(quality.Int64),
			primary: deriveItemPrimaryStat(statsRaw.String),
		}
		out[id] = profile
	}
	return out, rows.Err()
}

// returns "" for items with no primary (tabards, cosmetics, secondary-only).
// JSON shape: {"<ilvl_offset>": {"stats": {"<stat_id>": <value>}}}.
// any one ilvl entry is fine - primary stat IDs don't change per upgrade tier
func deriveItemPrimaryStat(statsJSON string) wow.PrimaryStat {
	if statsJSON == "" {
		return ""
	}
	var byIlvl map[string]struct {
		Stats map[string]int64 `json:"stats"`
	}
	if err := json.Unmarshal([]byte(statsJSON), &byIlvl); err != nil {
		return ""
	}
	// first hit wins - all entries share the same primary
	for _, entry := range byIlvl {
		if _, ok := entry.Stats[statKeyStr]; ok {
			return wow.PrimaryStatStr
		}
		if _, ok := entry.Stats[statKeyAgi]; ok {
			return wow.PrimaryStatAgi
		}
		if _, ok := entry.Stats[statKeyInt]; ok {
			return wow.PrimaryStatInt
		}
	}
	return ""
}

// validates each top-N set against the spec's expected primary stat, then
// tallies surviving items per slot
func buildSpecGear(db *sql.DB, season, specID int, expected wow.PrimaryStat, itemProfiles map[int]itemStatProfile) (GearSpecBucket, error) {
	playerIDs, err := topPlayersForSpec(db, season, specID, gearTopPlayers)
	if err != nil {
		return GearSpecBucket{}, fmt.Errorf("top players: %w", err)
	}
	if len(playerIDs) == 0 {
		return GearSpecBucket{}, nil
	}

	playerEquipment, err := loadPlayerEquipment(db, playerIDs)
	if err != nil {
		return GearSpecBucket{}, fmt.Errorf("equipment: %w", err)
	}

	// aggregating by item name so ilvl variants of the same piece collapse
	// into one bar; byID tracks counts per variant so we can pick the
	// most-equipped one for the wowhead link
	type nameAgg struct {
		count   int
		quality int
		icon    string
		byID    map[int]int
	}
	slotItemCounts := make(map[string]map[string]*nameAgg)
	slotPlayerCounts := make(map[string]int)
	totalPlayers := 0

	for _, playerID := range playerIDs {
		eq := playerEquipment[playerID]
		if len(eq) == 0 {
			continue
		}
		if !playerSetMatchesSpec(eq, expected, itemProfiles) {
			continue
		}
		totalPlayers++

		// avoid double-counting FINGER when both FINGER_1 and FINGER_2 land
		// in the same output slot
		seenSlot := make(map[string]bool)

		for _, item := range eq {
			outSlot := gearSlotOutputName(item.SlotType)
			if !isTrackedSlot(item.SlotType) {
				continue
			}
			profile, hasProfile := itemProfiles[item.ItemID]
			// items with no primary (necks, some trinkets) pass through
			if hasProfile && profile.primary != "" && profile.primary != expected {
				continue
			}
			if !hasProfile || profile.name == "" {
				continue
			}
			byName, ok := slotItemCounts[outSlot]
			if !ok {
				byName = make(map[string]*nameAgg)
				slotItemCounts[outSlot] = byName
			}
			agg := byName[profile.name]
			if agg == nil {
				agg = &nameAgg{
					quality: profile.quality,
					icon:    profile.icon,
					byID:    make(map[int]int),
				}
				byName[profile.name] = agg
			}
			agg.count++
			agg.byID[item.ItemID]++
			if !seenSlot[outSlot] {
				slotPlayerCounts[outSlot]++
				seenSlot[outSlot] = true
			}
		}
	}

	bucket := GearSpecBucket{
		TotalPlayers: totalPlayers,
		Slots:        make(map[string]GearSlotBucket),
	}
	for slot, byName := range slotItemCounts {
		entries := make([]GearItemEntry, 0, len(byName))
		for name, agg := range byName {
			// most-equipped variant id is the representative
			var repID, bestCount int
			for id, c := range agg.byID {
				if c > bestCount {
					repID = id
					bestCount = c
				}
			}
			entries = append(entries, GearItemEntry{
				ItemID:  repID,
				Name:    name,
				Icon:    agg.icon,
				Quality: agg.quality,
				Count:   agg.count,
			})
		}
		// insertion sort desc - slots top out at ~50 distinct items
		for i := 1; i < len(entries); i++ {
			for j := i; j > 0 && entries[j].Count > entries[j-1].Count; j-- {
				entries[j], entries[j-1] = entries[j-1], entries[j]
			}
		}
		bucket.Slots[slot] = GearSlotBucket{
			PlayersWithSlot: slotPlayerCounts[slot],
			Items:           entries,
		}
	}
	return bucket, nil
}

func isTrackedSlot(rawSlot string) bool {
	for _, s := range gearSlots {
		if s == rawSlot {
			return true
		}
	}
	return false
}

type equipmentRow struct {
	PlayerID int64
	SlotType string
	ItemID   int
}

func topPlayersForSpec(db *sql.DB, season, specID, limit int) ([]int64, error) {
	rows, err := db.Query(`
		SELECT player_id
		FROM player_profiles
		WHERE season_id = ?
		  AND main_spec_id = ?
		  AND has_complete_coverage = 1
		  AND combined_best_time IS NOT NULL
		ORDER BY combined_best_time ASC
		LIMIT ?
	`, season, specID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func loadPlayerEquipment(db *sql.DB, playerIDs []int64) (map[int64][]equipmentRow, error) {
	if len(playerIDs) == 0 {
		return map[int64][]equipmentRow{}, nil
	}
	placeholders := ""
	args := make([]any, 0, len(playerIDs))
	for i, id := range playerIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		SELECT player_id, slot_type, item_id
		FROM player_equipment
		WHERE player_id IN (%s) AND item_id IS NOT NULL
	`, placeholders)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]equipmentRow, len(playerIDs))
	for rows.Next() {
		var r equipmentRow
		if err := rows.Scan(&r.PlayerID, &r.SlotType, &r.ItemID); err != nil {
			return nil, err
		}
		out[r.PlayerID] = append(out[r.PlayerID], r)
	}
	return out, rows.Err()
}

// drops players who logged out in their offspec set: their gear majority
// has to match the spec's expected primary
func playerSetMatchesSpec(eq []equipmentRow, expected wow.PrimaryStat, itemProfiles map[int]itemStatProfile) bool {
	counts := map[wow.PrimaryStat]int{
		wow.PrimaryStatStr: 0,
		wow.PrimaryStatAgi: 0,
		wow.PrimaryStatInt: 0,
	}
	for _, item := range eq {
		profile, ok := itemProfiles[item.ItemID]
		if !ok || profile.primary == "" {
			continue
		}
		counts[profile.primary]++
	}
	// no primary-stat items at all: not enough signal, keep them
	total := counts[wow.PrimaryStatStr] + counts[wow.PrimaryStatAgi] + counts[wow.PrimaryStatInt]
	if total == 0 {
		return true
	}
	var modal wow.PrimaryStat
	var modalCount int
	for stat, c := range counts {
		if c > modalCount {
			modal = stat
			modalCount = c
		}
	}
	return modal == expected
}
