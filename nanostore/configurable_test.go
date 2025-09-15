package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestConfigurableStore(t *testing.T) {
	t.Run("custom dimensions", func(t *testing.T) {
		config := nanostore.Config{
			Dimensions: []nanostore.DimensionConfig{
				{
					Name:         "priority",
					Type:         nanostore.Enumerated,
					Values:       []string{"low", "normal", "high"},
					Prefixes:     map[string]string{"high": "h"},
					DefaultValue: "normal",
				},
				{
					Name:         "status",
					Type:         nanostore.Enumerated,
					Values:       []string{"todo", "in_progress", "done"},
					Prefixes:     map[string]string{"in_progress": "p", "done": "d"},
					DefaultValue: "todo",
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
			t.Fatalf("failed to create configurable store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add some documents with default dimensions
		doc1, err := store.Add("First Task", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		doc2, err := store.Add("Second Task", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// List documents - should have default IDs
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}

		// First document should have ID "1" (no prefix for defaults)
		if docs[0].UserFacingID != "1" {
			t.Errorf("expected first document ID to be '1', got '%s'", docs[0].UserFacingID)
		}

		// Test setting custom status
		if err := store.Update(doc1, nanostore.UpdateRequest{
			Dimensions: map[string]interface{}{"status": "done"},
		}); err != nil {
			t.Fatalf("failed to set status: %v", err)
		}

		// List again to see the updated ID
		docs, err = store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		// Document with 'done' status should have 'd' prefix
		foundDone := false
		t.Logf("After setting status to 'done':")
		for _, doc := range docs {
			status, _ := doc.Dimensions["status"].(string)
			t.Logf("  %s: %s (status: %s)", doc.UserFacingID, doc.Title, status)
			if doc.UUID == doc1 {
				if doc.UserFacingID == "d1" {
					foundDone = true
				} else {
					t.Logf("  ^ This document should have ID 'd1' but has '%s'", doc.UserFacingID)
				}
			}
		}
		if !foundDone {
			t.Error("expected document with 'done' status to have ID 'd1'")
		}

		// Test hierarchical IDs
		child1, err := store.Add("Subtask", map[string]interface{}{"parent_uuid": doc2})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// List again
		docs, err = store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		// Find the child document
		foundChild := false
		// First, let's see what all the IDs are
		t.Logf("Document IDs after adding child:")
		for _, doc := range docs {
			parentUUID, _ := doc.Dimensions["parent_uuid"].(string)
			t.Logf("  %s: %s (parent: %v)", doc.UserFacingID, doc.Title, parentUUID)
		}

		// Find parent's ID first
		var parentID string
		for _, doc := range docs {
			if doc.UUID == doc2 {
				parentID = doc.UserFacingID
				break
			}
		}

		for _, doc := range docs {
			if doc.UUID == child1 {
				// Should have hierarchical ID based on parent
				expectedID := parentID + ".1"
				if doc.UserFacingID != expectedID {
					t.Errorf("expected child ID to be '%s', got '%s'", expectedID, doc.UserFacingID)
				}
				foundChild = true
				break
			}
		}
		if !foundChild {
			t.Error("child document not found")
		}
	})

	t.Run("priority prefixes", func(t *testing.T) {
		config := nanostore.Config{
			Dimensions: []nanostore.DimensionConfig{
				{
					Name:         "priority",
					Type:         nanostore.Enumerated,
					Values:       []string{"low", "normal", "high"},
					Prefixes:     map[string]string{"high": "h", "low": "l"},
					DefaultValue: "normal",
				},
			},
		}

		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Add documents - will use AddWithDimensions when available
		// For now, just test basic functionality
		_, err = store.Add("Normal priority task", nil)
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}

		// Normal priority (default) should have no prefix
		if docs[0].UserFacingID != "1" {
			t.Errorf("expected ID '1' for normal priority, got '%s'", docs[0].UserFacingID)
		}
	})

}

func TestConfigurableIDResolution(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "normal", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "normal",
			},
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
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents to test ID resolution
	root1, err := store.Add("Root 1", nil)
	if err != nil {
		t.Fatalf("failed to add root: %v", err)
	}

	// Set as completed high priority
	if err := store.Update(root1, nanostore.UpdateRequest{
		Dimensions: map[string]interface{}{"status": "completed"},
	}); err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	// Test resolving the ID
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	userFacingID := docs[0].UserFacingID

	// Resolve back to UUID
	resolvedUUID, err := store.ResolveUUID(userFacingID)
	if err != nil {
		t.Fatalf("failed to resolve ID '%s': %v", userFacingID, err)
	}

	if resolvedUUID != root1 {
		t.Errorf("resolved UUID %s doesn't match original %s", resolvedUUID, root1)
	}

	// Test resolving different permutations if they exist
	// This depends on how the ID was generated (alphabetical ordering)
	// For example, if the ID is "hc1", we could also try "ch1"
	// Both should resolve to the same document
}
