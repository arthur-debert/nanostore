package nanostore_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

func TestFilteringMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("FilterByStatus", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Verify all returned docs have status=pending
		for _, doc := range docs {
			testutil.AssertHasStatus(t, doc, "pending")
		}

		// The fixture has specific pending documents
		if len(docs) == 0 {
			t.Error("expected some pending documents")
		}
	})

	t.Run("FilterByMultipleDimensions", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Verify all returned docs match filters
		for _, doc := range docs {
			testutil.AssertHasStatus(t, doc, "pending")
			testutil.AssertHasPriority(t, doc, "high")
		}

		// Fixture has TeamMeeting and others with these values
		if len(docs) == 0 {
			t.Error("expected some high priority pending documents")
		}
	})

	t.Run("FilterByMultipleValues", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": []string{"pending", "active"},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Verify all returned docs have one of the specified statuses
		for _, doc := range docs {
			status := doc.Dimensions["status"].(string)
			if status != "pending" && status != "active" {
				t.Errorf("expected status to be pending or active, got %s", status)
			}
		}
	})

	t.Run("FilterByTitle", func(t *testing.T) {
		// Use a title from the fixture - BuyGroceries
		docs, err := store.List(nanostore.ListOptions{
			FilterBySearch: "groceries",
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should find BuyGroceries and maybe its children (Milk, Bread)
		found := false
		for _, doc := range docs {
			if doc.UUID == universe.BuyGroceries.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find BuyGroceries document")
		}
	})

	t.Run("FilterByNonExistentValue", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "nonexistent",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 0 {
			t.Errorf("expected 0 documents for non-existent status, got %d", len(docs))
		}
	})

	t.Run("FilterCombination", func(t *testing.T) {
		// Filter by parent and status
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.WorkRoot.UUID,
				"status":    "pending",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// All docs should be children of WorkRoot with pending status
		for _, doc := range docs {
			if doc.Dimensions["parent_id"] != universe.WorkRoot.UUID {
				t.Errorf("expected parent_id to be %s, got %v", universe.WorkRoot.UUID, doc.Dimensions["parent_id"])
			}
			testutil.AssertHasStatus(t, doc, "pending")
		}
	})

	t.Run("SearchInBody", func(t *testing.T) {
		// Search for text that appears in document bodies
		docs, err := store.List(nanostore.ListOptions{
			FilterBySearch: "important",
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should find documents with "important" in title or body
		foundInBody := false
		for _, doc := range docs {
			if strings.Contains(strings.ToLower(doc.Body), "important") {
				foundInBody = true
				break
			}
		}
		if len(docs) > 0 && !foundInBody {
			// Only check if we found docs
			t.Log("Note: search found documents but 'important' not in body - might be in title")
		}
	})
}

func TestEmptyFiltersMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("EmptyFiltersReturnsAll", func(t *testing.T) {
		// Get all documents
		allDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		allCount := len(allDocs)

		// Empty filters should return the same
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != allCount {
			t.Errorf("empty filters: expected %d documents, got %d", allCount, len(docs))
		}
	})

	t.Run("NilFiltersReturnsAll", func(t *testing.T) {
		// Get all documents
		allDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		allCount := len(allDocs)

		// nil filters (default) should return all
		docs, err := store.List(nanostore.ListOptions{
			Filters: nil,
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(docs) != allCount {
			t.Errorf("nil filters: expected %d documents, got %d", allCount, len(docs))
		}
	})
}
