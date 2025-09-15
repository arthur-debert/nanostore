package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TestBatchIDResolution demonstrates the behavior when completing multiple items
// by their user-facing IDs. This test documents issue #16.
func TestBatchIDResolution(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.DefaultTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create three documents
	uuid1, _ := store.Add("First todo", nil)
	_, _ = store.Add("Second todo", nil)
	uuid3, _ := store.Add("Third todo", nil)

	// Verify initial IDs
	docs, _ := store.List(nanostore.ListOptions{})
	if len(docs) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(docs))
	}
	if docs[0].UserFacingID != "1" || docs[1].UserFacingID != "2" || docs[2].UserFacingID != "3" {
		t.Errorf("unexpected initial IDs: %s, %s, %s", docs[0].UserFacingID, docs[1].UserFacingID, docs[2].UserFacingID)
	}

	t.Run("completing items one at a time", func(t *testing.T) {
		// This demonstrates the safe approach

		// Complete item 1
		resolved1, err := store.ResolveUUID("1")
		if err != nil {
			t.Fatalf("failed to resolve ID 1: %v", err)
		}
		if resolved1 != uuid1 {
			t.Errorf("expected ID 1 to resolve to %s, got %s", uuid1, resolved1)
		}

		err = nanostore.TestSetStatusUpdate(store, resolved1, "completed")
		if err != nil {
			t.Fatalf("failed to complete first item: %v", err)
		}

		// After completing 1, IDs shift: 2→1, 3→2
		docs, _ = store.List(nanostore.ListOptions{Filters: map[string]interface{}{"status": "pending"}})
		if len(docs) != 2 {
			t.Fatalf("expected 2 pending documents, got %d", len(docs))
		}
		if docs[0].UserFacingID != "1" || docs[0].Title != "Second todo" {
			t.Errorf("expected 'Second todo' to be ID 1, got ID %s with title %s", docs[0].UserFacingID, docs[0].Title)
		}

		// Now complete what is currently ID 2 (originally ID 3)
		resolved2, err := store.ResolveUUID("2")
		if err != nil {
			t.Fatalf("failed to resolve ID 2: %v", err)
		}
		if resolved2 != uuid3 {
			t.Errorf("expected current ID 2 to resolve to %s (Third todo), got %s", uuid3, resolved2)
		}
	})
}

// TestBatchIDResolutionPattern demonstrates the correct pattern for batch operations
func TestBatchIDResolutionPattern(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.DefaultTestConfig())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents
	_, _ = store.Add("Task A", nil)
	_, _ = store.Add("Task B", nil)
	_, _ = store.Add("Task C", nil)
	_, _ = store.Add("Task D", nil)

	t.Run("correct batch completion pattern", func(t *testing.T) {
		// User wants to complete items 1 and 3
		targetIDs := []string{"1", "3"}

		// CORRECT: Resolve all IDs first
		var resolvedUUIDs []string
		for _, id := range targetIDs {
			uuid, err := store.ResolveUUID(id)
			if err != nil {
				t.Fatalf("failed to resolve ID %s: %v", id, err)
			}
			resolvedUUIDs = append(resolvedUUIDs, uuid)
		}

		// Then perform all operations
		for i, uuid := range resolvedUUIDs {
			err := nanostore.TestSetStatusUpdate(store, uuid, "completed")
			if err != nil {
				t.Fatalf("failed to complete item %s: %v", targetIDs[i], err)
			}
		}

		// Verify results
		pending, _ := store.List(nanostore.ListOptions{Filters: map[string]interface{}{"status": "pending"}})
		completed, _ := store.List(nanostore.ListOptions{Filters: map[string]interface{}{"status": "completed"}})

		if len(pending) != 2 {
			t.Errorf("expected 2 pending items, got %d", len(pending))
		}
		if len(completed) != 2 {
			t.Errorf("expected 2 completed items, got %d", len(completed))
		}

		// Verify the correct items were completed
		completedTitles := make(map[string]bool)
		for _, doc := range completed {
			completedTitles[doc.Title] = true
		}

		if !completedTitles["Task A"] || !completedTitles["Task C"] {
			t.Errorf("expected Task A and Task C to be completed, got: %v", completedTitles)
		}
	})

	t.Run("incorrect batch pattern - resolving IDs one at a time", func(t *testing.T) {
		// Reset by creating new store
		store2, _ := nanostore.New(":memory:", nanostore.DefaultTestConfig())
		defer func() { _ = store2.Close() }()

		uuid1, _ := store2.Add("Item 1", nil)
		_, _ = store2.Add("Item 2", nil)
		uuid3, _ := store2.Add("Item 3", nil)

		// INCORRECT: Resolving and completing one at a time
		// User wants to complete 1 and 3, but resolves after each operation

		// First complete ID 1
		_ = nanostore.TestSetStatusUpdate(store2, uuid1, "completed")

		// Now try to resolve ID 3 - but it's now ID 2!
		resolved3, err := store2.ResolveUUID("3")
		if err == nil {
			// ID 3 no longer exists, this should fail
			t.Errorf("expected ID 3 to not exist after completing ID 1, but resolved to %s", resolved3)
		}

		// What was ID 3 is now ID 2
		resolved2, _ := store2.ResolveUUID("2")
		if resolved2 != uuid3 {
			t.Errorf("expected current ID 2 to be original item 3 (uuid %s), got %s", uuid3, resolved2)
		}
	})
}

// TestBatchOperationStrategies demonstrates different strategies for handling batch operations
func TestBatchOperationStrategies(t *testing.T) {
	t.Run("reverse order strategy", func(t *testing.T) {
		store, _ := nanostore.New(":memory:", nanostore.DefaultTestConfig())
		defer func() { _ = store.Close() }()

		// Create items
		_, _ = store.Add("Item 1", nil)
		_, _ = store.Add("Item 2", nil)
		_, _ = store.Add("Item 3", nil)
		_, _ = store.Add("Item 4", nil)

		// Strategy: Complete in reverse order (4, 2, 1)
		// This avoids ID shifting affecting subsequent operations
		idsToComplete := []string{"4", "2", "1"}

		for _, id := range idsToComplete {
			uuid, err := store.ResolveUUID(id)
			if err != nil {
				t.Fatalf("failed to resolve ID %s: %v", id, err)
			}

			err = nanostore.TestSetStatusUpdate(store, uuid, "completed")
			if err != nil {
				t.Fatalf("failed to complete ID %s: %v", id, err)
			}
		}

		// Verify only Item 3 remains pending
		pending, _ := store.List(nanostore.ListOptions{Filters: map[string]interface{}{"status": "pending"}})
		if len(pending) != 1 || pending[0].Title != "Item 3" {
			t.Errorf("expected only 'Item 3' to remain pending, got %d items", len(pending))
		}
	})

	t.Run("pre-resolution with validation", func(t *testing.T) {
		store, _ := nanostore.New(":memory:", nanostore.DefaultTestConfig())
		defer func() { _ = store.Close() }()

		// Create items
		uuids := make(map[string]string)
		uuids["A"], _ = store.Add("Task A", nil)
		uuids["B"], _ = store.Add("Task B", nil)
		uuids["C"], _ = store.Add("Task C", nil)

		// Helper function that safely completes multiple items
		completeMultiple := func(ids []string) error {
			// Step 1: Resolve all IDs
			type resolved struct {
				userID string
				uuid   string
				title  string
			}
			var items []resolved

			// Get current state to help with debugging
			currentDocs, _ := store.List(nanostore.ListOptions{})
			idToDoc := make(map[string]nanostore.Document)
			for _, doc := range currentDocs {
				idToDoc[doc.UserFacingID] = doc
			}

			// Resolve each ID
			for _, id := range ids {
				uuid, err := store.ResolveUUID(id)
				if err != nil {
					return err
				}

				doc, exists := idToDoc[id]
				if !exists {
					t.Logf("Warning: ID %s exists but not in current listing", id)
				}

				items = append(items, resolved{
					userID: id,
					uuid:   uuid,
					title:  doc.Title,
				})
			}

			// Step 2: Complete all items
			for _, item := range items {
				t.Logf("Completing ID %s (UUID: %s, Title: %s)", item.userID, item.uuid, item.title)
				err := nanostore.TestSetStatusUpdate(store, item.uuid, "completed")
				if err != nil {
					return err
				}
			}

			return nil
		}

		// Use the helper to complete items 1 and 3
		err := completeMultiple([]string{"1", "3"})
		if err != nil {
			t.Fatalf("failed to complete items: %v", err)
		}

		// Verify results
		pending, _ := store.List(nanostore.ListOptions{Filters: map[string]interface{}{"status": "pending"}})
		if len(pending) != 1 || pending[0].Title != "Task B" {
			t.Errorf("expected only Task B to remain pending")
		}
	})
}
