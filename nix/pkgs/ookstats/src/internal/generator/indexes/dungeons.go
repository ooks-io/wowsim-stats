package indexes

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"

	"ookstats/internal/writer"
)

var dungeonShortNames = map[string]string{
	"temple-of-the-jade-serpent": "TJS",
	"stormstout-brewery":         "SB",
	"shado-pan-monastery":        "SPM",
	"mogu-shan-palace":           "MSP",
	"siege-of-niuzao-temple":     "SNT",
	"gate-of-the-setting-sun":    "GSS",
	"scarlet-halls":              "SH",
	"scarlet-monastery":          "SM",
	"scholomance":                "SCHOLO",
}

func generateShortName(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return ""
	}

	var acronym strings.Builder
	for _, word := range words {
		if len(word) > 0 && word != "of" && word != "the" {
			acronym.WriteRune([]rune(strings.ToUpper(word))[0])
		}
	}

	result := acronym.String()
	if len(result) > 6 {
		result = result[:6]
	}
	return result
}

func loadDungeons(db *sql.DB) ([]DungeonData, error) {
	rows, err := db.Query(`
		SELECT id, slug, name, map_challenge_mode_id
		FROM dungeons
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query dungeons: %w", err)
	}
	defer rows.Close()

	var dungeons []DungeonData
	for rows.Next() {
		var id int
		var slug, name string
		var mapChallengeModeID sql.NullInt64

		if err := rows.Scan(&id, &slug, &name, &mapChallengeModeID); err != nil {
			return nil, fmt.Errorf("scan dungeon: %w", err)
		}

		shortName := dungeonShortNames[slug]
		if shortName == "" {
			shortName = generateShortName(name)
		}

		var mapChallengeID *int
		if mapChallengeModeID.Valid {
			val := int(mapChallengeModeID.Int64)
			mapChallengeID = &val
		}

		dungeons = append(dungeons, DungeonData{
			ID:                 id,
			Slug:               slug,
			Name:               name,
			ShortName:          shortName,
			MapChallengeModeID: mapChallengeID,
		})
	}

	return dungeons, nil
}

// GenerateGlobalDungeonsIndex generates index of dungeons for global leaderboards
func GenerateGlobalDungeonsIndex(db *sql.DB, outDir string, seasonID int) error {
	log.Info("Generating global dungeons index", "season", seasonID)

	dungeons, err := loadDungeons(db)
	if err != nil {
		return err
	}

	// Add links for global scope
	for i := range dungeons {
		dungeons[i].Links = DungeonLinks{
			Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/global/%s/{page}.json", seasonID, dungeons[i].Slug)},
		}
	}

	index := DungeonsIndex{
		Data:     dungeons,
		Metadata: NewIndexMetadata(len(dungeons)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "global", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated global dungeons index", "season", seasonID, "count", len(dungeons), "path", outPath)
	return nil
}

// GenerateRealmDungeonsIndex generates index of dungeons for a specific realm
func GenerateRealmDungeonsIndex(db *sql.DB, outDir string, seasonID int, region, realm string) error {
	dungeons, err := loadDungeons(db)
	if err != nil {
		return err
	}

	// Add links for realm scope
	for i := range dungeons {
		dungeons[i].Links = DungeonLinks{
			Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/%s/%s/%s/{page}.json", seasonID, region, realm, dungeons[i].Slug)},
		}
	}

	index := RealmDungeonsIndex{
		Data:     dungeons,
		Metadata: NewIndexMetadata(len(dungeons)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), region, realm, "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	return nil
}
