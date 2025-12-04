package indexes

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/log"

	"ookstats/internal/wow"
	"ookstats/internal/writer"
)

var classKeys = map[string]int{
	"warrior":      1,
	"paladin":      2,
	"hunter":       3,
	"rogue":        4,
	"priest":       5,
	"death_knight": 6,
	"shaman":       7,
	"mage":         8,
	"warlock":      9,
	"monk":         10,
	"druid":        11,
}

// GeneratePlayersScopeIndex generates top-level players scope index
func GeneratePlayersScopeIndex(outDir string, seasonID int) error {
	log.Info("Generating players scope index", "season", seasonID)

	scopes := []PlayerScopeData{
		{
			Scope: "global",
			Links: PlayerScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/global/{page}.json", seasonID)},
			},
		},
		{
			Scope: "regional",
			Links: PlayerScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/regional/index.json", seasonID)},
			},
		},
		{
			Scope: "realm",
			Links: PlayerScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/realm/index.json", seasonID)},
			},
		},
		{
			Scope: "class",
			Links: PlayerScopeLinks{
				Leaderboard: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/index.json", seasonID)},
			},
		},
	}

	index := PlayersScopeIndex{
		Data:     scopes,
		Metadata: NewIndexMetadata(len(scopes)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated players scope index", "season", seasonID, "path", outPath)
	return nil
}

// GeneratePlayersRegionalIndex generates regional scope index (lists regions)
func GeneratePlayersRegionalIndex(outDir string, seasonID int) error {
	log.Info("Generating players regional index", "season", seasonID)

	regions := []RegionData{
		{Region: "us", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/regional/us/{page}.json", seasonID)},
		{Region: "eu", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/regional/eu/{page}.json", seasonID)},
		{Region: "kr", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/regional/kr/{page}.json", seasonID)},
		{Region: "tw", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/regional/tw/{page}.json", seasonID)},
	}

	index := RegionsIndex{
		Data:     regions,
		Metadata: NewIndexMetadata(len(regions)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "regional", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated players regional index", "season", seasonID, "path", outPath)
	return nil
}

// GeneratePlayersRealmRegionIndex generates realm scope region list
func GeneratePlayersRealmRegionIndex(outDir string, seasonID int) error {
	log.Info("Generating players realm region index", "season", seasonID)

	regions := []RegionData{
		{Region: "us", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/realm/us/index.json", seasonID)},
		{Region: "eu", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/realm/eu/index.json", seasonID)},
		{Region: "kr", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/realm/kr/index.json", seasonID)},
		{Region: "tw", Href: fmt.Sprintf("/api/leaderboard/season/%d/players/realm/tw/index.json", seasonID)},
	}

	index := RegionsIndex{
		Data:     regions,
		Metadata: NewIndexMetadata(len(regions)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "realm", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated players realm region index", "season", seasonID, "path", outPath)
	return nil
}

// GeneratePlayersRealmListIndex generates list of realms for a region (player leaderboards)
func GeneratePlayersRealmListIndex(db *sql.DB, outDir string, seasonID int, region string) error {
	log.Info("Generating players realm list index", "season", seasonID, "region", region)

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
			Href:   fmt.Sprintf("/api/leaderboard/season/%d/players/realm/%s/%s/{page}.json", seasonID, region, slug),
		})
	}

	index := RegionsIndex{
		Data:     realms,
		Metadata: NewIndexMetadata(len(realms)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "realm", region, "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated players realm list index", "season", seasonID, "region", region, "count", len(realms))
	return nil
}

// GeneratePlayersClassIndex generates class index
func GeneratePlayersClassIndex(outDir string, seasonID int) error {
	log.Info("Generating players class index", "season", seasonID)

	classMap := make(map[string]map[string]bool)
	for _, info := range wow.SpecByID {
		if classMap[info.ClassName] == nil {
			classMap[info.ClassName] = make(map[string]bool)
		}
		classMap[info.ClassName][info.SpecName] = true
	}

	var classes []ClassData
	for className, specsMap := range classMap {
		classKey := strings.ToLower(strings.ReplaceAll(className, " ", "_"))
		classID, ok := classKeys[classKey]
		if !ok {
			log.Warn("Unknown class key", "class", className, "key", classKey)
			continue
		}

		specs := make([]string, 0, len(specsMap))
		for spec := range specsMap {
			specs = append(specs, spec)
		}
		sort.Strings(specs)

		classes = append(classes, ClassData{
			ID:    classID,
			Key:   classKey,
			Name:  className,
			Specs: specs,
			Links: ClassLinks{
				Scopes: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/players/class/%s/index.json", seasonID, classKey)},
			},
		})
	}

	sort.Slice(classes, func(i, j int) bool {
		return classes[i].ID < classes[j].ID
	})

	index := PlayersClassIndex{
		Data:     classes,
		Metadata: NewIndexMetadata(len(classes)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), "players", "class", "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated players class index", "season", seasonID, "count", len(classes), "path", outPath)
	return nil
}
