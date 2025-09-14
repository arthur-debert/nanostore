package todo_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestHierarchyFiltering(t *testing.T) {
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

	// Test 1: List all pending
	t.Logf("\nTest 1: All pending items:")
	docs, _ := store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
	})
	for _, doc := range docs {
		t.Logf("  ID: %-5s Title: %s", doc.UserFacingID, doc.Title)
	}

	// Test 2: List only root pending items
	t.Logf("\nTest 2: Root pending items only:")
	emptyParent := ""
	docs, _ = store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
		FilterByParent: &emptyParent,
	})
	for _, doc := range docs {
		t.Logf("  ID: %-5s Title: %s", doc.UserFacingID, doc.Title)
	}

	// Test 3: List children of first root
	t.Logf("\nTest 3: Children of Groceries:")
	docs, _ = store.List(nanostore.ListOptions{
		FilterByStatus: []nanostore.Status{nanostore.StatusPending},
		FilterByParent: &rootID,
	})
	for _, doc := range docs {
		t.Logf("  ID: %-5s Title: %s", doc.UserFacingID, doc.Title)
	}
}
