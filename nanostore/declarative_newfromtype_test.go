package nanostore

import (
	"strings"
	"testing"
)

func TestNewFromType(t *testing.T) {
	t.Run("valid struct with enum dimensions", func(t *testing.T) {
		type TestDoc struct {
			Document
			Status   string `values:"pending,active,done" default:"pending"`
			Priority string `values:"low,medium,high"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Check that store was created
		if store.store == nil {
			t.Fatal("expected store to be created")
		}

		// Check config was built correctly
		if len(store.config.Dimensions) != 2 {
			t.Errorf("expected 2 dimensions, got %d", len(store.config.Dimensions))
		}

		// Check field indices
		if len(store.fieldIndices) != 2 {
			t.Errorf("expected 2 field indices, got %d", len(store.fieldIndices))
		}
		if _, exists := store.fieldIndices["status"]; !exists {
			t.Error("expected 'status' in field indices")
		}
		if _, exists := store.fieldIndices["priority"]; !exists {
			t.Error("expected 'priority' in field indices")
		}
	})

	t.Run("struct with hierarchical dimension", func(t *testing.T) {
		type TestDoc struct {
			Document
			Status   string `values:"pending,active,done"`
			ParentID string `dimension:"parent_id,ref"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Find parent dimension
		var hasHierarchical bool
		for _, dim := range store.config.Dimensions {
			if dim.Type == Hierarchical {
				hasHierarchical = true
				if dim.Name != "parent_id" {
					t.Errorf("expected hierarchical dimension name 'parent_id', got %q", dim.Name)
				}
				break
			}
		}
		if !hasHierarchical {
			t.Error("expected hierarchical dimension")
		}
	})

	t.Run("struct with prefixes", func(t *testing.T) {
		type TestDoc struct {
			Document
			Priority string `values:"low,medium,high" prefix:"high=h,medium=m"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Check prefixes
		priorityDim := store.config.Dimensions[0]
		if priorityDim.Prefixes["high"] != "h" {
			t.Errorf("expected prefix 'h' for 'high', got %q", priorityDim.Prefixes["high"])
		}
	})

	t.Run("struct with excluded field", func(t *testing.T) {
		type TestDoc struct {
			Document
			Status   string `values:"pending,active,done"`
			Internal string `dimension:"-"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Should only have one dimension
		if len(store.config.Dimensions) != 1 {
			t.Errorf("expected 1 dimension (excluded one), got %d", len(store.config.Dimensions))
		}

		// Internal should not be in field indices
		if _, exists := store.fieldIndices["internal"]; exists {
			t.Error("expected 'internal' to be excluded from field indices")
		}
	})

	t.Run("struct without Document embedding", func(t *testing.T) {
		type TestDoc struct {
			Status string `values:"pending,active,done"`
		}

		_, err := NewFromType[TestDoc](":memory:")
		if err == nil {
			t.Fatal("expected error for struct without Document embedding")
		}
		if !strings.Contains(err.Error(), "must embed nanostore.Document") {
			t.Errorf("expected error about Document embedding, got: %v", err)
		}
	})

	t.Run("struct with invalid field type", func(t *testing.T) {
		type TestDoc struct {
			Document
			Count int
		}

		_, err := NewFromType[TestDoc](":memory:")
		if err == nil {
			t.Fatal("expected error for non-string field")
		}
		if !strings.Contains(err.Error(), "only string dimensions") {
			t.Errorf("expected error about string dimensions, got: %v", err)
		}
	})

	t.Run("custom string type without values", func(t *testing.T) {
		type Priority string
		type TestDoc struct {
			Document
			Priority Priority
		}

		_, err := NewFromType[TestDoc](":memory:")
		if err == nil {
			t.Fatal("expected error for custom string type without values")
		}
		if !strings.Contains(err.Error(), "requires 'values' tag") {
			t.Errorf("expected error about missing values tag, got: %v", err)
		}
	})

	t.Run("empty struct with only Document", func(t *testing.T) {
		type TestDoc struct {
			Document
		}

		_, err := NewFromType[TestDoc](":memory:")
		if err == nil {
			t.Fatal("expected error for struct with no dimensions")
		}
		if !strings.Contains(err.Error(), "at least one dimension") {
			t.Errorf("expected error about dimension requirement, got: %v", err)
		}
	})

	t.Run("implicit dimension naming", func(t *testing.T) {
		type TestDoc struct {
			Document
			ParentID   string `dimension:",ref"`
			StatusCode string `values:"ok,error"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Check dimension names
		var foundParent, foundStatus bool
		for _, dim := range store.config.Dimensions {
			if dim.Name == "parent_id" {
				foundParent = true
			}
			if dim.Name == "status_code" {
				foundStatus = true
			}
		}
		if !foundParent {
			t.Error("expected dimension 'parent_id' from implicit naming")
		}
		if !foundStatus {
			t.Error("expected dimension 'status_code' from implicit naming")
		}
	})

	t.Run("complex naming conversions", func(t *testing.T) {
		type TestDoc struct {
			Document
			HTTPCode  string `values:"200,400,500"`
			XMLParser string `values:"sax,dom"`
			APIKey    string `values:"valid,invalid"`
		}

		store, err := NewFromType[TestDoc](":memory:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Check dimension names
		expectedNames := map[string]bool{
			"http_code":  false,
			"xml_parser": false,
			"api_key":    false,
		}

		for _, dim := range store.config.Dimensions {
			if _, exists := expectedNames[dim.Name]; exists {
				expectedNames[dim.Name] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("expected dimension name %q not found", name)
			}
		}
	})
}
