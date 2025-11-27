package merge

import (
	"encoding/json"
	"fmt"
	"os"
)

// PlayerIdentity represents a player by name, realm, and region
type PlayerIdentity struct {
	Name   string `json:"name"`
	Realm  string `json:"realm"`
	Region string `json:"region"`
}

// MergeEntry defines a single merge operation
type MergeEntry struct {
	From PlayerIdentity `json:"from"`
	To   PlayerIdentity `json:"to"`
}

// MergeConfig is the root configuration structure
type MergeConfig struct {
	Merges []MergeEntry `json:"merges"`
}

// LoadConfig reads and parses a merge configuration file
func LoadConfig(path string) (*MergeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config MergeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(config.Merges) == 0 {
		return nil, fmt.Errorf("config contains no merge entries")
	}

	return &config, nil
}
