package todo

import (
	"testing"
)

func TestCompleteMultiple(t *testing.T) {
	app, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create test todos
	app.Add("First", nil)
	app.Add("Second", nil)
	app.Add("Third", nil)
	app.Add("Fourth", nil)

	// Complete items 1 and 3
	err = app.CompleteMultiple([]string{"1", "3"})
	if err != nil {
		t.Fatalf("failed to complete multiple: %v", err)
	}

	// Verify results
	pending, err := app.List(ListOptions{ShowAll: false})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("expected 2 pending items, got %d", len(pending))
	}

	// Check that Second and Fourth are the remaining items
	if pending[0].Title != "Second" {
		t.Errorf("expected first pending item to be 'Second', got '%s'", pending[0].Title)
	}
	if pending[1].Title != "Fourth" {
		t.Errorf("expected second pending item to be 'Fourth', got '%s'", pending[1].Title)
	}

	// Verify completed items when showing all
	all, err := app.List(ListOptions{ShowAll: true})
	if err != nil {
		t.Fatalf("failed to list all: %v", err)
	}

	if len(all) != 4 {
		t.Errorf("expected 4 total items, got %d", len(all))
	}

	// Count completed items
	completed := 0
	for _, item := range all {
		if item.IsCompleted {
			completed++
		}
	}

	if completed != 2 {
		t.Errorf("expected 2 completed items, got %d", completed)
	}
}

func TestCompleteMultipleWithInvalidID(t *testing.T) {
	app, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create test todos
	app.Add("First", nil)
	app.Add("Second", nil)

	// Try to complete with one invalid ID
	err = app.CompleteMultiple([]string{"1", "99"})
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}

	// Verify no items were completed due to early error
	pending, _ := app.List(ListOptions{ShowAll: false})
	if len(pending) != 2 {
		t.Errorf("expected 2 pending items (none completed due to error), got %d", len(pending))
	}
}

func TestCompleteMultipleEmpty(t *testing.T) {
	app, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Complete with empty list should succeed
	err = app.CompleteMultiple([]string{})
	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
}

func TestCompleteMultipleHierarchical(t *testing.T) {
	app, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create hierarchical todos
	id1, _ := app.Add("Parent 1", nil)
	app.Add("Child 1.1", &id1)
	app.Add("Child 1.2", &id1)
	id2, _ := app.Add("Parent 2", nil)
	app.Add("Child 2.1", &id2)

	// Complete multiple items across hierarchy
	err = app.CompleteMultiple([]string{"1.1", "2", "2.1"})
	if err != nil {
		t.Fatalf("failed to complete multiple: %v", err)
	}

	// Verify results
	pending, err := app.List(ListOptions{ShowAll: false})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	// Should have Parent 1 and Child 1.2 remaining
	if len(pending) != 1 {
		t.Errorf("expected 1 pending parent, got %d", len(pending))
	}

	if pending[0].Title != "Parent 1" {
		t.Errorf("expected 'Parent 1' to remain, got '%s'", pending[0].Title)
	}

	if len(pending[0].Children) != 1 {
		t.Errorf("expected 1 pending child, got %d", len(pending[0].Children))
	}

	if pending[0].Children[0].Title != "Child 1.2" {
		t.Errorf("expected 'Child 1.2' to remain, got '%s'", pending[0].Children[0].Title)
	}
}
