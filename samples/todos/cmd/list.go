package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	showAll        bool
	statusFilter   string
	priorityFilter string
	rootsOnly      bool
	searchText     string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List todo items",
	Long: `List todo items with various filtering options.

By default, shows the canonical view (active todos excluding completed ones).
Use --all to see all todos including completed ones.

Examples:
  todos list                    # Show active todos
  todos list --all             # Show all todos including completed
  todos list --status pending  # Show only pending todos
  todos list --priority high   # Show only high priority todos
  todos list --roots-only      # Show only root-level todos
  todos list --search "grocery" # Search for todos containing "grocery"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := NewTodoApp(storePath)
		if err != nil {
			return fmt.Errorf("failed to open todos: %w", err)
		}
		defer app.Close()

		var todos []TodoItem

		// Handle search
		if searchText != "" {
			todos, err = app.SearchTodos(searchText)
			if err != nil {
				return fmt.Errorf("failed to search todos: %w", err)
			}
			fmt.Printf("Search results for '%s':\n", searchText)
		} else if rootsOnly {
			todos, err = app.GetRootTodos()
			if err != nil {
				return fmt.Errorf("failed to get root todos: %w", err)
			}
			fmt.Println("Root todos:")
		} else if priorityFilter == "high" {
			todos, err = app.GetHighPriorityTodos()
			if err != nil {
				return fmt.Errorf("failed to get high priority todos: %w", err)
			}
			fmt.Println("High priority todos:")
		} else if showAll {
			todos, err = app.GetAllTodos()
			if err != nil {
				return fmt.Errorf("failed to get all todos: %w", err)
			}
			fmt.Println("All todos:")
		} else {
			todos, err = app.GetAllActiveTodos()
			if err != nil {
				return fmt.Errorf("failed to get active todos: %w", err)
			}
			fmt.Println("Active todos:")
		}

		// Apply additional filters
		if statusFilter != "" && searchText == "" && priorityFilter != "high" {
			filteredTodos := []TodoItem{}
			for _, todo := range todos {
				if todo.Status == statusFilter {
					filteredTodos = append(filteredTodos, todo)
				}
			}
			todos = filteredTodos
		}

		if priorityFilter != "" && priorityFilter != "high" {
			filteredTodos := []TodoItem{}
			for _, todo := range todos {
				if todo.Priority == priorityFilter {
					filteredTodos = append(filteredTodos, todo)
				}
			}
			todos = filteredTodos
		}

		if len(todos) == 0 {
			fmt.Println("  (no todos found)")
			return nil
		}

		// Print todos in tree format
		printTodoTree(todos, showAll)

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all todos including completed ones")
	listCmd.Flags().StringVarP(&statusFilter, "status", "", "", "Filter by status (pending, active, done)")
	listCmd.Flags().StringVarP(&priorityFilter, "priority", "", "", "Filter by priority (low, medium, high)")
	listCmd.Flags().BoolVarP(&rootsOnly, "roots-only", "r", false, "Show only root-level todos")
	listCmd.Flags().StringVarP(&searchText, "search", "", "", "Search todos by text")

	rootCmd.AddCommand(listCmd)
}

// printTodoTree prints todos in a hierarchical tree format
func printTodoTree(todos []TodoItem, showCompleted bool) {
	if len(todos) == 0 {
		fmt.Println("  (no todos)")
		return
	}

	// Group todos by parent
	roots := []TodoItem{}
	childrenMap := make(map[string][]TodoItem)

	for _, todo := range todos {
		if todo.ParentID == "" {
			roots = append(roots, todo)
		} else {
			childrenMap[todo.ParentID] = append(childrenMap[todo.ParentID], todo)
		}
	}

	// Sort roots by SimpleID
	sort.Slice(roots, func(i, j int) bool {
		return compareIDs(roots[i].SimpleID, roots[j].SimpleID)
	})

	// Print each root and its children
	for _, root := range roots {
		printTodoWithChildren(root, childrenMap, "", showCompleted)
	}
}

// printTodoWithChildren recursively prints a todo and its children
func printTodoWithChildren(todo TodoItem, childrenMap map[string][]TodoItem, indent string, showCompleted bool) {
	// Skip completed items in canonical view
	if !showCompleted && todo.Status == "done" {
		return
	}

	// Determine status icon
	icon := getStatusIcon(todo, childrenMap)

	// Build the todo line
	var line strings.Builder
	line.WriteString(fmt.Sprintf("  %s%s %s. %s", indent, icon, todo.SimpleID, todo.Title))

	// Add metadata if verbose
	if verbose {
		metadata := []string{}
		if todo.Priority != "medium" {
			metadata = append(metadata, fmt.Sprintf("priority:%s", todo.Priority))
		}
		if todo.Status != "pending" {
			metadata = append(metadata, fmt.Sprintf("status:%s", todo.Status))
		}
		if todo.AssignedTo != "" {
			metadata = append(metadata, fmt.Sprintf("assigned:%s", todo.AssignedTo))
		}
		if todo.Tags != "" {
			metadata = append(metadata, fmt.Sprintf("tags:%s", todo.Tags))
		}
		if len(metadata) > 0 {
			line.WriteString(fmt.Sprintf(" [%s]", strings.Join(metadata, ", ")))
		}
	}

	fmt.Println(line.String())

	// Get and sort children
	children := childrenMap[todo.UUID]
	sort.Slice(children, func(i, j int) bool {
		return compareIDs(children[i].SimpleID, children[j].SimpleID)
	})

	// Print children with increased indentation
	for _, child := range children {
		printTodoWithChildren(child, childrenMap, indent+"  ", showCompleted)
	}
}

// getStatusIcon returns the appropriate icon for a todo's status
func getStatusIcon(todo TodoItem, childrenMap map[string][]TodoItem) string {
	switch todo.Status {
	case "done":
		return "●"
	case "active":
		return "◐"
	case "pending":
		// Check if it has children in mixed states
		children := childrenMap[todo.UUID]
		if len(children) > 0 {
			hasCompleted := false
			hasPending := false
			for _, child := range children {
				if child.Status == "done" {
					hasCompleted = true
				} else {
					hasPending = true
				}
			}
			if hasCompleted && hasPending {
				return "◐" // Mixed state
			}
		}
		return "○"
	default:
		return "○"
	}
}

// compareIDs compares two SimpleIDs for sorting
func compareIDs(a, b string) bool {
	return a < b
}
