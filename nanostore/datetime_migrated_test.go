package nanostore_test

import (
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/testutil"
)

func TestDateTimeFilteringMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("UpdateChangesTimestamp", func(t *testing.T) {
		// Create a new document
		docID, err := store.Add("Timestamp test", map[string]interface{}{
			"status":   "active",
			"priority": "medium",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get the document to check initial timestamps
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": docID,
			},
		})
		if err != nil || len(docs) != 1 {
			t.Fatal("failed to get document")
		}
		originalDoc := docs[0]

		// Wait a moment to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Update the document
		newTitle := "Updated timestamp test"
		err = store.Update(docID, nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get the updated document
		updatedDocs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": docID,
			},
		})
		if err != nil || len(updatedDocs) != 1 {
			t.Fatal("failed to get updated document")
		}
		updatedDoc := updatedDocs[0]

		// Verify updated_at changed
		if !updatedDoc.UpdatedAt.After(originalDoc.UpdatedAt) {
			t.Error("expected updated_at to be after original updated_at")
		}

		// Clean up
		_ = store.Delete(docID, false)
	})
}

func TestDateTimeOrderingMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("OrderByCreatedAtAscending", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: false},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify documents are ordered by created_at ascending
		for i := 1; i < len(docs); i++ {
			if docs[i-1].CreatedAt.After(docs[i].CreatedAt) {
				t.Errorf("documents not in created_at ascending order at positions %d,%d", i-1, i)
			}
		}
	})

	t.Run("OrderByCreatedAtDescending", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify documents are ordered by created_at descending
		for i := 1; i < len(docs); i++ {
			if docs[i-1].CreatedAt.Before(docs[i].CreatedAt) {
				t.Errorf("documents not in created_at descending order at positions %d,%d", i-1, i)
			}
		}
	})

	t.Run("FilterAndOrderByDates", func(t *testing.T) {
		// Get some recent documents and filter/order them
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify all are pending and ordered by date
		for i, doc := range docs {
			testutil.AssertHasStatus(t, doc, "pending")
			if i > 0 && docs[i-1].CreatedAt.Before(docs[i].CreatedAt) {
				t.Error("filtered results not properly ordered by created_at desc")
			}
		}
	})

	t.Run("MixedDateAndOtherOrdering", func(t *testing.T) {
		// Order by status first, then by created_at within each status
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "status", Descending: false},
				{Column: "created_at", Descending: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify within each status group, dates are descending
		var lastStatus string
		var lastTime time.Time
		for _, doc := range docs {
			status := doc.Dimensions["status"].(string)
			if status == lastStatus && !lastTime.IsZero() && doc.CreatedAt.After(lastTime) {
				t.Errorf("within status %q, created_at not in descending order", status)
			}
			if status != lastStatus {
				lastStatus = status
				lastTime = time.Time{} // Reset for new group
			} else {
				lastTime = doc.CreatedAt
			}
		}
	})
}

func TestDateTimeEdgeCasesMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("FilterByFutureDate", func(t *testing.T) {
		// Filter by a date far in the future
		futureDate := time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"created_at": futureDate,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != 0 {
			t.Error("expected no documents with future created_at")
		}
	})

	t.Run("FilterByVeryOldDate", func(t *testing.T) {
		// Filter by a date in the past
		oldDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"created_at": oldDate,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != 0 {
			t.Error("expected no documents with 1970 created_at")
		}
	})
}
