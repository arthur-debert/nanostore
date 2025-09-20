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
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/testutil"
)

func TestOrderingMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("OrderByTitle", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check that titles are in ascending order
		for i := 1; i < len(docs); i++ {
			if docs[i-1].Title > docs[i].Title {
				t.Errorf("titles not in ascending order: %q > %q at positions %d,%d",
					docs[i-1].Title, docs[i].Title, i-1, i)
			}
		}
	})

	t.Run("OrderByTitleDescending", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check that titles are in descending order
		for i := 1; i < len(docs); i++ {
			if docs[i-1].Title < docs[i].Title {
				t.Errorf("titles not in descending order: %q < %q at positions %d,%d",
					docs[i-1].Title, docs[i].Title, i-1, i)
			}
		}
	})

	t.Run("OrderByDimension", func(t *testing.T) {
		// Order by status (alphabetical: active, done, pending)
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "status", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check status order
		var lastStatus string
		for i, doc := range docs {
			status := doc.Dimensions["status"].(string)
			if i > 0 && lastStatus > status {
				t.Errorf("status not in ascending order: %q > %q", lastStatus, status)
			}
			lastStatus = status
		}
	})

	t.Run("OrderByMultipleColumns", func(t *testing.T) {
		// Order by status, then by priority
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "status", Descending: false},
				{Column: "priority", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check that within each status group, priority is ordered
		var lastStatus, lastPriority string
		for _, doc := range docs {
			status := doc.Dimensions["status"].(string)
			priority := doc.Dimensions["priority"].(string)

			if status == lastStatus && priority < lastPriority {
				t.Errorf("within status %q, priority not ordered: %q < %q",
					status, priority, lastPriority)
			}
			lastStatus = status
			lastPriority = priority
		}
	})

	t.Run("OrderByCreatedAt", func(t *testing.T) {
		// Order by created_at timestamp
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check that creation times are in ascending order
		for i := 1; i < len(docs); i++ {
			if docs[i-1].CreatedAt.After(docs[i].CreatedAt) {
				t.Error("created_at not in ascending order")
			}
		}
	})

	t.Run("OrderWithFilters", func(t *testing.T) {
		// Filter by status and order by title
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// All should be pending and ordered by title
		for i, doc := range docs {
			testutil.AssertHasStatus(t, doc, "pending")
			if i > 0 && docs[i-1].Title > doc.Title {
				t.Error("filtered results not ordered by title")
			}
		}
	})

	t.Run("OrderWithLimitAndOffset", func(t *testing.T) {
		// Get all docs ordered by title
		allDocs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Get page 2 (limit 5, offset 5)
		limit := 5
		offset := 5
		pageDocs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify we got the right slice
		if len(allDocs) > offset {
			expectedCount := limit
			if len(allDocs)-offset < limit {
				expectedCount = len(allDocs) - offset
			}
			if len(pageDocs) != expectedCount {
				t.Errorf("expected %d documents, got %d", expectedCount, len(pageDocs))
			}

			// Verify first doc matches
			if len(pageDocs) > 0 && pageDocs[0].UUID != allDocs[offset].UUID {
				t.Error("offset not working correctly with ordering")
			}
		}
	})

	t.Run("ComplexOrderingScenario", func(t *testing.T) {
		// Complex query: filter by multiple values, order by multiple columns
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"priority": []string{"high", "medium"},
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "priority", Descending: false},   // alphabetical order
				{Column: "created_at", Descending: false}, // oldest first within priority
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Verify filtering worked
		for _, doc := range docs {
			priority := doc.Dimensions["priority"].(string)
			if priority != "high" && priority != "medium" {
				t.Errorf("unexpected priority: %s", priority)
			}
		}

		// Verify ordering within each priority group by created_at
		var lastPriority string
		var lastCreatedAt string
		for _, doc := range docs {
			priority := doc.Dimensions["priority"].(string)

			if priority == lastPriority {
				// Within same priority, check created_at order
				if doc.CreatedAt.Format(time.RFC3339) < lastCreatedAt {
					t.Error("within same priority, created_at not in ascending order")
				}
			}
			lastPriority = priority
			lastCreatedAt = doc.CreatedAt.Format(time.RFC3339)
		}
	})

	t.Run("OrderByParentChild", func(t *testing.T) {
		// Get children of PersonalRoot ordered by title
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_id": universe.PersonalRoot.UUID,
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// All should be children of PersonalRoot
		for _, doc := range docs {
			if doc.Dimensions["parent_id"] != universe.PersonalRoot.UUID {
				t.Error("document is not a child of PersonalRoot")
			}
		}

		// Should be ordered by title
		for i := 1; i < len(docs); i++ {
			if docs[i-1].Title > docs[i].Title {
				t.Error("children not ordered by title")
			}
		}
	})
}

func TestOrderingEdgeCasesMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	t.Run("EmptyOrderBy", func(t *testing.T) {
		// Empty OrderBy should not cause error
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{},
		})
		if err != nil {
			t.Fatalf("empty OrderBy caused error: %v", err)
		}
		if len(docs) == 0 {
			t.Error("expected some documents")
		}
	})

	t.Run("OrderByInvalidColumn", func(t *testing.T) {
		// This might or might not error depending on implementation
		// Just verify it doesn't panic
		_, _ = store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "nonexistent_column", Descending: false},
			},
		})
		// No assertion - just checking it doesn't panic
	})

	t.Run("OrderByWithNullValues", func(t *testing.T) {
		// Add a document with null/empty values
		docID, err := store.Add("Null test", map[string]interface{}{
			"status":       "active",
			"priority":     "medium",
			"_data.custom": nil,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Order by the field that has null
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "_data.custom", Descending: false},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Just verify we got results and our doc is included
		found := false
		for _, doc := range docs {
			if doc.UUID == docID {
				found = true
				break
			}
		}
		if !found {
			t.Error("document with null value not found in ordered results")
		}

		// Clean up
		_ = store.Delete(docID, false)
	})
}
