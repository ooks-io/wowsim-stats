package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"ookstats/internal/playerid"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage curator-supplied account overrides",
	Long: `Read/write the manual account-link file consumed by the
account-grouping pipeline. Edits take effect on the next
'ookstats process accounts' run.`,
}

var accountsLinkCmd = &cobra.Command{
	Use:   "link <region/realm/name> <region/realm/name>",
	Short: "Declare two characters share an account",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("file")
		note, _ := cmd.Flags().GetString("note")

		a, err := parseCharRef(args[0])
		if err != nil {
			return err
		}
		b, err := parseCharRef(args[1])
		if err != nil {
			return err
		}

		links, err := playerid.LoadManualLinks(path)
		if err != nil {
			return err
		}
		newLink := playerid.ManualLink{A: a, B: b, Note: note}
		for _, existing := range links {
			if playerid.SameLink(existing, newLink) {
				fmt.Printf("Already linked: %s <-> %s\n", a, b)
				return nil
			}
		}
		links = append(links, newLink)
		if err := playerid.WriteManualLinks(path, links); err != nil {
			return err
		}
		fmt.Printf("Linked %s <-> %s\n", a, b)
		fmt.Printf("Run 'ookstats process accounts' to apply.\n")
		return nil
	},
}

var accountsUnlinkCmd = &cobra.Command{
	Use:   "unlink <region/realm/name> <region/realm/name>",
	Short: "Remove a manual account link",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("file")

		a, err := parseCharRef(args[0])
		if err != nil {
			return err
		}
		b, err := parseCharRef(args[1])
		if err != nil {
			return err
		}

		links, err := playerid.LoadManualLinks(path)
		if err != nil {
			return err
		}
		target := playerid.ManualLink{A: a, B: b}
		filtered := links[:0]
		removed := false
		for _, l := range links {
			if playerid.SameLink(l, target) {
				removed = true
				continue
			}
			filtered = append(filtered, l)
		}
		if !removed {
			fmt.Printf("No matching link: %s <-> %s\n", a, b)
			return nil
		}
		if err := playerid.WriteManualLinks(path, filtered); err != nil {
			return err
		}
		fmt.Printf("Unlinked %s <-> %s\n", a, b)
		fmt.Printf("Run 'ookstats process accounts' to apply.\n")
		return nil
	},
}

var accountsListLinksCmd = &cobra.Command{
	Use:   "list-links",
	Short: "Print all manual account links",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("file")
		links, err := playerid.LoadManualLinks(path)
		if err != nil {
			return err
		}
		if len(links) == 0 {
			fmt.Printf("No manual links configured (%s)\n", path)
			return nil
		}
		fmt.Printf("%d manual link(s) in %s:\n", len(links), path)
		for _, l := range links {
			line := fmt.Sprintf("  %s  <->  %s", l.A, l.B)
			if l.Note != "" {
				line += "  # " + l.Note
			}
			fmt.Println(line)
		}
		return nil
	},
}

func parseCharRef(s string) (playerid.CharRef, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return playerid.CharRef{}, fmt.Errorf("expected region/realm/name, got %q", s)
	}
	return playerid.CharRef{Region: parts[0], Realm: parts[1], Name: parts[2]}, nil
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.PersistentFlags().String("file", "account-manual-links.json", "Path to manual links JSON")

	accountsCmd.AddCommand(accountsLinkCmd)
	accountsLinkCmd.Flags().String("note", "", "Optional human-readable reason for the link")

	accountsCmd.AddCommand(accountsUnlinkCmd)
	accountsCmd.AddCommand(accountsListLinksCmd)
}
