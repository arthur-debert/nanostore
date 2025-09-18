package api_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeclarativeDelete(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := nanostore.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("DeleteSingleDocument", func(t *testing.T) {
		// Create a todo
		id, err := store.Create("To be deleted", &TodoItem{
			Priority: "high",
		})
		if err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}

		// Verify it exists
		todo, err := store.Get(id)
		if err != nil {
			t.Fatalf("failed to get todo: %v", err)
		}
		if todo.Title != "To be deleted" {
			t.Errorf("expected title 'To be deleted', got %s", todo.Title)
		}

		// Delete it
		err = store.Delete(id, false)
		if err != nil {
			t.Fatalf("failed to delete todo: %v", err)
		}

		// Verify it's gone
		_, err = store.Get(id)
		if err == nil {
			t.Error("expected error when getting deleted todo, got nil")
		}
	})

	t.Run("DeleteNonExistentDocument", func(t *testing.T) {
		// Try to delete a non-existent ID
		err := store.Delete("non-existent-id", false)
		// Should error - document not found
		if err == nil {
			t.Error("expected error when deleting non-existent document, got nil")
		}
	})

	t.Run("DeleteWithChildren", func(t *testing.T) {
		// Create parent
		parentID, err := store.Create("Parent todo", &TodoItem{})
		if err != nil {
			t.Fatalf("failed to create parent: %v", err)
		}

		// Create children
		child1ID, err := store.Create("Child 1", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 1: %v", err)
		}

		child2ID, err := store.Create("Child 2", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 2: %v", err)
		}

		// Delete parent without cascade - should fail
		err = store.Delete(parentID, false)
		if err == nil {
			t.Error("expected error when deleting parent with children and cascade=false, got nil")
		}

		// Verify parent still exists
		parent, err := store.Get(parentID)
		if err != nil {
			t.Errorf("expected parent to still exist after failed delete, got error: %v", err)
		}
		if parent.Title != "Parent todo" {
			t.Errorf("expected parent title 'Parent todo', got %s", parent.Title)
		}

		// Now delete children first
		err = store.Delete(child1ID, false)
		if err != nil {
			t.Fatalf("failed to delete child 1: %v", err)
		}

		err = store.Delete(child2ID, false)
		if err != nil {
			t.Fatalf("failed to delete child 2: %v", err)
		}

		// Now parent can be deleted
		err = store.Delete(parentID, false)
		if err != nil {
			t.Fatalf("failed to delete parent after children removed: %v", err)
		}

		// Verify all are gone
		_, err = store.Get(parentID)
		if err == nil {
			t.Error("expected error when getting deleted parent, got nil")
		}
	})

	t.Run("DeleteMultipleDocuments", func(t *testing.T) {
		// Create multiple todos
		ids := make([]string, 3)
		for i := 0; i < 3; i++ {
			id, err := store.Create("Batch delete todo", &TodoItem{
				Priority: "low",
			})
			if err != nil {
				t.Fatalf("failed to create todo %d: %v", i, err)
			}
			ids[i] = id
		}

		// Delete all of them
		for _, id := range ids {
			err := store.Delete(id, false)
			if err != nil {
				t.Fatalf("failed to delete todo %s: %v", id, err)
			}
		}

		// Verify all are gone
		for _, id := range ids {
			_, err := store.Get(id)
			if err == nil {
				t.Errorf("expected error when getting deleted todo %s, got nil", id)
			}
		}
	})

	t.Run("DeleteWithCascade", func(t *testing.T) {
		// Create parent
		parentID, err := store.Create("Parent with cascade", &TodoItem{})
		if err != nil {
			t.Fatalf("failed to create parent: %v", err)
		}

		// Create children
		child1ID, err := store.Create("Cascade child 1", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 1: %v", err)
		}

		child2ID, err := store.Create("Cascade child 2", &TodoItem{
			ParentID: parentID,
		})
		if err != nil {
			t.Fatalf("failed to create child 2: %v", err)
		}

		// Create grandchild
		grandchildID, err := store.Create("Grandchild", &TodoItem{
			ParentID: child1ID,
		})
		if err != nil {
			t.Fatalf("failed to create grandchild: %v", err)
		}

		// Delete parent with cascade
		err = store.Delete(parentID, true)
		if err != nil {
			t.Fatalf("failed to delete parent with cascade: %v", err)
		}

		// Verify all are gone
		_, err = store.Get(parentID)
		if err == nil {
			t.Error("expected error when getting deleted parent, got nil")
		}

		_, err = store.Get(child1ID)
		if err == nil {
			t.Error("expected error when getting cascade deleted child 1, got nil")
		}

		_, err = store.Get(child2ID)
		if err == nil {
			t.Error("expected error when getting cascade deleted child 2, got nil")
		}

		_, err = store.Get(grandchildID)
		if err == nil {
			t.Error("expected error when getting cascade deleted grandchild, got nil")
		}
	})
}
