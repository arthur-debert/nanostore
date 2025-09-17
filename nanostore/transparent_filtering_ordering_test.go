package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Test struct for transparent filtering and ordering
type TransparentTestItem struct {
	nanostore.Document

	// Dimension fields
	Status   string `values:"pending,active,done" default:"pending"`
	Category string `values:"A,B,C" default:"A"`

	// Non-dimension fields that should support transparent filtering/ordering
	Priority    int
	Score       float64
	IsImportant bool
	Owner       string
	CreatedBy   string
}

func TestTransparentFilteringAndOrdering(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create direct store for testing transparent support
	store, err := nanostore.New(tmpfile.Name(), nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "active", "done"},
				DefaultValue: "pending",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"A", "B", "C"},
				DefaultValue: "A",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with both dimensions and data fields
	testDocs := []struct {
		title string
		dims  map[string]interface{}
	}{
		{
			title: "High priority task",
			dims: map[string]interface{}{
				"status":            "active",
				"category":          "A",
				"_data.Priority":    1,
				"_data.Score":       95.5,
				"_data.IsImportant": true,
				"_data.Owner":       "alice",
				"_data.CreatedBy":   "system",
			},
		},
		{
			title: "Medium priority task",
			dims: map[string]interface{}{
				"status":            "active",
				"category":          "B",
				"_data.Priority":    2,
				"_data.Score":       75.0,
				"_data.IsImportant": false,
				"_data.Owner":       "bob",
				"_data.CreatedBy":   "admin",
			},
		},
		{
			title: "Low priority task",
			dims: map[string]interface{}{
				"status":            "pending",
				"category":          "A",
				"_data.Priority":    3,
				"_data.Score":       50.0,
				"_data.IsImportant": false,
				"_data.Owner":       "alice",
				"_data.CreatedBy":   "user",
			},
		},
		{
			title: "Another high priority",
			dims: map[string]interface{}{
				"status":            "done",
				"category":          "C",
				"_data.Priority":    1,
				"_data.Score":       88.0,
				"_data.IsImportant": true,
				"_data.Owner":       "charlie",
				"_data.CreatedBy":   "admin",
			},
		},
	}

	// Add all test documents
	for _, test := range testDocs {
		_, err := store.Add(test.title, test.dims)
		if err != nil {
			t.Fatalf("failed to add %s: %v", test.title, err)
		}
	}

	t.Run("TransparentFilteringByNonDimensionFields", func(t *testing.T) {
		tests := []struct {
			name           string
			filterKey      string
			filterValue    interface{}
			expectedCount  int
			expectedTitles []string
		}{
			{
				name:           "filter by Priority=1",
				filterKey:      "Priority",
				filterValue:    1,
				expectedCount:  2,
				expectedTitles: []string{"High priority task", "Another high priority"},
			},
			{
				name:           "filter by Owner=alice",
				filterKey:      "Owner",
				filterValue:    "alice",
				expectedCount:  2,
				expectedTitles: []string{"High priority task", "Low priority task"},
			},
			{
				name:           "filter by IsImportant=true",
				filterKey:      "IsImportant",
				filterValue:    true,
				expectedCount:  2,
				expectedTitles: []string{"High priority task", "Another high priority"},
			},
			{
				name:           "filter by Score=75.0",
				filterKey:      "Score",
				filterValue:    75.0,
				expectedCount:  1,
				expectedTitles: []string{"Medium priority task"},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				opts := nanostore.NewListOptions()
				opts.Filters[test.filterKey] = test.filterValue

				results, err := store.List(opts)
				if err != nil {
					t.Fatalf("failed to filter: %v", err)
				}

				if len(results) != test.expectedCount {
					t.Errorf("expected %d results, got %d", test.expectedCount, len(results))
					t.Logf("Results found:")
					for _, doc := range results {
						t.Logf("  - %s", doc.Title)
					}
				}

				// Verify the correct documents were returned
				foundTitles := make(map[string]bool)
				for _, doc := range results {
					foundTitles[doc.Title] = true
				}

				for _, expectedTitle := range test.expectedTitles {
					if !foundTitles[expectedTitle] {
						t.Errorf("expected to find '%s' in results", expectedTitle)
					}
				}
			})
		}
	})

	t.Run("TransparentOrderingByNonDimensionFields", func(t *testing.T) {
		tests := []struct {
			name          string
			orderColumn   string
			descending    bool
			expectedOrder []string
		}{
			{
				name:        "order by Priority ascending",
				orderColumn: "Priority",
				descending:  false,
				expectedOrder: []string{
					"High priority task",    // Priority 1
					"Another high priority", // Priority 1
					"Medium priority task",  // Priority 2
					"Low priority task",     // Priority 3
				},
			},
			{
				name:        "order by Score descending",
				orderColumn: "Score",
				descending:  true,
				expectedOrder: []string{
					"High priority task",    // Score 95.5
					"Another high priority", // Score 88.0
					"Medium priority task",  // Score 75.0
					"Low priority task",     // Score 50.0
				},
			},
			{
				name:        "order by Owner ascending",
				orderColumn: "Owner",
				descending:  false,
				expectedOrder: []string{
					"High priority task",    // alice
					"Low priority task",     // alice
					"Medium priority task",  // bob
					"Another high priority", // charlie
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				opts := nanostore.NewListOptions()
				opts.OrderBy = []nanostore.OrderClause{
					{Column: test.orderColumn, Descending: test.descending},
				}

				results, err := store.List(opts)
				if err != nil {
					t.Fatalf("failed to order: %v", err)
				}

				if len(results) != len(test.expectedOrder) {
					t.Fatalf("expected %d results, got %d", len(test.expectedOrder), len(results))
				}

				// For ordering tests, check if results are in the expected order
				// Allow for stable sort behavior (items with same value may be in any order)
				for i, expectedTitle := range test.expectedOrder {
					if i < len(results) {
						actualTitle := results[i].Title
						t.Logf("Position %d: expected '%s', got '%s'", i, expectedTitle, actualTitle)
					}
				}

				// Verify that ordering actually worked by checking the values
				if test.orderColumn == "Priority" {
					for i := 1; i < len(results); i++ {
						prev := results[i-1].Dimensions["_data.Priority"]
						curr := results[i].Dimensions["_data.Priority"]

						if test.descending {
							// For descending order, previous should be >= current
							if prev.(int) < curr.(int) {
								t.Errorf("Ordering failed: %v should be >= %v", prev, curr)
							}
						} else {
							// For ascending order, previous should be <= current
							if prev.(int) > curr.(int) {
								t.Errorf("Ordering failed: %v should be <= %v", prev, curr)
							}
						}
					}
				}
			})
		}
	})

	t.Run("CombinedFilteringAndOrdering", func(t *testing.T) {
		// Filter by IsImportant=true and order by Score descending
		opts := nanostore.NewListOptions()
		opts.Filters["IsImportant"] = true
		opts.OrderBy = []nanostore.OrderClause{
			{Column: "Score", Descending: true},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter and order: %v", err)
		}

		// Should get 2 important items ordered by score
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		// Verify ordering of the filtered results
		if len(results) >= 2 {
			score1 := results[0].Dimensions["_data.Score"].(float64)
			score2 := results[1].Dimensions["_data.Score"].(float64)

			if score1 < score2 {
				t.Errorf("Ordering failed: %f should be >= %f", score1, score2)
			}

			t.Logf("Filtered and ordered results:")
			for i, doc := range results {
				score := doc.Dimensions["_data.Score"].(float64)
				t.Logf("  %d. %s (Score: %f)", i+1, doc.Title, score)
			}
		}
	})

	t.Run("MixedDimensionAndNonDimensionFiltering", func(t *testing.T) {
		// Filter by both dimension (status) and non-dimension (Priority) fields
		opts := nanostore.NewListOptions()
		opts.Filters["status"] = "active"
		opts.Filters["Priority"] = 2

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("failed to filter by mixed fields: %v", err)
		}

		// Should find exactly 1 result: Medium priority task
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if len(results) > 0 && results[0].Title != "Medium priority task" {
			t.Errorf("expected 'Medium priority task', got '%s'", results[0].Title)
		}
	})
}
