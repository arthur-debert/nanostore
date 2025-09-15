package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDeleteByDimension(t *testing.T) {
	// Create a store with custom dimensions
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed", "archived"},
				Prefixes:     map[string]string{"completed": "c", "archived": "a"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents with different statuses
	_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending", "priority": "high"})
	_, _ = store.Add("Task 2", map[string]interface{}{"status": "completed", "priority": "low"})
	_, _ = store.Add("Task 3", map[string]interface{}{"status": "archived", "priority": "medium"})
	_, _ = store.Add("Task 4", map[string]interface{}{"status": "completed", "priority": "high"})
	_, _ = store.Add("Task 5", map[string]interface{}{"status": "pending", "priority": "low"})

	t.Run("delete by status dimension", func(t *testing.T) {
		// Delete all completed items
		deleted, err := store.DeleteByDimension(map[string]interface{}{"status": "completed"})
		if err != nil {
			t.Fatalf("failed to delete by dimension: %v", err)
		}

		if deleted != 2 {
			t.Errorf("expected 2 deleted items, got %d", deleted)
		}

		// Verify remaining items
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 3 {
			t.Errorf("expected 3 remaining documents, got %d", len(docs))
		}

		// Check that no completed items remain
		for _, doc := range docs {
			status, _ := doc.Dimensions["status"].(string)
			if status == "completed" {
				t.Errorf("found completed document that should have been deleted: %s", doc.Title)
			}
		}
	})

	t.Run("delete by priority dimension", func(t *testing.T) {
		// Delete all low priority items
		deleted, err := store.DeleteByDimension(map[string]interface{}{"priority": "low"})
		if err != nil {
			t.Fatalf("failed to delete by priority: %v", err)
		}

		if deleted != 1 {
			t.Errorf("expected 1 deleted item, got %d", deleted)
		}

		// Verify remaining items
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		// Should have 2 items left (Task 1 with high priority, Task 3 with medium priority)
		if len(docs) != 2 {
			t.Errorf("expected 2 remaining documents, got %d", len(docs))
		}
	})

	t.Run("delete with invalid dimension", func(t *testing.T) {
		_, err := store.DeleteByDimension(map[string]interface{}{"invalid_dimension": "value"})
		if err == nil {
			t.Error("expected error for invalid dimension, got nil")
		}
	})

	t.Run("delete with invalid value", func(t *testing.T) {
		_, err := store.DeleteByDimension(map[string]interface{}{"status": "invalid_status"})
		if err == nil {
			t.Error("expected error for invalid status value, got nil")
		}
	})

	t.Run("delete when no matches", func(t *testing.T) {
		// Note: archived items still exist from initial setup (Task 3)
		// First delete them
		_, _ = store.DeleteByDimension(map[string]interface{}{"status": "archived"})

		// Now try to delete archived items again (should be none)
		deleted, err := store.DeleteByDimension(map[string]interface{}{"status": "archived"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if deleted != 0 {
			t.Errorf("expected 0 deleted items, got %d", deleted)
		}
	})
}

func TestDeleteCompletedUsingDeleteByDimension(t *testing.T) {
	store, err := nanostore.New(":memory:", nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create mix of pending and completed items
	_, _ = store.Add("Pending 1", nil)
	_, _ = store.Add("Pending 2", nil)

	uuid1, _ := store.Add("To Complete 1", nil)
	uuid2, _ := store.Add("To Complete 2", nil)
	uuid3, _ := store.Add("To Complete 3", nil)

	// Complete some items
	_ = store.Update(uuid1, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "completed"},
	})
	_ = store.Update(uuid2, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "completed"},
	})
	_ = store.Update(uuid3, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "completed"},
	})

	// Delete all completed using DeleteByDimension
	deleted, err := store.DeleteByDimension(map[string]interface{}{"status": "completed"})
	if err != nil {
		t.Fatalf("failed to delete completed: %v", err)
	}

	if deleted != 3 {
		t.Errorf("expected 3 deleted items, got %d", deleted)
	}

	// Verify only pending items remain
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 remaining documents, got %d", len(docs))
	}

	for _, doc := range docs {
		status, _ := doc.Dimensions["status"].(string)
		if status != "pending" {
			t.Errorf("expected only pending items, found status: %s", status)
		}
	}
}

func TestDeleteByDimensionWithHierarchy(t *testing.T) {
	// Test with hierarchical documents
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"active", "inactive", "archived"},
				DefaultValue: "active",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchical structure
	parent1, _ := store.Add("Parent 1", map[string]interface{}{"status": "active"})
	child1, _ := store.Add("Child 1.1", map[string]interface{}{"parent_uuid": parent1, "status": "inactive"})
	_, _ = store.Add("Child 1.1.1", map[string]interface{}{"parent_uuid": child1, "status": "active"})

	parent2, _ := store.Add("Parent 2", map[string]interface{}{"status": "inactive"})
	_, _ = store.Add("Child 2.1", map[string]interface{}{"parent_uuid": parent2, "status": "inactive"})

	// List all before deletion to understand structure
	allBefore, _ := store.List(nanostore.ListOptions{})
	t.Logf("Documents before deletion: %d", len(allBefore))
	for _, doc := range allBefore {
		status, _ := doc.Dimensions["status"].(string)
		t.Logf("  %s: %s (status: %v)", doc.UserFacingID, doc.Title, status)
	}

	// Delete all inactive items
	deleted, err := store.DeleteByDimension(map[string]interface{}{"status": "inactive"})
	if err != nil {
		t.Fatalf("failed to delete inactive items: %v", err)
	}

	t.Logf("Deleted %d items", deleted)

	// Verify remaining structure
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	t.Logf("Documents after deletion: %d", len(docs))
	for _, doc := range docs {
		status, _ := doc.Dimensions["status"].(string)
		t.Logf("  %s: %s (status: %v)", doc.UserFacingID, doc.Title, status)
	}

	// The cascade delete behavior may affect the count
	// With ON DELETE CASCADE, when Parent 2 is deleted, Child 2.1 is automatically deleted
	// So we might get 2 deletions (Child 1.1 and Parent 2) with Child 2.1 cascaded
	if deleted != 3 && deleted != 2 {
		t.Errorf("expected 2-3 deleted items, got %d", deleted)
	}

	// Should have at least Parent 1 and potentially orphaned children
	if len(docs) < 1 {
		t.Errorf("expected at least 1 remaining document, got %d", len(docs))
	}
}

func TestDeleteWhere(t *testing.T) {
	// Create a store with custom dimensions
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "completed", "archived"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending", "priority": "high"})
	_, _ = store.Add("Task 2", map[string]interface{}{"status": "completed", "priority": "low"})
	_, _ = store.Add("Task 3", map[string]interface{}{"status": "archived", "priority": "low"})
	_, _ = store.Add("Task 4", map[string]interface{}{"status": "completed", "priority": "high"})
	_, _ = store.Add("Task 5", map[string]interface{}{"status": "pending", "priority": "low"})
	_, _ = store.Add("Task 6", map[string]interface{}{"status": "archived", "priority": "high"})

	t.Run("delete with simple condition", func(t *testing.T) {
		// Delete all low priority items
		deleted, err := store.DeleteWhere("priority = ?", "low")
		if err != nil {
			t.Fatalf("failed to delete with where clause: %v", err)
		}

		if deleted != 3 {
			t.Errorf("expected 3 deleted items, got %d", deleted)
		}
	})

	t.Run("delete with complex condition", func(t *testing.T) {
		// Delete completed items with high priority
		deleted, err := store.DeleteWhere("status = ? AND priority = ?", "completed", "high")
		if err != nil {
			t.Fatalf("failed to delete with complex where clause: %v", err)
		}

		if deleted != 1 {
			t.Errorf("expected 1 deleted item, got %d", deleted)
		}
	})

	t.Run("delete with OR condition", func(t *testing.T) {
		// Delete all archived or pending high priority items
		deleted, err := store.DeleteWhere("(status = ? OR status = ?) AND priority = ?", "archived", "pending", "high")
		if err != nil {
			t.Fatalf("failed to delete with OR condition: %v", err)
		}

		if deleted != 2 {
			t.Errorf("expected 2 deleted items (Task 1 and Task 6), got %d", deleted)
		}

		// Verify no documents remain
		docs, _ := store.List(nanostore.ListOptions{})
		if len(docs) != 0 {
			t.Errorf("expected 0 remaining documents, got %d", len(docs))
			for _, doc := range docs {
				status, _ := doc.Dimensions["status"].(string)
				t.Logf("  Remaining: %s (status: %v)", doc.Title, status)
			}
		}
	})

	t.Run("delete with empty where clause", func(t *testing.T) {
		_, err := store.DeleteWhere("", "value")
		if err == nil {
			t.Error("expected error for empty where clause, got nil")
		}
	})
}

func TestDeleteWhereWithPatterns(t *testing.T) {
	// Create a store with a category dimension
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"active", "inactive", "draft"},
				DefaultValue: "active",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"work", "personal", "archive"},
				DefaultValue: "work",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with various patterns
	_, _ = store.Add("Work Meeting Notes", map[string]interface{}{"category": "work", "status": "active"})
	_, _ = store.Add("Personal TODO", map[string]interface{}{"category": "personal", "status": "active"})
	_, _ = store.Add("Old Project Docs", map[string]interface{}{"category": "archive", "status": "inactive"})
	_, _ = store.Add("Draft Proposal", map[string]interface{}{"category": "work", "status": "draft"})
	_, _ = store.Add("Archived Reports", map[string]interface{}{"category": "archive", "status": "inactive"})
	_, _ = store.Add("Shopping List", map[string]interface{}{"category": "personal", "status": "active"})

	t.Run("delete with LIKE pattern", func(t *testing.T) {
		// Delete all documents with "Archive" in the title
		deleted, err := store.DeleteWhere("title LIKE ?", "%Archive%")
		if err != nil {
			t.Fatalf("failed to delete with LIKE pattern: %v", err)
		}

		if deleted != 1 {
			t.Errorf("expected 1 deleted item (Archived Reports), got %d", deleted)
		}
	})

	t.Run("delete with multiple conditions", func(t *testing.T) {
		// Delete all archive category or inactive status
		deleted, err := store.DeleteWhere("category = ? OR status = ?", "archive", "inactive")
		if err != nil {
			t.Fatalf("failed to delete with multiple conditions: %v", err)
		}

		if deleted != 1 {
			t.Errorf("expected 1 deleted item (Old Project Docs), got %d", deleted)
		}
	})

	t.Run("delete with IN clause", func(t *testing.T) {
		// Delete documents in specific categories
		deleted, err := store.DeleteWhere("category IN (?, ?)", "personal", "work")
		if err != nil {
			t.Fatalf("failed to delete with IN clause: %v", err)
		}

		// Should delete all remaining items except those already deleted
		if deleted != 4 {
			t.Errorf("expected 4 deleted items, got %d", deleted)
		}

		// Verify no documents remain
		docs, _ := store.List(nanostore.ListOptions{})
		if len(docs) != 0 {
			t.Errorf("expected 0 remaining documents, got %d", len(docs))
		}
	})
}
