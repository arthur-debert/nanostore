package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateParent(t *testing.T) {
	t.Run("move document to new parent", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create two potential parents and a child
		parent1ID, err := store.Add("Parent 1", nil, nil)
		if err != nil {
			t.Fatalf("failed to add parent 1: %v", err)
		}

		parent2ID, err := store.Add("Parent 2", nil, nil)
		if err != nil {
			t.Fatalf("failed to add parent 2: %v", err)
		}

		childID, err := store.Add("Child", &parent1ID, nil)
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Move child from parent1 to parent2
		err = store.Update(childID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": parent2ID},
		})
		if err != nil {
			t.Errorf("failed to update parent: %v", err)
		}

		// Verify the move
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		for _, doc := range docs {
			if doc.UUID == childID {
				if doc.GetParentUUID() == nil || *doc.GetParentUUID() != parent2ID {
					t.Errorf("child parent not updated correctly")
				}
				// Check that the ID reflects the new parent
				if doc.UserFacingID != "2.1" {
					t.Errorf("expected child ID to be 2.1, got %s", doc.UserFacingID)
				}
			}
		}
	})

	t.Run("make child document a root", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent and child
		parentID, err := store.Add("Parent", nil, nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		childID, err := store.Add("Child", &parentID, nil)
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Make child a root document
		err = store.Update(childID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": ""},
		})
		if err != nil {
			t.Errorf("failed to make child root: %v", err)
		}

		// Verify it's now a root document
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		for _, doc := range docs {
			if doc.UUID == childID {
				if doc.GetParentUUID() != nil {
					t.Errorf("child still has parent after update")
				}
				// Should now have a root-level ID
				if doc.UserFacingID != "2" {
					t.Errorf("expected root ID 2, got %s", doc.UserFacingID)
				}
			}
		}
	})

	t.Run("make root document a child", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create two root documents
		root1ID, err := store.Add("Root 1", nil, nil)
		if err != nil {
			t.Fatalf("failed to add root 1: %v", err)
		}

		root2ID, err := store.Add("Root 2", nil, nil)
		if err != nil {
			t.Fatalf("failed to add root 2: %v", err)
		}

		// Make root2 a child of root1
		err = store.Update(root2ID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": root1ID},
		})
		if err != nil {
			t.Errorf("failed to make root a child: %v", err)
		}

		// Verify the change
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		for _, doc := range docs {
			if doc.UUID == root2ID {
				if doc.GetParentUUID() == nil || *doc.GetParentUUID() != root1ID {
					t.Errorf("root2 not made child of root1")
				}
				// Should now have hierarchical ID
				if doc.UserFacingID != "1.1" {
					t.Errorf("expected child ID 1.1, got %s", doc.UserFacingID)
				}
			}
		}
	})

	t.Run("prevent self-parent", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document
		docID, err := store.Add("Document", nil, nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Try to make it its own parent
		err = store.Update(docID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": docID},
		})
		if err == nil {
			t.Error("expected error when setting document as its own parent")
		}
		if err.Error() != "cannot set document as its own parent" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("prevent circular reference", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a chain: A -> B -> C
		aID, err := store.Add("A", nil, nil)
		if err != nil {
			t.Fatalf("failed to add A: %v", err)
		}

		bID, err := store.Add("B", &aID, nil)
		if err != nil {
			t.Fatalf("failed to add B: %v", err)
		}

		cID, err := store.Add("C", &bID, nil)
		if err != nil {
			t.Fatalf("failed to add C: %v", err)
		}

		// Try to make A a child of C (would create cycle)
		err = store.Update(aID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": cID},
		})
		if err == nil {
			t.Error("expected error when creating circular reference")
		}
		if err.Error() != "cannot set parent: would create circular reference" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("update parent with other fields", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent and child
		parentID, err := store.Add("Parent", nil, nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		childID, err := store.Add("Child", &parentID, nil)
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Update title and make it a root
		newTitle := "Updated Child"
		err = store.Update(childID, nanostore.UpdateRequest{
			Title:      &newTitle,
			Dimensions: map[string]string{"parent_uuid": ""},
		})
		if err != nil {
			t.Errorf("failed to update: %v", err)
		}

		// Verify both changes
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		for _, doc := range docs {
			if doc.UUID == childID {
				if doc.Title != newTitle {
					t.Errorf("title not updated: got %s, want %s", doc.Title, newTitle)
				}
				if doc.GetParentUUID() != nil {
					t.Error("document still has parent")
				}
			}
		}
	})

	t.Run("update to non-existent parent", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document
		docID, err := store.Add("Document", nil, nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Try to set non-existent parent
		fakeParent := "non-existent-uuid"
		err = store.Update(docID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": fakeParent},
		})
		if err == nil {
			t.Error("expected error when setting non-existent parent")
		}
	})

	t.Run("nil parent means no change", func(t *testing.T) {
		store, err := nanostore.NewTestStore(":memory:")
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent and child
		parentID, err := store.Add("Parent", nil, nil)
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		childID, err := store.Add("Child", &parentID, nil)
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Update with nil parent (should not change parent)
		newTitle := "Updated Title"
		err = store.Update(childID, nanostore.UpdateRequest{
			Title:      &newTitle,
			Dimensions: nil, // Explicitly nil
		})
		if err != nil {
			t.Errorf("failed to update: %v", err)
		}

		// Verify parent unchanged
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		for _, doc := range docs {
			if doc.UUID == childID {
				if doc.GetParentUUID() == nil || *doc.GetParentUUID() != parentID {
					t.Error("parent changed when it shouldn't have")
				}
			}
		}
	})
}

func TestUpdateParentComplexHierarchy(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a complex hierarchy:
	// Root1
	//   ├── Child1
	//   │   └── Grandchild1
	//   └── Child2
	// Root2
	//   └── Child3

	root1ID, _ := store.Add("Root1", nil, nil)
	root2ID, _ := store.Add("Root2", nil, nil)
	child1ID, _ := store.Add("Child1", &root1ID, nil)
	child2ID, _ := store.Add("Child2", &root1ID, nil)
	child3ID, _ := store.Add("Child3", &root2ID, nil)
	grandchild1ID, _ := store.Add("Grandchild1", &child1ID, nil)

	t.Run("move subtree to different root", func(t *testing.T) {
		// Move Child1 (and its subtree) to Root2
		err := store.Update(child1ID, nanostore.UpdateRequest{
			Dimensions: map[string]string{"parent_uuid": root2ID},
		})
		if err != nil {
			t.Errorf("failed to move subtree: %v", err)
		}

		// Verify the IDs are updated correctly
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		// IDs are based on creation order within each parent, not display order
		expectedIDs := map[string]string{
			root1ID:       "1",
			root2ID:       "2",
			child2ID:      "1.1",   // Still child of Root1
			child1ID:      "2.1",   // First created child of Root2 (after move)
			child3ID:      "2.2",   // Second child of Root2 (created after child1)
			grandchild1ID: "2.1.1", // Follows its parent child1
		}

		for _, doc := range docs {
			if expected, ok := expectedIDs[doc.UUID]; ok {
				if doc.UserFacingID != expected {
					t.Errorf("document %s: expected ID %s, got %s", doc.Title, expected, doc.UserFacingID)
				}
			}
		}
	})
}
