package indexes

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"

	"ookstats/internal/writer"
)

func GenerateRootIndex(outDir string) error {
	log.Info("Generating root index")

	root := RootIndex{
		Links: RootIndexLinks{
			Self: Link{Href: "/api/index.json"},
		},
		Indexes: map[string]string{
			"seasons": "/api/leaderboard/season/index.json",
		},
		Endpoints: map[string]string{
			"dungeon_leaderboard": "/api/leaderboard/season/{season_id}/{scope}/{dungeon}/{page}.json",
			"player_leaderboard":  "/api/leaderboard/season/{season_id}/players/{scope}/{page}.json",
			"player_profile":      "/api/player/{region}/{realm}/{name}.json",
			"search":              "/api/search/players-{shard}.json",
		},
		APIVersion:  "1.0",
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
	}

	outPath := filepath.Join(outDir, "api", "index.json")
	if err := writer.WriteJSONFile(outPath, root); err != nil {
		return err
	}

	log.Info("Generated root index", "path", outPath)
	return nil
}
