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
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestTypedQueryData(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with custom data fields
	uuid1, err := store.AddRaw("Task with assignee Alice", map[string]interface{}{
		"status":         "active",
		"priority":       "high",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.tags":     "urgent,backend",
		"_data.estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.AddRaw("Task with assignee Bob", map[string]interface{}{
		"status":         "pending",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "bob",
		"_data.tags":     "frontend,ui",
		"_data.estimate": 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.AddRaw("Task with assignee Alice", map[string]interface{}{
		"status":         "done",
		"priority":       "low",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.tags":     "documentation",
		"_data.estimate": 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Also create a regular task without custom data
	uuid4, err := store.Create("Regular Task", &TodoItem{
		Status:   "active",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("FilterByDataField", func(t *testing.T) {
		// Filter by assignee Alice
		aliceTasks, err := store.Query().Data("assignee", "alice").Find()
		if err != nil {
			t.Fatalf("failed to filter by assignee: %v", err)
		}

		if len(aliceTasks) != 2 {
			t.Errorf("expected 2 tasks for Alice, got %d", len(aliceTasks))
		}

		// Verify the correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range aliceTasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 for Alice")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 for Alice")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find Bob's task for Alice")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find regular task without assignee")
		}
	})

	t.Run("FilterByDataFieldBob", func(t *testing.T) {
		// Filter by assignee Bob
		bobTasks, err := store.Query().Data("assignee", "bob").Find()
		if err != nil {
			t.Fatalf("failed to filter by assignee Bob: %v", err)
		}

		if len(bobTasks) != 1 {
			t.Errorf("expected 1 task for Bob, got %d", len(bobTasks))
		}

		if len(bobTasks) > 0 && bobTasks[0].UUID != uuid2 {
			t.Errorf("expected Bob's task %s, got %s", uuid2, bobTasks[0].UUID)
		}
	})

	t.Run("FilterByDataFieldNumeric", func(t *testing.T) {
		// Filter by estimate value
		smallTasks, err := store.Query().Data("estimate", 1).Find()
		if err != nil {
			t.Fatalf("failed to filter by estimate: %v", err)
		}

		if len(smallTasks) != 1 {
			t.Errorf("expected 1 task with estimate 1, got %d", len(smallTasks))
		}

		if len(smallTasks) > 0 && smallTasks[0].UUID != uuid3 {
			t.Errorf("expected task 3, got %s", smallTasks[0].UUID)
		}
	})

	t.Run("FilterByDataFieldTags", func(t *testing.T) {
		// Filter by tags containing "backend"
		backendTasks, err := store.Query().Data("tags", "urgent,backend").Find()
		if err != nil {
			t.Fatalf("failed to filter by tags: %v", err)
		}

		if len(backendTasks) != 1 {
			t.Errorf("expected 1 backend task, got %d", len(backendTasks))
		}

		if len(backendTasks) > 0 && backendTasks[0].UUID != uuid1 {
			t.Errorf("expected task 1, got %s", backendTasks[0].UUID)
		}
	})

	t.Run("FilterByNonExistentDataField", func(t *testing.T) {
		// Filter by field that doesn't exist - should now return validation error
		_, err := store.Query().Data("nonexistent", "value").Find()
		if err == nil {
			t.Error("expected validation error for nonexistent field, but got none")
		}

		if err != nil {
			// Verify it's a validation error with helpful message
			errMsg := err.Error()
			if !strings.Contains(errMsg, "nonexistent") {
				t.Errorf("Error should mention the invalid field name, got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "not found") {
				t.Errorf("Error should indicate field not found, got: %s", errMsg)
			}
		}
	})

	t.Run("FilterByNonExistentDataValue", func(t *testing.T) {
		// Filter by field that exists but value that doesn't
		noTasks, err := store.Query().Data("assignee", "charlie").Find()
		if err != nil {
			t.Fatalf("failed to filter by nonexistent value: %v", err)
		}

		if len(noTasks) != 0 {
			t.Errorf("expected 0 tasks for Charlie, got %d", len(noTasks))
		}
	})

	t.Run("CombineDataFilterWithDimensionFilter", func(t *testing.T) {
		// Combine data filter with dimension filter
		activeAliceTasks, err := store.Query().
			Status("active").
			Data("assignee", "alice").
			Find()
		if err != nil {
			t.Fatalf("failed to combine filters: %v", err)
		}

		if len(activeAliceTasks) != 1 {
			t.Errorf("expected 1 active task for Alice, got %d", len(activeAliceTasks))
		}

		if len(activeAliceTasks) > 0 && activeAliceTasks[0].UUID != uuid1 {
			t.Errorf("expected task 1, got %s", activeAliceTasks[0].UUID)
		}
	})

	t.Run("MultipleDataFilters", func(t *testing.T) {
		// Multiple data filters
		specificTasks, err := store.Query().
			Data("assignee", "alice").
			Data("estimate", 5).
			Find()
		if err != nil {
			t.Fatalf("failed to apply multiple data filters: %v", err)
		}

		if len(specificTasks) != 1 {
			t.Errorf("expected 1 task matching both data filters, got %d", len(specificTasks))
		}

		if len(specificTasks) > 0 && specificTasks[0].UUID != uuid1 {
			t.Errorf("expected task 1, got %s", specificTasks[0].UUID)
		}
	})

	t.Run("ChainDataFilterMethods", func(t *testing.T) {
		// Test method chaining works correctly
		tasks, err := store.Query().
			Status("active").
			Data("assignee", "alice").
			Priority("high").
			Find()
		if err != nil {
			t.Fatalf("failed to chain data filter methods: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task with all filters, got %d", len(tasks))
		}

		if len(tasks) > 0 && tasks[0].UUID != uuid1 {
			t.Errorf("expected task 1, got %s", tasks[0].UUID)
		}
	})
}

func TestTypedQueryOrderByData(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with different data field values for ordering
	_, err = store.AddRaw("Task C", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "charlie",
		"_data.estimate": 8,
		"_data.score":    75,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddRaw("Task A", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.estimate": 3,
		"_data.score":    95,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddRaw("Task B", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "bob",
		"_data.estimate": 5,
		"_data.score":    85,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("OrderByDataAscending", func(t *testing.T) {
		// Order by assignee name (ascending)
		tasks, err := store.Query().
			Status("active").
			OrderByData("assignee").
			Find()
		if err != nil {
			t.Fatalf("failed to order by data field: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}

		// Should be ordered: alice, bob, charlie
		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, task := range tasks {
			if task.Title != expectedTitles[i] {
				t.Errorf("position %d: expected %q, got %q", i, expectedTitles[i], task.Title)
			}
		}
	})

	t.Run("OrderByDataDescending", func(t *testing.T) {
		// Order by score (highest first)
		tasks, err := store.Query().
			Status("active").
			OrderByDataDesc("score").
			Find()
		if err != nil {
			t.Fatalf("failed to order by data field descending: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}

		// Should be ordered: 95, 85, 75 (Task A, Task B, Task C)
		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, task := range tasks {
			if task.Title != expectedTitles[i] {
				t.Errorf("position %d: expected %q, got %q", i, expectedTitles[i], task.Title)
			}
		}
	})

	t.Run("OrderByDataNumeric", func(t *testing.T) {
		// Order by estimate (ascending)
		tasks, err := store.Query().
			Status("active").
			OrderByData("estimate").
			Find()
		if err != nil {
			t.Fatalf("failed to order by numeric data field: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}

		// Should be ordered: 3, 5, 8 (Task A, Task B, Task C)
		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, task := range tasks {
			if task.Title != expectedTitles[i] {
				t.Errorf("position %d: expected %q, got %q", i, expectedTitles[i], task.Title)
			}
		}
	})

	t.Run("CombineDataOrderWithDimensionOrder", func(t *testing.T) {
		// Combine data ordering with dimension ordering
		tasks, err := store.Query().
			Status("active").
			OrderByData("assignee").   // Primary sort by assignee
			OrderByDesc("created_at"). // Secondary sort by creation time
			Find()
		if err != nil {
			t.Fatalf("failed to combine ordering: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}

		// Should still be primarily ordered by assignee
		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, task := range tasks {
			if task.Title != expectedTitles[i] {
				t.Errorf("position %d: expected %q, got %q", i, expectedTitles[i], task.Title)
			}
		}
	})

	t.Run("FilterAndOrderByData", func(t *testing.T) {
		// Combine data filtering and ordering
		tasks, err := store.Query().
			Data("assignee", "alice").
			Data("assignee", "bob"). // This should override the previous filter
			OrderByDataDesc("score").
			Find()
		if err != nil {
			t.Fatalf("failed to filter and order by data: %v", err)
		}

		// Should only get Bob's task since the second Data() call overrides the first
		if len(tasks) != 1 {
			t.Errorf("expected 1 task (Bob's), got %d", len(tasks))
		}

		if len(tasks) > 0 && tasks[0].Title != "Task B" {
			t.Errorf("expected Task B, got %q", tasks[0].Title)
		}
	})

	t.Run("OrderByNonExistentDataField", func(t *testing.T) {
		// Order by field that doesn't exist - should now return validation error
		_, err := store.Query().
			Status("active").
			OrderByData("nonexistent").
			Find()
		if err == nil {
			t.Error("expected validation error for nonexistent field in OrderByData, but got none")
		}

		if err != nil {
			// Verify it's a validation error
			errMsg := err.Error()
			if !strings.Contains(errMsg, "nonexistent") {
				t.Errorf("Error should mention the invalid field name, got: %s", errMsg)
			}
		}
	})
}
