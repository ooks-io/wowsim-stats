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

// GetPlayerIDs extracts player IDs from a slice of PlayerData
func GetPlayerIDs(players []PlayerData) []int64 {
	ids := make([]int64, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}
	return ids
}
