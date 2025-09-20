package query_test

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

func TestTransparentFilteringAndOrderingMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	// Add test documents with custom data fields
	doc1, err := store.Add("High priority task", map[string]interface{}{
		"status":            "active",
		"priority":          "high",
		"_data.Priority":    1,
		"_data.Score":       95.5,
		"_data.IsImportant": true,
		"_data.Owner":       "alice",
		"_data.CreatedBy":   "system",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc2, err := store.Add("Medium priority task", map[string]interface{}{
		"status":            "active",
		"priority":          "medium",
		"_data.Priority":    2,
		"_data.Score":       75.0,
		"_data.IsImportant": false,
		"_data.Owner":       "bob",
		"_data.CreatedBy":   "admin",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc3, err := store.Add("Low priority task", map[string]interface{}{
		"status":            "pending",
		"priority":          "low",
		"_data.Priority":    3,
		"_data.Score":       50.0,
		"_data.IsImportant": false,
		"_data.Owner":       "alice",
		"_data.CreatedBy":   "user",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc4, err := store.Add("Another high priority", map[string]interface{}{
		"status":            "done",
		"priority":          "high",
		"_data.Priority":    1,
		"_data.Score":       88.0,
		"_data.IsImportant": true,
		"_data.Owner":       "bob",
		"_data.CreatedBy":   "system",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("filter by non-dimension string field", func(t *testing.T) {
		// Filter by Owner (non-dimension field)
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.Owner": "alice",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 documents owned by alice, got %d", len(results))
		}
		for _, doc := range results {
			if doc.Dimensions["_data.Owner"] != "alice" {
				t.Errorf("expected owner to be alice, got %v", doc.Dimensions["_data.Owner"])
			}
		}
	})

	t.Run("filter by non-dimension numeric field", func(t *testing.T) {
		// Filter by Priority value
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.Priority": 1,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 documents with Priority=1, got %d", len(results))
		}
	})

	t.Run("filter by non-dimension boolean field", func(t *testing.T) {
		// Filter by IsImportant
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.IsImportant": true,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 important documents, got %d", len(results))
		}
	})

	t.Run("combined dimension and non-dimension filters", func(t *testing.T) {
		// Filter by status (dimension) AND Owner (non-dimension)
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":      "active",
				"_data.Owner": "bob",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 document (active AND bob), got %d", len(results))
		}
		if len(results) > 0 && results[0].UUID != doc2 {
			t.Error("wrong document returned")
		}
	})

	t.Run("order by non-dimension numeric field", func(t *testing.T) {
		// Order by Score descending
		results, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "_data.Score", Descending: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Just check our test documents are in correct order
		testDocIDs := map[string]bool{doc1: true, doc2: true, doc3: true, doc4: true}
		scores := []float64{}
		for _, doc := range results {
			if testDocIDs[doc.UUID] {
				if score, ok := doc.Dimensions["_data.Score"].(float64); ok {
					scores = append(scores, score)
				}
			}
		}

		// Verify descending order
		for i := 1; i < len(scores); i++ {
			if scores[i] > scores[i-1] {
				t.Error("scores not in descending order")
			}
		}
	})

	t.Run("order by non-dimension string field", func(t *testing.T) {
		// Order by Owner ascending
		results, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "_data.Owner", Descending: false},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Check our test documents
		testDocIDs := map[string]bool{doc1: true, doc2: true, doc3: true, doc4: true}
		var firstTestDoc, lastTestDoc *nanostore.Document
		for _, doc := range results {
			if testDocIDs[doc.UUID] {
				if firstTestDoc == nil {
					firstTestDoc = &doc
				}
				lastTestDoc = &doc
			}
		}

		// alice should come before bob
		if firstTestDoc != nil && firstTestDoc.Dimensions["_data.Owner"] != "alice" {
			t.Error("expected alice to come first in ascending order")
		}
		if lastTestDoc != nil && lastTestDoc.Dimensions["_data.Owner"] != "bob" {
			t.Error("expected bob to come last in ascending order")
		}
	})

	t.Run("multiple order by clauses", func(t *testing.T) {
		// Get all results ordered by Priority asc, then Score desc
		results, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "_data.Priority", Descending: false},
				{Column: "_data.Score", Descending: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// Find our test documents in results
		testDocIDs := map[string]bool{doc1: true, doc2: true, doc3: true, doc4: true}
		var testDocs []nanostore.Document
		for _, doc := range results {
			if testDocIDs[doc.UUID] {
				testDocs = append(testDocs, doc)
			}
		}

		if len(testDocs) != 4 {
			t.Fatalf("expected 4 test documents, found %d", len(testDocs))
		}

		// Check ordering: Priority 1 docs should come first
		if testDocs[0].Dimensions["_data.Priority"] != 1 {
			t.Error("expected Priority=1 documents first")
		}
		// Among Priority 1 docs, higher score should come first
		if testDocs[0].Dimensions["_data.Score"].(float64) < testDocs[1].Dimensions["_data.Score"].(float64) {
			t.Error("within same priority, higher scores should come first")
		}
	})

	t.Run("filter with limit and offset", func(t *testing.T) {
		limit := 2
		offset := 1

		// Create a specific filter for our test docs
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.Priority": []interface{}{1, 2, 3}, // Our test docs have these priorities
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "_data.Score", Descending: true},
			},
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Just verify we got results with limit
		if len(results) > limit {
			t.Errorf("expected at most %d results with limit, got %d", limit, len(results))
		}
	})

	// Clean up test documents
	for _, id := range []string{doc1, doc2, doc3, doc4} {
		_ = store.Delete(id, false)
	}
}

func TestTransparentFilteringEdgeCasesMigrated(t *testing.T) {
	store, _ := testutil.LoadUniverse(t)

	// Add documents with edge case values
	doc1, err := store.Add("Null values doc", map[string]interface{}{
		"status":          "active",
		"priority":        "medium",
		"_data.NullField": nil,
		"_data.EmptyStr":  "",
		"_data.Zero":      0,
		"_data.False":     false,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc2, err := store.Add("Mixed types doc", map[string]interface{}{
		"status":           "active",
		"priority":         "medium",
		"_data.StringNum":  "123",
		"_data.ActualNum":  123,
		"_data.Float":      123.0,
		"_data.BoolString": "true",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("filter by null value", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.NullField": nil,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == doc1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("document with null field not found")
		}
	})

	t.Run("filter by empty string", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.EmptyStr": "",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == doc1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("document with empty string not found")
		}
	})

	t.Run("filter by zero value", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.Zero": 0,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == doc1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("document with zero value not found")
		}
	})

	t.Run("filter by false boolean", func(t *testing.T) {
		results, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"_data.False": false,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, doc := range results {
			if doc.UUID == doc1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("document with false value not found")
		}
	})

	// Clean up
	_ = store.Delete(doc1, false)
	_ = store.Delete(doc2, false)
}
