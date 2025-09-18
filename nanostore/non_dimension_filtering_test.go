package nanostore_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// Test struct with both dimension and non-dimension fields for filtering tests
type FilterableItem struct {
	nanostore.Document

	// Dimension fields
	Status   string `values:"pending,active,done" default:"pending"`
	Category string `values:"A,B,C" default:"A"`

	// Non-dimension fields that should support filtering
	Priority    int
	Score       float64
	IsImportant bool
	Tags        string
	Owner       string
}

func TestTransparentFilteringForNonDimensionFields(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := nanostore.NewFromType[FilterableItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data with various combinations
	testItems := []struct {
		title string
		item  FilterableItem
	}{
		{
			title: "High priority task",
			item: FilterableItem{
				Status:      "active",
				Category:    "A",
				Priority:    1,
				Score:       95.5,
				IsImportant: true,
				Tags:        "urgent",
				Owner:       "alice",
			},
		},
		{
			title: "Medium priority task",
			item: FilterableItem{
				Status:      "active",
				Category:    "B",
				Priority:    2,
				Score:       75.0,
				IsImportant: false,
				Tags:        "normal",
				Owner:       "bob",
			},
		},
		{
			title: "Low priority task",
			item: FilterableItem{
				Status:      "pending",
				Category:    "A",
				Priority:    3,
				Score:       50.0,
				IsImportant: false,
				Tags:        "low",
				Owner:       "alice",
			},
		},
		{
			title: "Another high priority",
			item: FilterableItem{
				Status:      "done",
				Category:    "C",
				Priority:    1,
				Score:       88.0,
				IsImportant: true,
				Tags:        "urgent",
				Owner:       "charlie",
			},
		},
	}

	// Create all test items
	for _, test := range testItems {
		_, err := store.Create(test.title, &test.item)
		if err != nil {
			t.Fatalf("failed to create %s: %v", test.title, err)
		}
	}

	// Test filtering using the direct store API (simulating what the TypedQuery uses internally)
	t.Run("FilterByNonDimensionIntField", func(t *testing.T) {
		// This should test if the store supports filtering by non-dimension fields transparently
		query := store.Query()

		// We'll need to add a custom filter method or use the store directly
		// For now, let's test with the internal store API
		opts := nanostore.NewListOptions()
		opts.Filters["Priority"] = 1

		// We need to test this against the underlying store to see current behavior
		// The TypedStore doesn't expose the Store() method, so we'll use a workaround
		results, err := query.Find() // Start with all results
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}

		// Filter manually to see what we should expect
		var expectedCount int
		for _, item := range testItems {
			if item.item.Priority == 1 {
				expectedCount++
			}
		}

		t.Logf("Current behavior: found %d total results, expected %d with Priority=1",
			len(results), expectedCount)

		// This test will help us understand current behavior before we fix it
	})

	t.Run("FilterByNonDimensionStringField", func(t *testing.T) {
		opts := nanostore.NewListOptions()
		opts.Filters["Owner"] = "alice"

		// This will test the current filtering behavior
		results, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}

		// Count expected results
		var expectedCount int
		for _, item := range testItems {
			if item.item.Owner == "alice" {
				expectedCount++
			}
		}

		t.Logf("Current behavior: found %d total results, expected %d with Owner=alice",
			len(results), expectedCount)
	})

	t.Run("FilterByNonDimensionBoolField", func(t *testing.T) {
		opts := nanostore.NewListOptions()
		opts.Filters["IsImportant"] = true

		results, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}

		// Count expected results
		var expectedCount int
		for _, item := range testItems {
			if item.item.IsImportant {
				expectedCount++
			}
		}

		t.Logf("Current behavior: found %d total results, expected %d with IsImportant=true",
			len(results), expectedCount)
	})

	t.Run("DirectStoreFiltering", func(t *testing.T) {
		// Let's create a direct store instance to test the filtering behavior
		store2, err := nanostore.New(tmpfile.Name(), nanostore.Config{
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
			t.Fatalf("failed to create direct store: %v", err)
		}
		defer func() { _ = store2.Close() }()

		// Add a document with both dimension and non-dimension data
		dims := map[string]interface{}{
			"status":         "active",
			"category":       "A",
			"_data.Priority": 1,
			"_data.Owner":    "alice",
		}

		docID, err := store2.Add("Test doc", dims)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Test filtering by dimension field (should work)
		opts1 := nanostore.NewListOptions()
		opts1.Filters["status"] = "active"

		results1, err := store2.List(opts1)
		if err != nil {
			t.Fatalf("failed to filter by dimension: %v", err)
		}

		if len(results1) == 0 {
			t.Error("dimension filtering should work")
		}

		// Test filtering by non-dimension field (current behavior)
		opts2 := nanostore.NewListOptions()
		opts2.Filters["Priority"] = 1

		results2, err := store2.List(opts2)
		if err != nil {
			t.Fatalf("failed to filter by non-dimension: %v", err)
		}

		// This tells us if non-dimension filtering currently works
		t.Logf("Non-dimension filtering current behavior: found %d results (doc ID: %s)",
			len(results2), docID)

		// Now test with the _data prefix (what should work internally)
		opts3 := nanostore.NewListOptions()
		opts3.Filters["_data.Priority"] = 1

		results3, err := store2.List(opts3)
		if err != nil {
			t.Fatalf("failed to filter by _data prefix: %v", err)
		}

		t.Logf("_data prefix filtering: found %d results", len(results3))
	})
}
