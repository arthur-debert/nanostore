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

func TestUpdateMethodsAfterRefactoring(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	task1 := &TodoItem{
		Status:   "pending",
		Priority: "high",
		Activity: "active",
	}
	task1ID, err := store.Create("Task 1", task1)
	if err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}

	task2 := &TodoItem{
		Status:   "pending",
		Priority: "medium",
		Activity: "active",
	}
	task2ID, err := store.Create("Task 2", task2)
	if err != nil {
		t.Fatalf("Failed to create task2: %v", err)
	}

	task3 := &TodoItem{
		Status:   "active",
		Priority: "low",
		Activity: "active",
	}
	task3ID, err := store.Create("Task 3", task3)
	if err != nil {
		t.Fatalf("Failed to create task3: %v", err)
	}

	t.Run("Update", func(t *testing.T) {
		// Test single document update
		updateData := &TodoItem{
			Status:   "done",
			Priority: "high",
			Activity: "archived",
		}
		count, err := store.Update(task1ID, updateData)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 document updated, got %d", count)
		}

		// Verify the update
		updated, err := store.Get(task1ID)
		if err != nil {
			t.Fatalf("Get after update failed: %v", err)
		}
		if updated.Status != "done" {
			t.Errorf("Expected status 'done', got %s", updated.Status)
		}
	})

	t.Run("UpdateByDimension", func(t *testing.T) {
		// Test bulk update by dimension
		updateData := &TodoItem{
			Priority: "high",
		}
		count, err := store.UpdateByDimension(map[string]interface{}{
			"status": "pending",
		}, updateData)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != 1 { // Only task2 should match (task1 was changed to "done")
			t.Errorf("Expected 1 update, got %d", count)
		}

		// Verify the update
		updated, err := store.Get(task2ID)
		if err != nil {
			t.Fatalf("Get after UpdateByDimension failed: %v", err)
		}
		if updated.Priority != "high" {
			t.Errorf("Expected priority 'high', got %s", updated.Priority)
		}
	})

	t.Run("UpdateWhere", func(t *testing.T) {
		// Test bulk update with WHERE clause
		updateData := &TodoItem{
			Activity: "deleted",
		}
		count, err := store.UpdateWhere("priority = ?", updateData, "low")
		if err != nil {
			t.Fatalf("UpdateWhere failed: %v", err)
		}
		if count != 1 { // Only task3 should match
			t.Errorf("Expected 1 update, got %d", count)
		}

		// Verify the update
		updated, err := store.Get(task3ID)
		if err != nil {
			t.Fatalf("Get after UpdateWhere failed: %v", err)
		}
		if updated.Activity != "deleted" {
			t.Errorf("Expected activity 'deleted', got %s", updated.Activity)
		}
	})

	t.Run("UpdateByUUIDs", func(t *testing.T) {
		// Get UUIDs for task1 and task2
		task1Raw, err := store.GetRaw(task1ID)
		if err != nil {
			t.Fatalf("GetRaw task1 failed: %v", err)
		}
		task2Raw, err := store.GetRaw(task2ID)
		if err != nil {
			t.Fatalf("GetRaw task2 failed: %v", err)
		}

		// Test bulk update by UUIDs
		updateData := &TodoItem{
			Priority: "low",
		}
		count, err := store.UpdateByUUIDs([]string{task1Raw.UUID, task2Raw.UUID}, updateData)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 updates, got %d", count)
		}

		// Verify both updates
		updated1, err := store.Get(task1ID)
		if err != nil {
			t.Fatalf("Get task1 after UpdateByUUIDs failed: %v", err)
		}
		if updated1.Priority != "low" {
			t.Errorf("Expected task1 priority 'low', got %s", updated1.Priority)
		}

		updated2, err := store.Get(task2ID)
		if err != nil {
			t.Fatalf("Get task2 after UpdateByUUIDs failed: %v", err)
		}
		if updated2.Priority != "low" {
			t.Errorf("Expected task2 priority 'low', got %s", updated2.Priority)
		}
	})
}
