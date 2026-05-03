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

// GearJSON is the top-level shape of gear.json.
type GearJSON struct {
	GeneratedAt int64                          `json:"generated_at"`
	// Outer key: season key ("season_1" | "season_2"). Inner key: spec_id (string).
	Scopes map[string]map[string]GearSpecBucket `json:"scopes"`
}

// GearSpecBucket — gear popularity for a single (season, spec) pool.
// `total_players` is the count of qualifying players (after spec-stat
// validation) whose gear was tallied. `slots` is keyed by canonical slot name
// (HEAD, NECK, FINGER, TRINKET, etc.).
type GearSpecBucket struct {
	TotalPlayers int                          `json:"total_players"`
	Slots        map[string]GearSlotBucket    `json:"slots"`
}

// GearSlotBucket — items equipped in a slot, sorted by frequency.
type GearSlotBucket struct {
	// PlayersWithSlot is the number of qualifying players who had any item
	// equipped in this slot (denominator for share %). For paired slots
	// (FINGER, TRINKET) a player can contribute up to 2 to the slot tally so
	// PlayersWithSlot * 2 is the entry-count ceiling.
	PlayersWithSlot int             `json:"players_with_slot"`
	Items           []GearItemEntry `json:"items"`
}

// GearItemEntry — one item's tally within a (spec, slot) bucket.
// Aggregation key is `name`, not `item_id`: gear has multiple item ids for the
// same piece at different ilvls (LFR / normal / heroic / upgraded) and CM
// players treat them as the same gear. `item_id` is the most-equipped variant,
// used for the wowhead link and the icon.
type GearItemEntry struct {
	ItemID  int    `json:"item_id"`
	Name    string `json:"name"`
	Icon    string `json:"icon,omitempty"`
	Quality int    `json:"quality"`
	Count   int    `json:"count"`
}

// Pool size — how many top players per spec to consider.
const gearTopPlayers = 100

// Slots we tally. Cosmetic slots (SHIRT, TABARD) excluded. Paired slots
// (FINGER_1+FINGER_2, TRINKET_1+TRINKET_2) get merged in the output to a
// single "FINGER" / "TRINKET" bucket since the suffix has no meaning.
var gearSlots = []string{
	"HEAD", "NECK", "SHOULDER", "BACK", "CHEST", "WRIST",
	"HANDS", "WAIST", "LEGS", "FEET",
	"FINGER_1", "FINGER_2",
	"TRINKET_1", "TRINKET_2",
	"MAIN_HAND", "OFF_HAND",
}

// gearSlotOutputName collapses paired slots to their merged output name.
func gearSlotOutputName(rawSlot string) string {
	switch rawSlot {
	case "FINGER_1", "FINGER_2":
		return "FINGER"
	case "TRINKET_1", "TRINKET_2":
		return "TRINKET"
	}
	return rawSlot
}

// Stat IDs in items.stats JSON — derived from inspecting known items.
const (
	statKeyStr = "0"
	statKeyAgi = "1"
	statKeyInt = "3"
)

// itemStatProfile holds the primary stat we extracted from an item's stats JSON.
// `primary` is "str"/"agi"/"int"/"" (none — typically secondary-stat-only items).
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

	// Load every item's stats once into memory — there are ~12k items in the
	// DB and we'll touch every one of them across the spec loop, so a single
	// scan beats per-spec joins.
	itemProfiles, err := loadItemProfiles(db)
	if err != nil {
		return fmt.Errorf("load item profiles: %w", err)
	}

	for _, season := range []int{1, 2} {
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

// loadItemProfiles preloads a (item_id -> profile) map. We pull every item
// rather than per-spec since the same item may appear across many specs.
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

// deriveItemPrimaryStat parses an item's stats JSON and returns its primary
// stat, or "" if the item carries none (e.g. a tabard, blank cosmetic, or a
// piece with only secondary stats).
//
// Stats JSON shape: {"<ilvl_offset>": {"stats": {"<stat_id>": <value>, ...}, ...}, ...}
// We only need to look at one ilvl entry — primary stat IDs don't change per
// upgrade tier. Picking the highest ilvl entry is the safest (lowest entries
// can be sparse for some items).
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
	// Walk all entries and check for known primary stat keys. Multiple entries
	// will have the same primary stat — first hit wins.
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

// buildSpecGear walks the top-N players for a (season, spec) pair, validates
// each one's set against the spec's expected primary stat, and tallies surviving
// items per slot.
func buildSpecGear(db *sql.DB, season, specID int, expected wow.PrimaryStat, itemProfiles map[int]itemStatProfile) (GearSpecBucket, error) {
	playerIDs, err := topPlayersForSpec(db, season, specID, gearTopPlayers)
	if err != nil {
		return GearSpecBucket{}, fmt.Errorf("top players: %w", err)
	}
	if len(playerIDs) == 0 {
		return GearSpecBucket{}, nil
	}

	// Equipment for the candidate pool, indexed by player.
	playerEquipment, err := loadPlayerEquipment(db, playerIDs)
	if err != nil {
		return GearSpecBucket{}, fmt.Errorf("equipment: %w", err)
	}

	// (slot -> name -> aggregate). Aggregating by item NAME so that variants
	// of the same piece at different ilvls collapse into a single bar.
	type nameAgg struct {
		count   int
		quality int
		icon    string
		// variant id -> count, used to pick the most-equipped variant id as
		// the representative for the wowhead link.
		byID map[int]int
	}
	slotItemCounts := make(map[string]map[string]*nameAgg)
	// (slot -> count of qualifying players who had any item in that slot).
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

		// Track which output-slots this player contributed to (so we don't
		// double-count e.g. FINGER if both FINGER_1 and FINGER_2 are equipped).
		seenSlot := make(map[string]bool)

		for _, item := range eq {
			outSlot := gearSlotOutputName(item.SlotType)
			if !isTrackedSlot(item.SlotType) {
				continue
			}
			profile, hasProfile := itemProfiles[item.ItemID]
			// Drop items whose primary stat clearly doesn't match the spec.
			// Items with no primary stat (necks, some trinkets) pass through.
			if hasProfile && profile.primary != "" && profile.primary != expected {
				continue
			}
			// Skip unknown items (no profile) — name-based aggregation needs a
			// name, and a missing profile means we have no name either.
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
			// Pick the most-equipped variant id as the representative.
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
		// Sort desc by count (insertion sort — slot has at most ~50 distinct items).
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

// equipmentRow mirrors the player_equipment columns we care about.
type equipmentRow struct {
	PlayerID int64
	SlotType string
	ItemID   int
}

// topPlayersForSpec returns the player_ids of the top-N best-coverage players
// in the given (season, spec) pair, ordered by combined_best_time ascending.
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

// loadPlayerEquipment fetches every tracked-slot item for the given player ids
// in a single query. Result is keyed by player_id.
func loadPlayerEquipment(db *sql.DB, playerIDs []int64) (map[int64][]equipmentRow, error) {
	if len(playerIDs) == 0 {
		return map[int64][]equipmentRow{}, nil
	}
	// Build IN clause with placeholders.
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

// playerSetMatchesSpec checks if the player's gear majority-aligns with the
// spec's expected primary stat. Used to drop "logged out in their offspec set"
// entries before tallying their items.
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
	// If we found no primary-stat items at all, fall through and keep — not
	// enough signal to exclude.
	total := counts[wow.PrimaryStatStr] + counts[wow.PrimaryStatAgi] + counts[wow.PrimaryStatInt]
	if total == 0 {
		return true
	}
	// Find the modal stat.
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
