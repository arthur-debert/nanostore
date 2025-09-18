package api_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeclarativeParentFiltering(t *testing.T) {
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

	// Create some root todos
	root1ID, _ := store.Create("Root 1", &TodoItem{})
	root2ID, _ := store.Create("Root 2", &TodoItem{})

	// Create children for root1
	_, _ = store.Create("Child 1.1", &TodoItem{ParentID: root1ID})
	_, _ = store.Create("Child 1.2", &TodoItem{ParentID: root1ID})

	// Create a child for root2
	_, _ = store.Create("Child 2.1", &TodoItem{ParentID: root2ID})

	t.Run("FilterByNoParent", func(t *testing.T) {
		// Query todos without parent
		roots, err := store.Query().
			ParentIDNotExists().
			Find()
		if err != nil {
			t.Fatalf("failed to query roots: %v", err)
		}

		// Should have exactly 2 root todos
		if len(roots) != 2 {
			t.Errorf("expected 2 root todos, got %d", len(roots))
			for _, todo := range roots {
				t.Logf("- %s (parent: %s)", todo.Title, todo.ParentID)
			}
		}

		// Check that we got the right todos
		foundRoot1 := false
		foundRoot2 := false
		for _, todo := range roots {
			if todo.Title == "Root 1" {
				foundRoot1 = true
			}
			if todo.Title == "Root 2" {
				foundRoot2 = true
			}
		}

		if !foundRoot1 || !foundRoot2 {
			t.Error("didn't find expected root todos")
		}
	})

	t.Run("FilterBySpecificParent", func(t *testing.T) {
		// Query children of root1
		children, err := store.Query().
			ParentID(root1ID).
			Find()
		if err != nil {
			t.Fatalf("failed to query children: %v", err)
		}

		if len(children) != 2 {
			t.Errorf("expected 2 children of root1, got %d", len(children))
		}

		// All should have root1 as parent
		for _, child := range children {
			if child.ParentID != root1ID {
				t.Errorf("expected parent %s, got %s", root1ID, child.ParentID)
			}
		}
	})
}
