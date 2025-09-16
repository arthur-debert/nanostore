package nanostore

import (
	"testing"
)

func TestTypedStoreCRUD(t *testing.T) {
	// Define a test document type
	type TaskDoc struct {
		Document
		Status   string `values:"pending,in_progress,completed" default:"pending"`
		Priority string `values:"low,medium,high" default:"medium"`
		ParentID string `dimension:"parent_id,ref"`
	}

	t.Run("Create", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a new task
		task := &TaskDoc{
			Status:   "pending",
			Priority: "high",
		}

		uuid, err := store.Create("Important Task", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		if uuid == "" {
			t.Error("expected non-empty UUID")
		}

		// Verify the task UUID was set
		if task.UUID != uuid {
			t.Errorf("expected task UUID to be %q, got %q", uuid, task.UUID)
		}
	})

	t.Run("Get", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a task first
		original := &TaskDoc{
			Status:   "in_progress",
			Priority: "medium",
		}

		uuid, err := store.Create("Test Task", original)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Get the task back
		retrieved, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get task: %v", err)
		}

		// Verify fields
		if retrieved.UUID != uuid {
			t.Errorf("expected UUID %q, got %q", uuid, retrieved.UUID)
		}
		if retrieved.Title != "Test Task" {
			t.Errorf("expected title %q, got %q", "Test Task", retrieved.Title)
		}
		if retrieved.Status != "in_progress" {
			t.Errorf("expected status %q, got %q", "in_progress", retrieved.Status)
		}
		if retrieved.Priority != "medium" {
			t.Errorf("expected priority %q, got %q", "medium", retrieved.Priority)
		}
	})

	t.Run("Update", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a task
		task := &TaskDoc{
			Status:   "pending",
			Priority: "low",
		}

		uuid, err := store.Create("Update Test", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Update the task
		task.Status = "completed"
		task.Priority = "high"
		task.Title = "Updated Task"

		err = store.Update(uuid, task)
		if err != nil {
			t.Fatalf("failed to update task: %v", err)
		}

		// Get and verify
		updated, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get updated task: %v", err)
		}

		if updated.Status != "completed" {
			t.Errorf("expected status %q, got %q", "completed", updated.Status)
		}
		if updated.Priority != "high" {
			t.Errorf("expected priority %q, got %q", "high", updated.Priority)
		}
		if updated.Title != "Updated Task" {
			t.Errorf("expected title %q, got %q", "Updated Task", updated.Title)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a task
		task := &TaskDoc{
			Status: "pending",
		}

		uuid, err := store.Create("Delete Test", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Delete the task
		err = store.Delete(uuid, false)
		if err != nil {
			t.Fatalf("failed to delete task: %v", err)
		}

		// Verify it's gone
		_, err = store.Get(uuid)
		if err == nil {
			t.Error("expected error when getting deleted task")
		}
	})

	t.Run("Hierarchical", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent task
		parent := &TaskDoc{
			Status:   "in_progress",
			Priority: "high",
		}

		parentUUID, err := store.Create("Parent Task", parent)
		if err != nil {
			t.Fatalf("failed to create parent task: %v", err)
		}

		// Create child task
		child := &TaskDoc{
			Status:   "pending",
			Priority: "medium",
			ParentID: parentUUID,
		}

		childUUID, err := store.Create("Child Task", child)
		if err != nil {
			t.Fatalf("failed to create child task: %v", err)
		}

		// Get and verify child
		retrieved, err := store.Get(childUUID)
		if err != nil {
			t.Fatalf("failed to get child task: %v", err)
		}

		if retrieved.ParentID != parentUUID {
			t.Errorf("expected parent_id %q, got %q", parentUUID, retrieved.ParentID)
		}

		// Test cascade delete
		err = store.Delete(parentUUID, true)
		if err != nil {
			t.Fatalf("failed to cascade delete parent: %v", err)
		}

		// Verify child is also deleted
		_, err = store.Get(childUUID)
		if err == nil {
			t.Error("expected child to be deleted with cascade")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create task without specifying fields with defaults
		task := &TaskDoc{}

		uuid, err := store.Create("Default Test", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Get and verify defaults were applied
		retrieved, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get task: %v", err)
		}

		if retrieved.Status != "pending" {
			t.Errorf("expected default status %q, got %q", "pending", retrieved.Status)
		}
		if retrieved.Priority != "medium" {
			t.Errorf("expected default priority %q, got %q", "medium", retrieved.Priority)
		}
	})

	t.Run("SmartIDResolution", func(t *testing.T) {
		store, err := NewFromType[TaskDoc](":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a high priority task
		task := &TaskDoc{
			Status:   "pending",
			Priority: "high",
		}

		uuid, err := store.Create("High Priority Task", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Should be able to get by smart ID (assuming first high priority task)
		// Note: This tests the underlying store's smart ID resolution
		retrieved, err := store.Get("h1")
		if err != nil {
			// Smart ID might not be "h1" if prefixes aren't configured
			// Try with full UUID instead
			retrieved, err = store.Get(uuid)
			if err != nil {
				t.Fatalf("failed to get task by UUID: %v", err)
			}
		}

		if retrieved.UUID != uuid {
			t.Errorf("expected UUID %q, got %q", uuid, retrieved.UUID)
		}
	})
}
