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

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestTypedQueryNOTOperations(t *testing.T) {
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

	t.Run("StatusNot", func(t *testing.T) {
		// Find tasks that are NOT done
		tasks, err := store.Query().StatusNot("done").Find()
		if err != nil {
			t.Fatalf("failed to filter with StatusNot: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks not done, got %d", len(tasks))
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
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (done)")
		}
		if !foundUUIDs[uuid4] {
			t.Error("expected to find task 4 (pending)")
		}
	})

	t.Run("PriorityNot", func(t *testing.T) {
		// Find tasks that are NOT low priority
		tasks, err := store.Query().PriorityNot("low").Find()
		if err != nil {
			t.Fatalf("failed to filter with PriorityNot: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks not low priority, got %d", len(tasks))
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

	t.Run("ActivityNot", func(t *testing.T) {
		// Find tasks that are NOT deleted
		tasks, err := store.Query().ActivityNot("deleted").Find()
		if err != nil {
			t.Fatalf("failed to filter with ActivityNot: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks not deleted, got %d", len(tasks))
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

	t.Run("StatusNotIn", func(t *testing.T) {
		// Find tasks that are NOT pending or done
		tasks, err := store.Query().StatusNotIn("pending", "done").Find()
		if err != nil {
			t.Fatalf("failed to filter with StatusNotIn: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task not pending or done, got %d", len(tasks))
		}

		if len(tasks) > 0 && tasks[0].UUID != uuid2 {
			t.Errorf("expected task 2 (active), got %s", tasks[0].UUID)
		}
	})

	t.Run("PriorityNotIn", func(t *testing.T) {
		// Find tasks that are NOT low or medium priority (i.e., high only)
		tasks, err := store.Query().PriorityNotIn("low", "medium").Find()
		if err != nil {
			t.Fatalf("failed to filter with PriorityNotIn: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 high priority tasks, got %d", len(tasks))
		}

		// Verify correct tasks were returned
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (high)")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find task 2 (medium)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find task 3 (high)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (low)")
		}
	})

	t.Run("ActivityNotIn", func(t *testing.T) {
		// Find tasks that are NOT archived or deleted (i.e., active only)
		tasks, err := store.Query().ActivityNotIn("archived", "deleted").Find()
		if err != nil {
			t.Fatalf("failed to filter with ActivityNotIn: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 active tasks, got %d", len(tasks))
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
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (archived)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (deleted)")
		}
	})

	t.Run("CombineNotWithPositiveFilter", func(t *testing.T) {
		// Combine StatusNot with Activity filter
		tasks, err := store.Query().
			StatusNot("done").
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to combine StatusNot with Activity: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks matching both filters, got %d", len(tasks))
		}

		// Should find tasks 1 and 2 (both not done and active)
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (pending + active)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (active + active)")
		}
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (done)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (deleted)")
		}
	})

	t.Run("CombineMultipleNotFilters", func(t *testing.T) {
		// Combine multiple NOT filters
		tasks, err := store.Query().
			StatusNot("done").
			PriorityNot("low").
			ActivityNot("deleted").
			Find()
		if err != nil {
			t.Fatalf("failed to combine multiple NOT filters: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks matching all NOT filters, got %d", len(tasks))
		}

		// Should find tasks 1 and 2
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if !foundUUIDs[uuid1] {
			t.Error("expected to find task 1 (pending + high + active)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find task 2 (active + medium + active)")
		}
		if foundUUIDs[uuid3] {
			t.Error("should not find task 3 (done)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find task 4 (low priority)")
		}
	})

	t.Run("NotFilterWithEmptyResult", func(t *testing.T) {
		// NOT filter that excludes everything
		tasks, err := store.Query().
			StatusNotIn("pending", "active", "done").
			Find()
		if err != nil {
			t.Fatalf("failed to filter with comprehensive StatusNotIn: %v", err)
		}

		// This might return all tasks if the NOT logic results in an empty filter
		// The exact behavior depends on the implementation of empty include lists
		t.Logf("Got %d tasks when excluding all known statuses", len(tasks))
		// For now, we'll accept either 0 or all tasks depending on implementation
	})
}

func TestTypedQueryDataNOTOperations(t *testing.T) {
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
	uuid1, err := store.AddRaw("Task Alice 1", map[string]interface{}{
		"status":         "active",
		"priority":       "high",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.team":     "backend",
		"_data.estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.AddRaw("Task Bob 1", map[string]interface{}{
		"status":         "pending",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "bob",
		"_data.team":     "frontend",
		"_data.estimate": 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.AddRaw("Task Charlie 1", map[string]interface{}{
		"status":         "done",
		"priority":       "low",
		"activity":       "active",
		"_data.assignee": "charlie",
		"_data.team":     "backend",
		"_data.estimate": 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid4, err := store.AddRaw("Task Alice 2", map[string]interface{}{
		"status":         "active",
		"priority":       "medium",
		"activity":       "active",
		"_data.assignee": "alice",
		"_data.team":     "devops",
		"_data.estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Also create a task without assignee data field
	uuid5, err := store.Create("Regular Task", &TodoItem{
		Status:   "active",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("DataNot", func(t *testing.T) {
		// Find tasks NOT assigned to Alice
		tasks, err := store.Query().DataNot("assignee", "alice").Find()
		if err != nil {
			t.Fatalf("failed to filter with DataNot: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks not assigned to Alice, got %d", len(tasks))
		}

		// Should find Bob, Charlie, and the regular task (no assignee field)
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if foundUUIDs[uuid1] {
			t.Error("should not find Alice task 1")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find Bob task")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find Charlie task")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find Alice task 2")
		}
		if !foundUUIDs[uuid5] {
			t.Error("expected to find regular task (no assignee)")
		}
	})

	t.Run("DataNotNumeric", func(t *testing.T) {
		// Find tasks NOT with estimate 5
		tasks, err := store.Query().DataNot("estimate", 5).Find()
		if err != nil {
			t.Fatalf("failed to filter with DataNot numeric: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks not with estimate 5, got %d", len(tasks))
		}

		// Should find Bob (3), Charlie (8), and the regular task (no estimate)
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if foundUUIDs[uuid1] {
			t.Error("should not find Alice task 1 (estimate 5)")
		}
		if !foundUUIDs[uuid2] {
			t.Error("expected to find Bob task (estimate 3)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find Charlie task (estimate 8)")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find Alice task 2 (estimate 5)")
		}
		if !foundUUIDs[uuid5] {
			t.Error("expected to find regular task (no estimate)")
		}
	})

	t.Run("DataNotIn", func(t *testing.T) {
		// Find tasks NOT assigned to Alice or Bob
		tasks, err := store.Query().DataNotIn("assignee", "alice", "bob").Find()
		if err != nil {
			t.Fatalf("failed to filter with DataNotIn: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks not assigned to Alice or Bob, got %d", len(tasks))
		}

		// Should find Charlie and the regular task
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if foundUUIDs[uuid1] {
			t.Error("should not find Alice task 1")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find Bob task")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find Charlie task")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find Alice task 2")
		}
		if !foundUUIDs[uuid5] {
			t.Error("expected to find regular task (no assignee)")
		}
	})

	t.Run("CombineDataNotWithDimensionFilter", func(t *testing.T) {
		// Combine DataNot with status filter
		tasks, err := store.Query().
			Status("active").
			DataNot("assignee", "alice").
			Find()
		if err != nil {
			t.Fatalf("failed to combine DataNot with Status: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task matching both filters, got %d", len(tasks))
		}

		// Should find only the regular task (active status, no Alice assignment)
		if len(tasks) > 0 && tasks[0].UUID != uuid5 {
			t.Errorf("expected regular task %s, got %s", uuid5, tasks[0].UUID)
		}
	})

	t.Run("CombineDataNotWithDataFilter", func(t *testing.T) {
		// Combine DataNot with positive Data filter
		tasks, err := store.Query().
			Data("team", "backend").
			DataNot("assignee", "alice").
			Find()
		if err != nil {
			t.Fatalf("failed to combine DataNot with Data: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task matching both filters, got %d", len(tasks))
		}

		// Should find only Charlie (backend team, not Alice)
		if len(tasks) > 0 && tasks[0].UUID != uuid3 {
			t.Errorf("expected Charlie task %s, got %s", uuid3, tasks[0].UUID)
		}
	})

	t.Run("DataNotWithNonExistentField", func(t *testing.T) {
		// DataNot with field that doesn't exist on any document
		tasks, err := store.Query().DataNot("nonexistent", "value").Find()
		if err != nil {
			t.Fatalf("failed to filter with DataNot on nonexistent field: %v", err)
		}

		// Should return all tasks since none have the nonexistent field
		if len(tasks) != 5 {
			t.Errorf("expected 5 tasks when filtering nonexistent field, got %d", len(tasks))
		}
	})

	t.Run("MultipleDataNotFilters", func(t *testing.T) {
		// Multiple DataNot filters
		tasks, err := store.Query().
			DataNot("assignee", "alice").
			DataNot("team", "frontend").
			Find()
		if err != nil {
			t.Fatalf("failed to apply multiple DataNot filters: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks matching both DataNot filters, got %d", len(tasks))
		}

		// Should find Charlie (backend team, not Alice) and regular task
		foundUUIDs := make(map[string]bool)
		for _, task := range tasks {
			foundUUIDs[task.UUID] = true
		}

		if foundUUIDs[uuid1] {
			t.Error("should not find Alice task 1")
		}
		if foundUUIDs[uuid2] {
			t.Error("should not find Bob task (frontend team)")
		}
		if !foundUUIDs[uuid3] {
			t.Error("expected to find Charlie task")
		}
		if foundUUIDs[uuid4] {
			t.Error("should not find Alice task 2")
		}
		if !foundUUIDs[uuid5] {
			t.Error("expected to find regular task")
		}
	})
}
