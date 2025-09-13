package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestListEmpty(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestListWithIDs(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some pending documents
	id1, err := store.Add("First task", nil)
	if err != nil {
		t.Fatalf("failed to add first document: %v", err)
	}

	id2, err := store.Add("Second task", nil)
	if err != nil {
		t.Fatalf("failed to add second document: %v", err)
	}

	// Add a completed document
	id3, err := store.Add("Completed task", nil)
	if err != nil {
		t.Fatalf("failed to add third document: %v", err)
	}

	err = store.SetStatus(id3, nanostore.StatusCompleted)
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}

	// Check IDs are generated correctly
	expectedIDs := map[string]string{
		id1: "1",  // First pending
		id2: "2",  // Second pending
		id3: "c1", // First completed
	}

	for _, doc := range docs {
		expectedID, ok := expectedIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected document UUID: %s", doc.UUID)
			continue
		}

		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for document %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}

func TestListHierarchical(t *testing.T) {
	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a parent document
	parentID, err := store.Add("Parent task", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	// Add child documents
	child1ID, err := store.Add("Child 1", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 1: %v", err)
	}

	child2ID, err := store.Add("Child 2", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 2: %v", err)
	}

	// Add a completed child
	child3ID, err := store.Add("Completed child", &parentID)
	if err != nil {
		t.Fatalf("failed to add child 3: %v", err)
	}
	err = store.SetStatus(child3ID, nanostore.StatusCompleted)
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	// List all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 4 {
		t.Errorf("expected 4 documents, got %d", len(docs))
	}

	// Check hierarchical IDs
	expectedIDs := map[string]string{
		parentID: "1",    // Parent
		child1ID: "1.1",  // First child
		child2ID: "1.2",  // Second child
		child3ID: "1.c1", // First completed child
	}

	for _, doc := range docs {
		expectedID, ok := expectedIDs[doc.UUID]
		if !ok {
			t.Errorf("unexpected document UUID: %s", doc.UUID)
			continue
		}

		if doc.UserFacingID != expectedID {
			t.Errorf("expected ID %s for document %s, got %s", expectedID, doc.UUID, doc.UserFacingID)
		}
	}
}
