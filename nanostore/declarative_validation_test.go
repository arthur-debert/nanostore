package nanostore

import (
	"testing"
)

func TestDeclarativeValidation(t *testing.T) {
	t.Run("Invalid default value", func(t *testing.T) {
		type BadDefault struct {
			Document
			Status string `values:"pending,done" default:"invalid"`
		}

		_, err := NewFromType[BadDefault](":memory:")
		if err == nil {
			t.Fatal("expected error for invalid default value")
		}
		if !containsSubstring(err.Error(), "default value") || !containsSubstring(err.Error(), "not in the list of valid values") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Prefix for non-existent value", func(t *testing.T) {
		type BadPrefix struct {
			Document
			Status string `values:"pending,done" prefix:"invalid=i"`
		}

		_, err := NewFromType[BadPrefix](":memory:")
		if err == nil {
			t.Fatal("expected error for prefix on invalid value")
		}
		if !containsSubstring(err.Error(), "prefix defined for invalid value") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Duplicate prefixes", func(t *testing.T) {
		type DuplicatePrefix struct {
			Document
			Status string `values:"pending,done,completed" prefix:"done=d,completed=d"`
		}

		_, err := NewFromType[DuplicatePrefix](":memory:")
		if err == nil {
			t.Fatal("expected error for duplicate prefixes")
		}
		if !containsSubstring(err.Error(), "duplicate prefix") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Empty prefix", func(t *testing.T) {
		type EmptyPrefix struct {
			Document
			Status string `values:"pending,done" prefix:"done="`
		}

		_, err := NewFromType[EmptyPrefix](":memory:")
		if err == nil {
			t.Fatal("expected error for empty prefix")
		}
		if !containsSubstring(err.Error(), "empty prefix") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Hierarchical with default value", func(t *testing.T) {
		type BadHierarchical struct {
			Document
			ParentID string `dimension:"parent_id,ref" default:"something"`
		}

		_, err := NewFromType[BadHierarchical](":memory:")
		if err == nil {
			t.Fatal("expected error for hierarchical dimension with default")
		}
		if !containsSubstring(err.Error(), "hierarchical dimension") || !contains(err.Error(), "cannot have a default value") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Hierarchical with values", func(t *testing.T) {
		type BadHierarchical struct {
			Document
			ParentID string `dimension:"parent_id,ref" values:"a,b,c"`
		}

		_, err := NewFromType[BadHierarchical](":memory:")
		if err == nil {
			t.Fatal("expected error for hierarchical dimension with values")
		}
		if !containsSubstring(err.Error(), "hierarchical dimension") || !contains(err.Error(), "cannot have enumerated values") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Hierarchical with prefixes", func(t *testing.T) {
		type BadHierarchical struct {
			Document
			ParentID string `dimension:"parent_id,ref" prefix:"a=p"`
		}

		_, err := NewFromType[BadHierarchical](":memory:")
		if err == nil {
			t.Fatal("expected error for hierarchical dimension with prefixes")
		}
		if !containsSubstring(err.Error(), "hierarchical dimension") || !containsSubstring(err.Error(), "cannot have prefixes") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Multiple hierarchical dimensions", func(t *testing.T) {
		type MultipleHierarchical struct {
			Document
			ParentID   string `dimension:"parent_id,ref"`
			CategoryID string `dimension:"category_id,ref"`
		}

		_, err := NewFromType[MultipleHierarchical](":memory:")
		if err == nil {
			t.Fatal("expected error for multiple hierarchical dimensions")
		}
		if !containsSubstring(err.Error(), "only one hierarchical dimension is supported") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("No dimensions", func(t *testing.T) {
		type NoDimensions struct {
			Document
			Title       string `dimension:"-"`
			Description string `dimension:"-"`
		}

		_, err := NewFromType[NoDimensions](":memory:")
		if err == nil {
			t.Fatal("expected error for no dimensions")
		}
		if !containsSubstring(err.Error(), "at least one dimension must be defined") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Valid configurations", func(t *testing.T) {
		// Should not error
		type ValidConfig struct {
			Document
			Status   string `values:"pending,done" default:"pending" prefix:"done=d"`
			Priority string `values:"low,medium,high"`
			ParentID string `dimension:"parent_id,ref"`
		}

		store, err := NewFromType[ValidConfig](":memory:")
		if err != nil {
			t.Fatalf("expected valid config to work: %v", err)
		}
		_ = store.Close()
	})
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsSubstring(s[1:], substr)
}
