package indexes

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"

	"ookstats/internal/writer"
)

// GenerateRegionalRealmsIndex generates index of realms for a region + "all" endpoint
func GenerateRegionalRealmsIndex(db *sql.DB, outDir string, seasonID int, region string) error {
	log.Info("Generating regional realms index", "season", seasonID, "region", region)

	rows, err := db.Query(`
		SELECT
			r.slug,
			r.name,
			r.connected_realm_id,
			r.parent_realm_slug,
			COUNT(DISTINCT p.id) as player_count
		FROM realms r
		LEFT JOIN players p ON p.realm_id = r.id
		WHERE r.region = ?
		GROUP BY r.slug
		ORDER BY r.slug
	`, region)
	if err != nil {
		return fmt.Errorf("query realms for region %s: %w", region, err)
	}
	defer rows.Close()

	var realmsData []RealmData

	for rows.Next() {
		var slug, name string
		var connectedRealmID sql.NullInt64
		var parentRealmSlug sql.NullString
		var playerCount int

		if err := rows.Scan(&slug, &name, &connectedRealmID, &parentRealmSlug, &playerCount); err != nil {
			return fmt.Errorf("scan realm: %w", err)
		}

		var connectedID *int
		if connectedRealmID.Valid {
			val := int(connectedRealmID.Int64)
			connectedID = &val
		}

		var parentRealm *string
		if parentRealmSlug.Valid && parentRealmSlug.String != "" {
			parentRealm = &parentRealmSlug.String
		}

		realmsData = append(realmsData, RealmData{
			Slug:             slug,
			Name:             name,
			ConnectedRealmID: connectedID,
			ParentRealm:      parentRealm,
			PlayerCount:      playerCount,
			Links: RealmLinks{
				Dungeons: Link{Href: fmt.Sprintf("/api/leaderboard/season/%d/%s/%s/index.json", seasonID, region, slug)},
			},
		})
	}

	index := RegionalRealmsIndex{
		All: RegionalAllLink{
			Href: fmt.Sprintf("/api/leaderboard/season/%d/%s/all/{dungeon}/{page}.json", seasonID, region),
			Note: "Regional aggregate leaderboard (all realms combined)",
		},
		Data:     realmsData,
		Metadata: NewIndexMetadata(len(realmsData)),
	}

	outPath := filepath.Join(outDir, "api", "leaderboard", "season", fmt.Sprintf("%d", seasonID), region, "index.json")
	if err := writer.WriteJSONFile(outPath, index); err != nil {
		return err
	}

	log.Info("Generated regional realms index", "season", seasonID, "region", region, "count", len(realmsData), "path", outPath)
	return nil
}
