package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestAddWithDimensions(t *testing.T) {
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "normal", "high"},
				Prefixes:     map[string]string{"high": "h", "low": "l"},
				DefaultValue: "normal",
			},
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"todo", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "todo",
			},
		},
	}

	store, err := nanostore.New(":memory:", config)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test adding with high priority
	doc1, err := store.Add("Important Task", nil, map[string]string{
		"priority": "high",
	})
	if err != nil {
		t.Fatalf("failed to add document with high priority: %v", err)
	}

	// Test adding with low priority
	doc2, err := store.Add("Minor Task", nil, map[string]string{
		"priority": "low",
	})
	if err != nil {
		t.Fatalf("failed to add document with low priority: %v", err)
	}

	// Test adding with default (no dimensions specified)
	doc3, err := store.Add("Regular Task", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document with defaults: %v", err)
	}

	// Test invalid dimension value
	_, err = store.Add("Bad Task", nil, map[string]string{
		"priority": "urgent", // not in allowed values
	})
	if err == nil {
		t.Error("expected error for invalid dimension value")
	}

	// List and verify IDs
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(docs))
	}

	// Check IDs match priorities
	for _, doc := range docs {
		switch doc.UUID {
		case doc1:
			if doc.UserFacingID != "h1" {
				t.Errorf("high priority doc should have ID 'h1', got %s", doc.UserFacingID)
			}
		case doc2:
			if doc.UserFacingID != "l1" {
				t.Errorf("low priority doc should have ID 'l1', got %s", doc.UserFacingID)
			}
		case doc3:
			if doc.UserFacingID != "1" {
				t.Errorf("normal priority doc should have ID '1', got %s", doc.UserFacingID)
			}
		}
	}
}
