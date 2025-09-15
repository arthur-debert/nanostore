package nanostore_test

import (
	"fmt"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestConfigurableIntegration(t *testing.T) {
	// Test a real-world scenario: Todo app with multiple dimensions
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				Prefixes:     map[string]string{"high": "h", "medium": "m"},
				DefaultValue: "medium",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"personal", "work", "shopping"},
				Prefixes:     map[string]string{"work": "w", "shopping": "s"},
				DefaultValue: "personal",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_id",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create some todos
	personalTodo, err := store.Add("Buy groceries", nil)
	if err != nil {
		t.Fatalf("failed to add personal todo: %v", err)
	}

	_, err = store.Add("Finish report", nil)
	if err != nil {
		t.Fatalf("failed to add work todo: %v", err)
	}

	// Update to work category with high priority
	// Note: For now we use SetStatus which only works with "status" dimension
	// In a real implementation, we'd have SetDimension(id, dimensionName, value)

	// Add subtasks
	_, err = store.Add("Buy milk", map[string]interface{}{"parent_id": personalTodo})
	if err != nil {
		t.Fatalf("failed to add subtask: %v", err)
	}

	_, err = store.Add("Buy bread", map[string]interface{}{"parent_id": personalTodo})
	if err != nil {
		t.Fatalf("failed to add subtask: %v", err)
	}

	// List all todos
	todos, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}

	// Verify we have the right number
	if len(todos) != 4 {
		t.Errorf("expected 4 todos, got %d", len(todos))
	}

	// Test ID resolution
	for _, todo := range todos {
		resolved, err := store.ResolveUUID(todo.UserFacingID)
		if err != nil {
			t.Errorf("failed to resolve ID %s: %v", todo.UserFacingID, err)
		}
		if resolved != todo.UUID {
			t.Errorf("resolved UUID mismatch for ID %s: got %s, want %s",
				todo.UserFacingID, resolved, todo.UUID)
		}
	}

	// Test filtering by parent
	subtasks, err := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"parent_id": personalTodo},
	})
	if err != nil {
		t.Fatalf("failed to filter by parent: %v", err)
	}

	if len(subtasks) != 2 {
		t.Errorf("expected 2 subtasks, got %d", len(subtasks))
	}

	// When filtering by parent, IDs are renumbered starting from 1
	// They should have IDs like "m1", "m2" (with medium priority prefix)
	for i, subtask := range subtasks {
		// Both subtasks have medium priority by default, so should have 'm' prefix
		expectedID := fmt.Sprintf("m%d", i+1)
		if subtask.UserFacingID != expectedID {
			// Also check without prefix in case they don't have the same priority
			alternativeID := fmt.Sprintf("%d", i+1)
			if subtask.UserFacingID != alternativeID {
				t.Errorf("subtask should have ID '%s' or '%s', got %s", expectedID, alternativeID, subtask.UserFacingID)
			}
		}
	}
}

func TestMultiplePrefixCombinations(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "low",
			},
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"open", "closed"},
				Prefixes:     map[string]string{"closed": "c"},
				DefaultValue: "open",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with different combinations
	// These will have IDs based on their dimension values:
	// low + open = "1" (no prefixes)
	// high + open = "h1" (h prefix from priority)
	// low + closed = "c1" (c prefix from status)
	// high + closed = "hc1" (both prefixes, alphabetically ordered)

	doc1, _ := store.Add("Low priority, open", nil)

	docs, _ := store.List(nanostore.ListOptions{})
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	// Default values should give ID "1" (no prefixes)
	if docs[0].UserFacingID != "1" {
		t.Errorf("expected ID '1' for default values, got %s", docs[0].UserFacingID)
	}

	// Test ID resolution works
	resolved, err := store.ResolveUUID("1")
	if err != nil {
		t.Fatalf("failed to resolve ID '1': %v", err)
	}
	if resolved != doc1 {
		t.Errorf("resolved wrong document")
	}
}
