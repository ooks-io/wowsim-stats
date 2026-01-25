package database

import (
	"database/sql"
	"fmt"
	"ookstats/internal/blizzard"
	"sort"
	"strings"
)

// GetEligiblePlayersForProfileFetch returns players with complete coverage (9/9 dungeons)
// staleBefore: Unix timestamp in milliseconds. If > 0, returns players whose profile is NULL or older than this.
// If 0, returns only players who have never had a profile fetched.
func (ds *DatabaseService) GetEligiblePlayersForProfileFetch(staleBefore int64) ([]blizzard.PlayerInfo, error) {
	var rows *sql.Rows
	var err error

	if staleBefore > 0 {
		rows, err = ds.db.Query(`
			SELECT p.id, p.name, r.slug as realm_slug, r.region
			FROM players p
			JOIN player_profiles pp ON p.id = pp.player_id
			JOIN realms r ON p.realm_id = r.id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.has_complete_coverage = 1
			  AND COALESCE(p.is_valid, 1) = 1
			  AND (pd.last_updated IS NULL OR pd.last_updated < ?)
			ORDER BY pp.global_ranking
		`, staleBefore)
	} else {
		rows, err = ds.db.Query(`
			SELECT p.id, p.name, r.slug as realm_slug, r.region
			FROM players p
			JOIN player_profiles pp ON p.id = pp.player_id
			JOIN realms r ON p.realm_id = r.id
			LEFT JOIN player_details pd ON p.id = pd.player_id
			WHERE pp.has_complete_coverage = 1
			  AND COALESCE(p.is_valid, 1) = 1
			  AND pd.last_updated IS NULL
			ORDER BY pp.global_ranking
		`)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query eligible players: %w", err)
	}
	defer rows.Close()

	var players []blizzard.PlayerInfo
	for rows.Next() {
		var player blizzard.PlayerInfo
		err := rows.Scan(&player.ID, &player.Name, &player.RealmSlug, &player.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player row: %w", err)
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating player rows: %w", err)
	}

	return players, nil
}

// InsertPlayerProfileData inserts player profile data and returns counts
func (ds *DatabaseService) InsertPlayerProfileData(result blizzard.PlayerProfileResult, timestamp int64) (int, int, error) {
	if result.Error != nil {
		return 0, 0, result.Error
	}

	profilesUpdated := 0
	equipmentUpdated := 0

	tx, err := ds.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if result.Summary != nil {
		err := ds.insertPlayerDetailsTx(tx, result.PlayerID, result.Summary, result.Media, timestamp)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert player details: %w", err)
		}
		profilesUpdated++
	}

	if result.Equipment != nil {
		itemCount, err := ds.insertPlayerEquipmentTx(tx, result.PlayerID, result.Equipment, timestamp)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert player equipment: %w", err)
		}
		equipmentUpdated += itemCount
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return profilesUpdated, equipmentUpdated, nil
}

// insertPlayerDetailsTx inserts player summary data within a transaction
func (ds *DatabaseService) insertPlayerDetailsTx(tx *sql.Tx, playerID int, summary *blizzard.CharacterSummaryResponse, media *blizzard.CharacterMediaResponse, timestamp int64) error {
	var avatarURL *string
	if media != nil {
		for _, asset := range media.Assets {
			if asset.Key == "avatar" {
				avatarURL = &asset.Value
				break
			}
		}
	}

	var guildName *string
	if summary.Guild != nil {
		guildName = &summary.Guild.Name
	}

	_, err := tx.Exec(`
        INSERT INTO player_details (
            player_id, race_id, race_name, gender, class_id, class_name,
            active_spec_id, active_spec_name, guild_name, level,
            average_item_level, equipped_item_level, avatar_url, last_updated
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(player_id) DO UPDATE SET
            race_id = excluded.race_id,
            race_name = excluded.race_name,
            gender = excluded.gender,
            class_id = excluded.class_id,
            class_name = excluded.class_name,
            active_spec_id = excluded.active_spec_id,
            active_spec_name = excluded.active_spec_name,
            guild_name = excluded.guild_name,
            level = excluded.level,
            average_item_level = excluded.average_item_level,
            equipped_item_level = excluded.equipped_item_level,
            avatar_url = excluded.avatar_url,
            last_updated = excluded.last_updated
        WHERE
            player_details.race_id               IS NOT excluded.race_id OR
            player_details.race_name            IS NOT excluded.race_name OR
            player_details.gender               IS NOT excluded.gender OR
            player_details.class_id             IS NOT excluded.class_id OR
            player_details.class_name           IS NOT excluded.class_name OR
            player_details.active_spec_id       IS NOT excluded.active_spec_id OR
            player_details.active_spec_name     IS NOT excluded.active_spec_name OR
            player_details.guild_name           IS NOT excluded.guild_name OR
            player_details.level                IS NOT excluded.level OR
            player_details.average_item_level   IS NOT excluded.average_item_level OR
            player_details.equipped_item_level  IS NOT excluded.equipped_item_level OR
            player_details.avatar_url           IS NOT excluded.avatar_url
    `,
		playerID,
		summary.Race.ID,
		summary.Race.Name,
		summary.Gender.Type,
		summary.CharacterClass.ID,
		summary.CharacterClass.Name,
		summary.ActiveSpec.ID,
		summary.ActiveSpec.Name,
		guildName,
		summary.Level,
		summary.AverageItemLevel,
		summary.EquippedItemLevel,
		avatarURL,
		timestamp,
	)

	return err
}

// insertPlayerEquipmentTx inserts player equipment data within a transaction
func (ds *DatabaseService) insertPlayerEquipmentTx(tx *sql.Tx, playerID int, equipment *blizzard.CharacterEquipmentResponse, timestamp int64) (int, error) {
	if equipment == nil || len(equipment.EquippedItems) == 0 {
		return 0, nil
	}

	equipmentCount := 0

	for _, item := range equipment.EquippedItems {
		// check latest snapshot for this slot; skip writing if unchanged
		var prevID sql.NullInt64
		var prevItemID sql.NullInt64
		var prevUpgradeID sql.NullInt64
		var prevQuality, prevName sql.NullString
		if err := tx.QueryRow(
			`SELECT id, item_id, upgrade_id, quality, item_name
             FROM player_equipment
             WHERE player_id = ? AND slot_type = ?
             ORDER BY snapshot_timestamp DESC
             LIMIT 1`,
			playerID, item.Slot.Type,
		).Scan(&prevID, &prevItemID, &prevUpgradeID, &prevQuality, &prevName); err != nil && err != sql.ErrNoRows {
			return 0, fmt.Errorf("failed to query latest equipment: %w", err)
		}

		unchanged := false
		if prevID.Valid {
			prevUpg := 0
			if prevUpgradeID.Valid {
				prevUpg = int(prevUpgradeID.Int64)
			}
			curUpg := 0
			if item.UpgradeID != nil {
				curUpg = *item.UpgradeID
			}
			sameBasics := prevItemID.Valid && int(prevItemID.Int64) == item.Item.ID && prevQuality.Valid && prevQuality.String == item.Quality.Type && prevName.Valid && prevName.String == item.Name && prevUpg == curUpg

			if sameBasics {
				// compare enchantments as a canonical sorted signature
				dbRows, qerr := tx.Query(
					`SELECT
                        COALESCE(enchantment_id, -1) as eid,
                        COALESCE(source_item_id, -1) as sid,
                        COALESCE(slot_id, -1) as slotId,
                        COALESCE(slot_type, '') as slotType,
                        COALESCE(spell_id, -1) as spellId,
                        COALESCE(display_string, '') as disp
                     FROM player_equipment_enchantments
                     WHERE equipment_id = ?`, prevID.Int64)
				if qerr != nil {
					return 0, fmt.Errorf("failed to load existing enchantments: %w", qerr)
				}
				var dbSigs []string
				for dbRows.Next() {
					var eid, sid, slotId, spellId int
					var slotType, disp string
					if err := dbRows.Scan(&eid, &sid, &slotId, &slotType, &spellId, &disp); err != nil {
						dbRows.Close()
						return 0, fmt.Errorf("failed to scan enchantment: %w", err)
					}
					dbSigs = append(dbSigs, fmt.Sprintf("%d|%d|%d|%s|%d|%s", eid, sid, slotId, slotType, spellId, disp))
				}
				dbRows.Close()
				sort.Strings(dbSigs)

				var curSigs []string
				for _, ench := range item.Enchantments {
					eid := -1
					if ench.EnchantmentID != nil {
						eid = *ench.EnchantmentID
					}
					sid := -1
					if ench.SourceItem != nil {
						sid = ench.SourceItem.ID
					}
					slotId := -1
					var slotType string
					if ench.EnchantmentSlot != nil {
						slotId = ench.EnchantmentSlot.ID
						slotType = ench.EnchantmentSlot.Type
					}
					spellId := -1
					if ench.Spell != nil {
						spellId = ench.Spell.Spell.ID
					}
					disp := ench.DisplayString
					curSigs = append(curSigs, fmt.Sprintf("%d|%d|%d|%s|%d|%s", eid, sid, slotId, slotType, spellId, disp))
				}
				sort.Strings(curSigs)

				if strings.Join(dbSigs, ";") == strings.Join(curSigs, ";") {
					unchanged = true
				}
			}
		}

		if unchanged {
			continue
		}

		result, err := tx.Exec(`
            INSERT INTO player_equipment (
                player_id, slot_type, item_id, upgrade_id, quality, item_name, snapshot_timestamp
            ) VALUES (?, ?, ?, ?, ?, ?, ?)
        `,
			playerID,
			item.Slot.Type,
			item.Item.ID,
			item.UpgradeID,
			item.Quality.Type,
			item.Name,
			timestamp,
		)

		if err != nil {
			return 0, fmt.Errorf("failed to insert equipment item: %w", err)
		}

		equipmentID, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get equipment ID: %w", err)
		}

		equipmentCount++

		for _, enchant := range item.Enchantments {
			var sourceItemID *int
			var sourceItemName *string
			if enchant.SourceItem != nil {
				sourceItemID = &enchant.SourceItem.ID
				sourceItemName = &enchant.SourceItem.Name
			}

			var spellID *int
			if enchant.Spell != nil {
				spellID = &enchant.Spell.Spell.ID
			}

			var slotID *int
			var slotType *string
			if enchant.EnchantmentSlot != nil {
				slotID = &enchant.EnchantmentSlot.ID
				slotType = &enchant.EnchantmentSlot.Type
			}

			_, err := tx.Exec(`
				INSERT INTO player_equipment_enchantments (
					equipment_id, enchantment_id, slot_id, slot_type,
					display_string, source_item_id, source_item_name, spell_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`,
				equipmentID,
				enchant.EnchantmentID,
				slotID,
				slotType,
				enchant.DisplayString,
				sourceItemID,
				sourceItemName,
				spellID,
			)

			if err != nil {
				return 0, fmt.Errorf("failed to insert enchantment: %w", err)
			}
		}
	}

	return equipmentCount, nil
}

// CountPlayersMissingFingerprints returns how many valid players still lack a fingerprint
func (ds *DatabaseService) CountPlayersMissingFingerprints() (int, error) {
	var count int
	err := ds.db.QueryRow(`
		SELECT COUNT(*)
		FROM players p
		LEFT JOIN player_fingerprints pf ON pf.player_id = p.id
		WHERE pf.player_id IS NULL AND COALESCE(p.is_valid, 1) != 0
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count missing fingerprints: %w", err)
	}
	return count, nil
}

// CountPlayersNeedingStatusCheck returns how many players require a new status check
func (ds *DatabaseService) CountPlayersNeedingStatusCheck(staleBefore int64) (int, error) {
	var count int
	var err error
	if staleBefore > 0 {
		err = ds.db.QueryRow(`
			SELECT COUNT(*)
			FROM players
			WHERE status_checked_at IS NULL OR status_checked_at < ?
		`, staleBefore).Scan(&count)
	} else {
		err = ds.db.QueryRow(`
			SELECT COUNT(*)
			FROM players
			WHERE status_checked_at IS NULL
		`).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("count stale statuses: %w", err)
	}
	return count, nil
}

// UpdatePlayerStatus updates the cached status flags for a player
func (ds *DatabaseService) UpdatePlayerStatus(playerID int64, isValid bool, checkedAt int64, blizzardID *int) error {
	statusInt := 0
	if isValid {
		statusInt = 1
	}

	if blizzardID != nil {
		return retryOnBusy(func() error {
			_, err := ds.db.Exec(`
				UPDATE players
				SET is_valid = ?, status_checked_at = ?, blizzard_character_id = ?
				WHERE id = ?
			`, statusInt, checkedAt, *blizzardID, playerID)
			return err
		})
	}

	return retryOnBusy(func() error {
		_, err := ds.db.Exec(`
			UPDATE players
			SET is_valid = ?, status_checked_at = ?
			WHERE id = ?
		`, statusInt, checkedAt, playerID)
		return err
	})
}

// GetPlayersNeedingFingerprints returns a batch of players lacking fingerprint data
func (ds *DatabaseService) GetPlayersNeedingFingerprints(limit int) ([]PlayerFingerprintCandidate, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			p.id,
			p.name,
			r.region,
			r.slug,
			p.blizzard_character_id,
			(
				SELECT rm.spec_id
				FROM run_members rm
				WHERE rm.player_id = p.id AND rm.spec_id IS NOT NULL
				ORDER BY rm.run_id DESC
				LIMIT 1
			) AS latest_spec_id,
			pd.class_id,
			(
				SELECT MIN(cr.completed_timestamp)
				FROM run_members rm
				JOIN challenge_runs cr ON cr.id = rm.run_id
				WHERE rm.player_id = p.id
			) AS first_run_ts,
			(
				SELECT MAX(cr.completed_timestamp)
				FROM run_members rm
				JOIN challenge_runs cr ON cr.id = rm.run_id
				WHERE rm.player_id = p.id
			) AS last_run_ts
		FROM players p
		JOIN realms r ON r.id = p.realm_id
		LEFT JOIN player_fingerprints pf ON pf.player_id = p.id
		LEFT JOIN player_details pd ON pd.player_id = p.id
		WHERE pf.player_id IS NULL AND COALESCE(p.is_valid, 1) != 0
		ORDER BY last_run_ts DESC
		LIMIT ?
	`

	rows, err := ds.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query fingerprint candidates: %w", err)
	}
	defer rows.Close()

	var candidates []PlayerFingerprintCandidate
	for rows.Next() {
		var c PlayerFingerprintCandidate
		if err := rows.Scan(
			&c.PlayerID,
			&c.Name,
			&c.Region,
			&c.RealmSlug,
			&c.BlizzardCharacterID,
			&c.LatestSpecID,
			&c.DetailsClassID,
			&c.FirstRunTimestamp,
			&c.LastRunTimestamp,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

// GetPlayersNeedingStatusCheck returns players whose status check is stale
func (ds *DatabaseService) GetPlayersNeedingStatusCheck(limit int, staleBefore int64) ([]PlayerStatusCandidate, error) {
	if limit <= 0 {
		limit = 100
	}

	var rows *sql.Rows
	var err error
	if staleBefore > 0 {
		rows, err = ds.db.Query(`
			SELECT p.id, p.name, r.region, r.slug, p.status_checked_at, p.blizzard_character_id
			FROM players p
			JOIN realms r ON r.id = p.realm_id
			WHERE p.status_checked_at IS NULL OR p.status_checked_at < ?
			ORDER BY CASE WHEN p.status_checked_at IS NULL THEN 0 ELSE 1 END, p.status_checked_at ASC
			LIMIT ?
		`, staleBefore, limit)
	} else {
		rows, err = ds.db.Query(`
			SELECT p.id, p.name, r.region, r.slug, p.status_checked_at, p.blizzard_character_id
			FROM players p
			JOIN realms r ON r.id = p.realm_id
			WHERE p.status_checked_at IS NULL
			ORDER BY p.id
			LIMIT ?
		`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query status candidates: %w", err)
	}
	defer rows.Close()

	var candidates []PlayerStatusCandidate
	for rows.Next() {
		var c PlayerStatusCandidate
		if err := rows.Scan(&c.PlayerID, &c.Name, &c.Region, &c.RealmSlug, &c.StatusCheckedAt, &c.BlizzardCharacterID); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

// GetPlayerIDByFingerprintHash returns the player_id owning a fingerprint hash
func (ds *DatabaseService) GetPlayerIDByFingerprintHash(hash string) (int64, error) {
	if strings.TrimSpace(hash) == "" {
		return 0, nil
	}
	var playerID int64
	err := retryOnBusy(func() error {
		return ds.db.QueryRow(`SELECT player_id FROM player_fingerprints WHERE fingerprint_hash = ?`, hash).Scan(&playerID)
	})
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return playerID, err
}

// UpsertPlayerFingerprint inserts or updates a player's fingerprint record
func (ds *DatabaseService) UpsertPlayerFingerprint(fp PlayerFingerprint) error {
	return retryOnBusy(func() error {
		_, err := ds.db.Exec(`
			INSERT INTO player_fingerprints (
				player_id,
				fingerprint_hash,
				class_id,
				level85_timestamp,
				level90_timestamp,
				earliest_heroic_timestamp,
				last_seen_name,
				last_seen_realm_slug,
				last_seen_timestamp,
				first_run_timestamp,
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(player_id) DO UPDATE SET
				fingerprint_hash = excluded.fingerprint_hash,
				class_id = excluded.class_id,
				level85_timestamp = excluded.level85_timestamp,
				level90_timestamp = excluded.level90_timestamp,
				earliest_heroic_timestamp = excluded.earliest_heroic_timestamp,
				last_seen_name = excluded.last_seen_name,
				last_seen_realm_slug = excluded.last_seen_realm_slug,
				last_seen_timestamp = excluded.last_seen_timestamp,
				first_run_timestamp = excluded.first_run_timestamp
		`, fp.PlayerID, fp.FingerprintHash, fp.ClassID, fp.Level85Timestamp, fp.Level90Timestamp, fp.EarliestHeroicTimestamp,
			fp.LastSeenName, fp.LastSeenRealmSlug, fp.LastSeenTimestamp, fp.FirstRunTimestamp, fp.CreatedAt)
		return err
	})
}

// GetAllFingerprintHashes loads all existing fingerprint hashes into memory for collision detection
func (ds *DatabaseService) GetAllFingerprintHashes() (map[string]int64, error) {
	query := `SELECT fingerprint_hash, player_id FROM player_fingerprints`

	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query all fingerprint hashes: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var hash string
		var playerID int64
		if err := rows.Scan(&hash, &playerID); err != nil {
			return nil, err
		}
		result[hash] = playerID
	}
	return result, rows.Err()
}

// DeletePlayerFingerprint removes a player's fingerprint record
func (ds *DatabaseService) DeletePlayerFingerprint(playerID int64) error {
	return retryOnBusy(func() error {
		_, err := ds.db.Exec(`DELETE FROM player_fingerprints WHERE player_id = ?`, playerID)
		return err
	})
}

// MigratePlayerRuns updates all run_members records from one player to another
func (ds *DatabaseService) MigratePlayerRuns(fromPlayerID, toPlayerID int64) (int, error) {
	var rowsAffected int
	err := retryOnBusy(func() error {
		result, err := ds.db.Exec(`
			UPDATE run_members
			SET player_id = ?
			WHERE player_id = ?
		`, toPlayerID, fromPlayerID)

		if err != nil {
			return fmt.Errorf("migrate runs %d→%d: %w", fromPlayerID, toPlayerID, err)
		}

		rows, _ := result.RowsAffected()
		rowsAffected = int(rows)
		return nil
	})
	return rowsAffected, err
}

// InvalidatePlayerProfile deletes cached profile to force rebuild
func (ds *DatabaseService) InvalidatePlayerProfile(playerID int64) error {
	return retryOnBusy(func() error {
		_, err := ds.db.Exec(`
			DELETE FROM player_profiles
			WHERE player_id = ?
		`, playerID)
		return err
	})
}

// GetPlayerByNameRealmRegion looks up a player by name, realm slug, and region
func (ds *DatabaseService) GetPlayerByNameRealmRegion(name, realmSlug, region string) (int64, error) {
	var playerID int64
	err := retryOnBusy(func() error {
		return ds.db.QueryRow(`
			SELECT p.id
			FROM players p
			JOIN realms r ON r.id = p.realm_id
			WHERE p.name_lower = LOWER(?)
			  AND r.slug = ?
			  AND r.region = ?
			LIMIT 1
		`, name, realmSlug, region).Scan(&playerID)
	})

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("player not found: %s-%s (%s)", name, realmSlug, region)
	}
	if err != nil {
		return 0, fmt.Errorf("query player: %w", err)
	}
	return playerID, nil
}

// getExistingPlayerIDTx looks up an existing player by name+realm within a transaction.
// Returns 0 if no player found (not an error).
func getExistingPlayerIDTx(tx *sql.Tx, name string, realmID int) (int64, error) {
	var playerID int64
	err := tx.QueryRow(`
		SELECT id FROM players
		WHERE name_lower = LOWER(?) AND realm_id = ?
		LIMIT 1
	`, name, realmID).Scan(&playerID)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return playerID, nil
}
