package database

import "fmt"

type PlayerAccountFingerprint struct {
	PlayerID   int64
	Region     string
	TuplesJSON string
	FetchedAt  int64
}

type AccountGroup struct {
	Region    string
	PlayerIDs []int64
}

func (ds *DatabaseService) UpsertPlayerAccountFingerprint(playerID int64, region, tuplesJSON string, fetchedAt int64) error {
	return retryOnBusy(func() error {
		_, err := ds.db.Exec(`
			INSERT INTO player_account_fingerprint (player_id, region, tuples_json, fetched_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(player_id) DO UPDATE SET
				region      = excluded.region,
				tuples_json = excluded.tuples_json,
				fetched_at  = excluded.fetched_at
		`, playerID, region, tuplesJSON, fetchedAt)
		return err
	})
}

func (ds *DatabaseService) LoadAllPlayerAccountFingerprints() ([]PlayerAccountFingerprint, error) {
	rows, err := ds.db.Query(`
		SELECT player_id, region, tuples_json, fetched_at
		FROM player_account_fingerprint
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlayerAccountFingerprint
	for rows.Next() {
		var f PlayerAccountFingerprint
		if err := rows.Scan(&f.PlayerID, &f.Region, &f.TuplesJSON, &f.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type AccountBackfillCandidate struct {
	PlayerID  int64
	Name      string
	RealmSlug string
	Region    string
}

func (ds *DatabaseService) PlayersMissingAccountFingerprint(retryBefore int64) ([]AccountBackfillCandidate, error) {
	rows, err := ds.db.Query(`
		SELECT pf.player_id, p.name, r.slug, r.region
		FROM player_fingerprints pf
		JOIN players p ON p.id = pf.player_id
		JOIN realms r  ON r.id = p.realm_id
		LEFT JOIN player_account_fingerprint paf ON paf.player_id = pf.player_id
		WHERE paf.player_id IS NULL
		  AND p.is_valid = 1
		  AND (p.account_fp_attempted_at IS NULL OR p.account_fp_attempted_at < ?)
	`, retryBefore)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AccountBackfillCandidate
	for rows.Next() {
		var c AccountBackfillCandidate
		if err := rows.Scan(&c.PlayerID, &c.Name, &c.RealmSlug, &c.Region); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// single tx so a failure mid-way leaves the previous grouping intact
func (ds *DatabaseService) RebuildAccounts(groups []AccountGroup, computedAt int64) (int, error) {
	var totalChars int
	err := retryOnBusy(func() error {
		tx, err := ds.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`UPDATE players SET account_id = NULL WHERE account_id IS NOT NULL`); err != nil {
			return fmt.Errorf("clear account_id: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM accounts`); err != nil {
			return fmt.Errorf("delete accounts: %w", err)
		}

		insAcct, err := tx.Prepare(`INSERT INTO accounts (region, character_count, computed_at) VALUES (?, ?, ?)`)
		if err != nil {
			return err
		}
		defer insAcct.Close()
		updPlayer, err := tx.Prepare(`UPDATE players SET account_id = ? WHERE id = ?`)
		if err != nil {
			return err
		}
		defer updPlayer.Close()

		totalChars = 0
		for _, g := range groups {
			if len(g.PlayerIDs) == 0 {
				continue
			}
			res, err := insAcct.Exec(g.Region, len(g.PlayerIDs), computedAt)
			if err != nil {
				return fmt.Errorf("insert account: %w", err)
			}
			accountID, err := res.LastInsertId()
			if err != nil {
				return err
			}
			for _, pid := range g.PlayerIDs {
				if _, err := updPlayer.Exec(accountID, pid); err != nil {
					return fmt.Errorf("update players.account_id pid=%d: %w", pid, err)
				}
			}
			totalChars += len(g.PlayerIDs)
		}
		return tx.Commit()
	})
	if err != nil {
		return 0, err
	}
	return totalChars, nil
}

// MarkAccountFingerprintAttempts stamps candidates so failed fetches wait out the retry cooldown
func (ds *DatabaseService) MarkAccountFingerprintAttempts(playerIDs []int64, ts int64) error {
	if len(playerIDs) == 0 {
		return nil
	}
	return retryOnBusy(func() error {
		tx, err := ds.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		stmt, err := tx.Prepare(`UPDATE players SET account_fp_attempted_at = ? WHERE id = ?`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, id := range playerIDs {
			if _, err := stmt.Exec(ts, id); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

func (ds *DatabaseService) CountPlayerAccountFingerprints() (int, error) {
	var n int
	err := ds.db.QueryRow(`SELECT COUNT(*) FROM player_account_fingerprint`).Scan(&n)
	return n, err
}

func (ds *DatabaseService) CountPlayersMissingAccountFingerprint() (int, error) {
	var n int
	err := ds.db.QueryRow(`
		SELECT COUNT(*) FROM player_fingerprints pf
		LEFT JOIN player_account_fingerprint paf ON paf.player_id = pf.player_id
		JOIN players p ON p.id = pf.player_id
		WHERE paf.player_id IS NULL AND p.is_valid = 1
	`).Scan(&n)
	return n, err
}

