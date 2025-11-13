package loader

import (
	"database/sql"
	"fmt"
	"strings"
)

// EquipmentData represents a piece of equipment
type EquipmentData struct {
	ID         int64
	SlotType   string
	ItemID     sql.NullInt64
	UpgradeID  sql.NullInt64
	Quality    string
	ItemName   string
	SnapshotTs int64
	ItemIcon   sql.NullString
	ItemType   sql.NullString
}

// EnchantmentData represents an enchantment or gem on equipment
type EnchantmentData struct {
	EquipmentID    int64
	EnchantmentID  sql.NullInt64
	SlotID         sql.NullInt64
	SlotType       sql.NullString
	DisplayString  sql.NullString
	SourceItemID   sql.NullInt64
	SourceItemName sql.NullString
	SpellID        sql.NullInt64
	GemIconSlug    sql.NullString
}

// LoadAllEquipment loads equipment and enchantments for a set of players
// Returns: map[playerID]map[timestamp][]EquipmentData, map[equipmentID][]EnchantmentData, error
func LoadAllEquipment(db *sql.DB, playerIDs []int64) (map[int64]map[int64][]EquipmentData, map[int64][]EnchantmentData, error) {
	if len(playerIDs) == 0 {
		return make(map[int64]map[int64][]EquipmentData), make(map[int64][]EnchantmentData), nil
	}

	// Build IN clause
	placeholders := make([]string, len(playerIDs))
	args := make([]any, len(playerIDs))
	for i, id := range playerIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	// Get latest timestamp per player first
	latestQuery := fmt.Sprintf(`
        SELECT player_id, MAX(snapshot_timestamp) 
        FROM player_equipment 
        WHERE player_id IN (%s) 
        GROUP BY player_id
    `, strings.Join(placeholders, ","))

	latestRows, err := db.Query(latestQuery, args...)
	if err != nil {
		return nil, nil, err
	}
	defer latestRows.Close()

	playerTimestamps := make(map[int64]int64)
	for latestRows.Next() {
		var playerID, timestamp int64
		if err := latestRows.Scan(&playerID, &timestamp); err != nil {
			return nil, nil, err
		}
		playerTimestamps[playerID] = timestamp
	}

	if len(playerTimestamps) == 0 {
		return make(map[int64]map[int64][]EquipmentData), make(map[int64][]EnchantmentData), nil
	}

	// Load equipment for latest timestamps
	equipmentMap := make(map[int64]map[int64][]EquipmentData)
	var allEquipmentIDs []int64

	for playerID, timestamp := range playerTimestamps {
		rows, err := db.Query(`
            SELECT e.id, e.slot_type, e.item_id, e.upgrade_id, e.quality, e.item_name, e.snapshot_timestamp,
                   i.icon AS item_icon_slug, i.type AS item_type
            FROM player_equipment e
            LEFT JOIN items i ON e.item_id = i.id
            WHERE e.player_id = ? AND e.snapshot_timestamp = ?
            ORDER BY e.slot_type
        `, playerID, timestamp)
		if err != nil {
			return nil, nil, err
		}

		if equipmentMap[playerID] == nil {
			equipmentMap[playerID] = make(map[int64][]EquipmentData)
		}

		for rows.Next() {
			var eq EquipmentData
			if err := rows.Scan(
				&eq.ID, &eq.SlotType, &eq.ItemID, &eq.UpgradeID, &eq.Quality, &eq.ItemName, &eq.SnapshotTs,
				&eq.ItemIcon, &eq.ItemType); err != nil {
				rows.Close()
				return nil, nil, fmt.Errorf("scan equipment: %w", err)
			}
			equipmentMap[playerID][timestamp] = append(equipmentMap[playerID][timestamp], eq)
			allEquipmentIDs = append(allEquipmentIDs, eq.ID)
		}
		rows.Close()
	}

	// Load enchantments in batches
	enchantmentsMap := make(map[int64][]EnchantmentData)
	if len(allEquipmentIDs) > 0 {
		const enchBatchSize = 10000
		for i := 0; i < len(allEquipmentIDs); i += enchBatchSize {
			end := i + enchBatchSize
			if end > len(allEquipmentIDs) {
				end = len(allEquipmentIDs)
			}

			batch := allEquipmentIDs[i:end]
			placeholders := make([]string, len(batch))
			args := make([]any, len(batch))
			for j, id := range batch {
				placeholders[j] = "?"
				args[j] = id
			}

			enchQuery := fmt.Sprintf(`
                SELECT pee.equipment_id, pee.enchantment_id, pee.slot_id, pee.slot_type, pee.display_string,
                       pee.source_item_id, pee.source_item_name, pee.spell_id, i.icon as gem_icon_slug
                FROM player_equipment_enchantments pee
                LEFT JOIN items i ON pee.source_item_id = i.id
                WHERE pee.equipment_id IN (%s)
                ORDER BY pee.equipment_id, pee.slot_id
            `, strings.Join(placeholders, ","))

			enchRows, err := db.Query(enchQuery, args...)
			if err != nil {
				return nil, nil, fmt.Errorf("enchantments batch %d: %w", i/enchBatchSize+1, err)
			}

			for enchRows.Next() {
				var ench EnchantmentData
				if err := enchRows.Scan(
					&ench.EquipmentID, &ench.EnchantmentID, &ench.SlotID, &ench.SlotType, &ench.DisplayString,
					&ench.SourceItemID, &ench.SourceItemName, &ench.SpellID, &ench.GemIconSlug); err != nil {
					enchRows.Close()
					return nil, nil, fmt.Errorf("scan enchantment: %w", err)
				}
				enchantmentsMap[ench.EquipmentID] = append(enchantmentsMap[ench.EquipmentID], ench)
			}
			enchRows.Close()
		}
	}

	return equipmentMap, enchantmentsMap, nil
}
