package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	_ "github.com/tursodatabase/go-libsql"
	"ookstats/internal/database"
	"ookstats/internal/generator"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate static artifacts",
	Long:  `Generate static API endpoints and other build artifacts from the local database.`,
}

var generateAPICmd = &cobra.Command{
	Use:   "api",
	Short: "Generate static JSON API endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		onlyPlayers, _ := cmd.Flags().GetBool("players")
		doLeaderboards, _ := cmd.Flags().GetBool("leaderboards")
		doSearch, _ := cmd.Flags().GetBool("search")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		shardSize, _ := cmd.Flags().GetInt("shard-size")
		regionsCSV, _ := cmd.Flags().GetString("regions")

		if strings.TrimSpace(outDir) == "" {
			return errors.New("--out is required")
		}
		// Connect to local DB (file:)
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}
		defer db.Close()

		base := filepath.Join(outDir, "api")
		if err := os.MkdirAll(base, 0o755); err != nil {
			return fmt.Errorf("mkdir base: %w", err)
		}

		if onlyPlayers {
			if err := generator.GeneratePlayers(db, filepath.Join(base, "player"), ""); err != nil {
				return err
			}
		}

		if doLeaderboards {
			regions := []string{}
			if strings.TrimSpace(regionsCSV) != "" {
				for _, r := range strings.Split(regionsCSV, ",") {
					rr := strings.TrimSpace(r)
					if rr != "" {
						regions = append(regions, rr)
					}
				}
			}
			if err := generator.GenerateLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil {
				return err
			}
			if err := generator.GeneratePlayerLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions); err != nil {
				return err
			}
		}

		if doSearch {
			if err := generator.GenerateSearchIndex(db, filepath.Join(base, "search"), shardSize); err != nil {
				return err
			}
		}

		fmt.Printf("\nStatic API generated at %s\n", base)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateAPICmd)
	generateAPICmd.Flags().String("out", "public", "Output directory for static API")
	generateAPICmd.Flags().Bool("players", true, "Generate player profile JSON endpoints")
	generateAPICmd.Flags().Bool("leaderboards", true, "Generate leaderboard JSON endpoints")
	generateAPICmd.Flags().Bool("search", true, "Generate search index JSON shards")
	generateAPICmd.Flags().Int("page-size", 25, "Leaderboard page size")
	generateAPICmd.Flags().Int("shard-size", 5000, "Search index shard size")
	generateAPICmd.Flags().String("regions", "us,eu,kr,tw", "Regions to include for regional leaderboards")
}
