package realms

import (
	"database/sql"
	"fmt"
	"strings"
)

// ParentSlugByRegion defines merged realm mappings keyed by region and child slug
var ParentSlugByRegion = map[string]map[string]string{
	"us": {
		"nazgrim":   "pagle",
		"galakras":  "pagle",
		"raden":     "pagle",
		"ra-den":    "pagle",
		"lei-shen":  "pagle",
		"leishen":   "pagle",
		"immerseus": "pagle",
	},
	"eu": {
		"shekzeer":  "mirage-raceway",
		"garalon":   "mirage-raceway",
		"norushen":  "mirage-raceway",
		"hoptallus": "mirage-raceway",
		"hotallus":  "mirage-raceway",
		"ook-ook":   "everlook",
		"ookook":    "everlook",
	},
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// EffectiveSlug returns the leaderboard slug (parent) for the given region+slug pair.
func EffectiveSlug(region, slug string) string {
	normalizedRegion := normalize(region)
	normalizedSlug := normalize(slug)
	if normalizedSlug == "" {
		return ""
	}

	if regionMap, ok := ParentSlugByRegion[normalizedRegion]; ok {
		if parent, ok := regionMap[normalizedSlug]; ok {
			return parent
		}
	}
	return normalizedSlug
}

// SyncRealmGroups populates the realm_groups table so child -> parent mappings are available in SQL.
func SyncRealmGroups(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM realm_groups`); err != nil {
		return fmt.Errorf("failed clearing realm_groups: %w", err)
	}

	rows, err := tx.Query(`SELECT id, slug, region FROM realms`)
	if err != nil {
		return fmt.Errorf("failed loading realms for mapping: %w", err)
	}
	defer rows.Close()

	type realmRow struct {
		id     int64
		slug   string
		region string
	}

	realmIndex := make(map[string]map[string]int64)
	for rows.Next() {
		var rr realmRow
		if err := rows.Scan(&rr.id, &rr.slug, &rr.region); err != nil {
			return err
		}
		regionKey := normalize(rr.region)
		slugKey := normalize(rr.slug)
		if _, ok := realmIndex[regionKey]; !ok {
			realmIndex[regionKey] = make(map[string]int64)
		}
		realmIndex[regionKey][slugKey] = rr.id
	}

	for region, children := range ParentSlugByRegion {
		known := realmIndex[region]
		if known == nil {
			continue
		}
		for childSlug, parentSlug := range children {
			childID := known[childSlug]
			parentID := known[parentSlug]
			if childID == 0 || parentID == 0 {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO realm_groups (child_realm_id, parent_realm_id) VALUES (?, ?)`, childID, parentID); err != nil {
				return fmt.Errorf("failed inserting realm group for %s -> %s: %w", childSlug, parentSlug, err)
			}
		}
	}

	return nil
}
