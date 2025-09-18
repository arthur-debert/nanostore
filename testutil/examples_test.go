// Package testutil_test demonstrates how to use the test fixture for various testing scenarios.
// This file contains runnable examples of common test patterns.
package testutil_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/testutil"
)

// TestFilteringWithFixture shows how to test filtering without creating any test data
func TestFilteringWithFixture(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	// Test single dimension filter
	t.Run("filter by status", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// We know from fixture there are exactly 8 pending documents
		if len(results) != 8 {
			t.Errorf("expected 8 pending documents, got %d", len(results))
		}
	})

	// Test multiple dimension filters
	t.Run("filter by status and priority", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "active",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify all results match both criteria
		for _, doc := range results {
			if doc.Dimensions["status"] != "active" {
				t.Errorf("expected status=active, got %v", doc.Dimensions["status"])
			}
			if doc.Dimensions["priority"] != "high" {
				t.Errorf("expected priority=high, got %v", doc.Dimensions["priority"])
			}
		}
	})

	// Test filtering with known document
	t.Run("verify specific document in results", func(t *testing.T) {
		// BuyGroceries is pending, so it should appear in pending filter
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == universe.BuyGroceries.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Error("BuyGroceries should be in pending results")
		}
	})
}

// TestHierarchicalRelationships demonstrates testing parent-child queries
func TestHierarchicalRelationships(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("get direct children", func(t *testing.T) {
		// Query children of PersonalRoot
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// PersonalRoot has exactly 3 children
		if len(results) != 3 {
			t.Errorf("expected 3 children, got %d", len(results))
		}

		// Verify the children are correct
		childTitles := make(map[string]bool)
		for _, doc := range results {
			childTitles[doc.Title] = true
		}

		expectedTitles := []string{"Buy groceries", "Exercise routine", "Read book"}
		for _, title := range expectedTitles {
			if !childTitles[title] {
				t.Errorf("missing expected child: %s", title)
			}
		}
	})

	t.Run("test deep hierarchy", func(t *testing.T) {
		// Verify the 5-level deep structure exists
		if universe.Level5Task.SimpleID == "" {
			t.Fatal("Level5Task should have an ID")
		}

		// Check it has the correct parent
		if universe.Level5Task.Dimensions["parent_id"] != universe.Level4Task.UUID {
			t.Error("Level5Task should have Level4Task as parent")
		}

		// ID should show hierarchical structure
		t.Logf("Deep nested ID: %s", universe.Level5Task.SimpleID)
	})

	t.Run("get all roots", func(t *testing.T) {
		roots := universe.GetRootDocuments()

		// Fixture has exactly 10 root documents
		if len(roots) != 10 {
			t.Errorf("expected 10 roots, got %d", len(roots))
		}
	})
}

// TestSearchFunctionality demonstrates text search capabilities
func TestSearchFunctionality(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("search for pack", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			FilterBySearch: "pack",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Should find PackForTrip and PackLunch
		if len(results) != 2 {
			t.Errorf("expected 2 results for 'pack', got %d", len(results))
		}

		// Verify both documents are found
		foundIDs := make(map[string]bool)
		for _, doc := range results {
			foundIDs[doc.UUID] = true
		}

		if !foundIDs[universe.PackForTrip.UUID] {
			t.Error("PackForTrip should be in search results")
		}
		if !foundIDs[universe.PackLunch.UUID] {
			t.Error("PackLunch should be in search results")
		}
	})

	t.Run("search in body text", func(t *testing.T) {
		// Search for "development" which appears in TeamMeeting body
		results, err := store.List(nanostore.ListOptions{
			FilterBySearch: "development",
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == universe.TeamMeeting.UUID {
				found = true
				break
			}
		}
		if !found {
			// Log all results to debug
			t.Logf("Search results for 'development': %d documents", len(results))
			for i, doc := range results {
				t.Logf("[%d] %s: %s (Body: %s)", i, doc.UUID, doc.Title, doc.Body)
			}
			t.Error("should find TeamMeeting when searching for 'development'")
		}
	})
}

// TestEdgeCases verifies handling of edge cases
func TestEdgeCases(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("empty title", func(t *testing.T) {
		// Empty title document should exist
		if universe.EmptyTitle.Title != "" {
			t.Errorf("expected empty title, got %q", universe.EmptyTitle.Title)
		}

		// Should still be queryable by other fields
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"category": "other",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == universe.EmptyTitle.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Error("empty title document should be queryable")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		// Document with special chars should exist and be queryable
		if universe.SpecialChars.Title == "" {
			t.Error("SpecialChars should have a title")
		}

		// Should handle special characters in title correctly
		if universe.SpecialChars.Dimensions["status"] != "active" {
			t.Error("SpecialChars should be active")
		}
	})

	t.Run("unicode emoji", func(t *testing.T) {
		// Unicode should be preserved
		expectedTitle := "🚀 Launch project 💥"
		if universe.UnicodeEmoji.Title != expectedTitle {
			t.Errorf("expected %q, got %q", expectedTitle, universe.UnicodeEmoji.Title)
		}
	})
}

// TestMixedStateScenarios tests complex parent-child state combinations
func TestMixedStateScenarios(t *testing.T) {
	_, universe := testutil.LoadUniverse(t)

	t.Run("parent with mixed child states", func(t *testing.T) {
		// Get children of MixedParent
		children := universe.GetChildrenOf("mixed-parent")

		if len(children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(children))
		}

		// Count by status
		statusCount := make(map[string]int)
		for _, child := range children {
			status := child.Dimensions["status"].(string)
			statusCount[status]++
		}

		// Should have one of each status
		if statusCount["active"] != 1 {
			t.Errorf("expected 1 active child, got %d", statusCount["active"])
		}
		if statusCount["pending"] != 1 {
			t.Errorf("expected 1 pending child, got %d", statusCount["pending"])
		}
		if statusCount["done"] != 1 {
			t.Errorf("expected 1 done child, got %d", statusCount["done"])
		}
	})

	t.Run("deleted parent with active children", func(t *testing.T) {
		// DeletedParent has activity="deleted" but has active children
		if universe.DeletedParent.Dimensions["activity"] != "deleted" {
			t.Error("DeletedParent should have activity=deleted")
		}

		// Its child should be active
		if universe.OrphanChild.Dimensions["activity"] != "active" {
			t.Error("OrphanChild should be active despite deleted parent")
		}

		// Verify the relationship exists
		if universe.OrphanChild.Dimensions["parent_id"] != universe.DeletedParent.UUID {
			t.Error("OrphanChild should reference DeletedParent")
		}
	})
}

// TestOrderingAndPagination demonstrates sorting and pagination
func TestOrderingAndPagination(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("order by title ascending", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
			Limit: intPtr(5),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Should get at most 5 results
		if len(results) > 5 {
			t.Errorf("expected at most 5 results, got %d", len(results))
		}

		// Verify ascending order
		for i := 1; i < len(results); i++ {
			if results[i-1].Title > results[i].Title {
				t.Errorf("titles not in ascending order: %q > %q",
					results[i-1].Title, results[i].Title)
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		// Get first page
		page1, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: false},
			},
			Limit:  intPtr(10),
			Offset: intPtr(0),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get second page
		page2, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: false},
			},
			Limit:  intPtr(10),
			Offset: intPtr(10),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify no overlap
		page1IDs := make(map[string]bool)
		for _, doc := range page1 {
			page1IDs[doc.UUID] = true
		}

		for _, doc := range page2 {
			if page1IDs[doc.UUID] {
				t.Error("found duplicate document across pages")
			}
		}
	})
}

// Helper function for int pointers
func intPtr(i int) *int {
	return &i
}
