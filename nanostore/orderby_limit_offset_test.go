package nanostore

import (
	"testing"
)

func TestListWithOrderBy(t *testing.T) {
	// Create test store with sample config
	store, err := New(":memory:", Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "active", "completed"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add test documents with different values
	docs := []struct {
		title    string
		status   string
		priority string
	}{
		{"Task C", "completed", "high"},
		{"Task A", "pending", "low"},
		{"Task B", "active", "medium"},
		{"Task D", "pending", "high"},
	}

	for _, doc := range docs {
		_, err := store.Add(doc.title, map[string]interface{}{
			"status":   doc.status,
			"priority": doc.priority,
		})
		if err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	t.Run("order by title ascending", func(t *testing.T) {
		opts := ListOptions{
			Filters: map[string]interface{}{},
			OrderBy: []OrderClause{
				{Column: "title", Descending: false},
			},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 4 {
			t.Fatalf("Expected 4 results, got %d", len(results))
		}

		expectedTitles := []string{"Task A", "Task B", "Task C", "Task D"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("order by title descending", func(t *testing.T) {
		opts := ListOptions{
			Filters: map[string]interface{}{},
			OrderBy: []OrderClause{
				{Column: "title", Descending: true},
			},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		expectedTitles := []string{"Task D", "Task C", "Task B", "Task A"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("order by status then priority", func(t *testing.T) {
		opts := ListOptions{
			Filters: map[string]interface{}{},
			OrderBy: []OrderClause{
				{Column: "status", Descending: false},
				{Column: "priority", Descending: true}, // high -> low
			},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		// Expected order based on config enum order:
		// status order: pending (0), active (1), completed (2)
		// priority order (DESC): high (2), medium (1), low (0)
		// So: pending high (Task D), pending low (Task A), active medium (Task B), completed high (Task C)
		expectedTitles := []string{"Task D", "Task A", "Task B", "Task C"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})
}

func TestListWithLimit(t *testing.T) {
	store, err := New(":memory:", Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "active", "completed"},
				DefaultValue: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add 5 test documents
	for i := 1; i <= 5; i++ {
		_, err := store.Add("Task "+string(rune('A'+i-1)), map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatalf("Failed to add document %d: %v", i, err)
		}
	}

	t.Run("limit to 3 documents", func(t *testing.T) {
		limit := 3
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Limit:   &limit,
			OrderBy: []OrderClause{{Column: "title", Descending: false}},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("limit of 0 returns no documents", func(t *testing.T) {
		limit := 0
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Limit:   &limit,
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 0 {
			t.Fatalf("Expected 0 results, got %d", len(results))
		}
	})
}

func TestListWithOffset(t *testing.T) {
	store, err := New(":memory:", Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "active", "completed"},
				DefaultValue: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add 5 test documents
	for i := 1; i <= 5; i++ {
		_, err := store.Add("Task "+string(rune('A'+i-1)), map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatalf("Failed to add document %d: %v", i, err)
		}
	}

	t.Run("offset by 2 documents", func(t *testing.T) {
		offset := 2
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Offset:  &offset,
			OrderBy: []OrderClause{{Column: "title", Descending: false}},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		expectedTitles := []string{"Task C", "Task D", "Task E"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("offset larger than total returns empty", func(t *testing.T) {
		offset := 10
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Offset:  &offset,
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 0 {
			t.Fatalf("Expected 0 results, got %d", len(results))
		}
	})
}

func TestListWithPagination(t *testing.T) {
	store, err := New(":memory:", Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "active", "completed"},
				DefaultValue: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add 10 test documents
	for i := 1; i <= 10; i++ {
		_, err := store.Add("Task "+string(rune('A'+i-1)), map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Fatalf("Failed to add document %d: %v", i, err)
		}
	}

	t.Run("page 1 (limit 3, offset 0)", func(t *testing.T) {
		limit := 3
		offset := 0
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Limit:   &limit,
			Offset:  &offset,
			OrderBy: []OrderClause{{Column: "title", Descending: false}},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		expectedTitles := []string{"Task A", "Task B", "Task C"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("page 2 (limit 3, offset 3)", func(t *testing.T) {
		limit := 3
		offset := 3
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Limit:   &limit,
			Offset:  &offset,
			OrderBy: []OrderClause{{Column: "title", Descending: false}},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		expectedTitles := []string{"Task D", "Task E", "Task F"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
		}
	})

	t.Run("last page (limit 3, offset 9)", func(t *testing.T) {
		limit := 3
		offset := 9
		opts := ListOptions{
			Filters: map[string]interface{}{},
			Limit:   &limit,
			Offset:  &offset,
			OrderBy: []OrderClause{{Column: "title", Descending: false}},
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if results[0].Title != "Task J" {
			t.Errorf("Expected title 'Task J', got %s", results[0].Title)
		}
	})
}

func TestListWithCombinedFeatures(t *testing.T) {
	store, err := New(":memory:", Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "active", "completed"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add test documents
	docs := []struct {
		title    string
		status   string
		priority string
	}{
		{"High Task 1", "pending", "high"},
		{"High Task 2", "pending", "high"},
		{"High Task 3", "active", "high"},
		{"Medium Task 1", "pending", "medium"},
		{"Low Task 1", "pending", "low"},
	}

	for _, doc := range docs {
		_, err := store.Add(doc.title, map[string]interface{}{
			"status":   doc.status,
			"priority": doc.priority,
		})
		if err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	t.Run("filter + order + pagination", func(t *testing.T) {
		limit := 2
		offset := 0
		opts := ListOptions{
			Filters: map[string]interface{}{
				"status": "pending",
			},
			OrderBy: []OrderClause{
				{Column: "priority", Descending: true}, // high first
				{Column: "title", Descending: false},   // then alphabetical
			},
			Limit:  &limit,
			Offset: &offset,
		}

		results, err := store.List(opts)
		if err != nil {
			t.Fatalf("Failed to list documents: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}

		// Should get first 2 pending items ordered by priority desc, title asc
		expectedTitles := []string{"High Task 1", "High Task 2"}
		for i, result := range results {
			if result.Title != expectedTitles[i] {
				t.Errorf("Expected title %s at position %d, got %s", expectedTitles[i], i, result.Title)
			}
			if result.Dimensions["status"] != "pending" {
				t.Errorf("Expected status 'pending' for %s, got %s", result.Title, result.Dimensions["status"])
			}
		}
	})
}
