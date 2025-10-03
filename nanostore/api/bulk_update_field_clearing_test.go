package api_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestBulkClearingItem represents a test item for bulk update field clearing testing
type TestBulkClearingItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`
	PinnedAt time.Time

	// Data fields
	Assignee      string
	Description   string
	EstimateHours int
}

func TestBulkUpdateFieldClearing(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test_bulk_update_field_clearing*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestBulkClearingItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create initial test documents with non-zero values
	uuid1, err := store.Create("Task 1", &TestBulkClearingItem{
		Status:        "active",
		Priority:      "high",
		PinnedAt:      time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Assignee:      "alice",
		Description:   "Important task",
		EstimateHours: 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid2, err := store.Create("Task 2", &TestBulkClearingItem{
		Status:        "pending",
		Priority:      "medium",
		PinnedAt:      time.Date(2024, 1, 2, 14, 0, 0, 0, time.UTC),
		Assignee:      "bob",
		Description:   "Regular task",
		EstimateHours: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuid3, err := store.Create("Task 3", &TestBulkClearingItem{
		Status:        "done",
		Priority:      "low",
		PinnedAt:      time.Date(2024, 1, 3, 9, 0, 0, 0, time.UTC),
		Assignee:      "charlie",
		Description:   "Completed task",
		EstimateHours: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	uuids := []string{uuid1, uuid2, uuid3}

	t.Run("UpdateByUUIDs should clear string data fields with zero values", func(t *testing.T) {
		// This test should FAIL initially - demonstrates the bug
		// Zero values should clear fields, not be ignored

		// Attempt to clear assignee and description fields for all tasks
		updates := &TestBulkClearingItem{
			Assignee:    "", // Zero value - should clear the field
			Description: "", // Zero value - should clear the field
		}

		count, err := store.UpdateByUUIDs(uuids, updates)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 documents updated, got %d", count)
		}

		// Verify that fields were actually cleared
		for i, uuid := range uuids {
			retrieved, err := store.Get(uuid)
			if err != nil {
				t.Fatalf("Get failed for uuid %s: %v", uuid, err)
			}

			// These should be empty (cleared) but currently they retain original values
			if retrieved.Assignee != "" {
				t.Errorf("Task %d: Expected assignee to be cleared, got: %q", i+1, retrieved.Assignee)
			}

			if retrieved.Description != "" {
				t.Errorf("Task %d: Expected description to be cleared, got: %q", i+1, retrieved.Description)
			}
		}
	})

	t.Run("UpdateByUUIDs should clear numeric data fields with zero values", func(t *testing.T) {
		// Attempt to clear estimate hours for all tasks
		updates := &TestBulkClearingItem{
			EstimateHours: 0, // Zero value - should clear the field
		}

		count, err := store.UpdateByUUIDs(uuids, updates)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 documents updated, got %d", count)
		}

		// Verify that numeric field was cleared
		for i, uuid := range uuids {
			retrieved, err := store.Get(uuid)
			if err != nil {
				t.Fatalf("Get failed for uuid %s: %v", uuid, err)
			}

			if retrieved.EstimateHours != 0 {
				t.Errorf("Task %d: Expected estimate hours to be cleared (0), got: %d", i+1, retrieved.EstimateHours)
			}
		}
	})

	t.Run("UpdateByUUIDs should clear time fields with zero values", func(t *testing.T) {
		// Attempt to clear pinned time for all tasks
		updates := &TestBulkClearingItem{
			PinnedAt: time.Time{}, // Zero value - should clear the field
		}

		count, err := store.UpdateByUUIDs(uuids, updates)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 documents updated, got %d", count)
		}

		// Verify that time field was cleared
		for i, uuid := range uuids {
			retrieved, err := store.Get(uuid)
			if err != nil {
				t.Fatalf("Get failed for uuid %s: %v", uuid, err)
			}

			if !retrieved.PinnedAt.IsZero() {
				t.Errorf("Task %d: Expected pinned time to be cleared (zero), got: %v", i+1, retrieved.PinnedAt)
			}
		}
	})

	t.Run("UpdateByDimension should clear fields with zero values", func(t *testing.T) {
		// First, set some values back to test UpdateByDimension
		for _, uuid := range uuids {
			retrieved, err := store.Get(uuid)
			if err != nil {
				t.Fatal(err)
			}
			retrieved.Assignee = "test-assignee"
			_, err = store.Update(uuid, retrieved)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Now attempt to clear with UpdateByDimension
		updates := &TestBulkClearingItem{
			Assignee: "", // Zero value - should clear the field
		}

		// Update all tasks with priority "high", "medium", or "low" (all of them)
		filters := map[string]interface{}{
			"status": "active", // Only task 1 should match
		}

		count, err := store.UpdateByDimension(filters, updates)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 document updated by dimension, got %d", count)
		}

		// Verify the active task had its assignee cleared
		retrieved, err := store.Get(uuid1) // Task 1 is active
		if err != nil {
			t.Fatal(err)
		}

		if retrieved.Assignee != "" {
			t.Errorf("Expected assignee to be cleared for active task, got: %q", retrieved.Assignee)
		}
	})

	t.Run("Individual Update should work correctly with zero values (baseline)", func(t *testing.T) {
		// This test verifies that individual updates work correctly
		// This serves as a baseline to compare bulk update behavior

		// First, set a value
		retrieved, err := store.Get(uuid1)
		if err != nil {
			t.Fatal(err)
		}
		retrieved.Assignee = "individual-test"
		_, err = store.Update(uuid1, retrieved)
		if err != nil {
			t.Fatal(err)
		}

		// Verify it was set
		retrieved, err = store.Get(uuid1)
		if err != nil {
			t.Fatal(err)
		}
		if retrieved.Assignee != "individual-test" {
			t.Errorf("Expected assignee to be set to 'individual-test', got: %q", retrieved.Assignee)
		}

		// Now clear it with individual update
		retrieved.Assignee = ""
		_, err = store.Update(uuid1, retrieved)
		if err != nil {
			t.Fatalf("Individual update failed: %v", err)
		}

		// Verify it was cleared
		retrieved, err = store.Get(uuid1)
		if err != nil {
			t.Fatal(err)
		}
		if retrieved.Assignee != "" {
			t.Errorf("Expected assignee to be cleared with individual update, got: %q", retrieved.Assignee)
		}
	})

	t.Run("Non-zero values should still update correctly in bulk operations", func(t *testing.T) {
		// Verify that non-zero values still work correctly (shouldn't break existing functionality)

		updates := &TestBulkClearingItem{
			Assignee:      "bulk-assignee",
			Description:   "bulk-description",
			EstimateHours: 10,
		}

		count, err := store.UpdateByUUIDs(uuids, updates)
		if err != nil {
			t.Fatalf("UpdateByUUIDs with non-zero values failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 documents updated, got %d", count)
		}

		// Verify non-zero values were set correctly
		for i, uuid := range uuids {
			retrieved, err := store.Get(uuid)
			if err != nil {
				t.Fatalf("Get failed for uuid %s: %v", uuid, err)
			}

			if retrieved.Assignee != "bulk-assignee" {
				t.Errorf("Task %d: Expected assignee 'bulk-assignee', got: %q", i+1, retrieved.Assignee)
			}

			if retrieved.Description != "bulk-description" {
				t.Errorf("Task %d: Expected description 'bulk-description', got: %q", i+1, retrieved.Description)
			}

			if retrieved.EstimateHours != 10 {
				t.Errorf("Task %d: Expected estimate hours 10, got: %d", i+1, retrieved.EstimateHours)
			}
		}
	})
}

func TestDimensionFieldClearingInBulkUpdates(t *testing.T) {
	// Test clearing of dimension fields (enumerated fields) in bulk updates

	tmpfile, err := os.CreateTemp("", "test_dimension_clearing*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TestBulkClearingItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create test document with non-default dimension values
	uuid1, err := store.Create("Task 1", &TestBulkClearingItem{
		Status:   "active",
		Priority: "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Bulk update should handle dimension fields with default values", func(t *testing.T) {
		// Note: This might be a separate issue, but testing for completeness
		// When updating with default values, should they be applied or ignored?

		updates := &TestBulkClearingItem{
			Status:   "pending", // Reset to default
			Priority: "medium",  // Reset to default
		}

		count, err := store.UpdateByUUIDs([]string{uuid1}, updates)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 document updated, got %d", count)
		}

		retrieved, err := store.Get(uuid1)
		if err != nil {
			t.Fatal(err)
		}

		// These should be updated to the new values
		if retrieved.Status != "pending" {
			t.Errorf("Expected status to be updated to 'pending', got: %q", retrieved.Status)
		}

		if retrieved.Priority != "medium" {
			t.Errorf("Expected priority to be updated to 'medium', got: %q", retrieved.Priority)
		}
	})
}
