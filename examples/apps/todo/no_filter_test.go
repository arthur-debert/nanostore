package todo_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestNoFilterHierarchy(t *testing.T) {
	// Create store directly with default config
	store, err := nanostore.New(":memory:", nanostore.DefaultTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Add documents
	rootID, _ := store.Add("Groceries", nil, nil)
	store.Add("Milk", &rootID, nil)
	store.Add("Bread", &rootID, nil)
	store.Add("Eggs", &rootID, nil)
	store.Add("Pack for Trip", nil, nil)

	// List WITHOUT any filters
	t.Logf("\nListing without filters:")
	docs, _ := store.List(nanostore.ListOptions{})
	for _, doc := range docs {
		t.Logf("  ID: %-5s Title: %s", doc.UserFacingID, doc.Title)
	}
}
