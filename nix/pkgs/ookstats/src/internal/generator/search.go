package generator

import (
	"database/sql"
	"fmt"
	"ookstats/internal/writer"
	"path/filepath"
	"time"
)

// SearchEntry represents a player in the search index
type SearchEntry struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	RealmSlug     string `json:"realm_slug"`
	RealmName     string `json:"realm_name"`
	ClassName     string `json:"class_name,omitempty"`
	GlobalRanking *int   `json:"global_ranking,omitempty"`
	GlobalBracket string `json:"global_ranking_bracket,omitempty"`
}

// GenerateSearchIndex generates sharded JSON files for the player search index
func GenerateSearchIndex(db *sql.DB, out string, shardSize int) error {
	if shardSize <= 0 {
		shardSize = 5000
	}
	if err := writer.EnsureDir(out); err != nil {
		return err
	}

	rows, err := db.Query(`
        SELECT p.id, p.name, r.region, r.slug, r.name,
               COALESCE(pp.class_name, ''), pp.global_ranking, COALESCE(pp.global_ranking_bracket,'')
        FROM players p
        JOIN player_profiles pp ON p.id = pp.player_id
        JOIN realms r ON p.realm_id = r.id
        WHERE pp.has_complete_coverage = 1
          AND pp.season_id = (SELECT MAX(season_number) FROM seasons)
        ORDER BY pp.global_ranking ASC, p.name ASC
    `)
	if err != nil {
		return err
	}
	defer rows.Close()

	shard := 0
	count := 0
	buf := []SearchEntry{}

	// Precompute total
	var totalPlayers int
	if err := db.QueryRow(`
      SELECT COUNT(*)
      FROM players p
      JOIN player_profiles pp ON p.id = pp.player_id
      JOIN realms r ON p.realm_id = r.id
      WHERE pp.has_complete_coverage = 1
        AND pp.season_id = (SELECT MAX(season_number) FROM seasons)
    `).Scan(&totalPlayers); err != nil {
		return err
	}

	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		path := filepath.Join(out, fmt.Sprintf("players-%03d.json", shard))
		meta := map[string]any{
			"total_players":    totalPlayers,
			"returned_players": len(buf),
			"offset":           shard * shardSize,
			"limit":            shardSize,
			"last_updated":     time.Now().Format(time.RFC3339),
		}
		if err := writer.WriteJSONFileCompact(path, map[string]any{"players": buf, "metadata": meta}); err != nil {
			return err
		}
		shard++
		buf = buf[:0]
		return nil
	}

	for rows.Next() {
		var e SearchEntry
		var className sql.NullString
		var gr sql.NullInt64
		var gb string
		if err := rows.Scan(&e.ID, &e.Name, &e.Region, &e.RealmSlug, &e.RealmName, &className, &gr, &gb); err != nil {
			return err
		}
		e.ClassName = className.String
		e.GlobalBracket = gb
		if gr.Valid {
			v := int(gr.Int64)
			e.GlobalRanking = &v
		}
		buf = append(buf, e)
		count++
		if len(buf) >= shardSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	fmt.Printf("[OK] Generated search index: %d players in %d shards\n", count, shard)
	return nil
}
