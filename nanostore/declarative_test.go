package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TodoItem represents a todo item with hierarchical support
type TodoItem struct {
	nanostore.Document

	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`
}

func TestDeclarativeAPI(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := nanostore.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("CreateAndRetrieve", func(t *testing.T) {
		// Create a todo
		id, err := store.Create("Buy groceries", &TodoItem{
			Priority: "high",
		})
		if err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}

		// Retrieve it
		todo, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get todo: %v", err)
		}

		// Check values
		if todo.Title != "Buy groceries" {
			t.Errorf("expected title 'Buy groceries', got %s", todo.Title)
		}
		if todo.Status != "pending" {
			t.Errorf("expected default status 'pending', got %s", todo.Status)
		}
		if todo.Priority != "high" {
			t.Errorf("expected priority 'high', got %s", todo.Priority)
		}
		if todo.Activity != "active" {
			t.Errorf("expected default activity 'active', got %s", todo.Activity)
		}
	})

	t.Run("HierarchicalTodos", func(t *testing.T) {
		// Create parent
		parentID, err := store.Create("Shopping", &TodoItem{})
		if err != nil {
			t.Fatalf("failed to create parent: %v", err)
		}

		// Create children
		_, err = store.Create("Milk", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 1: %v", err)
		}

		_, err = store.Create("Bread", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 2: %v", err)
		}

		// Query children
		children, err := store.Query().
			ParentID(parentID).
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query children: %v", err)
		}

		if len(children) != 2 {
			t.Errorf("expected 2 children, got %d", len(children))
		}

		// Query root todos (no parent)
		roots, err := store.Query().
			ParentIDNotExists().
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query roots: %v", err)
		}

		// Should have at least the original todos plus "Shopping"
		hasShoppingRoot := false
		for _, todo := range roots {
			if todo.Title == "Shopping" {
				hasShoppingRoot = true
				break
			}
		}
		if !hasShoppingRoot {
			t.Error("expected to find 'Shopping' in root todos")
		}
	})

	t.Run("QueryBuilder", func(t *testing.T) {
		// Create test data
		_, _ = store.Create("Task 1", &TodoItem{
			Status:   "active",
			Priority: "high",
		})
		_, _ = store.Create("Task 2", &TodoItem{
			Status:   "done",
			Priority: "low",
		})
		_, _ = store.Create("Task 3", &TodoItem{
			Status:   "pending",
			Priority: "high",
		})

		// Query high priority items
		highPriorityTodos, err := store.Query().
			Priority("high").
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query high priority: %v", err)
		}

		// Should have at least 2 high priority items
		if len(highPriorityTodos) < 2 {
			t.Errorf("expected at least 2 high priority todos, got %d", len(highPriorityTodos))
		}

		// Query with status filter
		activeTodos, err := store.Query().
			Status("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query active todos: %v", err)
		}

		if len(activeTodos) < 1 {
			t.Errorf("expected at least 1 active todo, got %d", len(activeTodos))
		}

		// Test Exists
		hasHighPriority, err := store.Query().
			Priority("high").
			Exists()
		if err != nil {
			t.Fatalf("failed to check existence: %v", err)
		}
		if !hasHighPriority {
			t.Error("expected high priority todos to exist")
		}
	})

	t.Run("UpdateWithTypes", func(t *testing.T) {
		// Create a todo
		id, err := store.Create("Update test", &TodoItem{
			Status: "pending",
		})
		if err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}

		// Update it
		err = store.Update(id, &TodoItem{
			Status:   "done",
			Priority: "high",
		})
		if err != nil {
			t.Fatalf("failed to update todo: %v", err)
		}

		// Verify update
		updated, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get updated todo: %v", err)
		}

		if updated.Status != "done" {
			t.Errorf("expected status 'done', got %s", updated.Status)
		}
		if updated.Priority != "high" {
			t.Errorf("expected priority 'high', got %s", updated.Priority)
		}
	})

	t.Run("SearchFunctionality", func(t *testing.T) {
		// Create searchable todos
		_, _ = store.Create("Pack for trip", &TodoItem{})
		_, _ = store.Create("Pack lunch", &TodoItem{})
		_, _ = store.Create("Buy tickets", &TodoItem{})

		// Search for "pack"
		results, err := store.Query().
			Search("pack").
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results for 'pack', got %d", len(results))
		}
	})
}

