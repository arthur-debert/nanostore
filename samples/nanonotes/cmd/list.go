package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	showAll    bool
	pinnedOnly bool
	limit      int
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long: `List notes with various filters.
	
By default, shows only active (non-deleted) notes.

Examples:
  nanonotes list                # List active notes
  nanonotes list --all          # List all notes including deleted
  nanonotes list --pinned       # List only pinned notes
  nanonotes list --limit 10     # List only the 10 most recent notes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		var notes []Note

		if limit > 0 {
			// Get recent notes
			notes, err = app.GetRecentNotes(limit, showAll)
			if err != nil {
				return fmt.Errorf("failed to get recent notes: %w", err)
			}
		} else {
			// Get all notes with filters
			notes, err = app.ListNotes(showAll, pinnedOnly)
			if err != nil {
				return fmt.Errorf("failed to list notes: %w", err)
			}
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		// Display header
		fmt.Printf("%-8s %-40s %-10s %-7s %-19s\n", "ID", "TITLE", "STATUS", "PINNED", "UPDATED")
		fmt.Println(strings.Repeat("-", 88))

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

			fmt.Printf("%-8s %-40s %-10s %-7s %-19s\n",
				note.SimpleID,
				title,
				status,
				pinned,
				formatTime(note.UpdatedAt),
			)

			// Show body preview if available
			if note.Body != "" && cmd.Flag("verbose").Changed {
				body := strings.TrimSpace(note.Body)
				if len(body) > 60 {
					body = body[:60] + "..."
				}
				fmt.Printf("         %s\n", body)
			}
		}

		// Show count summary
		fmt.Printf("\nShowing %d note(s)\n", len(notes))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "show all notes including deleted")
	listCmd.Flags().BoolVarP(&pinnedOnly, "pinned", "p", false, "show only pinned notes")
	listCmd.Flags().IntVarP(&limit, "limit", "l", 0, "limit number of notes shown (shows most recent)")
	listCmd.Flags().BoolP("verbose", "v", false, "show body preview")
}
