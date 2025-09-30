package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	parentID    string
	priority    string
	description string
	assignedTo  string
	tags        string
	dueDate     string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new todo item",
	Long: `Add a new todo item to the store.

The todo will be created with default status 'pending' and priority 'medium'
unless specified otherwise. You can create subtasks by specifying a parent ID.

Examples:
  todos add "Buy groceries"
  todos add --parent 1 "Milk"
  todos add --priority high "Important task"
  todos add --description "Weekly shopping" "Groceries"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		app, err := NewTodoApp(storePath)
		if err != nil {
			return fmt.Errorf("failed to open todos: %w", err)
		}
		defer app.Close()

		// Validate due date if provided
		if dueDate != "" {
			_, err := time.Parse("2006-01-02", dueDate)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
		}

		// Create the todo item
		todo := &TodoItem{
			Description: description,
			AssignedTo:  assignedTo,
			Tags:        tags,
			DueDate:     dueDate,
		}

		// Set optional fields
		if parentID != "" {
			// Resolve parent ID to UUID
			parentUUID, err := app.store.Store().ResolveUUID(parentID)
			if err != nil {
				return fmt.Errorf("invalid parent ID %s: %w", parentID, err)
			}
			todo.ParentID = parentUUID
		}

		if priority != "" {
			todo.Priority = priority
		}

		// Create the todo
		id, err := app.CreateTodo(title, todo)
		if err != nil {
			return fmt.Errorf("failed to create todo: %w", err)
		}

		// Get the created todo to show the generated ID
		createdTodo, err := app.GetTodo(id)
		if err != nil {
			return fmt.Errorf("failed to retrieve created todo: %w", err)
		}

		fmt.Printf("✅ Todo created: %s. %s\n", createdTodo.SimpleID, createdTodo.Title)
		if verbose {
			fmt.Printf("   UUID: %s\n", createdTodo.UUID)
			if createdTodo.ParentID != "" {
				fmt.Printf("   Parent: %s\n", parentID)
			}
			fmt.Printf("   Priority: %s\n", createdTodo.Priority)
			fmt.Printf("   Status: %s\n", createdTodo.Status)
		}

		return nil
	},
}

func init() {
	addCmd.Flags().StringP("parent", "p", "", "Parent todo ID for creating subtasks")
	addCmd.Flags().StringVar(&priority, "priority", "", "Priority level (low, medium, high)")
	addCmd.Flags().StringP("description", "d", "", "Detailed description")
	addCmd.Flags().StringP("assigned-to", "a", "", "Person assigned to this todo")
	addCmd.Flags().StringP("tags", "t", "", "Comma-separated tags")
	addCmd.Flags().StringVar(&dueDate, "due", "", "Due date (YYYY-MM-DD format)")

	rootCmd.AddCommand(addCmd)
}
