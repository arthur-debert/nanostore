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
	"github.com/arthur-debert/nanostore/types"
)

func TestTypedStoreList(t *testing.T) {
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

	// Create test data
	parentID, err := store.Create("Parent Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	childID1, err := store.Create("Child Task 1", &TodoItem{
		Status:   "pending",
		Priority: "medium",
		Activity: "active",
		ParentID: parentID,
	})
	if err != nil {
		t.Fatal(err)
	}

	childID2, err := store.Create("Child Task 2", &TodoItem{
		Status:   "done",
		Priority: "low",
		Activity: "active",
		ParentID: parentID,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("Deleted Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "deleted",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ListAllTasks", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list all tasks: %v", err)
		}

		if len(tasks) != 4 {
			t.Errorf("expected 4 tasks, got %d", len(tasks))
		}

		// Verify all tasks have proper type structure
		for _, task := range tasks {
			if task.UUID == "" {
				t.Error("task UUID should not be empty")
			}
			if task.SimpleID == "" {
				t.Error("task SimpleID should not be empty")
			}
			if task.Title == "" {
				t.Error("task Title should not be empty")
			}
			if task.Activity == "" {
				t.Error("task Activity should not be empty")
			}
		}
	})

	t.Run("ListWithActivityFilter", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"activity": "active",
			},
		})
		if err != nil {
			t.Fatalf("failed to list active tasks: %v", err)
		}

		if len(tasks) != 3 {
			t.Errorf("expected 3 active tasks, got %d", len(tasks))
		}

		// Verify all returned tasks are active
		for _, task := range tasks {
			if task.Activity != "active" {
				t.Errorf("expected activity 'active', got %q", task.Activity)
			}
		}
	})

	t.Run("ListWithStatusFilter", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatalf("failed to list pending tasks: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 pending task, got %d", len(tasks))
		}

		if len(tasks) > 0 {
			if tasks[0].Status != "pending" {
				t.Errorf("expected status 'pending', got %q", tasks[0].Status)
			}
			if tasks[0].UUID != childID1 {
				t.Errorf("expected UUID %q, got %q", childID1, tasks[0].UUID)
			}
		}
	})

	t.Run("ListWithParentIDFilter", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": parentID,
			},
		})
		if err != nil {
			t.Fatalf("failed to list child tasks: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 child tasks, got %d", len(tasks))
		}

		// Verify all returned tasks have correct parent
		for _, task := range tasks {
			if task.ParentID != parentID {
				t.Errorf("expected ParentID %q, got %q", parentID, task.ParentID)
			}
		}
	})

	t.Run("ListWithMultipleFilters", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"activity":  "active",
				"status":    "done",
				"parent_id": parentID,
			},
		})
		if err != nil {
			t.Fatalf("failed to list filtered tasks: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 filtered task, got %d", len(tasks))
		}

		if len(tasks) > 0 {
			task := tasks[0]
			if task.Activity != "active" {
				t.Errorf("expected activity 'active', got %q", task.Activity)
			}
			if task.Status != "done" {
				t.Errorf("expected status 'done', got %q", task.Status)
			}
			if task.ParentID != parentID {
				t.Errorf("expected ParentID %q, got %q", parentID, task.ParentID)
			}
			if task.UUID != childID2 {
				t.Errorf("expected UUID %q, got %q", childID2, task.UUID)
			}
		}
	})

	t.Run("ListWithLimit", func(t *testing.T) {
		limit := 2
		tasks, err := store.List(types.ListOptions{
			Limit: &limit,
		})
		if err != nil {
			t.Fatalf("failed to list limited tasks: %v", err)
		}

		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks with limit, got %d", len(tasks))
		}
	})

	t.Run("ListWithOffset", func(t *testing.T) {
		allTasks, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list all tasks: %v", err)
		}

		offset := 1
		tasks, err := store.List(types.ListOptions{
			Offset: &offset,
		})
		if err != nil {
			t.Fatalf("failed to list offset tasks: %v", err)
		}

		if len(tasks) != len(allTasks)-1 {
			t.Errorf("expected %d tasks with offset, got %d", len(allTasks)-1, len(tasks))
		}
	})

	t.Run("ListWithOrdering", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			OrderBy: []types.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list ordered tasks: %v", err)
		}

		if len(tasks) < 2 {
			t.Skip("need at least 2 tasks to test ordering")
		}

		// Verify ordering (should be alphabetical by title)
		for i := 1; i < len(tasks); i++ {
			if tasks[i-1].Title > tasks[i].Title {
				t.Errorf("tasks not properly ordered by title: %q should come before %q", tasks[i-1].Title, tasks[i].Title)
			}
		}
	})

	t.Run("ListEmptyResult", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"status": "nonexistent_status",
			},
		})
		if err != nil {
			t.Fatalf("failed to list tasks with no matches: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks for nonexistent status, got %d", len(tasks))
		}
	})

	t.Run("ListWithUUIDFilter", func(t *testing.T) {
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"uuid": parentID,
			},
		})
		if err != nil {
			t.Fatalf("failed to list task by UUID: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 task by UUID, got %d", len(tasks))
		}

		if len(tasks) > 0 {
			if tasks[0].UUID != parentID {
				t.Errorf("expected UUID %q, got %q", parentID, tasks[0].UUID)
			}
			if tasks[0].Title != "Parent Task" {
				t.Errorf("expected title 'Parent Task', got %q", tasks[0].Title)
			}
		}
	})
}

func TestTypedStoreListErrorHandling(t *testing.T) {
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

	t.Run("ListWithInvalidFilter", func(t *testing.T) {
		// This should work but return empty results, as the underlying store
		// handles invalid filters gracefully
		tasks, err := store.List(types.ListOptions{
			Filters: map[string]interface{}{
				"invalid_field": "some_value",
			},
		})
		if err != nil {
			t.Fatalf("list with invalid filter should not error: %v", err)
		}

		// Should return empty results rather than error
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks for invalid filter, got %d", len(tasks))
		}
	})

	t.Run("ListWithNilFilters", func(t *testing.T) {
		_, err := store.List(types.ListOptions{
			Filters: nil,
		})
		if err != nil {
			t.Fatalf("list with nil filters should not error: %v", err)
		}

		// Should work the same as empty filters (no error expected)
	})
}
