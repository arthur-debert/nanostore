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

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestItem represents a test item for bulk operations
type TestItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" default:"pending"`
	Priority string `values:"low,medium,high" default:"medium"`
}

func TestTypedStoreUpdateByUUIDs(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.NewFromType[TestItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("update multiple items by UUIDs", func(t *testing.T) {
		// Create test items
		id1, err := store.Create("Test Item 1", &TestItem{
			Status:   "pending",
			Priority: "low",
		})
		if err != nil {
			t.Fatal(err)
		}

		id2, err := store.Create("Test Item 2", &TestItem{
			Status:   "pending",
			Priority: "medium",
		})
		if err != nil {
			t.Fatal(err)
		}

		id3, err := store.Create("Test Item 3", &TestItem{
			Status:   "active",
			Priority: "high",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Update first two items
		targetUUIDs := []string{id1, id2}
		updateData := &TestItem{
			Status:   "done",
			Priority: "high",
		}
		updateData.Title = "Bulk Updated"

		count, err := store.UpdateByUUIDs(targetUUIDs, updateData)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected to update 2 items, updated %d", count)
		}

		// Verify updates
		item1, err := store.Get(id1)
		if err != nil {
			t.Fatal(err)
		}
		if item1.Title != "Bulk Updated" {
			t.Errorf("item1 title = %q, want %q", item1.Title, "Bulk Updated")
		}
		if item1.Status != "done" {
			t.Errorf("item1 status = %q, want %q", item1.Status, "done")
		}
		if item1.Priority != "high" {
			t.Errorf("item1 priority = %q, want %q", item1.Priority, "high")
		}

		item2, err := store.Get(id2)
		if err != nil {
			t.Fatal(err)
		}
		if item2.Status != "done" {
			t.Errorf("item2 status = %q, want %q", item2.Status, "done")
		}

		// Verify third item unchanged
		item3, err := store.Get(id3)
		if err != nil {
			t.Fatal(err)
		}
		if item3.Status != "active" {
			t.Errorf("item3 status should remain %q, got %q", "active", item3.Status)
		}
		if item3.Title == "Bulk Updated" {
			t.Error("item3 title should not have been updated")
		}
	})

	t.Run("update with empty UUID list", func(t *testing.T) {
		count, err := store.UpdateByUUIDs([]string{}, &TestItem{Status: "done"})
		if err != nil {
			t.Fatalf("UpdateByUUIDs with empty list failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to update 0 items, updated %d", count)
		}
	})

	t.Run("update with non-existent UUIDs", func(t *testing.T) {
		count, err := store.UpdateByUUIDs([]string{"non-existent-1", "non-existent-2"}, &TestItem{Status: "done"})
		if err != nil {
			t.Fatalf("UpdateByUUIDs with non-existent UUIDs failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to update 0 items, updated %d", count)
		}
	})

	t.Run("update only specific fields", func(t *testing.T) {
		// Create test item
		id, err := store.Create("Partial Update Test", &TestItem{
			Status:   "pending",
			Priority: "low",
		})
		if err != nil {
			t.Fatal(err)
		}

		originalItem, err := store.Get(id)
		if err != nil {
			t.Fatal(err)
		}

		// Update only priority
		updateData := &TestItem{Priority: "high"}
		count, err := store.UpdateByUUIDs([]string{id}, updateData)
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to update 1 item, updated %d", count)
		}

		// Verify only priority changed
		updatedItem, err := store.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if updatedItem.Priority != "high" {
			t.Errorf("priority = %q, want %q", updatedItem.Priority, "high")
		}
		if updatedItem.Status != originalItem.Status {
			t.Errorf("status should not have changed: got %q, want %q", updatedItem.Status, originalItem.Status)
		}
		if updatedItem.Title != originalItem.Title {
			t.Errorf("title should not have changed: got %q, want %q", updatedItem.Title, originalItem.Title)
		}
	})
}

func TestTypedStoreDeleteByUUIDs(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.NewFromType[TestItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("delete multiple items by UUIDs", func(t *testing.T) {
		// Create test items
		id1, err := store.Create("Delete Test 1", &TestItem{Status: "pending"})
		if err != nil {
			t.Fatal(err)
		}

		id2, err := store.Create("Delete Test 2", &TestItem{Status: "active"})
		if err != nil {
			t.Fatal(err)
		}

		id3, err := store.Create("Keep This", &TestItem{Status: "done"})
		if err != nil {
			t.Fatal(err)
		}

		// Delete first two
		count, err := store.DeleteByUUIDs([]string{id1, id2})
		if err != nil {
			t.Fatalf("DeleteByUUIDs failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected to delete 2 items, deleted %d", count)
		}

		// Verify they're gone
		_, err = store.Get(id1)
		if err == nil {
			t.Errorf("item %s should have been deleted", id1)
		}

		_, err = store.Get(id2)
		if err == nil {
			t.Errorf("item %s should have been deleted", id2)
		}

		// Verify third item still exists
		item3, err := store.Get(id3)
		if err != nil {
			t.Errorf("item %s should still exist: %v", id3, err)
		}
		if item3 == nil {
			t.Error("item3 should still exist")
		}
	})

	t.Run("delete with empty UUID list", func(t *testing.T) {
		count, err := store.DeleteByUUIDs([]string{})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with empty list failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to delete 0 items, deleted %d", count)
		}
	})

	t.Run("delete with non-existent UUIDs", func(t *testing.T) {
		count, err := store.DeleteByUUIDs([]string{"non-existent-1", "non-existent-2"})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with non-existent UUIDs failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to delete 0 items, deleted %d", count)
		}
	})

	t.Run("delete mixed existing and non-existent UUIDs", func(t *testing.T) {
		// Create test item
		id, err := store.Create("Mixed Delete Test", &TestItem{Status: "pending"})
		if err != nil {
			t.Fatal(err)
		}

		// Delete both existing and non-existent
		count, err := store.DeleteByUUIDs([]string{id, "non-existent-uuid"})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with mixed UUIDs failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to delete 1 item, deleted %d", count)
		}

		// Verify the existing one was deleted
		_, err = store.Get(id)
		if err == nil {
			t.Errorf("item %s should have been deleted", id)
		}
	})
}
