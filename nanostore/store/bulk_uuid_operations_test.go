package store_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

func TestUpdateByUUIDs(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("update multiple documents by UUIDs", func(t *testing.T) {
		// Get some documents to update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) < 2 {
			t.Skip("Need at least 2 pending documents for this test")
		}

		// Take first 2 documents
		targetUUIDs := []string{docs[0].UUID, docs[1].UUID}

		// Update them
		newTitle := "Bulk Updated Title"
		count, err := store.UpdateByUUIDs(targetUUIDs, nanostore.UpdateRequest{
			Title: &newTitle,
			Dimensions: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}

		if count != 2 {
			t.Errorf("expected to update 2 documents, updated %d", count)
		}

		// Verify the updates
		for _, uuid := range targetUUIDs {
			doc, err := store.GetByID(uuid)
			if err != nil {
				t.Fatalf("failed to get updated document %s: %v", uuid, err)
			}
			if doc.Title != newTitle {
				t.Errorf("document %s title = %q, want %q", uuid, doc.Title, newTitle)
			}
			if doc.Dimensions["status"] != "done" {
				t.Errorf("document %s status = %v, want %q", uuid, doc.Dimensions["status"], "done")
			}
		}
	})

	t.Run("update with empty UUID list", func(t *testing.T) {
		count, err := store.UpdateByUUIDs([]string{}, nanostore.UpdateRequest{
			Title: stringPtr("Should not update anything"),
		})
		if err != nil {
			t.Fatalf("UpdateByUUIDs with empty list failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to update 0 documents, updated %d", count)
		}
	})

	t.Run("update with non-existent UUIDs", func(t *testing.T) {
		count, err := store.UpdateByUUIDs([]string{"non-existent-uuid-1", "non-existent-uuid-2"}, nanostore.UpdateRequest{
			Title: stringPtr("Should not update anything"),
		})
		if err != nil {
			t.Fatalf("UpdateByUUIDs with non-existent UUIDs failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to update 0 documents, updated %d", count)
		}
	})

	t.Run("update only title", func(t *testing.T) {
		// Get a document
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) == 0 {
			t.Skip("Need at least 1 document for this test")
		}

		originalDoc := docs[0]
		newTitle := "Title Only Update"

		count, err := store.UpdateByUUIDs([]string{originalDoc.UUID}, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("UpdateByUUIDs failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to update 1 document, updated %d", count)
		}

		// Verify only title changed
		updatedDoc, err := store.GetByID(originalDoc.UUID)
		if err != nil {
			t.Fatalf("failed to get updated document: %v", err)
		}
		if updatedDoc.Title != newTitle {
			t.Errorf("title = %q, want %q", updatedDoc.Title, newTitle)
		}
		if updatedDoc.Body != originalDoc.Body {
			t.Errorf("body should not have changed: got %q, want %q", updatedDoc.Body, originalDoc.Body)
		}
	})
}

func TestDeleteByUUIDs(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("delete multiple documents by UUIDs", func(t *testing.T) {
		// Add some test documents first
		uuid1, err := store.Add("Test Document 1", map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatal(err)
		}

		uuid2, err := store.Add("Test Document 2", map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatal(err)
		}

		uuid3, err := store.Add("Test Document 3", map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Delete 2 of them
		targetUUIDs := []string{uuid1, uuid2}
		count, err := store.DeleteByUUIDs(targetUUIDs)
		if err != nil {
			t.Fatalf("DeleteByUUIDs failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected to delete 2 documents, deleted %d", count)
		}

		// Verify they're gone
		for _, uuid := range targetUUIDs {
			doc, err := store.GetByID(uuid)
			if err != nil {
				t.Fatalf("failed to check document %s: %v", uuid, err)
			}
			if doc != nil {
				t.Errorf("document %s should have been deleted", uuid)
			}
		}

		// Verify the third one still exists
		doc3, err := store.GetByID(uuid3)
		if err != nil {
			t.Errorf("document %s should still exist: %v", uuid3, err)
		}
		if doc3 == nil {
			t.Error("document 3 should still exist")
		}
	})

	t.Run("delete with empty UUID list", func(t *testing.T) {
		count, err := store.DeleteByUUIDs([]string{})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with empty list failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to delete 0 documents, deleted %d", count)
		}
	})

	t.Run("delete with non-existent UUIDs", func(t *testing.T) {
		count, err := store.DeleteByUUIDs([]string{"non-existent-uuid-1", "non-existent-uuid-2"})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with non-existent UUIDs failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to delete 0 documents, deleted %d", count)
		}
	})

	t.Run("delete mixed existing and non-existent UUIDs", func(t *testing.T) {
		// Add a test document
		uuid, err := store.Add("Test Document for Mixed Delete", map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Try to delete both existing and non-existent
		count, err := store.DeleteByUUIDs([]string{uuid, "non-existent-uuid"})
		if err != nil {
			t.Fatalf("DeleteByUUIDs with mixed UUIDs failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to delete 1 document, deleted %d", count)
		}

		// Verify the existing one was deleted
		doc, err := store.GetByID(uuid)
		if err != nil {
			t.Fatalf("failed to check document %s: %v", uuid, err)
		}
		if doc != nil {
			t.Errorf("document %s should have been deleted", uuid)
		}
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
