package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestTypedQueryWhereClause(t *testing.T) {
	// Create a temporary file for store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with known timestamps for WHERE clause testing
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	uuid1, err := store.Create("Recent Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.Create("Old Task", &TodoItem{
		Status:   "pending",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.Create("Important Task", &TodoItem{
		Status:   "done",
		Priority: "high",
		Activity: "archived",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create task with custom data for WHERE clause testing
	uuid4, err := store.AddRaw("Custom Task", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.estimate": 5,
		"_data.urgent":   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WhereWithSimpleCondition", func(t *testing.T) {
		// Test WHERE clause with simple condition
		// Note: This is testing the method signature and structure,
		// not actual SQL execution since that's not implemented yet
		tasks, err := store.Query().
			Where("status = ?", "active").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause query: %v", err)
		}

		// Since WHERE clause is currently a no-op (not actually filtering),
		// this should return all tasks regardless of the WHERE condition
		t.Logf("Found %d tasks with WHERE clause (expected behavior: no-op until SQL evaluation implemented)", len(tasks))

		// Verify we get some results (exact count depends on implementation)
		if len(tasks) == 0 {
			t.Error("expected some tasks, but got none")
		}
	})

	t.Run("WhereWithMultipleParameters", func(t *testing.T) {
		// Test WHERE clause with multiple parameters
		tasks, err := store.Query().
			Where("status = ? AND priority = ?", "active", "high").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with multiple params: %v", err)
		}

		t.Logf("Found %d tasks with multi-parameter WHERE clause", len(tasks))

		// Verify query structure was accepted
		if len(tasks) == 0 {
			t.Error("expected some tasks, but got none")
		}
	})

	t.Run("WhereWithTimeComparison", func(t *testing.T) {
		// Test WHERE clause with time comparison
		yesterday := baseTime.AddDate(0, 0, -1)
		tasks, err := store.Query().
			Where("created_at > ?", yesterday).
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with time comparison: %v", err)
		}

		t.Logf("Found %d tasks created after %s", len(tasks), yesterday)
	})

	t.Run("WhereWithDataFieldCondition", func(t *testing.T) {
		// Test WHERE clause with custom data field
		tasks, err := store.Query().
			Where("_data.assignee = ?", "alice").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with data field: %v", err)
		}

		t.Logf("Found %d tasks assigned to alice via WHERE clause", len(tasks))
	})

	t.Run("WhereWithComplexCondition", func(t *testing.T) {
		// Test WHERE clause with complex SQL
		tasks, err := store.Query().
			Where("(status = ? OR priority = ?) AND _data.urgent = ?", "active", "high", true).
			Find()
		if err != nil {
			t.Fatalf("failed to execute complex WHERE clause: %v", err)
		}

		t.Logf("Found %d tasks matching complex WHERE condition", len(tasks))
	})

	t.Run("WhereCombinedWithOtherFilters", func(t *testing.T) {
		// Test WHERE clause combined with typed filters
		tasks, err := store.Query().
			Status("active").
			Priority("high").
			Where("created_at > ?", baseTime.AddDate(0, 0, -7)).
			Find()
		if err != nil {
			t.Fatalf("failed to combine WHERE with other filters: %v", err)
		}

		t.Logf("Found %d tasks combining typed filters with WHERE clause", len(tasks))

		// Should respect the Status and Priority filters even if WHERE is no-op
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
			// Verify typed filters are working
			if task.Status != "active" {
				t.Errorf("expected active status, got %s", task.Status)
			}
			if task.Priority != "high" {
				t.Errorf("expected high priority, got %s", task.Priority)
			}
		}
	})

	t.Run("WhereWithStringLike", func(t *testing.T) {
		// Test WHERE clause with LIKE operation
		tasks, err := store.Query().
			Where("LOWER(title) LIKE ?", "%task%").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with LIKE: %v", err)
		}

		t.Logf("Found %d tasks with title containing 'task'", len(tasks))
	})

	t.Run("WhereWithNoParameters", func(t *testing.T) {
		// Test WHERE clause without parameters
		tasks, err := store.Query().
			Where("status IS NOT NULL").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause without params: %v", err)
		}

		t.Logf("Found %d tasks with non-null status", len(tasks))
	})

	t.Run("WhereWithNumericComparison", func(t *testing.T) {
		// Test WHERE clause with numeric data field
		tasks, err := store.Query().
			Where("_data.estimate > ?", 3).
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with numeric comparison: %v", err)
		}

		t.Logf("Found %d tasks with estimate > 3", len(tasks))
	})

	t.Run("MultipleWhereClauses", func(t *testing.T) {
		// Test behavior with multiple WHERE clauses (should use last one)
		tasks, err := store.Query().
			Where("status = ?", "active").
			Where("priority = ?", "high").
			Find()
		if err != nil {
			t.Fatalf("failed to execute multiple WHERE clauses: %v", err)
		}

		t.Logf("Found %d tasks with multiple WHERE clauses (last one should win)", len(tasks))
	})

	// Verify all test UUIDs are accessible for reference
	_ = uuid1
	_ = uuid2
	_ = uuid3
	_ = uuid4
}

func TestTypedQueryWhereEdgeCases(t *testing.T) {
	// Create a temporary file for store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create minimal test data
	_, err = store.Create("Test Task", &TodoItem{
		Status:   "active",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WhereWithEmptyClause", func(t *testing.T) {
		// Test WHERE clause with empty string
		tasks, err := store.Query().
			Where("").
			Find()
		if err != nil {
			t.Fatalf("failed to execute empty WHERE clause: %v", err)
		}

		t.Logf("Found %d tasks with empty WHERE clause", len(tasks))
	})

	t.Run("WhereWithNilArgs", func(t *testing.T) {
		// Test WHERE clause with nil arguments
		tasks, err := store.Query().
			Where("status IS NOT NULL", nil).
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with nil arg: %v", err)
		}

		t.Logf("Found %d tasks with nil WHERE argument", len(tasks))
	})

	t.Run("WhereWithNoArgs", func(t *testing.T) {
		// Test WHERE clause with no arguments at all
		tasks, err := store.Query().
			Where("1 = 1").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with no args: %v", err)
		}

		t.Logf("Found %d tasks with argumentless WHERE clause", len(tasks))
	})

	t.Run("WhereWithSpecialCharacters", func(t *testing.T) {
		// Test WHERE clause with special SQL characters
		tasks, err := store.Query().
			Where("title NOT LIKE '%[special]%'").
			Find()
		if err != nil {
			t.Fatalf("failed to execute WHERE clause with special chars: %v", err)
		}

		t.Logf("Found %d tasks with special character WHERE clause", len(tasks))
	})
}
