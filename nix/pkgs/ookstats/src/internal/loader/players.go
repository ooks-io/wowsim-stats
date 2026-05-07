package loader

import (
	"database/sql"
	"fmt"
	"strings"
)

// PlayerData represents a player loaded from the database with all profile information
type PlayerData struct {
	ID                int64
	Name              string
	RealmSlug         string
	RealmName         string
	Region            string
	ClassName         sql.NullString
	ActiveSpecName    sql.NullString
	AvatarURL         string
	GuildName         sql.NullString
	RaceName          sql.NullString
	AverageItemLevel  sql.NullInt64
	EquippedItemLevel sql.NullInt64
}

// PlayerSeasonData represents a player's stats for a specific season
type PlayerSeasonData struct {
	SeasonID          int
	MainSpecID        sql.NullInt64
	DungeonsCompleted int
	TotalRuns         int
	CombinedBest      sql.NullInt64
	GlobalRanking     sql.NullInt64
	RegionalRanking   sql.NullInt64
	RealmRanking      sql.NullInt64
	GlobalBracket     sql.NullString
	RegionalBracket   sql.NullString
	RealmBracket      sql.NullString
	LastUpdated       sql.NullInt64
}

// LoadAllCompleteCoveragePlayers loads all unique players who have complete coverage in ANY season
func LoadAllCompleteCoveragePlayers(db *sql.DB) ([]PlayerData, error) {
	rows, err := db.Query(`
        SELECT DISTINCT p.id, p.name, r.slug, r.name, r.region,
               pd.class_name, pd.active_spec_name,
               COALESCE(pd.avatar_url, ''),
               pd.guild_name, pd.race_name, pd.average_item_level, pd.equipped_item_level
        FROM players p
        JOIN realms r ON p.realm_id = r.id
        JOIN player_profiles pp ON p.id = pp.player_id
        LEFT JOIN player_details pd ON p.id = pd.player_id
        WHERE pp.has_complete_coverage = 1
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []PlayerData
	for rows.Next() {
		var player PlayerData
		if err := rows.Scan(
			&player.ID, &player.Name, &player.RealmSlug, &player.RealmName, &player.Region,
			&player.ClassName, &player.ActiveSpecName,
			&player.AvatarURL,
			&player.GuildName, &player.RaceName, &player.AverageItemLevel, &player.EquippedItemLevel); err != nil {
			return nil, fmt.Errorf("scan player: %w", err)
		}
		players = append(players, player)
	}
	return players, nil
}

// LoadAllPlayerSeasons loads all season data for a set of players
func LoadAllPlayerSeasons(db *sql.DB, playerIDs []int64) (map[int64][]PlayerSeasonData, error) {
	if len(playerIDs) == 0 {
		return make(map[int64][]PlayerSeasonData), nil
	}

	placeholders := make([]string, len(playerIDs))
	args := make([]any, len(playerIDs))
	for i, id := range playerIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
        SELECT player_id, season_id, main_spec_id, dungeons_completed, total_runs,
               combined_best_time, global_ranking, regional_ranking, realm_ranking,
               global_ranking_bracket, regional_ranking_bracket, realm_ranking_bracket,
               last_updated
        FROM player_profiles
        WHERE player_id IN (%s)
        ORDER BY player_id, season_id
    `, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seasonsMap := make(map[int64][]PlayerSeasonData)
	for rows.Next() {
		var playerID int64
		var season PlayerSeasonData
		if err := rows.Scan(
			&playerID, &season.SeasonID, &season.MainSpecID, &season.DungeonsCompleted, &season.TotalRuns,
			&season.CombinedBest, &season.GlobalRanking, &season.RegionalRanking, &season.RealmRanking,
			&season.GlobalBracket, &season.RegionalBracket, &season.RealmBracket,
			&season.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan player season: %w", err)
		}
		seasonsMap[playerID] = append(seasonsMap[playerID], season)
	}
	return seasonsMap, nil
}

func LoadPlayerAccountIDs(db *sql.DB) (map[int64]int64, error) {
	rows, err := db.Query(`SELECT id, account_id FROM players WHERE account_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("load player account ids: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]int64)
	for rows.Next() {
		var pid, aid int64
		if err := rows.Scan(&pid, &aid); err != nil {
			return nil, err
		}
		out[pid] = aid
	}
	return out, nil
}

// GetPlayerIDs extracts player IDs from a slice of PlayerData
func GetPlayerIDs(players []PlayerData) []int64 {
	ids := make([]int64, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}
	return ids
}

type AltSummary struct {
	PlayerID       int64
	Name           string
	RealmSlug      string
	RealmName      string
	Region         string
	ClassName      sql.NullString
	ActiveSpecName sql.NullString
	MainSpecID     sql.NullInt64
}

// correlated subqueries (not join on player_profiles) to avoid per-season fanout that would duplicate alts
func LoadAllAccountAlts(db *sql.DB) (map[int64][]AltSummary, error) {
	rows, err := db.Query(`
		SELECT p.account_id, p.id, p.name, r.slug, r.name, r.region,
		       pd.class_name, pd.active_spec_name,
		       (SELECT pp.main_spec_id FROM player_profiles pp
		         WHERE pp.player_id = p.id
		         ORDER BY pp.season_id DESC LIMIT 1) AS main_spec_id,
		       (SELECT COALESCE(SUM(pp.total_runs), 0) FROM player_profiles pp
		         WHERE pp.player_id = p.id) AS total_runs,
		       (SELECT MAX(pp.combined_best_time) FROM player_profiles pp
		         WHERE pp.player_id = p.id) AS combined_best
		FROM players p
		JOIN realms r ON p.realm_id = r.id
		LEFT JOIN player_details pd ON p.id = pd.player_id
		WHERE p.account_id IS NOT NULL AND p.is_valid = 1
	`)
	if err != nil {
		return nil, fmt.Errorf("load account alts: %w", err)
	}
	defer rows.Close()

	type rowEntry struct {
		AccountID  int64
		Alt        AltSummary
		HasAnyRuns bool
		HasAnyTime bool
	}

	byAccount := make(map[int64][]rowEntry)
	for rows.Next() {
		var (
			accountID    int64
			a            AltSummary
			totalRuns    sql.NullInt64
			combinedBest sql.NullInt64
		)
		if err := rows.Scan(
			&accountID, &a.PlayerID, &a.Name, &a.RealmSlug, &a.RealmName, &a.Region,
			&a.ClassName, &a.ActiveSpecName,
			&a.MainSpecID, &totalRuns, &combinedBest); err != nil {
			return nil, err
		}
		byAccount[accountID] = append(byAccount[accountID], rowEntry{
			AccountID:  accountID,
			Alt:        a,
			HasAnyRuns: totalRuns.Valid && totalRuns.Int64 > 0,
			HasAnyTime: combinedBest.Valid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[int64][]AltSummary, len(byAccount))
	for _, entries := range byAccount {
		for _, self := range entries {
			alts := make([]AltSummary, 0, len(entries)-1)
			for _, other := range entries {
				if other.Alt.PlayerID == self.Alt.PlayerID {
					continue
				}
				if !other.HasAnyTime && !other.HasAnyRuns && !other.Alt.ClassName.Valid {
					continue
				}
				alts = append(alts, other.Alt)
			}
			if len(alts) == 0 {
				continue
			}
			out[self.Alt.PlayerID] = alts
		}
	}
	return out, nil
}
