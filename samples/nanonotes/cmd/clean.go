package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var forceClean bool

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Permanently delete all soft-deleted notes",
	Long: `Clean removes all notes marked as deleted from the database.
	
This is a permanent operation and cannot be undone.
Use --force flag to skip confirmation.

Example:
  nanonotes clean
  nanonotes clean --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		// Get count of deleted notes first
		_, _, deleted, _, err := app.CountNotes()
		if err != nil {
			return fmt.Errorf("failed to count notes: %w", err)
		}

		if deleted == 0 {
			fmt.Println("No deleted notes to clean.")
			return nil
		}

		// Confirm unless force flag is set
		if !forceClean {
			fmt.Printf("This will permanently delete %d note(s). Continue? [y/N]: ", deleted)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Clean cancelled.")
				return nil
			}
		}

		count, err := app.Clean()
		if err != nil {
			return fmt.Errorf("failed to clean notes: %w", err)
		}

		fmt.Printf("Successfully cleaned %d deleted note(s)\n", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVar(&forceClean, "force", false, "skip confirmation prompt")
}
