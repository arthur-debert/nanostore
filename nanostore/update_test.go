package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdate(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a document
	id, err := store.Add("Original Title", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Update title only
	newTitle := "Updated Title"
	err = store.Update(id, nanostore.UpdateRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("failed to update title: %v", err)
	}

	// Verify update
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	if docs[0].Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
	}

	// Update body only
	newBody := "This is the body"
	err = store.Update(id, nanostore.UpdateRequest{
		Body: &newBody,
	})
	if err != nil {
		t.Fatalf("failed to update body: %v", err)
	}

	// Verify both updates
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if docs[0].Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
	}
	if docs[0].Body != newBody {
		t.Errorf("expected body %q, got %q", newBody, docs[0].Body)
	}
}

func TestUpdateNonExistent(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Try to update non-existent document
	title := "New Title"
	err = store.Update("non-existent-uuid", nanostore.UpdateRequest{
		Title: &title,
	})

	if err == nil {
		t.Fatal("expected error when updating non-existent document")
	}
}

func TestUpdateWithoutParentField(t *testing.T) {
	// Test that existing code without ParentID field continues to work
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

	// Update without specifying ParentID (backward compatibility)
	newTitle := "Updated Child"
	err = store.Update(childID, nanostore.UpdateRequest{
		Title: &newTitle,
		// ParentID not specified - should remain unchanged
	})
	if err != nil {
		t.Errorf("failed to update: %v", err)
	}

	// Verify parent relationship unchanged
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	for _, doc := range docs {
		if doc.UUID == childID {
			if doc.Title != newTitle {
				t.Errorf("title not updated")
			}
			if doc.GetParentUUID() == nil || *doc.GetParentUUID() != parentID {
				t.Error("parent relationship changed when it shouldn't have")
			}
		}
	}
}
