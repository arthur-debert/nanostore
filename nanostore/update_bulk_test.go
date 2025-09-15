package nanostore_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestUpdateByDimension(t *testing.T) {
	// Create a store with custom dimensions
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "in_progress", "completed", "archived"},
				Prefixes:     map[string]string{"completed": "c", "archived": "a", "in_progress": "p"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high", "urgent"},
				DefaultValue: "medium",
			},
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	t.Run("update status by dimension", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents with different statuses
		_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending", "priority": "high"})
		_, _ = store.Add("Task 2", map[string]interface{}{"status": "pending", "priority": "low"})
		_, _ = store.Add("Task 3", map[string]interface{}{"status": "completed", "priority": "medium"})
		_, _ = store.Add("Task 4", map[string]interface{}{"status": "pending", "priority": "high"})
		_, _ = store.Add("Task 5", map[string]interface{}{"status": "archived", "priority": "low"})

		// Update all pending items to in_progress
		newTitle := "Updated Task"
		updated, err := store.UpdateByDimension("status", "pending", nanostore.UpdateRequest{
			Title: &newTitle,
			Dimensions: map[string]interface{}{
				"status": "in_progress",
			},
		})
		if err != nil {
			t.Fatalf("failed to update by dimension: %v", err)
		}

		if updated != 3 {
			t.Errorf("expected 3 updated items, got %d", updated)
		}

		// Verify the updates
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"status": "in_progress"},
		})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 3 {
			t.Errorf("expected 3 in_progress documents, got %d", len(docs))
		}

		// Check that all updated documents have the new title
		for _, doc := range docs {
			if doc.Title != "Updated Task" {
				t.Errorf("expected title 'Updated Task', got '%s'", doc.Title)
			}
		}

		// Verify that non-matching documents were not updated
		completedDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"status": "completed"},
		})
		if len(completedDocs) != 1 || completedDocs[0].Title == "Updated Task" {
			t.Error("completed document was incorrectly updated")
		}
	})

	t.Run("update priority by dimension", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		_, _ = store.Add("Low Priority 1", map[string]interface{}{"priority": "low"})
		_, _ = store.Add("Low Priority 2", map[string]interface{}{"priority": "low"})
		_, _ = store.Add("High Priority", map[string]interface{}{"priority": "high"})

		// Update all low priority items to medium
		newBody := "This task has been escalated"
		updated, err := store.UpdateByDimension("priority", "low", nanostore.UpdateRequest{
			Body: &newBody,
			Dimensions: map[string]interface{}{
				"priority": "medium",
			},
		})
		if err != nil {
			t.Fatalf("failed to update by priority: %v", err)
		}

		if updated != 2 {
			t.Errorf("expected 2 updated items, got %d", updated)
		}

		// Verify updates
		mediumDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"priority": "medium"},
		})
		if len(mediumDocs) != 2 {
			t.Errorf("expected 2 medium priority documents, got %d", len(mediumDocs))
		}

		for _, doc := range mediumDocs {
			if doc.Body != newBody {
				t.Errorf("expected body '%s', got '%s'", newBody, doc.Body)
			}
		}
	})

	t.Run("update with invalid dimension", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _ = store.Add("Task", map[string]interface{}{})

		_, err = store.UpdateByDimension("invalid_dimension", "value", nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{"status": "completed"},
		})
		if err == nil {
			t.Error("expected error for invalid dimension, got nil")
		}
		if !strings.Contains(err.Error(), "dimension 'invalid_dimension' not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("update with invalid dimension value", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _ = store.Add("Task", map[string]interface{}{"status": "pending"})

		_, err = store.UpdateByDimension("status", "invalid_status", nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{"priority": "high"},
		})
		if err == nil {
			t.Error("expected error for invalid dimension value, got nil")
		}
		if !strings.Contains(err.Error(), "invalid value 'invalid_status'") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("update with no matching documents", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create documents but none with status=archived
		_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending"})
		_, _ = store.Add("Task 2", map[string]interface{}{"status": "completed"})

		newTitle := "Won't be applied"
		updated, err := store.UpdateByDimension("status", "archived", nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updated != 0 {
			t.Errorf("expected 0 updated items, got %d", updated)
		}
	})

	t.Run("update only non-dimension fields", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		_, _ = store.Add("Original Title 1", map[string]interface{}{"status": "pending"})
		_, _ = store.Add("Original Title 2", map[string]interface{}{"status": "pending"})

		// Update only title and body, not dimensions
		newTitle := "Updated Title"
		newBody := "Updated Body"
		updated, err := store.UpdateByDimension("status", "pending", nanostore.UpdateRequest{
			Title: &newTitle,
			Body:  &newBody,
		})
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		if updated != 2 {
			t.Errorf("expected 2 updated items, got %d", updated)
		}

		// Verify updates
		docs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"status": "pending"},
		})
		for _, doc := range docs {
			if doc.Title != newTitle || doc.Body != newBody {
				t.Errorf("document not properly updated: title=%s, body=%s", doc.Title, doc.Body)
			}
			// Status should remain pending
			if status, _ := doc.Dimensions["status"].(string); status != "pending" {
				t.Errorf("status was changed unexpectedly to %s", status)
			}
		}
	})
}

func TestUpdateWhere(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "in_progress", "completed"},
				Prefixes:     map[string]string{"completed": "c", "in_progress": "p"},
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

	t.Run("update with simple where clause", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending", "priority": "high"})
		_, _ = store.Add("Task 2", map[string]interface{}{"status": "pending", "priority": "low"})
		_, _ = store.Add("Task 3", map[string]interface{}{"status": "completed", "priority": "high"})

		// Update using WHERE clause
		newTitle := "High Priority Task"
		updated, err := store.UpdateWhere("priority = ?", nanostore.UpdateRequest{
			Title: &newTitle,
		}, "high")
		if err != nil {
			t.Fatalf("failed to update where: %v", err)
		}

		if updated != 2 {
			t.Errorf("expected 2 updated items, got %d", updated)
		}

		// Verify updates
		docs, _ := store.List(nanostore.ListOptions{})
		highPriorityCount := 0
		for _, doc := range docs {
			if priority, _ := doc.Dimensions["priority"].(string); priority == "high" {
				highPriorityCount++
				if doc.Title != newTitle {
					t.Errorf("high priority doc not updated: %s", doc.Title)
				}
			}
		}
		if highPriorityCount != 2 {
			t.Errorf("expected 2 high priority docs, found %d", highPriorityCount)
		}
	})

	t.Run("update with complex where clause", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		_, _ = store.Add("Task 1", map[string]interface{}{"status": "pending", "priority": "high"})
		_, _ = store.Add("Task 2", map[string]interface{}{"status": "pending", "priority": "low"})
		_, _ = store.Add("Task 3", map[string]interface{}{"status": "completed", "priority": "high"})
		_, _ = store.Add("Task 4", map[string]interface{}{"status": "in_progress", "priority": "high"})

		// Update documents with status pending OR in_progress AND high priority
		newBody := "Urgent task"
		updated, err := store.UpdateWhere("(status = ? OR status = ?) AND priority = ?", nanostore.UpdateRequest{
			Body: &newBody,
			Dimensions: map[string]interface{}{
				"priority": "medium", // Downgrade priority
			},
		}, "pending", "in_progress", "high")
		if err != nil {
			t.Fatalf("failed to update where: %v", err)
		}

		if updated != 2 {
			t.Errorf("expected 2 updated items, got %d", updated)
		}

		// Verify updates
		docs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"priority": "medium"},
		})
		if len(docs) != 2 {
			t.Errorf("expected 2 medium priority documents, got %d", len(docs))
		}
		for _, doc := range docs {
			if doc.Body != newBody {
				t.Errorf("document body not updated: %s", doc.Body)
			}
		}
	})

	t.Run("update with LIKE clause", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		_, _ = store.Add("Project Alpha", map[string]interface{}{})
		_, _ = store.Add("Project Beta", map[string]interface{}{})
		_, _ = store.Add("Task 1", map[string]interface{}{})
		_, _ = store.Add("Project Gamma", map[string]interface{}{})

		// Update all documents with title starting with "Project"
		newStatus := "in_progress"
		updated, err := store.UpdateWhere("title LIKE ?", nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status": newStatus,
			},
		}, "Project%")
		if err != nil {
			t.Fatalf("failed to update where: %v", err)
		}

		if updated != 3 {
			t.Errorf("expected 3 updated items, got %d", updated)
		}

		// Verify updates
		docs, _ := store.List(nanostore.ListOptions{})
		for _, doc := range docs {
			if strings.HasPrefix(doc.Title, "Project") {
				if status, _ := doc.Dimensions["status"].(string); status != newStatus {
					t.Errorf("project document not updated: %s", doc.Title)
				}
			}
		}
	})

	t.Run("update with empty where clause", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _ = store.Add("Task", map[string]interface{}{})

		newTitle := "Updated"
		_, err = store.UpdateWhere("", nanostore.UpdateRequest{
			Title: &newTitle,
		})
		if err == nil {
			t.Error("expected error for empty where clause, got nil")
		}
		if !strings.Contains(err.Error(), "where clause cannot be empty") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("update with no fields to update", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _ = store.Add("Task", map[string]interface{}{})

		_, err = store.UpdateWhere("status = ?", nanostore.UpdateRequest{}, "pending")
		if err == nil {
			t.Error("expected error for no update fields, got nil")
		}
		if !strings.Contains(err.Error(), "no fields to update") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("update with invalid dimension in update request", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _ = store.Add("Task", map[string]interface{}{})

		_, err = store.UpdateWhere("status = ?", nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"invalid_dimension": "value",
			},
		}, "pending")
		if err == nil {
			t.Error("expected error for invalid dimension in update, got nil")
		}
		if !strings.Contains(err.Error(), "dimension 'invalid_dimension' not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("update with IN clause", func(t *testing.T) {
		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create test documents
		id1, _ := store.Add("Task 1", map[string]interface{}{})
		id2, _ := store.Add("Task 2", map[string]interface{}{})
		id3, _ := store.Add("Task 3", map[string]interface{}{})
		_, _ = store.Add("Task 4", map[string]interface{}{})

		// Update specific documents by UUID
		newPriority := "high"
		updated, err := store.UpdateWhere("uuid IN (?, ?, ?)", nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{
				"priority": newPriority,
			},
		}, id1, id2, id3)
		if err != nil {
			t.Fatalf("failed to update where: %v", err)
		}

		if updated != 3 {
			t.Errorf("expected 3 updated items, got %d", updated)
		}

		// Verify updates
		highPriorityDocs, _ := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{"priority": "high"},
		})
		if len(highPriorityDocs) != 3 {
			t.Errorf("expected 3 high priority documents, got %d", len(highPriorityDocs))
		}
	})
}

func TestBulkUpdatePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "processing", "completed"},
				DefaultValue: "pending",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create many documents
	const docCount = 1000
	for i := 0; i < docCount; i++ {
		status := "pending"
		if i%3 == 0 {
			status = "processing"
		}
		_, err := store.Add(fmt.Sprintf("Task %d", i), map[string]interface{}{
			"status": status,
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
	}

	// Update all pending documents
	newStatus := "completed"
	updated, err := store.UpdateByDimension("status", "pending", nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{
			"status": newStatus,
		},
	})
	if err != nil {
		t.Fatalf("failed to bulk update: %v", err)
	}

	expectedUpdates := (docCount * 2) / 3 // approximately 666 documents
	if updated < expectedUpdates-10 || updated > expectedUpdates+10 {
		t.Errorf("expected approximately %d updates, got %d", expectedUpdates, updated)
	}

	// Verify all are now either processing or completed
	pendingDocs, _ := store.List(nanostore.ListOptions{
		Filters: map[string]interface{}{"status": "pending"},
	})
	if len(pendingDocs) != 0 {
		t.Errorf("found %d pending documents, expected 0", len(pendingDocs))
	}
}
