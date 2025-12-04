package indexes

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"ookstats/internal/writer"
)

// GenerateClassScopeIndex generates scope index for a specific class
func GenerateClassScopeIndex(outDir string, seasonID int, classKey string) error {
	scopes := []ClassScopeData{
		{
			Scope: "global",
			Href:  fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/global/{page}.json", seasonID, classKey),
		},
		{
			Scope: "regional",
			Href:  fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/regional/index.json", seasonID, classKey),
		},
		{
			Scope: "realm",
			Href:  fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/index.json", seasonID, classKey),
		},
	}

	index := ClassScopeIndex{
		Data:     scopes,
		Metadata: NewIndexMetadata(len(scopes)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "class", classKey, "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	return nil
}

// GenerateClassRegionalIndex generates regional index for a class
func GenerateClassRegionalIndex(outDir string, seasonID int, classKey string) error {
	regions := []RegionData{
		{Region: "us", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/regional/us/{page}.json", seasonID, classKey)},
		{Region: "eu", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/regional/eu/{page}.json", seasonID, classKey)},
		{Region: "kr", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/regional/kr/{page}.json", seasonID, classKey)},
		{Region: "tw", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/regional/tw/{page}.json", seasonID, classKey)},
	}

	index := RegionsIndex{
		Data:     regions,
		Metadata: NewIndexMetadata(len(regions)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "class", classKey, "regional", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	return nil
}

// GenerateClassRealmRegionIndex generates realm region index for a class
func GenerateClassRealmRegionIndex(outDir string, seasonID int, classKey string) error {
	regions := []RegionData{
		{Region: "us", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/us/index.json", seasonID, classKey)},
		{Region: "eu", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/eu/index.json", seasonID, classKey)},
		{Region: "kr", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/kr/index.json", seasonID, classKey)},
		{Region: "tw", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/tw/index.json", seasonID, classKey)},
	}

	index := RegionsIndex{
		Data:     regions,
		Metadata: NewIndexMetadata(len(regions)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "class", classKey, "realm", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	return nil
}

// GenerateClassRealmListIndex generates realm list for a class and region
func GenerateClassRealmListIndex(db *sql.DB, outDir string, seasonID int, classKey, region string) error {
	rows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, region)
	if err != nil {
		return fmt.Errorf("query realms: %w", err)
	}
	defer rows.Close()

	var realms []RegionData
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return err
		}
		realms = append(realms, RegionData{
			Region: slug,
			Href:   fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/realm/%s/%s/{page}.json", seasonID, classKey, region, slug),
		})
	}

	index := RegionsIndex{
		Data:     realms,
		Metadata: NewIndexMetadata(len(realms)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "class", classKey, "realm", region, "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	return nil
}
