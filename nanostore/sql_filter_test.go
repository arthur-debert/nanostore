package nanostore

import (
	"testing"
)

// TestSQLLevelFiltering tests that NOT, EXISTS, and NOT_EXISTS filters are
// properly pushed to the SQL layer instead of being filtered client-side
func TestSQLLevelFiltering(t *testing.T) {
	// Create store with TodoItem for testing
	store, err := NewFromType[TodoItem](":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Create test data with various statuses and parent relationships
	// This gives us documents to test NOT, EXISTS, and NOT_EXISTS filters

	// Root todos
	_, err = store.Create("Root Todo 1", &TodoItem{
		Status:   "pending",
		Priority: "medium",
	})
	if err != nil {
		t.Fatalf("Failed to create root todo 1: %v", err)
	}

	rootTodo2ID, err := store.Create("Root Todo 2", &TodoItem{
		Status:   "done",
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("Failed to create root todo 2: %v", err)
	}

	// Child todos
	_, err = store.Create("Child Todo 1", &TodoItem{
		Status:   "active",
		Priority: "low",
		ParentID: rootTodo2ID,
	})
	if err != nil {
		t.Fatalf("Failed to create child todo 1: %v", err)
	}

	_, err = store.Create("Child Todo 2", &TodoItem{
		Status:   "pending",
		Priority: "medium",
		ParentID: rootTodo2ID,
	})
	if err != nil {
		t.Fatalf("Failed to create child todo 2: %v", err)
	}

	// Create a todo with empty parent_id (should be NULL in DB)
	_, err = store.Create("Orphan Todo", &TodoItem{
		Status:   "active",
		Priority: "high",
		ParentID: "", // This should be NULL in the database
	})
	if err != nil {
		t.Fatalf("Failed to create orphan todo: %v", err)
	}

	t.Run("StatusNot filter", func(t *testing.T) {
		// Test StatusNot - should exclude documents with the specified status
		results, err := store.Query().
			StatusNot("pending").
			Find()
		if err != nil {
			t.Fatalf("StatusNot query failed: %v", err)
		}

		// Should get 3 results: root todo 2 (done), child todo 1 (active), orphan todo (active)
		if len(results) != 3 {
			t.Errorf("Expected 3 results for StatusNot('pending'), got %d", len(results))
		}

		// Verify none of the results have status "pending"
		for _, todo := range results {
			if todo.Status == "pending" {
				t.Errorf("Found todo with status 'pending' in StatusNot('pending') results: %v", todo.Title)
			}
		}
	})

	t.Run("ParentIDExists filter", func(t *testing.T) {
		// Test ParentIDExists - should only get documents that have a parent
		results, err := store.Query().
			ParentIDExists().
			Find()
		if err != nil {
			t.Fatalf("ParentIDExists query failed: %v", err)
		}

		// Should get 2 results: child todo 1 and child todo 2
		if len(results) != 2 {
			t.Errorf("Expected 2 results for ParentIDExists(), got %d", len(results))
		}

		// Verify all results have a non-empty parent_id
		for _, todo := range results {
			if todo.ParentID == "" {
				t.Errorf("Found todo without parent in ParentIDExists() results: %v", todo.Title)
			}
		}
	})

	t.Run("ParentIDNotExists filter", func(t *testing.T) {
		// Test ParentIDNotExists - should only get documents that don't have a parent
		results, err := store.Query().
			ParentIDNotExists().
			Find()
		if err != nil {
			t.Fatalf("ParentIDNotExists query failed: %v", err)
		}

		// Should get 3 results: root todo 1, root todo 2, orphan todo
		if len(results) != 3 {
			t.Errorf("Expected 3 results for ParentIDNotExists(), got %d", len(results))
		}

		// Verify all results have empty or null parent_id
		for _, todo := range results {
			if todo.ParentID != "" {
				t.Errorf("Found todo with parent in ParentIDNotExists() results: %v (parent: %s)", todo.Title, todo.ParentID)
			}
		}
	})

	t.Run("Combined filters", func(t *testing.T) {
		// Test combining NOT with other filters
		results, err := store.Query().
			StatusNot("done").
			ParentIDExists().
			Find()
		if err != nil {
			t.Fatalf("Combined query failed: %v", err)
		}

		// Should get 2 results: child todo 1 (active) and child todo 2 (pending)
		if len(results) != 2 {
			t.Errorf("Expected 2 results for combined query, got %d", len(results))
		}

		// Verify results match criteria: not done AND have parent
		for _, todo := range results {
			if todo.Status == "done" {
				t.Errorf("Found todo with status 'done' in StatusNot('done') results: %v", todo.Title)
			}
			if todo.ParentID == "" {
				t.Errorf("Found todo without parent in ParentIDExists() results: %v", todo.Title)
			}
		}
	})

	t.Run("Pagination with filters", func(t *testing.T) {
		// Test that pagination works correctly with SQL-level filtering
		// This was one of the main issues mentioned in the GitHub issue

		// Get first 2 non-pending todos
		results, err := store.Query().
			StatusNot("pending").
			Limit(2).
			Find()
		if err != nil {
			t.Fatalf("Pagination query failed: %v", err)
		}

		// Should get exactly 2 results
		if len(results) != 2 {
			t.Errorf("Expected 2 results for paginated StatusNot('pending'), got %d", len(results))
		}

		// Verify none have status "pending"
		for _, todo := range results {
			if todo.Status == "pending" {
				t.Errorf("Found todo with status 'pending' in paginated StatusNot('pending') results: %v", todo.Title)
			}
		}

		// Test with offset
		results, err = store.Query().
			StatusNot("pending").
			Limit(1).
			Offset(1).
			Find()
		if err != nil {
			t.Fatalf("Offset pagination query failed: %v", err)
		}

		// Should get exactly 1 result (the second one)
		if len(results) != 1 {
			t.Errorf("Expected 1 result for offset pagination StatusNot('pending'), got %d", len(results))
		}
	})
}

// TodoItem for testing - reusing the type from other tests
type TodoItem struct {
	Document
	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Activity string `values:"active,archived,deleted" default:"active"`
	ParentID string `dimension:"parent_id,ref"`
}
