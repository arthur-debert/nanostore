package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestBasicOperations(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "in_progress", "done"},
				DefaultValue: "todo",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("Add", func(t *testing.T) {
		// Add a document with default dimensions
		id1, err := store.Add("Task 1", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		if id1 == "" {
			t.Fatal("expected non-empty ID")
		}

		// Add a document with custom dimensions
		id2, err := store.Add("Task 2", map[string]interface{}{
			"status":   "in_progress",
			"priority": "high",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		if id2 == "" {
			t.Fatal("expected non-empty ID")
		}

		// Verify documents were added
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}
	})

	t.Run("Update", func(t *testing.T) {
		// Add a document
		id, err := store.Add("Update Test", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Update title
		newTitle := "Updated Title"
		err = store.Update(id, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("failed to update document: %v", err)
		}

		// Update dimensions
		err = store.Update(id, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status":   "done",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatalf("failed to update dimensions: %v", err)
		}

		// Verify updates
		docs, _ := store.List(nanostore.ListOptions{})
		for _, doc := range docs {
			if doc.UUID == id {
				if doc.Title != "Updated Title" {
					t.Errorf("title not updated: %s", doc.Title)
				}
				if doc.Dimensions["status"] != "done" {
					t.Errorf("status not updated: %v", doc.Dimensions["status"])
				}
				if doc.Dimensions["priority"] != "high" {
					t.Errorf("priority not updated: %v", doc.Dimensions["priority"])
				}
				break
			}
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Add a document
		id, err := store.Add("Delete Test", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Get count before delete
		docs, _ := store.List(nanostore.ListOptions{})
		countBefore := len(docs)

		// Delete the document
		err = store.Delete(id, false)
		if err != nil {
			t.Fatalf("failed to delete document: %v", err)
		}

		// Verify deletion
		docs, _ = store.List(nanostore.ListOptions{})
		if len(docs) != countBefore-1 {
			t.Errorf("document not deleted: expected %d, got %d", countBefore-1, len(docs))
		}
	})

	t.Run("InvalidDimensionValue", func(t *testing.T) {
		// Try to add with invalid status
		_, err := store.Add("Invalid", map[string]interface{}{
			"status": "invalid_status",
		})
		if err == nil {
			t.Error("expected error for invalid dimension value")
		}
	})
}

func TestHierarchicalOperations(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("ParentChild", func(t *testing.T) {
		// Add parent
		parentID, err := store.Add("Parent", nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		// Add children
		child1ID, err := store.Add("Child 1", map[string]interface{}{
			"parent_uuid": parentID,
		})
		if err != nil {
			t.Fatalf("failed to add child 1: %v", err)
		}

		child2ID, _ := store.Add("Child 2", map[string]interface{}{
			"parent_uuid": parentID,
		})

		// Try to delete parent without cascade
		err = store.Delete(parentID, false)
		if err == nil {
			t.Error("expected error when deleting parent without cascade")
		}

		// Delete parent with cascade
		err = store.Delete(parentID, true)
		if err != nil {
			t.Fatalf("failed to delete parent with cascade: %v", err)
		}

		// Verify all were deleted
		docs, _ := store.List(nanostore.ListOptions{})
		if len(docs) != 0 {
			t.Errorf("expected 0 documents after cascade delete, got %d", len(docs))
		}

		// Verify children were deleted by checking they can't be found
		for _, childID := range []string{child1ID, child2ID} {
			found := false
			for _, doc := range docs {
				if doc.UUID == childID {
					found = true
					break
				}
			}
			if found {
				t.Errorf("child %s not deleted", childID)
			}
		}
	})
}
