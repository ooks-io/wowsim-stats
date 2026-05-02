package cmd

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"ookstats/internal/blizzard"
	"ookstats/internal/database"
)

var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "Populate reference data",
	Long:  `Populate database with reference data like items, dungeons, etc.`,
}

var populateItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Populate item database",
	Long:  `Populate item database with data from WoW Sims database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Item Population ===")

		wowsimsDBPath, _ := cmd.Flags().GetString("wowsims-db")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		fmt.Printf("Connected to database: %s\n", database.DBFilePath())

		if err := populateItems(db, wowsimsDBPath); err != nil {
			return fmt.Errorf("failed to populate items: %w", err)
		}

		fmt.Printf("Item population complete!\n")
		return nil
	},
}

// WoWSims database structures
type WowSimsDatabase struct {
	Items    []WowSimsItem    `json:"items"`
	Gems     []WowSimsGem     `json:"gems"`
	Enchants []WowSimsEnchant `json:"enchants"`
}

type WowSimsItem struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Icon           string          `json:"icon"`
	Quality        int             `json:"quality"`
	Type           int             `json:"type"`
	ScalingOptions json.RawMessage `json:"scalingOptions,omitempty"`
	ItemEffect     json.RawMessage `json:"itemEffect,omitempty"`
}

type WowSimsGem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   int    `json:"color"`
	Stats   []int  `json:"stats"`
	Phase   int    `json:"phase"`
	Quality int    `json:"quality"`
}

type WowSimsEnchant struct {
	ID       int    `json:"id,omitempty"`
	EffectID int    `json:"effectId,omitempty"`
	ItemID   int    `json:"itemId,omitempty"`
	SpellID  int    `json:"spellId,omitempty"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Type     int    `json:"type"`
	Stats    []int  `json:"stats"`
	Quality  int    `json:"quality"`
}

func populateItems(db *sql.DB, wowsimsDBPath string) error {
	var wowsimsDB WowSimsDatabase

	if strings.TrimSpace(wowsimsDBPath) == "" {
		// Try environment-provided path via Nix wrapper
		if envPath := strings.TrimSpace(os.Getenv("OOKSTATS_WOWSIMS_DB")); envPath != "" {
			fmt.Printf("Loading items from OOKSTATS_WOWSIMS_DB=%s\n", envPath)
			file, err := os.Open(envPath)
			if err != nil {
				return fmt.Errorf("failed to open WoW Sims database file: %w", err)
			}
			defer file.Close()
			if err := json.NewDecoder(file).Decode(&wowsimsDB); err != nil {
				return fmt.Errorf("failed to parse WoW Sims database: %w", err)
			}
		} else {
			return fmt.Errorf("no items DB provided; set OOKSTATS_WOWSIMS_DB or use --wowsims-db")
		}
	} else {
		fmt.Printf("Loading items from %s\n", wowsimsDBPath)
		// load WoWSims database JSON from file
		file, err := os.Open(wowsimsDBPath)
		if err != nil {
			return fmt.Errorf("failed to open WoW Sims database file: %w", err)
		}
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&wowsimsDB); err != nil {
			return fmt.Errorf("failed to parse WoW Sims database: %w", err)
		}
	}

	fmt.Printf("Found %d items, %d gems, %d enchants\n",
		len(wowsimsDB.Items), len(wowsimsDB.Gems), len(wowsimsDB.Enchants))

	// begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// clear existing items
	fmt.Println("Clearing existing items...")
	if _, err := tx.Exec("DELETE FROM items"); err != nil {
		return fmt.Errorf("failed to clear existing items: %w", err)
	}

	fmt.Println("Inserting items...")
	insertCount := 0

	for _, item := range wowsimsDB.Items {
		if item.ID == 0 {
			continue
		}

		statsJSON := "{}"
		if len(item.ScalingOptions) > 0 {
			statsJSON = string(item.ScalingOptions)
		}

		var itemEffectJSON *string
		if len(item.ItemEffect) > 0 {
			s := string(item.ItemEffect)
			itemEffectJSON = &s
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO items (id, name, icon, quality, type, stats, item_effect)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, item.ID, item.Name, item.Icon, item.Quality, item.Type, statsJSON, itemEffectJSON)

		if err != nil {
			return fmt.Errorf("failed to insert item %d: %w", item.ID, err)
		}

		insertCount++
	}

	// insert gems (with stats array)
	for _, gem := range wowsimsDB.Gems {
		if gem.ID == 0 {
			continue
		}

		// convert stats array to JSON string
		statsJSON, err := json.Marshal(gem.Stats)
		if err != nil {
			fmt.Printf("Warning: failed to marshal stats for gem %d: %v\n", gem.ID, err)
			statsJSON = []byte("[]")
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO items (id, name, icon, quality, type, stats)
			VALUES (?, ?, ?, ?, ?, ?)
		`, gem.ID, gem.Name, gem.Icon, gem.Quality, 99, string(statsJSON)) // Use type 99 for gems

		if err != nil {
			return fmt.Errorf("failed to insert gem %d: %w", gem.ID, err)
		}

		insertCount++
	}

	// insert enchants (with stats array)
	for _, enchant := range wowsimsDB.Enchants {
		// use ItemID if available, otherwise EffectID, otherwise ID
		id := enchant.ItemID
		if id == 0 {
			id = enchant.EffectID
		}
		if id == 0 {
			id = enchant.ID
		}
		if id == 0 {
			continue
		}

		// convert stats array to JSON string
		statsJSON, err := json.Marshal(enchant.Stats)
		if err != nil {
			fmt.Printf("Warning: failed to marshal stats for enchant %d: %v\n", id, err)
			statsJSON = []byte("[]")
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO items (id, name, icon, quality, type, stats)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, enchant.Name, enchant.Icon, enchant.Quality, 98, string(statsJSON)) // Use type 98 for enchants

		if err != nil {
			return fmt.Errorf("failed to insert enchant %d: %w", id, err)
		}

		insertCount++
	}

	// progress reporting
	if insertCount%1000 == 0 {
		fmt.Printf("Processed %d total items...\n", insertCount)
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit items: %w", err)
	}

	fmt.Printf("[OK] Successfully inserted %d items\n", insertCount)
	return nil
}

var populateItemsEnrichCmd = &cobra.Command{
	Use:   "items-enrich",
	Short: "Enrich items with Blizzard spell descriptions",
	Long:  `Fetch spell descriptions from Blizzard API for items missing them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Item Enrichment ===")

		region, _ := cmd.Flags().GetString("region")

		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		client, err := blizzard.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create blizzard client: %w", err)
		}

		if err := enrichItemsWithDescriptions(db, client, region); err != nil {
			return fmt.Errorf("failed to enrich items: %w", err)
		}

		fmt.Printf("Item enrichment complete!\n")
		return nil
	},
}

func enrichItemsWithDescriptions(db *sql.DB, client *blizzard.Client, region string) error {
	// get all item IDs that are missing spell_description (only trinkets type 12)
	rows, err := db.Query(`
		SELECT id FROM items
		WHERE spell_description IS NULL
		  AND type = 12
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("query items: %w", err)
	}

	var itemIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan item id: %w", err)
		}
		itemIDs = append(itemIDs, id)
	}
	rows.Close()

	if len(itemIDs) == 0 {
		fmt.Println("All items already have spell descriptions")
		return nil
	}

	fmt.Printf("Found %d items missing spell descriptions\n", len(itemIDs))

	// prepare update statement
	updateStmt, err := db.Prepare(`UPDATE items SET spell_description = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare update: %w", err)
	}
	defer updateStmt.Close()

	fetched := 0
	skipped := 0
	notFound := 0
	startTime := time.Now()

	for i, itemID := range itemIDs {
		// progress every 100 items
		if (i+1)%100 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(i+1) / elapsed.Seconds()
			remaining := float64(len(itemIDs)-i-1) / rate
			fmt.Printf("  Progress: %d/%d (%.1f/sec, ~%.0fs remaining)\n",
				i+1, len(itemIDs), rate, remaining)
		}

		item, err := client.FetchItem(itemID, region)
		if err != nil {
			var apiErr *blizzard.APIError
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
				notFound++
				continue
			}
			fmt.Printf("  Warning: failed to fetch item %d: %v\n", itemID, err)
			skipped++
			continue
		}

		// extract spell descriptions
		var descriptions []string
		if item.PreviewItem != nil {
			for _, spell := range item.PreviewItem.Spells {
				if spell.Description != "" {
					descriptions = append(descriptions, spell.Description)
				}
			}
		}

		if len(descriptions) == 0 {
			continue
		}

		// debug: log first few items with descriptions
		if fetched < 5 {
			fmt.Printf("  DEBUG: Item %d (%s) has description: %.50s...\n", itemID, item.Name, descriptions[0])
		}

		// join multiple descriptions with newline
		desc := strings.Join(descriptions, "\n")
		result, err := updateStmt.Exec(desc, itemID)
		if err != nil {
			fmt.Printf("  Warning: failed to update item %d: %v\n", itemID, err)
			skipped++
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			fmt.Printf("  Warning: item %d update affected 0 rows\n", itemID)
		}

		fetched++
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[OK] Enriched %d items with spell descriptions (%.1fs)\n", fetched, elapsed.Seconds())
	fmt.Printf("     %d not found in Blizzard API, %d skipped due to errors\n", notFound, skipped)
	return nil
}

func init() {
	rootCmd.AddCommand(populateCmd)
	populateCmd.AddCommand(populateItemsCmd)
	populateCmd.AddCommand(populateItemsEnrichCmd)

	// add flag for wowsims database path
	populateItemsCmd.Flags().String("wowsims-db", "", "Path to WoW Sims database JSON file")

	// add flag for region
	populateItemsEnrichCmd.Flags().String("region", "us", "Blizzard API region (us, eu, etc)")
}
