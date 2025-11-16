package cmd

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"
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
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Quality int    `json:"quality"`
	Type    int    `json:"type"`
	// items don't have stats array, they have other fields like weaponType, etc.
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

	// insert items (no stats array)
	for _, item := range wowsimsDB.Items {
		if item.ID == 0 {
			continue
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO items (id, name, icon, quality, type, stats)
			VALUES (?, ?, ?, ?, ?, ?)
		`, item.ID, item.Name, item.Icon, item.Quality, item.Type, "{}")

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

func init() {
	rootCmd.AddCommand(populateCmd)
	populateCmd.AddCommand(populateItemsCmd)

	// add flag for wowsims database path
	populateItemsCmd.Flags().String("wowsims-db", "", "Path to WoW Sims database JSON file")
}
