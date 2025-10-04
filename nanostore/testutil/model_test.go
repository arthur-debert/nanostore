// Package testutil_test provides THE model test file demonstrating best practices
// for testing nanostore functionality.
//
// IMPORTANT: This file serves as the reference implementation for ALL tests in the
// nanostore codebase. If you're writing or modifying tests, start here to understand
// the proper patterns and available utilities.
//
// Key Principles:
// 1. Use LoadUniverse() for one-line test setup with comprehensive fixture data
// 2. Use assertion helpers instead of verbose manual checks
// 3. Leverage the fixture data instead of creating test data in each test
// 4. Only create a fresh store when testing store initialization or configuration
package testutil_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/store"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
	"github.com/arthur-debert/nanostore/types"
)

// =============================================================================
// BASIC PATTERN: LoadUniverse() provides everything you need
// =============================================================================

// TestBasicFilteringPattern demonstrates the fundamental testing pattern.
// This is how 90% of your tests should look - simple, focused, and leveraging
// the fixture data.
func TestBasicFilteringPattern(t *testing.T) {
	// ONE LINE SETUP - This gives you:
	// - A fully populated store with 28+ documents
	// - A universe object with typed access to specific test documents
	// - Automatic cleanup when the test ends
	store, universe := testutil.LoadUniverse(t)

	t.Run("filter by single dimension", func(t *testing.T) {
		// Query using standard store API
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Use assertion helpers for clear, concise verification
		// The fixture has exactly 8 pending documents - this is stable test data
		testutil.AssertDocumentCount(t, results, 8, "pending documents")

		// Verify specific documents are included
		testutil.AssertDocumentExists(t, results, universe.BuyGroceries.UUID)
		testutil.AssertDocumentExists(t, results, universe.TeamMeeting.UUID)

		// Verify completed documents are excluded
		testutil.AssertDocumentNotExists(t, results, universe.ReadBook.UUID)
	})

	t.Run("filter by multiple dimensions", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Every result should match both criteria
		testutil.AssertAllHaveDimension(t, results, "status", "pending")
		testutil.AssertAllHaveDimension(t, results, "priority", "high")
	})
}

// =============================================================================
// HIERARCHICAL RELATIONSHIPS: The fixture provides a rich hierarchy
// =============================================================================

// TestHierarchicalPatterns shows how to test parent-child relationships.
// The fixture includes documents up to 5 levels deep with various configurations.
func TestHierarchicalPatterns(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("query direct children", func(t *testing.T) {
		// The universe object provides typed access to key documents
		// PersonalRoot has exactly 3 children: BuyGroceries, ExerciseRoutine, ReadBook
		testutil.AssertChildCount(t, store, universe.PersonalRoot.UUID, 3)

		// You can also check children by status
		testutil.AssertPendingChildCount(t, store, universe.PersonalRoot.UUID, 1)
		testutil.AssertActiveChildCount(t, store, universe.PersonalRoot.UUID, 1)
		testutil.AssertDoneChildCount(t, store, universe.PersonalRoot.UUID, 1)
	})

	t.Run("verify parent relationships", func(t *testing.T) {
		// Assertions make parent-child verification simple
		testutil.AssertHasParent(t, universe.BuyGroceries, universe.PersonalRoot.UUID)
		testutil.AssertHasParent(t, universe.Milk, universe.BuyGroceries.UUID)

		// Root documents have no parent
		testutil.AssertIsRoot(t, universe.PersonalRoot)
		testutil.AssertIsRoot(t, universe.WorkRoot)
	})

	t.Run("deep hierarchy navigation", func(t *testing.T) {
		// The fixture includes a 5-level deep hierarchy for testing
		// Level5Task -> Level4Task -> Level3Task -> PrepareAgenda -> TeamMeeting
		testutil.AssertHasParent(t, universe.Level5Task, universe.Level4Task.UUID)

		// Verify the deep nesting
		if universe.Level5Task.SimpleID == "" {
			t.Error("deep nested document should have a SimpleID")
		}
	})
}

// =============================================================================
// SEARCH FUNCTIONALITY: Test text search without creating test data
// =============================================================================

// TestSearchPatterns demonstrates testing search functionality.
func TestSearchPatterns(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("search in titles", func(t *testing.T) {
		// The fixture includes documents designed for search testing
		testutil.AssertSearchFinds(t, store, "pack", 2)      // PackForTrip, PackLunch
		testutil.AssertSearchFinds(t, store, "groceries", 1) // BuyGroceries
		testutil.AssertSearchFinds(t, store, "nonexistent", 0)
	})

	t.Run("search with filters", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			FilterBySearch: "pack",
			Filters: map[string]interface{}{
				"status": "done",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Should only find PackLunch (done), not PackForTrip (pending)
		testutil.AssertDocumentCount(t, results, 1, "pack + done")
	})
}

// =============================================================================
// COMPLEX SCENARIOS: The fixture includes edge cases and special scenarios
// =============================================================================

// TestComplexScenarios shows how the fixture handles edge cases.
func TestComplexScenarios(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("mixed parent with children of all statuses", func(t *testing.T) {
		// MixedParent is designed to have exactly one child of each status
		testutil.AssertChildCount(t, store, universe.MixedParent.UUID, 3)
		testutil.AssertPendingChildCount(t, store, universe.MixedParent.UUID, 1)
		testutil.AssertActiveChildCount(t, store, universe.MixedParent.UUID, 1)
		testutil.AssertDoneChildCount(t, store, universe.MixedParent.UUID, 1)
	})

	t.Run("deleted parent with active children", func(t *testing.T) {
		// Tests the edge case of a deleted parent with active children
		if universe.DeletedParent.Dimensions["activity"] != "deleted" {
			t.Error("DeletedParent should have activity=deleted")
		}
		if universe.OrphanChild.Dimensions["activity"] != "active" {
			t.Error("OrphanChild should have activity=active")
		}
		testutil.AssertHasParent(t, universe.OrphanChild, universe.DeletedParent.UUID)
	})

	t.Run("special characters and unicode", func(t *testing.T) {
		// The fixture includes documents with special cases
		if universe.EmptyTitle.Title != "" {
			t.Error("EmptyTitle should have empty title")
		}

		expectedEmoji := "🚀 Launch project 💥"
		if universe.UnicodeEmoji.Title != expectedEmoji {
			t.Errorf("expected %q, got %q", expectedEmoji, universe.UnicodeEmoji.Title)
		}
	})
}

// =============================================================================
// ASSERTION SHOWCASE: Available assertion helpers
// =============================================================================

// TestAssertionHelpers demonstrates the full range of assertion helpers.
// These helpers make tests more readable and provide better error messages.
func TestAssertionHelpers(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("document assertions", func(t *testing.T) {
		docs, _ := store.List(nanostore.ListOptions{})

		// Count assertions with context
		testutil.AssertDocumentCount(t, docs, 28, "total documents")

		// Existence checks
		testutil.AssertDocumentExists(t, docs, universe.TeamMeeting.UUID)
		testutil.AssertDocumentNotExists(t, []nanostore.Document{}, "any-id")
	})

	t.Run("dimension assertions", func(t *testing.T) {
		// Single dimension checks
		testutil.AssertHasStatus(t, universe.BuyGroceries, "pending")
		testutil.AssertHasPriority(t, universe.ExerciseRoutine, "high")
		// Category assertion not implemented yet, check manually
		if universe.TeamMeeting.Dimensions["category"] != "work" {
			t.Error("TeamMeeting should have category=work")
		}

		// Multiple dimensions at once
		testutil.AssertDimensionValues(t, universe.TeamMeeting, map[string]string{
			"status":   "pending",
			"priority": "high",
			"category": "work",
		})

		// Verify all documents in a set have a dimension value
		highPriorityDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"priority": "high"},
		})
		testutil.AssertAllHaveDimension(t, highPriorityDocs, "priority", "high")
	})

	t.Run("ordering assertions", func(t *testing.T) {
		docs, _ := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{{Column: "title", Descending: false}},
		})

		// Verify ordering by any field
		testutil.AssertOrderedBy(t, docs, "title", true) // true = ascending

		// Verify specific order
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

		// Verify empty results
		testutil.AssertQueryEmpty(t, store,
			nanostore.ListOptions{
				Filters: map[string]interface{}{
					"status": "invalid", // This status doesn't exist
				},
			},
		)
	})

	t.Run("custom condition assertions", func(t *testing.T) {
		roots := universe.GetRootDocuments()

		// Verify all documents match a condition
		testutil.AssertAllDocuments(t, roots,
			func(d nanostore.Document) bool {
				_, hasParent := d.Dimensions["parent_id"]
				return !hasParent
			},
			"root document without parent_id",
		)

		// Find at least one document matching a condition
		allDocs, _ := store.List(nanostore.ListOptions{})
		testutil.AssertContainsDocument(t, allDocs,
			func(d nanostore.Document) bool {
				return len(d.Title) > 20 // Long title
			},
			"document with long title",
		)
	})
}

// =============================================================================
// WHEN TO CREATE A FRESH STORE (RARE CASES)
// =============================================================================

// TestWhenToCreateFreshStore documents the rare cases where you should NOT use
// the fixture and instead create a fresh store.
func TestWhenToCreateFreshStore(t *testing.T) {
	t.Run("testing store initialization", func(t *testing.T) {
		// ONLY create a fresh store when testing initialization/configuration
		tempFile := t.TempDir() + "/test.json"

		// Testing specific configuration
		config := types.Config{
			Dimensions: []types.DimensionConfig{
				{
					Name:         "custom",
					Type:         types.Enumerated,
					Values:       []string{"a", "b", "c"},
					DefaultValue: "a",
				},
			},
		}
		store, err := store.New(tempFile, &config)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		// Test configuration-specific behavior
	})

	t.Run("testing error conditions", func(t *testing.T) {
		// When testing validation errors or failure modes
		emptyConfig := types.Config{}
		_, err := store.New("", &emptyConfig) // Invalid empty path
		if err == nil {
			t.Error("expected error for empty path")
		}
	})

	// Other valid cases for fresh stores:
	// - Testing migration or schema changes
	// - Testing concurrent store access
	// - Testing specific data formats or encodings
	// - Testing performance with specific data patterns
}

// =============================================================================
// COMPARISON: Before and After Using Test Utilities
// =============================================================================

// TestWithoutUtilities shows what tests looked like before our testing utilities.
// DON'T WRITE TESTS LIKE THIS - This is an example of what to avoid.
func TestWithoutUtilities(t *testing.T) {
	t.Skip("This test demonstrates the OLD way - don't copy this pattern")

	// BAD: Creating test data manually in every test
	tempFile := t.TempDir() + "/test.json"
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{Name: "status", Type: types.Enumerated, Values: []string{"pending", "done"}},
		},
	}
	store, err := store.New(tempFile, &config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// BAD: Manually creating test documents
	id1, _ := store.Add("Test 1", map[string]interface{}{"status": "pending"})
	_, _ = store.Add("Test 2", map[string]interface{}{"status": "done"})
	_, _ = store.Add("Test 3", map[string]interface{}{"status": "pending"})

	// BAD: Verbose manual verification
	results, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	found := false
	for _, doc := range results {
		if doc.UUID == id1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("document not found")
	}

	// This is verbose, brittle, and hard to maintain!
}

// TestWithUtilities shows the CORRECT way using our utilities.
// THIS IS HOW YOU SHOULD WRITE TESTS.
func TestWithUtilities(t *testing.T) {
	// GOOD: One-line setup with rich test data
	store, universe := testutil.LoadUniverse(t)

	// GOOD: Clear, focused test using existing data
	results, _ := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})

	// GOOD: Concise assertions with good error messages
	testutil.AssertDocumentCount(t, results, 8, "pending documents")
	testutil.AssertDocumentExists(t, results, universe.BuyGroceries.UUID)

	// This is clear, maintainable, and reuses stable test data!
}

// =============================================================================
// SUMMARY: Key Takeaways
// =============================================================================
//
// 1. Always use LoadUniverse() unless you have a specific reason not to
// 2. Use assertion helpers for clearer tests and better error messages
// 3. Leverage the fixture's pre-built scenarios instead of creating test data
// 4. The fixture provides stable, comprehensive test data including:
//    - Documents with all dimension combinations
//    - Hierarchical relationships up to 5 levels deep
//    - Edge cases (empty titles, special characters, unicode)
//    - Mixed state scenarios
//    - Search-friendly content
// 5. Only create a fresh store for:
//    - Testing initialization/configuration
//    - Testing error conditions
//    - Testing migrations or schema changes
//
// For a complete list of available assertions, see testutil/assertions.go
// For the fixture data structure, see testdata/universe.json
