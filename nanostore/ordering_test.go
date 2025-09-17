package nanostore_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestOrdering(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "in_progress", "done"},
				DefaultValue: "todo",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get test store for time control
	testStore := nanostore.AsTestStore(store)
	if testStore == nil {
		t.Fatal("store doesn't support testing features")
	}

	// Set deterministic time
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	currentTime := baseTime
	testStore.SetTimeFunc(func() time.Time {
		t := currentTime
		currentTime = currentTime.Add(1 * time.Hour)
		return t
	})

	// Add test documents with different values
	_, _ = store.Add("Zebra task", map[string]interface{}{
		"priority": "low",
		"status":   "done",
	})
	_, _ = store.Add("Alpha task", map[string]interface{}{
		"priority": "high",
		"status":   "todo",
	})
	_, _ = store.Add("Beta task", map[string]interface{}{
		"priority": "medium",
		"status":   "in_progress",
	})
	_, _ = store.Add("Charlie task", map[string]interface{}{
		"priority": "high",
		"status":   "todo",
	})

	t.Run("OrderByTitle", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check order: Alpha, Beta, Charlie, Zebra
		expectedTitles := []string{"Alpha task", "Beta task", "Charlie task", "Zebra task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
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

		// Check reverse order: Zebra, Charlie, Beta, Alpha
		expectedTitles := []string{"Zebra task", "Charlie task", "Beta task", "Alpha task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
			}
		}
	})

	t.Run("OrderByDimension", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "priority", Descending: false}, // low, medium, high
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Check priority order alphabetically: high -> low -> medium
		expectedPriorities := []string{"high", "high", "low", "medium"}
		for i, expected := range expectedPriorities {
			if priority := docs[i].Dimensions["priority"]; priority != expected {
				t.Errorf("position %d: expected priority %s, got %v", i, expected, priority)
			}
		}
	})

	t.Run("OrderByMultipleColumns", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "priority", Descending: true}, // high, medium, low
				{Column: "title", Descending: false},   // alphabetical within priority
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Expected order with string comparison:
		// Priority descending: "medium" > "low" > "high" (alphabetically)
		// So: medium (Beta), low (Zebra), high (Alpha, Charlie - alphabetical)
		expectedTitles := []string{"Beta task", "Zebra task", "Alpha task", "Charlie task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
			}
		}
	})

	t.Run("OrderByCreatedAt", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should be in creation order: Zebra, Alpha, Beta, Charlie
		expectedTitles := []string{"Zebra task", "Alpha task", "Beta task", "Charlie task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
			}
		}

		// Verify timestamps are increasing
		for i := 1; i < len(docs); i++ {
			if !docs[i].CreatedAt.After(docs[i-1].CreatedAt) {
				t.Errorf("timestamps not in ascending order at position %d", i)
			}
		}
	})

	t.Run("OrderByCreatedAtDescending", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "created_at", Descending: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should be in reverse creation order: Charlie, Beta, Alpha, Zebra
		expectedTitles := []string{"Charlie task", "Beta task", "Alpha task", "Zebra task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
			}
		}
	})

	t.Run("OrderByWithFilter", func(t *testing.T) {
		// Filter for high priority tasks and order by title
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"priority": "high",
			},
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should only have 2 high priority tasks in alphabetical order
		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}

		if len(docs) >= 2 {
			if docs[0].Title != "Alpha task" {
				t.Errorf("expected Alpha task first, got %s", docs[0].Title)
			}
			if docs[1].Title != "Charlie task" {
				t.Errorf("expected Charlie task second, got %s", docs[1].Title)
			}
		}
	})

	t.Run("OrderByNonExistentColumn", func(t *testing.T) {
		// Should handle non-existent columns gracefully (treat as empty string)
		docs, err := store.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "nonexistent", Descending: false},
				{Column: "title", Descending: false}, // Secondary sort for predictability
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// All should have same value for nonexistent column, so should be ordered by title
		expectedTitles := []string{"Alpha task", "Beta task", "Charlie task", "Zebra task"}
		for i, expected := range expectedTitles {
			if docs[i].Title != expected {
				t.Errorf("position %d: expected %s, got %s", i, expected, docs[i].Title)
			}
		}
	})
}

func TestOrderingEdgeCases(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("EmptyOrderBy", func(t *testing.T) {
		_, _ = store.Add("B", nil)
		_, _ = store.Add("A", nil)
		_, _ = store.Add("C", nil)

		// No ordering specified - documents should be in their natural order
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 3 {
			t.Errorf("expected 3 documents, got %d", len(docs))
		}
	})

	t.Run("EmptyStore", func(t *testing.T) {
		// Create new empty store
		tmpfile2, _ := os.CreateTemp("", "test*.json")
		defer func() { _ = os.Remove(tmpfile2.Name()) }()
		_ = tmpfile2.Close()

		emptyStore, _ := nanostore.New(tmpfile2.Name(), config)
		defer func() { _ = emptyStore.Close() }()

		docs, err := emptyStore.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: false},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 0 {
			t.Errorf("expected 0 documents, got %d", len(docs))
		}
	})

	t.Run("SingleDocument", func(t *testing.T) {
		// Create new store with single document
		tmpfile3, _ := os.CreateTemp("", "test*.json")
		defer func() { _ = os.Remove(tmpfile3.Name()) }()
		_ = tmpfile3.Close()

		singleStore, _ := nanostore.New(tmpfile3.Name(), config)
		defer func() { _ = singleStore.Close() }()

		_, _ = singleStore.Add("Only task", nil)

		docs, err := singleStore.List(nanostore.ListOptions{
			OrderBy: []nanostore.OrderClause{
				{Column: "title", Descending: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
		if len(docs) > 0 && docs[0].Title != "Only task" {
			t.Errorf("expected 'Only task', got %s", docs[0].Title)
		}
	})
}