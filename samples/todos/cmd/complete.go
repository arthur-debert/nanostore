package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a todo as completed",
	Long: `Mark a todo item as completed (done status).

The todo's status will change to 'done' and its ID will be updated
to include the completion prefix if configured.

Examples:
  todos complete 1      # Complete todo with ID 1
  todos complete 1.2    # Complete subtask 1.2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		todoID := args[0]

		app, err := NewTodoApp(storePath)
		if err != nil {
			return fmt.Errorf("failed to open todos: %w", err)
		}
		defer app.Close()

		// Get the todo to update
		uuid, err := app.store.Store().ResolveUUID(todoID)
		if err != nil {
			return fmt.Errorf("todo not found: %s", todoID)
		}

		todo, err := app.GetTodo(uuid)
		if err != nil {
			return fmt.Errorf("failed to get todo: %w", err)
		}

		if todo.Status == "done" {
			fmt.Printf("Todo %s. %s is already completed\n", todo.SimpleID, todo.Title)
			return nil
		}

		// Update status to done
		todo.Status = "done"
		err = app.UpdateTodo(uuid, todo)
		if err != nil {
			return fmt.Errorf("failed to complete todo: %w", err)
		}

		// Get the updated todo to show new ID
		updatedTodo, err := app.GetTodo(uuid)
		if err != nil {
			return fmt.Errorf("failed to get updated todo: %w", err)
		}

		fmt.Printf("✅ Completed: %s. %s\n", updatedTodo.SimpleID, updatedTodo.Title)
		if verbose {
			fmt.Printf("   Status changed from '%s' to 'done'\n", todo.Status)
			if updatedTodo.SimpleID != todoID {
				fmt.Printf("   ID updated: %s → %s\n", todoID, updatedTodo.SimpleID)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(completeCmd)
}
