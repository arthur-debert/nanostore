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

func TestDeleteByDimensionMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("delete by single dimension", func(t *testing.T) {
		// Count documents with done status before deletion
		doneDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedDeleted := len(doneDocs)

		// Delete all documents with status "done"
		count, err := store.DeleteByDimension(map[string]interface{}{
			"status": "done",
		})
		if err != nil {
			t.Fatalf("DeleteByDimension failed: %v", err)
		}
		if count != expectedDeleted {
			t.Errorf("expected to delete %d documents, deleted %d", expectedDeleted, count)
		}

		// Verify deletion
		remainingDone, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingDone) != 0 {
			t.Errorf("expected 0 documents with status=done after deletion, got %d", len(remainingDone))
		}
	})

	// Reload for next test
	store, _ = testutil.LoadUniverse(t)

	t.Run("delete by multiple dimensions", func(t *testing.T) {
		// Count documents with pending status and high priority
		targetDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedDeleted := len(targetDocs)

		// Delete by multiple dimensions
		count, err := store.DeleteByDimension(map[string]interface{}{
			"status":   "pending",
			"priority": "high",
		})
		if err != nil {
			t.Fatalf("DeleteByDimension failed: %v", err)
		}
		if count != expectedDeleted {
			t.Errorf("expected to delete %d documents, deleted %d", expectedDeleted, count)
		}

		// Verify deletion
		remainingTarget, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingTarget) != 0 {
			t.Error("documents matching criteria still exist after deletion")
		}
	})

	// Reload for next test
	store, _ = testutil.LoadUniverse(t)

	t.Run("delete with no matches", func(t *testing.T) {
		// Try to delete with a non-existent status value
		count, err := store.DeleteByDimension(map[string]interface{}{
			"status": "nonexistent",
		})
		if err != nil {
			t.Fatalf("DeleteByDimension failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to delete 0 documents, deleted %d", count)
		}
	})

	// Reload for next test
	store, universe := testutil.LoadUniverse(t)

	t.Run("delete by parent_id", func(t *testing.T) {
		// Delete all children of PersonalRoot
		children, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedDeleted := len(children)

		count, err := store.DeleteByDimension(map[string]interface{}{
			"parent_id": universe.PersonalRoot.UUID,
		})
		if err != nil {
			t.Fatalf("DeleteByDimension failed: %v", err)
		}
		if count != expectedDeleted {
			t.Errorf("expected to delete %d documents, deleted %d", expectedDeleted, count)
		}

		// Verify all PersonalRoot children are gone
		remainingChildren, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(remainingChildren) != 0 {
			t.Errorf("expected 0 children after deletion, got %d", len(remainingChildren))
		}

		// But PersonalRoot itself should still exist
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		testutil.AssertDocumentCount(t, docs, 1)
	})
}

func TestDeleteByDimensionWithNonDimensionFieldsMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	// First, let's add some documents with custom data fields
	id1, err := store.Add("Version 1.0 Doc", map[string]interface{}{
		"status":        "active",
		"priority":      "medium",
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	id2, err := store.Add("Version 2.0 Doc", map[string]interface{}{
		"status":        "active",
		"priority":      "high",
		"_data.version": "2.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	id3, err := store.Add("Another Version 1.0 Doc", map[string]interface{}{
		"status":        "pending",
		"priority":      "low",
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Delete by non-dimension field
	count, err := store.DeleteByDimension(map[string]interface{}{
		"_data.version": "1.0",
	})
	if err != nil {
		t.Fatalf("DeleteByDimension failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected to delete 2 documents, deleted %d", count)
	}

	// Verify deletion - the 2.0 version should remain
	docs, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"uuid": id2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error("version 2.0 document was deleted")
	}

	// Verify the 1.0 versions are gone
	for _, id := range []string{id1, id3} {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": id,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 0 {
			t.Errorf("document %s with version 1.0 still exists", id)
		}
	}
}
