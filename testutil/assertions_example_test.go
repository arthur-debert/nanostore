// Package testutil_test demonstrates the assertion helpers in action
package testutil_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/testutil"
)

// TestAssertionHelpersDemo shows how assertion helpers simplify tests
func TestAssertionHelpersDemo(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("document count assertions", func(t *testing.T) {
		// Query pending documents
		pendingDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"status": "pending"},
		})

		// Simple assertion with context
		testutil.AssertDocumentCount(t, pendingDocs, 8, "with status=pending")

		// Verify specific document exists
		testutil.AssertDocumentExists(t, pendingDocs, universe.BuyGroceries.UUID)

		// Verify completed document is NOT in pending results
		testutil.AssertDocumentNotExists(t, pendingDocs, universe.ReadBook.UUID)
	})

	t.Run("hierarchy assertions", func(t *testing.T) {
		// Check child counts for PersonalRoot
		testutil.AssertChildCount(t, store, universe.PersonalRoot.UUID, 3)

		// Check specific status counts
		testutil.AssertPendingChildCount(t, store, universe.PersonalRoot.UUID, 1) // BuyGroceries
		testutil.AssertActiveChildCount(t, store, universe.PersonalRoot.UUID, 1)  // ExerciseRoutine
		testutil.AssertDoneChildCount(t, store, universe.PersonalRoot.UUID, 1)    // ReadBook

		// Verify root status
		testutil.AssertIsRoot(t, universe.PersonalRoot)

		// Verify parent relationships
		testutil.AssertHasParent(t, universe.BuyGroceries, universe.PersonalRoot.UUID)
	})

	t.Run("dimension assertions", func(t *testing.T) {
		// Get all high priority docs
		highPriorityDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"priority": "high"},
		})

		// All should have priority=high
		testutil.AssertAllHaveDimension(t, highPriorityDocs, "priority", "high")

		// Check multiple dimensions on a single doc
		testutil.AssertDimensionValues(t, universe.TeamMeeting, map[string]string{
			"status":   "pending",
			"priority": "high",
			"category": "work",
		})

		// Convenience assertions for common dimensions
		testutil.AssertHasStatus(t, universe.ReadBook, "done")
		testutil.AssertHasPriority(t, universe.ExerciseRoutine, "high")
	})

	t.Run("ordering assertions", func(t *testing.T) {
		// Get documents ordered by title
		orderedDocs, _ := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{{Column: "title", Descending: false}},
			Limit:   intPtr(5),
		})

		// Verify ordering
		testutil.AssertOrderedBy(t, orderedDocs, "title", true)

		// For specific order verification
		personalDocs := []nanostore.Document{
			universe.BuyGroceries,
			universe.ExerciseRoutine,
			universe.ReadBook,
		}
		expectedOrder := []string{
			universe.BuyGroceries.UUID,
			universe.ExerciseRoutine.UUID,
			universe.ReadBook.UUID,
		}
		testutil.AssertIDsInOrder(t, personalDocs, expectedOrder)
	})

	t.Run("query assertions", func(t *testing.T) {
		// Verify a query returns exactly the expected documents
		testutil.AssertQueryReturns(t, store,
			nanostore.ListOptions{
				Filters: map[string]interface{}{
					"parent_id": universe.PersonalRoot.UUID,
				},
			},
			universe.BuyGroceries.UUID,
			universe.ExerciseRoutine.UUID,
			universe.ReadBook.UUID,
		)

		// Verify an impossible query returns nothing
		testutil.AssertQueryEmpty(t, store,
			nanostore.ListOptions{
				Filters: map[string]interface{}{
					"status":   "done",
					"priority": "high",
					"category": "personal",
					"activity": "deleted", // No personal done high-priority deleted items
				},
			},
		)

		// Search assertions
		testutil.AssertSearchFinds(t, store, "pack", 2)
		testutil.AssertSearchFinds(t, store, "zzz-no-match", 0)
	})

	t.Run("complex condition assertions", func(t *testing.T) {
		allDocs, _ := store.List(nanostore.ListOptions{})

		// Find any document with emoji
		testutil.AssertContainsDocument(t, allDocs,
			func(d nanostore.Document) bool {
				return d.UUID == universe.UnicodeEmoji.UUID
			},
			"document with emoji in title",
		)

		// Verify all root documents have no parent_id
		roots := universe.GetRootDocuments()
		testutil.AssertAllDocuments(t, roots,
			func(d nanostore.Document) bool {
				_, hasParent := d.Dimensions["parent_id"]
				return !hasParent
			},
			"root document (no parent_id)",
		)
	})

	t.Run("mixed parent assertions", func(t *testing.T) {
		// The MixedParent has exactly one child of each status
		testutil.AssertChildCount(t, store, universe.MixedParent.UUID, 3)
		testutil.AssertPendingChildCount(t, store, universe.MixedParent.UUID, 1)
		testutil.AssertActiveChildCount(t, store, universe.MixedParent.UUID, 1)
		testutil.AssertDoneChildCount(t, store, universe.MixedParent.UUID, 1)
	})
}

// TestAssertionHelpersBefore shows what tests looked like without helpers
func TestAssertionHelpersBefore(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	// OLD WAY: Verbose and repetitive
	results, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 8 {
		t.Errorf("expected 8 documents, got %d", len(results))
	}

	found := false
	for _, doc := range results {
		if doc.UUID == universe.BuyGroceries.UUID {
			found = true
			break
		}
	}
	if !found {
		t.Error("BuyGroceries not found in results")
	}
}

// TestAssertionHelpersAfter shows the same test with helpers
func TestAssertionHelpersAfter(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	// NEW WAY: Clear and concise
	results, _ := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})

	testutil.AssertDocumentCount(t, results, 8)
	testutil.AssertDocumentExists(t, results, universe.BuyGroceries.UUID)
}
