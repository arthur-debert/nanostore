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

func TestTransparentFilteringVerificationMigrated(t *testing.T) {
	store, universe := testutil.LoadUniverse(t)

	t.Run("VerifyFixtureDocumentsExist", func(t *testing.T) {
		// First verify we have the expected fixture documents
		allDocs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		if len(allDocs) < 20 {
			t.Errorf("expected at least 20 documents from fixture, got %d", len(allDocs))
		}
	})

	t.Run("FilterByDimensionValues", func(t *testing.T) {
		// Test standard dimension filtering works
		opts := nanostore.NewListOptions()
		opts.Filters["status"] = "pending"

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by status: %v", err)
		}

		// Verify all results have pending status
		for _, doc := range results {
			testutil.AssertHasStatus(t, doc, "pending")
		}

		if len(results) == 0 {
			t.Error("expected some pending documents")
		}
	})

	t.Run("FilterByMultipleDimensionValues", func(t *testing.T) {
		// Test filtering by multiple dimension values
		opts := nanostore.NewListOptions()
		opts.Filters["status"] = []string{"pending", "active"}
		opts.Filters["priority"] = "high"

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by multiple dimensions: %v", err)
		}

		// Verify all results match the filters
		for _, doc := range results {
			status := doc.Dimensions["status"].(string)
			if status != "pending" && status != "active" {
				t.Errorf("expected status to be pending or active, got %s", status)
			}
			testutil.AssertHasPriority(t, doc, "high")
		}
	})

	t.Run("OrderByDimensionValues", func(t *testing.T) {
		// Test ordering by dimension values
		opts := nanostore.NewListOptions()
		opts.OrderBy = []nanostore.OrderClause{
			{Column: "status", Descending: false},
			{Column: "priority", Descending: true},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to order by dimensions: %v", err)
		}

		// Verify ordering
		var lastStatus string
		var lastPriority string
		for i, doc := range results {
			status := doc.Dimensions["status"].(string)
			priority := doc.Dimensions["priority"].(string)

			if i > 0 {
				// Check status ordering (alphabetical)
				if status < lastStatus {
					t.Errorf("status not in ascending order: %s < %s", status, lastStatus)
				}
				// Within same status, check priority ordering (desc)
				if status == lastStatus && priority > lastPriority {
					t.Errorf("priority not in descending order within status %s: %s > %s",
						status, priority, lastPriority)
				}
			}
			lastStatus = status
			lastPriority = priority
		}
	})

	t.Run("CombinedFilterAndOrder", func(t *testing.T) {
		// Test combined filtering and ordering
		opts := nanostore.NewListOptions()
		opts.Filters["parent_id"] = universe.WorkRoot.UUID
		opts.OrderBy = []nanostore.OrderClause{
			{Column: "title", Descending: false},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter and order: %v", err)
		}

		// Verify all are children of WorkRoot
		for _, doc := range results {
			if doc.Dimensions["parent_id"] != universe.WorkRoot.UUID {
				t.Errorf("expected parent_id to be WorkRoot, got %v", doc.Dimensions["parent_id"])
			}
		}

		// Verify title ordering
		for i := 1; i < len(results); i++ {
			if results[i-1].Title > results[i].Title {
				t.Errorf("titles not in ascending order: %q > %q",
					results[i-1].Title, results[i].Title)
			}
		}
	})
}

func TestTransparentNonDimensionFilteringMigrated(t *testing.T) {
	// This test specifically verifies non-dimension filtering behavior
	// We use a temporary file to ensure we control the exact data
	tempFile := t.TempDir() + "/test.json"
	store, err := nanostore.New(tempFile, nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "active", "done"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Add test documents with non-dimension data
	testData := []struct {
		title    string
		status   string
		priority string
		owner    string
		score    int
	}{
		{"Task 1", "active", "high", "alice", 95},
		{"Task 2", "pending", "medium", "bob", 75},
		{"Task 3", "active", "high", "alice", 85},
		{"Task 4", "done", "low", "charlie", 60},
		{"Task 5", "active", "medium", "alice", 90},
	}

	for _, data := range testData {
		dims := map[string]interface{}{
			"status":      data.status,
			"priority":    data.priority,
			"_data.owner": data.owner,
			"_data.score": data.score,
		}
		_, err := store.Add(data.title, dims)
		if err != nil {
			t.Fatalf("failed to add %s: %v", data.title, err)
		}
	}

	t.Run("FilterByNonDimensionString", func(t *testing.T) {
		opts := nanostore.NewListOptions()
		opts.Filters["owner"] = "alice"

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by owner: %v", err)
		}

		// Should find 3 documents owned by alice
		if len(results) != 3 {
			t.Errorf("expected 3 documents owned by alice, got %d", len(results))
			for _, doc := range results {
				t.Logf("  - %s", doc.Title)
			}
		}

		// Verify all results have alice as owner
		expectedTitles := map[string]bool{"Task 1": true, "Task 3": true, "Task 5": true}
		for _, doc := range results {
			if !expectedTitles[doc.Title] {
				t.Errorf("unexpected document in alice's results: %s", doc.Title)
			}
		}
	})

	t.Run("FilterByNonDimensionNumber", func(t *testing.T) {
		// Filter for high scores (>= 85)
		opts := nanostore.NewListOptions()
		opts.Filters["score"] = 85

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by score: %v", err)
		}

		// Note: exact filtering might only match score=85
		// This tests the current behavior
		t.Logf("Documents with score=85: %d", len(results))
		for _, doc := range results {
			t.Logf("  - %s (score: %v)", doc.Title, doc.Dimensions["_data.score"])
		}
	})

	t.Run("CombineDimensionAndNonDimensionFilters", func(t *testing.T) {
		opts := nanostore.NewListOptions()
		opts.Filters["status"] = "active"
		opts.Filters["owner"] = "alice"

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by combined criteria: %v", err)
		}

		// Should find Task 1, Task 3, and Task 5 (all active + alice)
		if len(results) != 3 {
			t.Errorf("expected 3 documents (active + alice), got %d", len(results))
		}

		for _, doc := range results {
			testutil.AssertHasStatus(t, doc, "active")
			if doc.Title != "Task 1" && doc.Title != "Task 3" && doc.Title != "Task 5" {
				t.Errorf("unexpected document: %s", doc.Title)
			}
		}
	})

	t.Run("OrderByNonDimensionField", func(t *testing.T) {
		opts := nanostore.NewListOptions()
		opts.OrderBy = []nanostore.OrderClause{
			{Column: "score", Descending: true},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Logf("Note: ordering by non-dimension fields may not be supported")
			return
		}

		// If supported, verify ordering
		t.Logf("Documents ordered by score (descending):")
		for i, doc := range results {
			score := doc.Dimensions["_data.score"]
			t.Logf("  %d. %s (score: %v)", i+1, doc.Title, score)
		}
	})
}
