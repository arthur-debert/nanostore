package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestTransactionBehavior(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test 1: Verify foreign key constraint is enforced with transactions
	invalidParent := "non-existent-uuid"
	_, err = store.Add("Orphan Document", &invalidParent)

	if err == nil {
		t.Error("expected foreign key constraint error, but got none")
	} else {
		t.Logf("Foreign key constraint properly enforced: %v", err)
	}

	// Verify no partial data was inserted
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("expected 0 documents after failed insert, got %d", len(docs))
	}

	// Test 2: Verify update rollback on non-existent document
	fakeID := "non-existent-document"
	title := "Should Not Work"
	err = store.Update(fakeID, nanostore.UpdateRequest{
		Title: &title,
	})

	if err == nil {
		t.Error("expected error updating non-existent document")
	} else {
		t.Logf("Update properly failed for non-existent document: %v", err)
	}

	// Test 3: Verify successful transaction
	id, err := store.Add("Valid Document", nil)
	if err != nil {
		t.Fatalf("failed to add valid document: %v", err)
	}

	// Update it
	newTitle := "Updated Title"
	newBody := "Updated Body"
	err = store.Update(id, nanostore.UpdateRequest{
		Title: &newTitle,
		Body:  &newBody,
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify the update was atomic
	docs, err = store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Title != newTitle || doc.Body != newBody {
		t.Error("update was not atomic - partial update detected")
	}

	// Verify updated_at was changed (might be same second, so just check it's not zero)
	if doc.UpdatedAt.IsZero() {
		t.Error("updated_at timestamp is zero")
	}
	t.Logf("Document timestamps - Created: %v, Updated: %v", doc.CreatedAt, doc.UpdatedAt)
}
