package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestTransparentFilteringVerification(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create direct store
	store, err := nanostore.New(tmpfile.Name(), nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "active", "done"},
				DefaultValue: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add multiple documents with different non-dimension values
	testData := []struct {
		title    string
		priority int
		owner    string
	}{
		{"Task 1", 1, "alice"},
		{"Task 2", 2, "bob"},
		{"Task 3", 1, "alice"},
		{"Task 4", 3, "charlie"},
		{"Task 5", 2, "alice"},
	}

	for _, data := range testData {
		dims := map[string]interface{}{
			"status":         "active",
			"_data.Priority": data.priority,
			"_data.Owner":    data.owner,
		}
		_, err := store.Add(data.title, dims)
		if err != nil {
			t.Fatalf("failed to add %s: %v", data.title, err)
		}
	}

	t.Run("VerifyTransparentFilteringWorks", func(t *testing.T) {
		// Test 1: Get all documents first
		allOpts := nanostore.NewListOptions()
		allResults, err := store.List(allOpts)
		if err != nil {
			t.Fatalf("failed to get all documents: %v", err)
		}

		if len(allResults) != 5 {
			t.Fatalf("expected 5 total documents, got %d", len(allResults))
		}

		// Test 2: Filter by Priority=1 (should get 2 documents)
		priorityOpts := nanostore.NewListOptions()
		priorityOpts.Filters["Priority"] = 1

		priorityResults, err := store.List(priorityOpts)
		if err != nil {
			t.Fatalf("failed to filter by Priority: %v", err)
		}

		if len(priorityResults) != 2 {
			t.Errorf("expected 2 documents with Priority=1, got %d", len(priorityResults))
			t.Logf("Priority=1 results:")
			for _, doc := range priorityResults {
				t.Logf("  - %s", doc.Title)
			}
		}

		// Verify the correct documents were returned
		expectedTitles := map[string]bool{"Task 1": true, "Task 3": true}
		for _, doc := range priorityResults {
			if !expectedTitles[doc.Title] {
				t.Errorf("unexpected document in Priority=1 results: %s", doc.Title)
			}
		}

		// Test 3: Filter by Owner="alice" (should get 3 documents)
		ownerOpts := nanostore.NewListOptions()
		ownerOpts.Filters["Owner"] = "alice"

		ownerResults, err := store.List(ownerOpts)
		if err != nil {
			t.Fatalf("failed to filter by Owner: %v", err)
		}

		if len(ownerResults) != 3 {
			t.Errorf("expected 3 documents with Owner=alice, got %d", len(ownerResults))
			t.Logf("Owner=alice results:")
			for _, doc := range ownerResults {
				t.Logf("  - %s", doc.Title)
			}
		}

		// Test 4: Combined filter (Priority=1 AND Owner="alice") should get 2 documents
		combinedOpts := nanostore.NewListOptions()
		combinedOpts.Filters["Priority"] = 1
		combinedOpts.Filters["Owner"] = "alice"

		combinedResults, err := store.List(combinedOpts)
		if err != nil {
			t.Fatalf("failed to filter by combined criteria: %v", err)
		}

		if len(combinedResults) != 2 {
			t.Errorf("expected 2 documents with Priority=1 AND Owner=alice, got %d", len(combinedResults))
			t.Logf("Combined filter results:")
			for _, doc := range combinedResults {
				t.Logf("  - %s", doc.Title)
			}
		}

		// Test 5: Filter by non-existent value (should get 0 documents)
		nonExistentOpts := nanostore.NewListOptions()
		nonExistentOpts.Filters["Owner"] = "nonexistent"

		nonExistentResults, err := store.List(nonExistentOpts)
		if err != nil {
			t.Fatalf("failed to filter by non-existent value: %v", err)
		}

		if len(nonExistentResults) != 0 {
			t.Errorf("expected 0 documents with Owner=nonexistent, got %d", len(nonExistentResults))
		}
	})

	t.Run("VerifyTransparentOrderingWorks", func(t *testing.T) {
		// Test ordering by Priority ascending
		opts := nanostore.NewListOptions()
		opts.OrderBy = []nanostore.OrderClause{
			{Column: "Priority", Descending: false},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to order by Priority: %v", err)
		}

		if len(results) != 5 {
			t.Fatalf("expected 5 documents, got %d", len(results))
		}

		// Verify the ordering is correct
		for i := 1; i < len(results); i++ {
			prevPriority := results[i-1].Dimensions["_data.Priority"].(int)
			currPriority := results[i].Dimensions["_data.Priority"].(int)

			if prevPriority > currPriority {
				t.Errorf("Ordering failed: document %d has Priority %d, document %d has Priority %d",
					i-1, prevPriority, i, currPriority)
			}
		}

		t.Logf("Documents ordered by Priority ascending:")
		for i, doc := range results {
			priority := doc.Dimensions["_data.Priority"].(int)
			t.Logf("  %d. %s (Priority: %d)", i+1, doc.Title, priority)
		}
	})

	t.Run("VerifyBothDimensionAndNonDimensionFiltering", func(t *testing.T) {
		// Add documents with different status values
		dims1 := map[string]interface{}{
			"status":         "pending",
			"_data.Priority": 1,
			"_data.Owner":    "alice",
		}
		_, err := store.Add("Pending task", dims1)
		if err != nil {
			t.Fatalf("failed to add pending task: %v", err)
		}

		// Filter by dimension (status) and non-dimension (Priority) combined
		opts := nanostore.NewListOptions()
		opts.Filters["status"] = "active" // dimension filter
		opts.Filters["Priority"] = 1      // non-dimension filter

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by mixed criteria: %v", err)
		}

		// Should find 2 documents: "Task 1" and "Task 3" (both have status="active" and Priority=1)
		if len(results) != 2 {
			t.Errorf("expected 2 documents with status=active AND Priority=1, got %d", len(results))
			t.Logf("Mixed filtering results:")
			for _, doc := range results {
				status := doc.Dimensions["status"]
				priority := doc.Dimensions["_data.Priority"]
				t.Logf("  - %s (status: %v, Priority: %v)", doc.Title, status, priority)
			}
		}

		// Verify the pending task is not included
		for _, doc := range results {
			if doc.Title == "Pending task" {
				t.Error("Pending task should not be included in active status filter")
			}
		}
	})
}
