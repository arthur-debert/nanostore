package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search notes by title or body",
	Long: `Search for notes containing the given text in title or body.

By default, searches only active notes. Use --all to include deleted notes.

Example:
  nanonotes search "meeting"
  nanonotes search "project" --all`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		query := strings.Join(args, " ")
		includeDeleted, _ := cmd.Flags().GetBool("all")

		notes, err := app.SearchNotes(query, includeDeleted)
		if err != nil {
			return fmt.Errorf("failed to search notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Printf("No notes found matching '%s'\n", query)
			return nil
		}

		// Display header
		fmt.Printf("%-8s %-40s %-10s %-7s\n", "ID", "TITLE", "STATUS", "PINNED")
		fmt.Println(strings.Repeat("-", 68))

		// Display notes
		for _, note := range notes {
			title := note.Title
			if len(title) > 37 {
				title = title[:37] + "..."
			}

			pinned := ""
			if note.Pinned {
				pinned = "📌 Yes"
			}

			status := note.Status
			if status == "deleted" {
				status = "🗑️  " + status
			}

			fmt.Printf("%-8s %-40s %-10s %-7s\n",
				note.SimpleID,
				title,
				status,
				pinned,
			)
		}

		fmt.Printf("\nFound %d note(s) matching '%s'\n", len(notes), query)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolP("all", "a", false, "include deleted notes in search")
}
