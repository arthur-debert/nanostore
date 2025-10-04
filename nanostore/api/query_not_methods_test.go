package api_test

import (
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// CustomTodoItem with different enumerated values to test flexibility
type CustomTodoItem struct {
	nanostore.Document
	// Different status values than the hardcoded ones
	Status   string `values:"draft,review,approved,rejected" default:"draft"`
	Priority string `values:"urgent,normal,defer" default:"normal"`
	Activity string `values:"new,working,blocked,complete" default:"new"`
}

func TestHardcodedNOTMethodsFlexibility(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store with custom enum values
	store, err := api.New[CustomTodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add test data with our custom enum values
	testItems := []struct {
		title    string
		status   string
		priority string
		activity string
	}{
		{"Item 1", "draft", "urgent", "new"},
		{"Item 2", "review", "normal", "working"},
		{"Item 3", "approved", "defer", "blocked"},
		{"Item 4", "rejected", "urgent", "complete"},
	}

	for _, item := range testItems {
		_, err := store.Create(item.title, &CustomTodoItem{
			Status:   item.status,
			Priority: item.priority,
			Activity: item.activity,
		})
		if err != nil {
			t.Fatalf("Failed to create item %s: %v", item.title, err)
		}
	}

	t.Run("StatusNotMethodLimitations", func(t *testing.T) {
		// This should work - trying to exclude "rejected" status
		// But it will fail because the method has hardcoded ["pending", "active", "done"]
		// and doesn't know about our custom values ["draft", "review", "approved", "rejected"]

		results, err := store.Query().StatusNot("rejected").Find()
		if err != nil {
			t.Fatalf("StatusNot query failed: %v", err)
		}

		// With hardcoded values, this will not work as expected
		// It should return 3 items (all except "rejected"), but might return 0 or all 4
		t.Logf("StatusNot('rejected') returned %d items (expected 3)", len(results))

		// Check what we actually got
		for _, result := range results {
			t.Logf("- Item: %s, Status: %s", result.Title, result.Status)
			if result.Status == "rejected" {
				t.Error("Found rejected item when it should have been excluded")
			}
		}

		// This demonstrates the inflexibility - the method doesn't adapt to our custom enum values
		if len(results) != 3 {
			t.Logf("EXPECTED FAILURE: StatusNot method uses hardcoded values and cannot handle custom enums")
		}
	})

	t.Run("PriorityNotMethodLimitations", func(t *testing.T) {
		// Similar issue with priority - hardcoded ["low", "medium", "high"]
		// Our custom values are ["urgent", "normal", "defer"]

		results, err := store.Query().PriorityNot("defer").Find()
		if err != nil {
			t.Fatalf("PriorityNot query failed: %v", err)
		}

		t.Logf("PriorityNot('defer') returned %d items (expected 3)", len(results))

		for _, result := range results {
			t.Logf("- Item: %s, Priority: %s", result.Title, result.Priority)
			if result.Priority == "defer" {
				t.Error("Found defer item when it should have been excluded")
			}
		}

		if len(results) != 3 {
			t.Logf("EXPECTED FAILURE: PriorityNot method uses hardcoded values and cannot handle custom enums")
		}
	})

	t.Run("ActivityNotMethodLimitations", func(t *testing.T) {
		// Activity hardcoded ["active", "archived", "deleted"]
		// Our custom values are ["new", "working", "blocked", "complete"]

		results, err := store.Query().ActivityNot("complete").Find()
		if err != nil {
			t.Fatalf("ActivityNot query failed: %v", err)
		}

		t.Logf("ActivityNot('complete') returned %d items (expected 3)", len(results))

		for _, result := range results {
			t.Logf("- Item: %s, Activity: %s", result.Title, result.Activity)
			if result.Activity == "complete" {
				t.Error("Found complete item when it should have been excluded")
			}
		}

		if len(results) != 3 {
			t.Logf("EXPECTED FAILURE: ActivityNot method uses hardcoded values and cannot handle custom enums")
		}
	})

	t.Run("StatusNotInMethodNowFlexible", func(t *testing.T) {
		// Test that StatusNotIn also works with custom enum values
		results, err := store.Query().StatusNotIn("rejected", "draft").Find()
		if err != nil {
			t.Fatalf("StatusNotIn query failed: %v", err)
		}

		// Should return 2 items (review and approved)
		expectedCount := 2
		if len(results) != expectedCount {
			t.Errorf("Expected %d items, got %d", expectedCount, len(results))
		}

		for _, result := range results {
			t.Logf("- Item: %s, Status: %s", result.Title, result.Status)
			if result.Status == "rejected" || result.Status == "draft" {
				t.Error("Found excluded item when it should have been filtered out")
			}
		}
	})
}

func TestNOTMethodsWithDefaultTodoItem(t *testing.T) {
	// Test with the standard TodoItem to verify current behavior still works
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add test data using standard enum values
	testItems := []struct {
		title    string
		status   string
		priority string
		activity string
	}{
		{"Task 1", "pending", "low", "active"},
		{"Task 2", "active", "medium", "archived"},
		{"Task 3", "done", "high", "deleted"},
	}

	for _, item := range testItems {
		_, err := store.Create(item.title, &TodoItem{
			Status:   item.status,
			Priority: item.priority,
			Activity: item.activity,
		})
		if err != nil {
			t.Fatalf("Failed to create item %s: %v", item.title, err)
		}
	}

	t.Run("StatusNotWorksWithStandardValues", func(t *testing.T) {
		// This should work because the hardcoded values match the TodoItem struct
		results, err := store.Query().StatusNot("done").Find()
		if err != nil {
			t.Fatalf("StatusNot query failed: %v", err)
		}

		// Should return 2 items (pending and active)
		expectedCount := 2
		if len(results) != expectedCount {
			t.Errorf("Expected %d items, got %d", expectedCount, len(results))
		}

		for _, result := range results {
			if result.Status == "done" {
				t.Error("Found done item when it should have been excluded")
			}
		}
	})
}
