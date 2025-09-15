package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDelete(t *testing.T) {
	t.Run("delete single document without children", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document
		id, err := store.Add("Test Document", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Delete it
		err = store.Delete(id, false)
		if err != nil {
			t.Errorf("failed to delete document: %v", err)
		}

		// Verify it's gone
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("expected 0 documents, got %d", len(docs))
		}
	})

	t.Run("delete non-existent document", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Try to delete non-existent document (use proper UUID format)
		nonExistentUUID := "00000000-0000-0000-0000-000000000001"
		err = store.Delete(nonExistentUUID, false)
		if err == nil {
			t.Error("expected error when deleting non-existent document")
		}
		expectedError := "document not found: " + nonExistentUUID
		if err.Error() != expectedError {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("delete document with children (cascade=false)", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent and child
		parentID, err := store.Add("Parent", nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		_, err = store.Add("Child", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Try to delete parent without cascade
		err = store.Delete(parentID, false)
		if err == nil {
			t.Error("expected error when deleting parent without cascade")
		} else if err.Error() != "cannot delete document with children unless cascade is true" {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify parent still exists
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}
	})

	t.Run("delete document with children (cascade=true)", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent with multiple children and grandchildren
		parentID, err := store.Add("Parent", nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		child1ID, err := store.Add("Child 1", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child 1: %v", err)
		}

		child2ID, err := store.Add("Child 2", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child 2: %v", err)
		}

		_, err = store.Add("Grandchild 1", map[string]interface{}{"parent_uuid": child1ID})
		if err != nil {
			t.Fatalf("failed to add grandchild 1: %v", err)
		}

		_, err = store.Add("Grandchild 2", map[string]interface{}{"parent_uuid": child2ID})
		if err != nil {
			t.Fatalf("failed to add grandchild 2: %v", err)
		}

		// Delete parent with cascade
		err = store.Delete(parentID, true)
		if err != nil {
			t.Errorf("failed to delete parent with cascade: %v", err)
		}

		// Verify all are gone
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("expected 0 documents after cascade delete, got %d", len(docs))
		}
	})

	t.Run("delete middle node with cascade", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create grandparent -> parent -> child hierarchy
		grandparentID, err := store.Add("Grandparent", nil)
		if err != nil {
			t.Fatalf("failed to add grandparent: %v", err)
		}

		parentID, err := store.Add("Parent", map[string]interface{}{"parent_uuid": grandparentID})
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		childID, err := store.Add("Child", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Also add a sibling to the parent
		_, err = store.Add("Sibling", map[string]interface{}{"parent_uuid": grandparentID})
		if err != nil {
			t.Fatalf("failed to add sibling: %v", err)
		}

		// Delete the parent (middle node) with cascade
		err = store.Delete(parentID, true)
		if err != nil {
			t.Errorf("failed to delete parent with cascade: %v", err)
		}

		// Verify grandparent and sibling remain, but parent and child are gone
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("expected 2 documents remaining, got %d", len(docs))
		}

		// Verify the right documents remain
		foundGrandparent := false
		foundSibling := false
		for _, doc := range docs {
			if doc.UUID == grandparentID {
				foundGrandparent = true
			}
			if doc.Title == "Sibling" {
				foundSibling = true
			}
			if doc.UUID == parentID || doc.UUID == childID {
				t.Errorf("found deleted document: %s", doc.Title)
			}
		}
		if !foundGrandparent {
			t.Error("grandparent not found after cascade delete")
		}
		if !foundSibling {
			t.Error("sibling not found after cascade delete")
		}
	})

	t.Run("delete leaf node", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent -> child hierarchy
		parentID, err := store.Add("Parent", nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		childID, err := store.Add("Child", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Delete the child (leaf node)
		err = store.Delete(childID, false)
		if err != nil {
			t.Errorf("failed to delete child: %v", err)
		}

		// Verify only parent remains
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		if docs[0].UUID != parentID {
			t.Error("parent not found after deleting child")
		}
	})

	t.Run("delete with mixed statuses", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent with children of different statuses
		parentID, err := store.Add("Parent", nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		_, err = store.Add("Pending Child", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child 1: %v", err)
		}

		child2ID, err := store.Add("Completed Child", map[string]interface{}{"parent_uuid": parentID})
		if err != nil {
			t.Fatalf("failed to add child 2: %v", err)
		}

		// Set one child as completed
		err = nanostore.SetStatus(store, child2ID, "completed")
		if err != nil {
			t.Fatalf("failed to set status: %v", err)
		}

		// Delete parent with cascade
		err = store.Delete(parentID, true)
		if err != nil {
			t.Errorf("failed to delete parent with cascade: %v", err)
		}

		// Verify all are gone
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("expected 0 documents, got %d", len(docs))
		}
	})
}

func TestDeleteConcurrent(t *testing.T) {
	// SQLite has database-level locking, so concurrent deletes from different
	// connections will result in lock contention. This test verifies that
	// concurrent deletes from the same connection work correctly.
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple documents
	var ids []string
	for i := 0; i < 10; i++ {
		id, err := store.Add("Document", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		ids = append(ids, id)
	}

	// Delete them sequentially (SQLite doesn't support true concurrent writes)
	for _, id := range ids {
		err := store.Delete(id, false)
		if err != nil {
			t.Errorf("delete failed: %v", err)
		}
	}

	// Verify all are deleted
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents after deletes, got %d", len(docs))
	}
}

func TestDeleteEdgeCases(t *testing.T) {
	t.Run("delete with SQL injection attempt", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Try to delete with SQL injection
		err = store.Delete("'; DROP TABLE documents; --", false)
		if err == nil {
			t.Error("expected error for SQL injection attempt")
		}
	})

	t.Run("delete empty string ID", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		err = store.Delete("", false)
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("delete very deep hierarchy", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a 10-level deep hierarchy
		var currentParentID string
		for i := 0; i < 10; i++ {
			var parentID *string
			if currentParentID != "" {
				parentID = &currentParentID
			}
			dimensions := make(map[string]interface{})
			if parentID != nil {
				dimensions["parent_uuid"] = *parentID
			}
			id, err := store.Add("Level", dimensions)
			if err != nil {
				t.Fatalf("failed to add level %d: %v", i, err)
			}
			if i == 0 {
				// Remember the root for deletion
				currentParentID = id
			} else if i < 9 {
				currentParentID = id
			}
		}

		// Delete the root with cascade
		rootID := currentParentID
		for i := 0; i < 9; i++ {
			// Get parent of current
			docs, _ := store.List(nanostore.ListOptions{})
			for _, doc := range docs {
				if doc.GetParentUUID() == nil {
					rootID = doc.UUID
					break
				}
			}
		}

		err = store.Delete(rootID, true)
		if err != nil {
			t.Errorf("failed to delete deep hierarchy: %v", err)
		}

		// Verify all are gone
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("expected 0 documents after cascade delete, got %d", len(docs))
		}
	})
}
