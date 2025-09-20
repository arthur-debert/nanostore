package store_test

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

func TestDeleteWithSmartIDMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("delete using UUID", func(t *testing.T) {
		// Count initial documents
		initialDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		initialCount := len(initialDocs)

		// Delete using UUID - use a leaf document with no children
		err = store.Delete(universe.SpecialChars.UUID, false)
		if err != nil {
			t.Fatalf("Delete with UUID failed: %v", err)
		}

		// Verify deletion
		remainingDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingDocs) != initialCount-1 {
			t.Errorf("expected %d documents after delete, got %d", initialCount-1, len(remainingDocs))
		}

		// Ensure the specific document was deleted
		for _, doc := range remainingDocs {
			if doc.UUID == universe.SpecialChars.UUID {
				t.Error("deleted document still exists")
			}
		}
	})

	// Reload universe for next test (since we deleted a document)
	store, universe = testutil.LoadUniverse(t)

	t.Run("delete using SimpleID", func(t *testing.T) {
		// Get fresh SimpleID for the document we want to delete
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.EmptyTitle.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}
		simpleID := docs[0].SimpleID

		// Count documents before deletion
		allDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		countBefore := len(allDocs)

		// Delete using SimpleID
		err = store.Delete(simpleID, false)
		if err != nil {
			t.Fatalf("Delete with SimpleID %q failed: %v", simpleID, err)
		}

		// Verify deletion
		remainingDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingDocs) != countBefore-1 {
			t.Errorf("expected %d documents after delete, got %d", countBefore-1, len(remainingDocs))
		}

		// Ensure the specific document was deleted
		for _, doc := range remainingDocs {
			if doc.UUID == universe.EmptyTitle.UUID {
				t.Error("deleted document still exists")
			}
		}
	})

	t.Run("delete with invalid ID", func(t *testing.T) {
		err := store.Delete("invalid-id-12345", false)
		if err == nil {
			t.Error("expected error for invalid ID, got nil")
		}
	})
}

func TestDeleteCascadeWithSmartIDMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("delete parent with cascade using SimpleID", func(t *testing.T) {
		// Get the parent's SimpleID
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.Level4Task.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}
		parentSimpleID := docs[0].SimpleID

		// Count children before deletion
		children, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.Level4Task.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		childCount := len(children)
		if childCount == 0 {
			t.Fatal("expected Level4Task to have children")
		}

		// Delete parent with cascade using SimpleID
		err = store.Delete(parentSimpleID, true)
		if err != nil {
			t.Fatalf("Delete with cascade using SimpleID %q failed: %v", parentSimpleID, err)
		}

		// Verify parent is deleted
		docs, err = store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.Level4Task.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Error("parent document still exists after cascade delete")
		}

		// Verify children are deleted
		children, err = store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.Level4Task.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(children) != 0 {
			t.Errorf("expected 0 children after cascade delete, found %d", len(children))
		}
	})

	// Reload universe for next test
	store, universe = testutil.LoadUniverse(t)

	t.Run("delete parent without cascade should fail if children exist", func(t *testing.T) {
		// Get the parent's SimpleID
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}
		parentSimpleID := docs[0].SimpleID

		// Verify it has children
		children, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(children) == 0 {
			t.Fatal("expected PersonalRoot to have children")
		}

		// Try to delete without cascade - should fail
		err = store.Delete(parentSimpleID, false)
		if err == nil {
			t.Error("expected error when deleting parent with children without cascade, got nil")
		}

		// Verify parent still exists
		docs, err = store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Error("parent document was deleted despite having children")
		}
	})
}
