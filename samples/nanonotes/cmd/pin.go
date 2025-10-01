package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pinCmd represents the pin command
var pinCmd = &cobra.Command{
	Use:   "pin <id>",
	Short: "Pin a note",
	Long: `Pin a note to mark it as important.
	
Pinned notes can be easily filtered using 'list --pinned'.

Example:
  nanonotes pin 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		id := args[0]

		// Check if note exists
		note, err := app.GetNote(id)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if note.Pinned {
			return fmt.Errorf("note %s is already pinned", id)
		}

		err = app.PinNote(id)
		if err != nil {
			return fmt.Errorf("failed to pin note: %w", err)
		}

		fmt.Printf("Note %s pinned successfully\n", id)
		return nil
	},
}

// unpinCmd represents the unpin command
var unpinCmd = &cobra.Command{
	Use:   "unpin <id>",
	Short: "Unpin a note",
	Long: `Unpin a previously pinned note.

Example:
  nanonotes unpin 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		id := args[0]

		// Check if note exists
		note, err := app.GetNote(id)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if !note.Pinned {
			return fmt.Errorf("note %s is not pinned", id)
		}

		err = app.UnpinNote(id)
		if err != nil {
			return fmt.Errorf("failed to unpin note: %w", err)
		}

		fmt.Printf("Note %s unpinned successfully\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)
}
