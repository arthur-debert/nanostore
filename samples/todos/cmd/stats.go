package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show todo statistics",
	Long: `Display statistics about your todos including counts by status,
priority, and other metrics.

Examples:
  todos stats           # Show all statistics
  todos stats --verbose # Show detailed breakdown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := NewTodoApp(storePath)
		if err != nil {
			return fmt.Errorf("failed to open todos: %w", err)
		}
		defer app.Close()

		stats, err := app.GetStatistics()
		if err != nil {
			return fmt.Errorf("failed to get statistics: %w", err)
		}

		fmt.Println("📊 Todo Statistics:")
		fmt.Println()

		// Basic counts
		fmt.Printf("Total active todos:   %d\n", stats["total"])
		fmt.Printf("Pending todos:        %d\n", stats["pending"])
		fmt.Printf("Active todos:         %d\n", stats["active"])
		fmt.Printf("Completed todos:      %d\n", stats["done"])
		fmt.Printf("High priority todos:  %d\n", stats["high_priority"])

		if verbose {
			fmt.Println()
			fmt.Println("📋 Detailed Breakdown:")

			// Get root vs subtask breakdown
			rootTodos, err := app.GetRootTodos()
			if err == nil {
				fmt.Printf("Root todos:           %d\n", len(rootTodos))

				// Count subtasks
				allTodos, err := app.GetAllTodos()
				if err == nil {
					subtaskCount := 0
					for _, todo := range allTodos {
						if todo.ParentID != "" {
							subtaskCount++
						}
					}
					fmt.Printf("Subtasks:             %d\n", subtaskCount)
				}
			}

			// Show completion rate
			if stats["total"] > 0 {
				completionRate := float64(stats["done"]) / float64(stats["total"]+stats["done"]) * 100
				fmt.Printf("Completion rate:      %.1f%%\n", completionRate)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
