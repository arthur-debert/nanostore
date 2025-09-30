package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show note statistics",
	Long: `Display statistics about your notes including counts by status and pinned state.

Example:
  nanonotes stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getApp()
		if err != nil {
			return err
		}
		defer app.Close()

		total, active, deleted, pinned, err := app.CountNotes()
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}

		fmt.Println("📊 Note Statistics")
		fmt.Println("==================")
		fmt.Printf("Total notes:    %d\n", total)
		fmt.Printf("Active notes:   %d\n", active)
		fmt.Printf("Deleted notes:  %d\n", deleted)
		fmt.Printf("Pinned notes:   %d\n", pinned)

		if active > 0 {
			fmt.Printf("\nPinned ratio:   %.1f%%\n", float64(pinned)/float64(active)*100)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
