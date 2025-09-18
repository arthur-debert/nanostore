package nanostore_test

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestFiltering(t *testing.T) {
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
			{
				Name:     "parent",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add test documents
	_, _ = store.Add("Fix bug in login", map[string]interface{}{
		"status":   "todo",
		"priority": "high",
	})
	doc2, _ := store.Add("Add feature X", map[string]interface{}{
		"status":   "in_progress",
		"priority": "medium",
	})
	_, _ = store.Add("Update documentation", map[string]interface{}{
		"status":   "done",
		"priority": "low",
	})
	_, _ = store.Add("Fix critical issue", map[string]interface{}{
		"status":   "todo",
		"priority": "high",
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "todo",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 todo documents, got %d", len(docs))
		}

		// Verify all returned docs have status=todo
		for _, doc := range docs {
			if doc.Dimensions["status"] != "todo" {
				t.Errorf("expected status=todo, got %v", doc.Dimensions["status"])
			}
		}
	})

	t.Run("FilterByMultipleDimensions", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status":   "todo",
				"priority": "high",
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 high priority todo documents, got %d", len(docs))
		}

		for _, doc := range docs {
			if doc.Dimensions["status"] != "todo" || doc.Dimensions["priority"] != "high" {
				t.Errorf("unexpected dimensions: %v", doc.Dimensions)
			}
		}
	})

	t.Run("FilterByUUID", func(t *testing.T) {
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"uuid": doc2,
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}

		if docs[0].UUID != doc2 {
			t.Errorf("expected UUID %s, got %s", doc2, docs[0].UUID)
		}
	})

	t.Run("FilterBySliceValues", func(t *testing.T) {
		// Filter by multiple status values (IN style)
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": []string{"todo", "done"},
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 3 {
			t.Errorf("expected 3 documents (2 todo + 1 done), got %d", len(docs))
		}

		// Verify no in_progress documents
		for _, doc := range docs {
			if doc.Dimensions["status"] == "in_progress" {
				t.Error("unexpected in_progress document")
			}
		}
	})

	t.Run("FilterBySearch", func(t *testing.T) {
		// Add documents with body text
		bodyTitle := "Search test"
		bodyText := "This document contains important information about searching"
		docWithBody, _ := store.Add(bodyTitle, nil)
		_ = store.Update(docWithBody, nanostore.UpdateRequest{
			Body: &bodyText,
		})

		// Search in title
		docs, err := store.List(nanostore.ListOptions{
			FilterBySearch: "fix",
		})
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents with 'fix' in title, got %d", len(docs))
		}

		// Search in body
		docs, err = store.List(nanostore.ListOptions{
			FilterBySearch: "important information",
		})
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document with text in body, got %d", len(docs))
		}

		// Case insensitive search
		docs, err = store.List(nanostore.ListOptions{
			FilterBySearch: "FIX",
		})
		if err != nil {
			t.Fatalf("failed to search: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents with case-insensitive 'FIX', got %d", len(docs))
		}
	})

	t.Run("CombineFiltersAndSearch", func(t *testing.T) {
		// Combine dimension filter with search
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"status": "todo",
			},
			FilterBySearch: "fix",
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		// Should only find todo items with "fix" in the title
		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}

		for _, doc := range docs {
			if doc.Dimensions["status"] != "todo" {
				t.Errorf("expected status=todo, got %v", doc.Dimensions["status"])
			}
			if !strings.Contains(strings.ToLower(doc.Title), "fix") {
				t.Errorf("expected 'fix' in title, got %s", doc.Title)
			}
		}
	})

	t.Run("FilterHierarchical", func(t *testing.T) {
		// Create parent-child documents
		parentID, _ := store.Add("Parent task", nil)
		child1ID, _ := store.Add("Child 1", map[string]interface{}{
			"parent_uuid": parentID,
		})
		child2ID, _ := store.Add("Child 2", map[string]interface{}{
			"parent_uuid": parentID,
		})

		// Filter by parent
		docs, err := store.List(nanostore.ListOptions{
			Filters: map[string]interface{}{
				"parent_uuid": parentID,
			},
		})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 children, got %d", len(docs))
		}

		// Verify we got the right children
		foundChild1 := false
		foundChild2 := false
		for _, doc := range docs {
			if doc.UUID == child1ID {
				foundChild1 = true
			}
			if doc.UUID == child2ID {
				foundChild2 = true
			}
		}

		if !foundChild1 || !foundChild2 {
			t.Error("didn't find expected child documents")
		}
	})
}

func TestEmptyFilters(t *testing.T) {
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

	// Add some documents
	_, _ = store.Add("Doc 1", nil)
	_, _ = store.Add("Doc 2", nil)
	_, _ = store.Add("Doc 3", nil)

	// List with no filters should return all
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("expected 3 documents, got %d", len(docs))
	}

	// List with nil filters should also return all
	docs, err = store.List(nanostore.ListOptions{
		Filters: nil,
	})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("expected 3 documents with nil filters, got %d", len(docs))
	}
}
