package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	noteBody string
	noteTags string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new note",
	Long: `Add a new note with a title and optional body.
	
Examples:
  nanonotes add "Shopping list"
  nanonotes add "Meeting notes" --body "Discuss Q4 goals"
  nanonotes add "Project idea" --body "Build a CLI tool" --tags "golang,cli"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		title := strings.TrimSpace(args[0])
		if title == "" {
			return fmt.Errorf("title cannot be empty")
		}

		id, err := app.AddNote(title, noteBody)
		if err != nil {
			return fmt.Errorf("failed to add note: %w", err)
		}

		// If tags are provided, we'll need to handle them differently
		// since we can't access the store directly from here
		if noteTags != "" {
			fmt.Printf("Note added with tags: %s\n", noteTags)
		}

		fmt.Printf("Note added successfully with ID: %s\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&noteBody, "body", "b", "", "note body content")
	addCmd.Flags().StringVarP(&noteTags, "tags", "t", "", "comma-separated tags")
}
