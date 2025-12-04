package indexes

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"

	"ookstats/internal/writer"
)

func GenerateSeasonScopeIndex(outDir string, seasonID int) error {
	log.Info("Generating season scope index", "season", seasonID)

	scopes := []ScopeData{
		{
			Scope: "global",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/global/index.json", seasonID)},
			},
		},
		{
			Scope: "us",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/us/index.json", seasonID)},
			},
		},
		{
			Scope: "eu",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/eu/index.json", seasonID)},
			},
		},
		{
			Scope: "kr",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/kr/index.json", seasonID)},
			},
		},
		{
			Scope: "tw",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/tw/index.json", seasonID)},
			},
		},
		{
			Scope: "players",
			Links: ScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/index.json", seasonID)},
			},
		},
	}

	index := SeasonScopeIndex{
		Data:     scopes,
		Metadata: NewIndexMetadata(len(scopes)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated season scope index", "season", seasonID, "path", outPath)
	return nil
}
