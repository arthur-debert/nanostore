package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateEmptyRequest(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a document
	docID, err := store.Add("Original Title", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Get original state
	origDocs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	var origDoc nanostore.Document
	for _, doc := range origDocs {
		if doc.UUID == docID {
			origDoc = doc
			break
		}
	}

	// Update with completely empty request (all fields nil)
	err = store.Update(docID, nanostore.UpdateRequest{})
	if err != nil {
		t.Errorf("failed to update with empty request: %v", err)
	}

	// Verify nothing changed except UpdatedAt
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents after update: %v", err)
	}

	for _, doc := range docs {
		if doc.UUID == docID {
			if doc.Title != origDoc.Title {
				t.Errorf("title changed: was %q, now %q", origDoc.Title, doc.Title)
			}
			if doc.Body != origDoc.Body {
				t.Errorf("body changed: was %q, now %q", origDoc.Body, doc.Body)
			}
			if doc.ParentUUID != origDoc.ParentUUID {
				t.Error("parent changed when it shouldn't have")
			}
			// UpdatedAt should be newer or the same (SQLite might optimize away no-op updates)
			if doc.UpdatedAt.Before(origDoc.UpdatedAt) {
				t.Error("UpdatedAt went backwards")
			}
		}
	}
}

func TestUpdateEmptyRequestWithParent(t *testing.T) {
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

	// Update child with empty request
	err = store.Update(childID, nanostore.UpdateRequest{
		Title:    nil,
		Body:     nil,
		ParentID: nil,
	})
	if err != nil {
		t.Errorf("failed to update with empty request: %v", err)
	}

	// Verify parent relationship maintained
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	for _, doc := range docs {
		if doc.UUID == childID {
			if doc.ParentUUID == nil || *doc.ParentUUID != parentID {
				t.Error("parent relationship changed with nil ParentID")
			}
		}
	}
}
