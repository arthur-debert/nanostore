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

func TestTypedQueryOROperations(t *testing.T) {
	// Create a temporary file for typed store
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

	// Create test data with various status, priority, and activity values
	uuid1, err := store.Create("Task 1", &TodoItem{
		Status:   "pending",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.Create("Task 2", &TodoItem{
		Status:   "active",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.Create("Task 3", &TodoItem{
		Status:   "done",
		Priority: "high",
		Activity: "archived",
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid4, err := store.Create("Task 4", &TodoItem{
		Status:   "pending",
		Priority: "low",
		Activity: "deleted",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("StatusIn", func(t *testing.T) {
		// Find tasks that are either pending or active
		tasks, err := store.Query().StatusIn("pending", "active").Find()
		if err != nil {
			t.Fatalf("failed to filter with StatusIn: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks with pending or active status, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (pending)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (active)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find task 4 (pending)")
		}
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (done)")
		}
	})

	t.Run("PriorityIn", func(t *testing.T) {
		// Find tasks that are either high or medium priority
		tasks, err := store.Query().PriorityIn("high", "medium").Find()
		if err != nil {
			t.Fatalf("failed to filter with PriorityIn: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks with high or medium priority, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (high)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (medium)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 (high)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (low)")
		}
	})

	t.Run("ActivityIn", func(t *testing.T) {
		// Find tasks that are either active or archived
		tasks, err := store.Query().ActivityIn("active", "archived").Find()
		if err != nil {
			t.Fatalf("failed to filter with ActivityIn: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks with active or archived activity, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (active)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (active)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 (archived)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (deleted)")
		}
	})

	t.Run("CombineInWithSingleFilter", func(t *testing.T) {
		// Combine StatusIn with single Priority filter
		tasks, err := store.Query().
			StatusIn("pending", "active").
			Priority("high").
			Find()
		if err != nil {
			t.Fatalf("failed to combine StatusIn with Priority: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task matching both filters, got %d", len(tasks))
		}

		if len(tasks) > 0 && tasks[0].UUID != uuid1 {
			t.Errorf("expected task 1, got %s", tasks[0].UUID)
		}
	})

	t.Run("CombineMultipleInFilters", func(t *testing.T) {
		// Combine multiple In filters
		tasks, err := store.Query().
			StatusIn("pending", "done").
			PriorityIn("high", "low").
			Find()
		if err != nil {
			t.Fatalf("failed to combine multiple In filters: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks matching both In filters, got %d", len(tasks))
		}

		// Should find task 1 (pending + high), task 3 (done + high), and task 4 (pending + low)
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (pending + high)")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find task 2 (active + medium)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 (done + high)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find task 4 (pending + low)")
		}
	})

	t.Run("SingleValueInArray", func(t *testing.T) {
		// Test In methods with single value
		tasks, err := store.Query().StatusIn("active").Find()
		if err != nil {
			t.Fatalf("failed to filter with single value in StatusIn: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task with single StatusIn value, got %d", len(tasks))
		}

		if len(tasks) > 0 && tasks[0].UUID != uuid2 {
			t.Errorf("expected task 2, got %s", tasks[0].UUID)
		}
	})

	t.Run("EmptyInFilter", func(t *testing.T) {
		// Test In methods with no values - should return no results
		tasks, err := store.Query().StatusIn().Find()
		if err != nil {
			t.Fatalf("failed to filter with empty StatusIn: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with empty StatusIn, got %d", len(tasks))
		}
	})

	t.Run("NonExistentValuesInFilter", func(t *testing.T) {
		// Test In methods with values that don't exist
		tasks, err := store.Query().StatusIn("nonexistent", "alsononexistent").Find()
		if err != nil {
			t.Fatalf("failed to filter with nonexistent StatusIn values: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with nonexistent StatusIn values, got %d", len(tasks))
		}
	})

	t.Run("MixExistentAndNonExistentValues", func(t *testing.T) {
		// Test In methods mixing real and fake values
		tasks, err := store.Query().StatusIn("active", "nonexistent", "pending").Find()
		if err != nil {
			t.Fatalf("failed to filter with mixed StatusIn values: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks with mixed StatusIn values (should ignore nonexistent), got %d", len(tasks))
		}

		// Should find all pending and active tasks
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (pending)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (active)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find task 4 (pending)")
		}
	})
}

func TestTypedQueryDataIn(t *testing.T) {
	// Create a temporary file for typed store
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

	// Create test data with custom data fields
	uuid1, err := store.AddRaw("Task Alice 1", map[string]interface{}{
		"status":         "active",
		"priority":       "high",
		"activity":       "active",
		"_data.Assignee": "alice",
		"_data.Team":     "backend",
		"_data.Estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.AddRaw("Task Bob 1", map[string]interface{}{
		"status":         "pending",
		"priority":       "medium",
		"activity":       "active",
		"_data.Assignee": "bob",
		"_data.Team":     "frontend",
		"_data.Estimate": 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.AddRaw("Task Charlie 1", map[string]interface{}{
		"status":         "done",
		"priority":       "low",
		"activity":       "active",
		"_data.Assignee": "charlie",
		"_data.Team":     "backend",
		"_data.Estimate": 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid4, err := store.AddRaw("Task Alice 2", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.Assignee": "alice",
		"_data.Team":     "devops",
		"_data.Estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("DataInString", func(t *testing.T) {
		// Find tasks assigned to Alice or Bob
		tasks, err := store.Query().DataIn("Assignee", "alice", "bob").Find()
		if err != nil {
			t.Fatalf("failed to filter with DataIn string: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks for Alice or Bob, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find Alice task 1")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find Bob task 1")
		}
		if foundUUIDs[uuid3] {
			t.Error("should not find Charlie task")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find Alice task 2")
		}
	})

	t.Run("DataInNumeric", func(t *testing.T) {
		// Find tasks with estimate 3 or 5
		tasks, err := store.Query().DataIn("Estimate", 3, 5).Find()
		if err != nil {
			t.Fatalf("failed to filter with DataIn numeric: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks with estimate 3 or 5, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (estimate 5)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (estimate 3)")
		}
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (estimate 8)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find task 4 (estimate 5)")
		}
	})

	t.Run("CombineDataInWithDimensionIn", func(t *testing.T) {
		// Combine DataIn with StatusIn
		tasks, err := store.Query().
			StatusIn("active", "done").
			DataIn("Assignee", "alice", "charlie").
			Find()
		if err != nil {
			t.Fatalf("failed to combine DataIn with StatusIn: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks matching both filters, got %d", len(tasks))
		}

		// Should find Alice's active tasks and Charlie's done task
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find Alice task 1 (active)")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find Bob task (wrong assignee)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find Charlie task (done)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find Alice task 2 (active)")
		}
	})

	t.Run("DataInSingleValue", func(t *testing.T) {
		// Test DataIn with single value
		tasks, err := store.Query().DataIn("Team", "backend").Find()
		if err != nil {
			t.Fatalf("failed to filter with single DataIn value: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 backend tasks, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (backend)")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find task 2 (frontend)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 (backend)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (devops)")
		}
	})

	t.Run("DataInMixedTypes", func(t *testing.T) {
		// Test DataIn with mixed types (string and number)
		tasks, err := store.Query().DataIn("Estimate", 3, "5").Find()
		if err != nil {
			t.Fatalf("failed to filter with mixed type DataIn: %v", err)
		}

		// This should work based on how the underlying store handles type conversion
		// The exact behavior may depend on implementation
		if len(tasks) > 0 {
			t.Logf("found %d tasks with mixed type DataIn", len(tasks))
		}
	})

	t.Run("DataInEmptyValues", func(t *testing.T) {
		// Test DataIn with no values
		tasks, err := store.Query().DataIn("Assignee").Find()
		if err != nil {
			t.Fatalf("failed to filter with empty DataIn: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks with empty DataIn, got %d", len(tasks))
		}
	})

	t.Run("DataInNonExistentField", func(t *testing.T) {
		// Test DataIn with field that doesn't exist - should now return validation error
		_, err := store.Query().DataIn("nonexistent", "alice", "bob").Find()
		if err == nil {
			t.Error("expected validation error for nonexistent field in DataIn, but got none")
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
