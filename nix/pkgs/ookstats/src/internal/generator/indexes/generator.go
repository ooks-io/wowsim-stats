package indexes

import (
	"database/sql"
	"fmt"

	"github.com/charmbracelet/log"
)

var allClasses = []string{
	"warrior", "paladin", "hunter", "rogue", "priest",
	"death_knight", "shaman", "mage", "warlock", "monk", "druid",
}

var allRegions = []string{"us", "eu", "kr", "tw"}

func GenerateAllIndexes(db *sql.DB, outDir string) error {
	log.Info("Starting comprehensive index generation")

	// 1. Root index
	if err := GenerateRootIndex(outDir); err != nil {
		return fmt.Errorf("root index: %w", err)
	}

	// 2. Seasons index
	if err := GenerateSeasonsIndex(db, outDir); err != nil {
		return fmt.Errorf("seasons index: %w", err)
	}

	// 3. Get current season for season-specific indexes
	seasonID, err := getCurrentSeasonID(db)
	if err != nil {
		return fmt.Errorf("get current season: %w", err)
	}
	if seasonID == 0 {
		log.Warn("No current season found, using season 1 as default")
		seasonID = 1
	}

	log.Info("Generating indexes for season", "season", seasonID)

	// 4. Season scope index
	if err := GenerateSeasonScopeIndex(outDir, seasonID); err != nil {
		return fmt.Errorf("season scope index: %w", err)
	}

	// 5. Global dungeons index
	if err := GenerateGlobalDungeonsIndex(db, outDir, seasonID); err != nil {
		return fmt.Errorf("global dungeons index: %w", err)
	}

	// 6. Regional indexes (realms + realm dungeons)
	for _, region := range allRegions {
		// Regional realms index
		if err := GenerateRegionalRealmsIndex(db, outDir, seasonID, region); err != nil {
			return fmt.Errorf("regional realms index for %s: %w", region, err)
		}

		// Get realms for this region
		realms, err := getRealmSlugsForRegion(db, region)
		if err != nil {
			return fmt.Errorf("get realms for %s: %w", region, err)
		}

		// Generate dungeons index for each realm
		for _, realm := range realms {
			if err := GenerateRealmDungeonsIndex(db, outDir, seasonID, region, realm); err != nil {
				log.Warn("Failed to generate realm dungeons index", "region", region, "realm", realm, "error", err)
			}
		}
	}

	// 7. Players scope index
	if err := GeneratePlayersScopeIndex(outDir, seasonID); err != nil {
		return fmt.Errorf("players scope index: %w", err)
	}

	// 8. Players regional index
	if err := GeneratePlayersRegionalIndex(outDir, seasonID); err != nil {
		return fmt.Errorf("players regional index: %w", err)
	}

	// 9. Players realm indexes
	if err := GeneratePlayersRealmRegionIndex(outDir, seasonID); err != nil {
		return fmt.Errorf("players realm region index: %w", err)
	}

	for _, region := range allRegions {
		if err := GeneratePlayersRealmListIndex(db, outDir, seasonID, region); err != nil {
			return fmt.Errorf("players realm list index for %s: %w", region, err)
		}
	}

	// 10. Players class index
	if err := GeneratePlayersClassIndex(outDir, seasonID); err != nil {
		return fmt.Errorf("players class index: %w", err)
	}

	// 11. Class-specific indexes
	for _, classKey := range allClasses {
		// Class scope index
		if err := GenerateClassScopeIndex(outDir, seasonID, classKey); err != nil {
			log.Warn("Failed to generate class scope index", "class", classKey, "error", err)
			continue
		}

		// Class regional index
		if err := GenerateClassRegionalIndex(outDir, seasonID, classKey); err != nil {
			log.Warn("Failed to generate class regional index", "class", classKey, "error", err)
		}

		// Class realm region index
		if err := GenerateClassRealmRegionIndex(outDir, seasonID, classKey); err != nil {
			log.Warn("Failed to generate class realm region index", "class", classKey, "error", err)
		}

		// Class realm list indexes for each region
		for _, region := range allRegions {
			if err := GenerateClassRealmListIndex(db, outDir, seasonID, classKey, region); err != nil {
				log.Warn("Failed to generate class realm list index", "class", classKey, "region", region, "error", err)
			}
		}
	}

	log.Info("Completed comprehensive index generation")
	return nil
}

func getCurrentSeasonID(db *sql.DB) (int, error) {
	var seasonID int
	err := db.QueryRow(`
		SELECT DISTINCT season_number
		FROM seasons
		WHERE end_timestamp IS NULL
		ORDER BY season_number DESC
		LIMIT 1
	`).Scan(&seasonID)

	if err == sql.ErrNoRows {
		err = db.QueryRow(`
			SELECT DISTINCT season_number
			FROM seasons
			ORDER BY season_number DESC
			LIMIT 1
		`).Scan(&seasonID)
	}

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return seasonID, err
}

func getRealmSlugsForRegion(db *sql.DB, region string) ([]string, error) {
	rows, err := db.Query(`SELECT slug FROM realms WHERE region = ? ORDER BY slug`, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}

	return slugs, nil
}
