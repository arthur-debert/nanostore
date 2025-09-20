package nanostore_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)


import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

func TestBasicOperationsMigrated(t *testing.T) {
	var store nanostore.Store
	var universe *testutil.UniverseData

	// Initial load
	store, _ = testutil.LoadUniverse(t)

	t.Run("Add", func(t *testing.T) {
		// Add a document with default dimensions
		id1, err := store.Add("New Task 1", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
		if id1 == "" {
			t.Fatal("expected non-empty ID")
		}

		// Add with specific dimensions
		id2, err := store.Add("New Task 2", map[string]interface{}{
			"status":   "done",
			"priority": "high",
		})
		if err != nil {
			t.Fatalf("failed to add document with dimensions: %v", err)
		}
		if id2 == "" {
			t.Fatal("expected non-empty ID")
		}

		// Verify the documents exist
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		// Count new docs (should have original fixture docs + 2 new ones)
		newDocsFound := 0
		for _, doc := range docs {
			if doc.UUID == id1 || doc.UUID == id2 {
				newDocsFound++
			}
		}
		if newDocsFound != 2 {
			t.Errorf("expected 2 new documents, found %d", newDocsFound)
		}
	})

	// Reload universe for clean state
	store, universe = testutil.LoadUniverse(t)

	t.Run("Update", func(t *testing.T) {
		// Update an existing document from fixture
		newTitle := "Updated Title"
		newBody := "Updated body content"

		err := store.Update(universe.BuyGroceries.UUID, nanostore.UpdateRequest{
			Title: &newTitle,
			Body:  &newBody,
			Dimensions: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatalf("failed to update document: %v", err)
		}

		// Verify the update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.BuyGroceries.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		testutil.AssertDocumentCount(t, docs, 1)
		if docs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
		}
		if docs[0].Body != newBody {
			t.Errorf("expected body %q, got %q", newBody, docs[0].Body)
		}
		testutil.AssertHasStatus(t, docs[0], "done")
	})

	// Reload for clean state
	store, universe = testutil.LoadUniverse(t)

	t.Run("Delete", func(t *testing.T) {
		// Count initial documents
		initialDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		initialCount := len(initialDocs)

		// Delete a leaf document (no children)
		err = store.Delete(universe.UnicodeEmoji.UUID, false)
		if err != nil {
			t.Fatalf("failed to delete document: %v", err)
		}

		// Verify deletion
		remainingDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingDocs) != initialCount-1 {
			t.Errorf("expected %d documents after delete, got %d", initialCount-1, len(remainingDocs))
		}

		// Ensure the document is gone
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.UnicodeEmoji.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Error("deleted document still exists")
		}
	})

	// Reload for clean state
	store, universe = testutil.LoadUniverse(t)

	t.Run("InvalidDimensionValue", func(t *testing.T) {
		// Try to add with invalid dimension value
		_, err := store.Add("Invalid Task", map[string]interface{}{
			"status": "invalid_status",
		})
		if err == nil {
			t.Fatal("expected error for invalid dimension value, got nil")
		}

		// Try to update with invalid dimension value
		err = store.Update(universe.ExerciseRoutine.UUID, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"priority": "invalid_priority",
			},
		})
		if err == nil {
			t.Fatal("expected error for invalid dimension value, got nil")
		}
	})

	// Reload for clean state
	store, universe = testutil.LoadUniverse(t)

	t.Run("ParentChild", func(t *testing.T) {
		// The fixture already has parent-child relationships
		// Test that we can query children of a parent
		children, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(children) == 0 {
			t.Error("expected PersonalRoot to have children")
		}

		// Try to delete a parent with children (should fail without cascade)
		err = store.Delete(universe.PersonalRoot.UUID, false)
		if err == nil {
			t.Error("expected error when deleting parent with children without cascade")
		}

		// Delete with cascade should work
		err = store.Delete(universe.WorkRoot.UUID, true)
		if err != nil {
			t.Fatalf("failed to delete parent with cascade: %v", err)
		}

		// Verify all WorkRoot children are gone
		workChildren, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.WorkRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(workChildren) != 0 {
			t.Errorf("expected 0 children after cascade delete, found %d", len(workChildren))
		}
	})
}
