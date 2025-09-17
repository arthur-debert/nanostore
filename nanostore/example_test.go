package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

// TestBasicCRUD demonstrates basic create, read, update, delete operations
func TestBasicCRUD(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New("/tmp/test.json", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create
	id, err := store.Add("Buy milk", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Read
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	// Update
	err = store.Update(id, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "done"},
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Delete
	err = store.Delete(id, false)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
}

// TestHierarchicalDocuments demonstrates parent-child relationships
func TestHierarchicalDocuments(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New("/tmp/test_hierarchy.json", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create parent
	parentID, _ := store.Add("Project", nil)

	// Create children
	store.Add("Task 1", map[string]interface{}{"parent_uuid": parentID})
	store.Add("Task 2", map[string]interface{}{"parent_uuid": parentID})

	// List should show hierarchical IDs like "1", "1.1", "1.2"
	docs, _ := store.List(nanostore.ListOptions{})
	if len(docs) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(docs))
	}
}

// TestFiltering demonstrates dimension-based filtering
func TestFiltering(t *testing.T) {
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
				Values:       []string{"low", "high"},
				DefaultValue: "low",
			},
		},
	}

	store, err := nanostore.New("/tmp/test_filter.json", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Add test data
	store.Add("Low priority todo", nil)
	store.Add("High priority todo", map[string]interface{}{"priority": "high"})
	store.Add("Done task", map[string]interface{}{"status": "done"})

	// Filter by status
	todos, _ := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "todo"},
	})
	if len(todos) != 2 {
		t.Errorf("expected 2 todo items, got %d", len(todos))
	}

	// Filter by multiple dimensions
	highPriorityTodos, _ := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{
			"status":   "todo",
			"priority": "high",
		},
	})
	if len(highPriorityTodos) != 1 {
		t.Errorf("expected 1 high priority todo, got %d", len(highPriorityTodos))
	}
}

// TestIDResolution demonstrates converting user-facing IDs to UUIDs
func TestIDResolution(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New("/tmp/test_resolve.json", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Add documents
	uuid1, _ := store.Add("Todo item", nil)
	uuid2, _ := store.Add("Done item", map[string]interface{}{"status": "done"})

	// Resolve user-facing IDs
	resolved1, _ := store.ResolveUUID("1")    // First todo item
	resolved2, _ := store.ResolveUUID("d1")   // First done item

	if resolved1 != uuid1 {
		t.Errorf("ID resolution failed for '1'")
	}
	if resolved2 != uuid2 {
		t.Errorf("ID resolution failed for 'd1'")
	}
}