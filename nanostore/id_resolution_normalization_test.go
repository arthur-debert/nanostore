package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestIDResolutionNormalization(t *testing.T) {
	t.Skip("Skipping test - ID resolution normalization not yet implemented")
	t.Run("two prefixes normalization", func(t *testing.T) {
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
					Values:       []string{"todo", "pending", "done"},
					Prefixes:     map[string]string{"pending": "p", "done": "d"},
					DefaultValue: "todo",
				},
			},
		}

		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document with high priority and pending status
		docID, err := store.Add("Important Task", nil, map[string]string{
			"priority": "high",
			"status":   "pending",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Verify the canonical ID is "hp1" (alphabetical by dimension name)
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}
		if docs[0].UserFacingID != "hp1" {
			t.Errorf("expected canonical ID 'hp1', got %s", docs[0].UserFacingID)
		}

		// Test that both "hp1" and "ph1" resolve to the same document
		uuid1, err := store.ResolveUUID("hp1")
		if err != nil {
			t.Fatalf("failed to resolve hp1: %v", err)
		}

		uuid2, err := store.ResolveUUID("ph1")
		if err != nil {
			t.Fatalf("failed to resolve ph1: %v", err)
		}

		if uuid1 != uuid2 {
			t.Errorf("hp1 and ph1 should resolve to the same document, got %s and %s", uuid1, uuid2)
		}

		if uuid1 != docID {
			t.Errorf("resolved UUID should match original document ID")
		}
	})

	t.Run("three prefixes normalization", func(t *testing.T) {
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
					Values:       []string{"todo", "pending", "done"},
					Prefixes:     map[string]string{"pending": "p"},
					DefaultValue: "todo",
				},
				{
					Name:         "category",
					Type:         nanostore.Enumerated,
					Values:       []string{"personal", "work", "urgent"},
					Prefixes:     map[string]string{"urgent": "u"},
					DefaultValue: "personal",
				},
			},
		}

		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document with all three prefix values
		docID, err := store.Add("Critical Task", nil, map[string]string{
			"priority": "high",
			"status":   "pending",
			"category": "urgent",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Test all 6 permutations of the three prefixes
		permutations := []string{
			"hpu1", "hup1", "phu1", "puh1", "uhp1", "uph1",
		}

		for _, id := range permutations {
			resolved, err := store.ResolveUUID(id)
			if err != nil {
				t.Errorf("failed to resolve %s: %v", id, err)
			}
			if resolved != docID {
				t.Errorf("ID %s should resolve to the same document", id)
			}
		}
	})

	t.Run("hierarchical ID normalization", func(t *testing.T) {
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
					Values:       []string{"todo", "done"},
					Prefixes:     map[string]string{"done": "d"},
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
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create parent with high priority and done status
		parentID, err := store.Add("Parent Task", nil, map[string]string{
			"priority": "high",
			"status":   "done",
		})
		if err != nil {
			t.Fatalf("failed to add parent: %v", err)
		}

		// Create child with high priority
		childID, err := store.Add("Child Task", &parentID, map[string]string{
			"priority": "high",
		})
		if err != nil {
			t.Fatalf("failed to add child: %v", err)
		}

		// Test that both "hd1.h1" and "dh1.h1" resolve to the same child
		uuid1, err := store.ResolveUUID("hd1.h1")
		if err != nil {
			t.Fatalf("failed to resolve hd1.h1: %v", err)
		}

		uuid2, err := store.ResolveUUID("dh1.h1")
		if err != nil {
			t.Fatalf("failed to resolve dh1.h1: %v", err)
		}

		if uuid1 != uuid2 {
			t.Errorf("hd1.h1 and dh1.h1 should resolve to the same child document")
		}

		if uuid1 != childID {
			t.Errorf("resolved UUID should match child document ID")
		}
	})

	t.Run("invalid prefix combinations", func(t *testing.T) {
		config := nanostore.Config{
			Dimensions: []nanostore.DimensionConfig{
				{
					Name:         "priority",
					Type:         nanostore.Enumerated,
					Values:       []string{"low", "normal", "high"},
					Prefixes:     map[string]string{"high": "h"},
					DefaultValue: "normal",
				},
			},
		}

		store, err := nanostore.New(":memory:", config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create a document
		_, err = store.Add("Task", nil, map[string]string{
			"priority": "high",
		})
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}

		// Test invalid prefix combinations
		invalidIDs := []string{
			"x1",   // Unknown prefix
			"hx1",  // Mix of valid and invalid
			"hhh1", // Too many of the same prefix
			"h1x",  // Invalid suffix
		}

		for _, id := range invalidIDs {
			_, err := store.ResolveUUID(id)
			if err == nil {
				t.Errorf("expected error for invalid ID %s, but got none", id)
			}
		}
	})
}
