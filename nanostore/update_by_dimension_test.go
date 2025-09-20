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

func TestUpdateByDimensionMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("update by single dimension", func(t *testing.T) {
		// Count documents with pending status before update
		pendingDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedCount := len(pendingDocs)

		// Update all pending to done
		newTitle := "Marked as Done"
		count, err := store.UpdateByDimension(
			map[string]interface{}{"status": "pending"},
			nanostore.UpdateRequest{
				Title: &newTitle,
				Dimensions: map[string]interface{}{
					"status": "done",
				},
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != expectedCount {
			t.Errorf("expected to update %d documents, updated %d", expectedCount, count)
		}

		// Verify update - check that pending are now done
		updatedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Count how many have our new title
		withNewTitle := 0
		for _, doc := range updatedDocs {
			if doc.Title == newTitle {
				withNewTitle++
			}
		}
		if withNewTitle != count {
			t.Errorf("expected %d documents with updated title, got %d", count, withNewTitle)
		}
	})

	// Reload for clean state
	store, _ = testutil.LoadUniverse(t)

	t.Run("update by multiple dimensions", func(t *testing.T) {
		// Find active+medium priority documents
		targetDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "active",
				"priority": "medium",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedCount := len(targetDocs)

		// Update body and priority
		newBody := "Escalated to high priority"
		count, err := store.UpdateByDimension(
			map[string]interface{}{
				"status":   "active",
				"priority": "medium",
			},
			nanostore.UpdateRequest{
				Body: &newBody,
				Dimensions: map[string]interface{}{
					"priority": "high",
				},
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != expectedCount {
			t.Errorf("expected to update %d documents, updated %d", expectedCount, count)
		}

		// Verify updates
		updatedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "active",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		// Should find the documents we just updated
		foundWithNewBody := 0
		for _, doc := range updatedDocs {
			if doc.Body == newBody {
				foundWithNewBody++
			}
		}
		if foundWithNewBody != count {
			t.Errorf("expected %d documents with new body, found %d", count, foundWithNewBody)
		}
	})

	t.Run("update with no matches", func(t *testing.T) {
		newTitle := "Should not update anything"
		count, err := store.UpdateByDimension(
			map[string]interface{}{"status": "nonexistent"},
			nanostore.UpdateRequest{
				Title: &newTitle,
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected to update 0 documents, updated %d", count)
		}
	})

	t.Run("invalid dimension value in update", func(t *testing.T) {
		_, err := store.UpdateByDimension(
			map[string]interface{}{"status": "active"},
			nanostore.UpdateRequest{
				Dimensions: map[string]interface{}{
					"status": "invalid_status",
				},
			},
		)
		if err == nil {
			t.Error("expected error for invalid dimension value, got nil")
		}
	})
}

func TestUpdateByDimensionWithNonDimensionFieldsMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	// Add test documents with custom data fields
	id1, err := store.Add("Version 1.0 Doc A", map[string]interface{}{
		"status":        "active",
		"priority":      "medium",
		"_data.version": 1.0,
		"_data.author":  "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	id2, err := store.Add("Version 2.0 Doc B", map[string]interface{}{
		"status":        "active",
		"priority":      "low",
		"_data.version": 2.0,
		"_data.author":  "bob",
	})
	if err != nil {
		t.Fatal(err)
	}

	id3, err := store.Add("Version 1.0 Doc C", map[string]interface{}{
		"status":        "pending",
		"priority":      "high",
		"_data.version": 1.0,
		"_data.author":  "alice",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("update by non-dimension field", func(t *testing.T) {
		// Update all documents authored by alice
		count, err := store.UpdateByDimension(
			map[string]interface{}{"_data.author": "alice"},
			nanostore.UpdateRequest{
				Dimensions: map[string]interface{}{
					"_data.version": 2.5,
					"_data.status":  "updated",
				},
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected to update 2 documents, updated %d", count)
		}

		// Verify updates
		for _, id := range []string{id1, id3} {
			docs, err := store.List(nanostore.ListOptions{
				Filters: map[string]interface{}{"uuid": id},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(docs) != 1 {
				t.Fatalf("document %s not found", id)
			}
			if docs[0].Dimensions["_data.version"] != 2.5 {
				t.Errorf("expected version 2.5, got %v", docs[0].Dimensions["_data.version"])
			}
			if docs[0].Dimensions["_data.status"] != "updated" {
				t.Errorf("expected status 'updated', got %v", docs[0].Dimensions["_data.status"])
			}
		}
	})

	t.Run("mixed dimension and non-dimension update", func(t *testing.T) {
		// Update documents matching both dimension and non-dimension criteria
		newTitle := "Mixed Update"
		count, err := store.UpdateByDimension(
			map[string]interface{}{
				"status":        "active",
				"_data.version": 2.5,
			},
			nanostore.UpdateRequest{
				Title: &newTitle,
				Dimensions: map[string]interface{}{
					"priority":       "high",
					"_data.category": "processed",
				},
			},
		)
		if err != nil {
			t.Fatalf("UpdateByDimension failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected to update 1 document, updated %d", count)
		}

		// Verify the update
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"uuid": id1},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatal("document not found")
		}
		if docs[0].Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, docs[0].Title)
		}
		testutil.AssertHasPriority(t, docs[0], "high")
		if docs[0].Dimensions["_data.category"] != "processed" {
			t.Errorf("expected category 'processed', got %v", docs[0].Dimensions["_data.category"])
		}
	})

	// Clean up test documents
	for _, id := range []string{id1, id2, id3} {
		_ = store.Delete(id, false)
	}
}
