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
	"ookstats/internal/generator/indexes"
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
		doIndexes, _ := cmd.Flags().GetBool("indexes")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		shardSize, _ := cmd.Flags().GetInt("shard-size")
		regionsCSV, _ := cmd.Flags().GetString("regions")
		workers, _ := cmd.Flags().GetInt("workers")

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
			if err := generator.GenerateLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions, workers); err != nil {
				return err
			}
			if err := generator.GeneratePlayerLeaderboards(db, filepath.Join(base, "leaderboard"), pageSize, regions, workers); err != nil {
				return err
			}
		}

		if doSearch {
			if err := generator.GenerateSearchIndex(db, filepath.Join(base, "search"), shardSize); err != nil {
				return err
			}
		}

		if doIndexes {
			if err := indexes.GenerateAllIndexes(db, outDir); err != nil {
				return err
			}
		}

		fmt.Printf("\nStatic API generated at %s\n", base)
		return nil
	},
}

var generateHomeCmd = &cobra.Command{
	Use:   "home",
	Short: "Generate home page JSON (top runs + top players + recent feed)",
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		if strings.TrimSpace(outDir) == "" {
			return errors.New("--out is required")
		}
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}
		defer db.Close()

		if err := os.MkdirAll(filepath.Join(outDir, "api"), 0o755); err != nil {
			return fmt.Errorf("mkdir base: %w", err)
		}

		if err := generator.GenerateHome(db, outDir); err != nil {
			return err
		}
		fmt.Printf("\nHome JSON generated at %s/api/home.json\n", outDir)
		return nil
	},
}

var generateStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Generate stats page JSON (totals, completion tiers, spec counts, top runners, weekly activity)",
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		if strings.TrimSpace(outDir) == "" {
			return errors.New("--out is required")
		}
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}
		defer db.Close()

		if err := os.MkdirAll(filepath.Join(outDir, "api"), 0o755); err != nil {
			return fmt.Errorf("mkdir base: %w", err)
		}

		if err := generator.GenerateStats(db, outDir); err != nil {
			return err
		}
		fmt.Printf("\nStats JSON generated at %s/api/stats.json\n", outDir)
		return nil
	},
}

var generateGearCmd = &cobra.Command{
	Use:   "gear",
	Short: "Generate gear popularity JSON (per-spec top items per slot, validated by spec primary stat)",
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		if strings.TrimSpace(outDir) == "" {
			return errors.New("--out is required")
		}
		db, err := database.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}
		defer db.Close()

		if err := os.MkdirAll(filepath.Join(outDir, "api"), 0o755); err != nil {
			return fmt.Errorf("mkdir base: %w", err)
		}

		if err := generator.GenerateGear(db, outDir); err != nil {
			return err
		}
		fmt.Printf("\nGear JSON generated at %s/api/gear.json\n", outDir)
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
	generateAPICmd.Flags().Bool("indexes", true, "Generate API discovery indexes")
	generateAPICmd.Flags().Int("page-size", 25, "Leaderboard page size")
	generateAPICmd.Flags().Int("shard-size", 5000, "Search index shard size")
	generateAPICmd.Flags().String("regions", "us,eu,kr,tw", "Regions to include for regional leaderboards")
	generateAPICmd.Flags().Int("workers", 10, "Number of parallel workers for leaderboard generation")

	generateCmd.AddCommand(generateHomeCmd)
	generateHomeCmd.Flags().String("out", "web/public", "Output directory (api/home.json will be written under it)")

	generateCmd.AddCommand(generateStatsCmd)
	generateStatsCmd.Flags().String("out", "web/public", "Output directory (api/stats.json will be written under it)")

	generateCmd.AddCommand(generateGearCmd)
	generateGearCmd.Flags().String("out", "web/public", "Output directory (api/gear.json will be written under it)")
}
