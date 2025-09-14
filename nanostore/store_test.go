package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestNewStore(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Basic smoke test - store should be created successfully
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestAddDocument(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a root document
	id, err := store.Add("Test Document", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	if id == "" {
		t.Fatal("expected non-empty UUID")
	}
}

func TestSetStatus(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a document
	id, err := store.Add("Test Document", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Change its status
	err = nanostore.SetStatus(store, id, "completed")
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}
}
