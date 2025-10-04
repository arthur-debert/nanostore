package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"testing"

	_ "github.com/arthur-debert/nanostore/nanostore" // for embedded Document type
	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestDeclarativeQueryModifiers(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create typed store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	setupTestData := func() {
		testData := []struct {
			title    string
			status   string
			priority string
			activity string
		}{
			{"Task 1", "pending", "high", "active"},
			{"Task 2", "active", "medium", "active"},
			{"Task 3", "done", "low", "active"},
			{"Task 4", "pending", "high", "archived"},
			{"Task 5", "active", "low", "active"},
			{"Task 6", "done", "high", "deleted"},
		}

		for _, data := range testData {
			_, err := store.Create(data.title, &TodoItem{
				Status:   data.status,
				Priority: data.priority,
				Activity: data.activity,
			})
			if err != nil {
				t.Fatalf("failed to create test data %s: %v", data.title, err)
			}
		}
	}

	setupTestData()

	t.Run("StatusIn", func(t *testing.T) {
		// Query for multiple statuses
		results, err := store.Query().
			StatusIn("pending", "active").
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query with StatusIn: %v", err)
		}

		// Should have Task 1, 2, 5 (not Task 4 because it's archived)
		if len(results) != 3 {
			t.Errorf("expected 3 results with StatusIn, got %d", len(results))
		}

		// Verify all results have correct status
		for _, todo := range results {
			if todo.Status != "pending" && todo.Status != "active" {
				t.Errorf("unexpected status %s in StatusIn results", todo.Status)
			}
			if todo.Activity != "active" {
				t.Errorf("unexpected activity %s, expected 'active'", todo.Activity)
			}
		}
	})

	t.Run("StatusNot", func(t *testing.T) {
		// Query excluding a specific status
		results, err := store.Query().
			StatusNot("done").
			Activity("active").
			Find()
		if err != nil {
			t.Fatalf("failed to query with StatusNot: %v", err)
		}

		// Should have Task 1, 2, 5 (all active tasks that are not done)
		if len(results) != 3 {
			t.Errorf("expected 3 results with StatusNot(done), got %d", len(results))
		}

		// Verify no results have "done" status
		for _, todo := range results {
			if todo.Status == "done" {
				t.Error("found 'done' status in StatusNot(done) results")
			}
		}
	})

	t.Run("OrderBy", func(t *testing.T) {
		// Query with ordering by priority
		results, err := store.Query().
			Activity("active").
			OrderBy("priority").
			Find()
		if err != nil {
			t.Fatalf("failed to query with OrderBy: %v", err)
		}

		// Should be ordered: high, high, low, medium (when sorted alphabetically)
		// But since 'high' < 'low' < 'medium' alphabetically
		if len(results) < 2 {
			t.Fatalf("expected at least 2 results, got %d", len(results))
		}

		// Test ordering by title
		titleResults, err := store.Query().
			OrderBy("title").
			Find()
		if err != nil {
			t.Fatalf("failed to query with OrderBy title: %v", err)
		}

		// Verify titles are in ascending order
		for i := 1; i < len(titleResults); i++ {
			if titleResults[i-1].Title > titleResults[i].Title {
				t.Errorf("titles not in ascending order: %s > %s",
					titleResults[i-1].Title, titleResults[i].Title)
			}
		}
	})

	t.Run("OrderByDesc", func(t *testing.T) {
		// Query with descending order
		results, err := store.Query().
			OrderByDesc("title").
			Find()
		if err != nil {
			t.Fatalf("failed to query with OrderByDesc: %v", err)
		}

		// Verify titles are in descending order
		for i := 1; i < len(results); i++ {
			if results[i-1].Title < results[i].Title {
				t.Errorf("titles not in descending order: %s < %s",
					results[i-1].Title, results[i].Title)
			}
		}
	})

	t.Run("Limit", func(t *testing.T) {
		// Query with limit
		results, err := store.Query().
			Activity("active").
			Limit(2).
			Find()
		if err != nil {
			t.Fatalf("failed to query with Limit: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected exactly 2 results with Limit(2), got %d", len(results))
		}
	})

	t.Run("Offset", func(t *testing.T) {
		// First get all results to know what to expect
		allResults, err := store.Query().
			OrderBy("title").
			Find()
		if err != nil {
			t.Fatalf("failed to get all results: %v", err)
		}

		// Query with offset
		offsetResults, err := store.Query().
			OrderBy("title").
			Offset(2).
			Find()
		if err != nil {
			t.Fatalf("failed to query with Offset: %v", err)
		}

		expectedCount := len(allResults) - 2
		if len(offsetResults) != expectedCount {
			t.Errorf("expected %d results with Offset(2), got %d", expectedCount, len(offsetResults))
		}

		// Verify we skipped the first 2
		if len(offsetResults) > 0 && len(allResults) > 2 {
			if offsetResults[0].Title != allResults[2].Title {
				t.Errorf("offset didn't skip correctly: expected %s, got %s",
					allResults[2].Title, offsetResults[0].Title)
			}
		}
	})

	t.Run("LimitAndOffset", func(t *testing.T) {
		// Combine limit and offset for pagination
		page1, err := store.Query().
			OrderBy("title").
			Limit(2).
			Offset(0).
			Find()
		if err != nil {
			t.Fatalf("failed to get page 1: %v", err)
		}

		page2, err := store.Query().
			OrderBy("title").
			Limit(2).
			Offset(2).
			Find()
		if err != nil {
			t.Fatalf("failed to get page 2: %v", err)
		}

		// Verify no overlap
		if len(page1) > 0 && len(page2) > 0 {
			if page1[0].UUID == page2[0].UUID {
				t.Error("pages have overlapping results")
			}
		}
	})

	t.Run("First", func(t *testing.T) {
		// Get first result
		first, err := store.Query().
			OrderBy("title").
			First()
		if err != nil {
			t.Fatalf("failed to get first: %v", err)
		}

		if first == nil {
			t.Fatal("expected a result from First(), got nil")
		}

		// Verify it's actually the first when ordered by title
		all, _ := store.Query().OrderBy("title").Find()
		if len(all) > 0 && first.UUID != all[0].UUID {
			t.Error("First() didn't return the actual first result")
		}
	})

	t.Run("Count", func(t *testing.T) {
		// Count all documents
		totalCount, err := store.Query().Count()
		if err != nil {
			t.Fatalf("failed to count all: %v", err)
		}

		if totalCount != 6 {
			t.Errorf("expected total count of 6, got %d", totalCount)
		}

		// Count with filter
		activeCount, err := store.Query().
			Activity("active").
			Count()
		if err != nil {
			t.Fatalf("failed to count active: %v", err)
		}

		if activeCount != 4 {
			t.Errorf("expected active count of 4, got %d", activeCount)
		}

		// Count with multiple filters
		highPriorityActiveCount, err := store.Query().
			Priority("high").
			Activity("active").
			Count()
		if err != nil {
			t.Fatalf("failed to count high priority active: %v", err)
		}

		if highPriorityActiveCount != 1 {
			t.Errorf("expected 1 high priority active task, got %d", highPriorityActiveCount)
		}
	})

	t.Run("ComplexQuery", func(t *testing.T) {
		// Test combining multiple modifiers
		results, err := store.Query().
			StatusIn("pending", "active").
			Priority("high").
			Activity("active").
			OrderByDesc("title").
			Limit(10).
			Find()
		if err != nil {
			t.Fatalf("failed complex query: %v", err)
		}

		// Should only have Task 1 (pending, high, active)
		if len(results) != 1 {
			t.Errorf("expected 1 result from complex query, got %d", len(results))
		}

		if len(results) > 0 && results[0].Title != "Task 1" {
			t.Errorf("expected 'Task 1', got %s", results[0].Title)
		}
	})

	t.Run("EmptyResults", func(t *testing.T) {
		// Query that should return no results
		results, err := store.Query().
			Status("done").
			Priority("low").
			Activity("archived").
			Find()
		if err != nil {
			t.Fatalf("failed empty query: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}

		// First on empty results
		first, err := store.Query().
			Status("nonexistent").
			First()
		// Should return an error when no documents found
		if err == nil {
			t.Error("expected error from First on empty results, got nil")
		}

		if first != nil {
			t.Error("expected nil from First() on empty results")
		}

		// Count on empty results
		count, err := store.Query().
			Status("nonexistent").
			Count()
		if err != nil {
			t.Fatalf("failed to count empty results: %v", err)
		}

		if count != 0 {
			t.Errorf("expected count 0 for empty results, got %d", count)
		}
	})
}
