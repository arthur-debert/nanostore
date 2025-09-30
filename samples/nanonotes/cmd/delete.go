package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a note (soft delete)",
	Long: `Delete a note by marking it as deleted.
	
This is a soft delete - the note is marked as deleted but not removed from the database.
Use 'clean' command to permanently remove deleted notes.

Example:
  nanonotes delete 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		id := args[0]

		// Check if note exists and is not already deleted
		note, err := app.GetNote(id)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if note.Status == "deleted" {
			return fmt.Errorf("note %s is already deleted", id)
		}

		err = app.DeleteNote(id)
		if err != nil {
			return fmt.Errorf("failed to delete note: %w", err)
		}

		fmt.Printf("Note %s marked as deleted\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
