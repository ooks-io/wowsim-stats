package indexes

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"

	"ookstats/internal/writer"
)

func GenerateSeasonsIndex(db *sql.DB, outDir string) error {
	log.Info("Generating seasons index")

	rows, err := db.Query(`
		SELECT
			season_number,
			season_name,
			MIN(start_timestamp) as start_ts,
			MAX(end_timestamp) as end_ts
		FROM seasons
		GROUP BY season_number
		ORDER BY season_number ASC
	`)
	if err != nil {
		return fmt.Errorf("query seasons: %w", err)
	}
	defer rows.Close()

	var seasonsData []SeasonData
	var currentSeason int

	for rows.Next() {
		var seasonNum int
		var seasonName sql.NullString
		var startTs, endTs sql.NullInt64

		if err := rows.Scan(&seasonNum, &seasonName, &startTs, &endTs); err != nil {
			return fmt.Errorf("scan season: %w", err)
		}

		// Season name fallback
		name := "Season " + fmt.Sprintf("%d", seasonNum)
		if seasonName.Valid && seasonName.String != "" {
			name = seasonName.String
		}

		var startTimestamp, endTimestamp *int64
		if startTs.Valid {
			startTimestamp = &startTs.Int64
		}
		if endTs.Valid {
			endTimestamp = &endTs.Int64
		}

		isCurrent := !endTs.Valid
		if isCurrent {
			currentSeason = seasonNum
		}

		seasonsData = append(seasonsData, SeasonData{
			ID:             seasonNum,
			Name:           name,
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
			IsCurrent:      isCurrent,
			Links: SeasonLinks{
				Self:   Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/index.json", seasonNum)},
				Scopes: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/index.json", seasonNum)},
			},
		})
	}

	if currentSeason == 0 && len(seasonsData) > 0 {
		seasonsData[len(seasonsData)-1].IsCurrent = true
	}

	index := SeasonsIndex{
		Data:     seasonsData,
		Metadata: NewIndexMetadata(len(seasonsData)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated seasons index", "count", len(seasonsData), "path", outPath)
	return nil
}
