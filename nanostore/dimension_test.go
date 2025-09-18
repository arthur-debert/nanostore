package nanostore_test

import (
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestDimensionHelpers(t *testing.T) {
	// Create a test configuration with various dimension types
	config := nanostore.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:         "status",
				Type:         nanostore.Enumerated,
				Values:       []string{"pending", "active", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         nanostore.Enumerated,
				Values:       []string{"low", "medium", "high"},
				Prefixes:     map[string]string{"high": "h"},
				DefaultValue: "medium",
			},
			{
				Name:     "parent_uuid",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
			{
				Name:         "category",
				Type:         nanostore.Enumerated,
				Values:       []string{"work", "personal", "other"},
				DefaultValue: "other",
			},
		},
	}

	t.Run("GetEnumeratedDimensions", func(t *testing.T) {
		enumerated := config.GetEnumeratedDimensions()

		// Should return 3 enumerated dimensions
		if len(enumerated) != 3 {
			t.Errorf("expected 3 enumerated dimensions, got %d", len(enumerated))
		}

		// Verify they are all enumerated type
		for _, dim := range enumerated {
			if dim.Type != nanostore.Enumerated {
				t.Errorf("expected enumerated type for %s, got %v", dim.Name, dim.Type)
			}
		}

		// Check specific dimensions are included
		foundDims := make(map[string]bool)
		for _, dim := range enumerated {
			foundDims[dim.Name] = true
		}

		expectedDims := []string{"status", "priority", "category"}
		for _, expected := range expectedDims {
			if !foundDims[expected] {
				t.Errorf("expected to find dimension %s in enumerated dimensions", expected)
			}
		}

		// Verify parent_uuid is NOT included
		if foundDims["parent_uuid"] {
			t.Error("parent_uuid should not be in enumerated dimensions")
		}
	})

	t.Run("GetHierarchicalDimensions", func(t *testing.T) {
		hierarchical := config.GetHierarchicalDimensions()

		// Should return 1 hierarchical dimension
		if len(hierarchical) != 1 {
			t.Errorf("expected 1 hierarchical dimension, got %d", len(hierarchical))
		}

		// Verify it's the correct one
		if len(hierarchical) > 0 {
			if hierarchical[0].Name != "parent_uuid" {
				t.Errorf("expected parent_uuid, got %s", hierarchical[0].Name)
			}
			if hierarchical[0].Type != nanostore.Hierarchical {
				t.Errorf("expected hierarchical type, got %v", hierarchical[0].Type)
			}
			if hierarchical[0].RefField != "parent_uuid" {
				t.Errorf("expected RefField parent_uuid, got %s", hierarchical[0].RefField)
			}
		}
	})

	t.Run("GetDimension", func(t *testing.T) {
		// Test getting existing dimensions
		statusDim, exists := config.GetDimension("status")
		if !exists {
			t.Error("expected to find status dimension")
		}
		if statusDim == nil {
			t.Fatal("status dimension is nil")
		}
		if statusDim.Name != "status" {
			t.Errorf("expected dimension name 'status', got %s", statusDim.Name)
		}
		if statusDim.DefaultValue != "pending" {
			t.Errorf("expected default value 'pending', got %s", statusDim.DefaultValue)
		}

		// Test getting hierarchical dimension
		parentDim, exists := config.GetDimension("parent_uuid")
		if !exists {
			t.Error("expected to find parent_uuid dimension")
		}
		if parentDim == nil {
			t.Fatal("parent_uuid dimension is nil")
		}
		if parentDim.Type != nanostore.Hierarchical {
			t.Errorf("expected hierarchical type for parent_uuid, got %v", parentDim.Type)
		}

		// Test getting non-existent dimension
		nonExistent, exists := config.GetDimension("non_existent")
		if exists {
			t.Error("expected not to find non_existent dimension")
		}
		if nonExistent != nil {
			t.Error("expected nil for non-existent dimension")
		}
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		emptyConfig := nanostore.Config{}

		// Should return empty slices
		enumerated := emptyConfig.GetEnumeratedDimensions()
		if len(enumerated) != 0 {
			t.Errorf("expected 0 enumerated dimensions for empty config, got %d", len(enumerated))
		}

		hierarchical := emptyConfig.GetHierarchicalDimensions()
		if len(hierarchical) != 0 {
			t.Errorf("expected 0 hierarchical dimensions for empty config, got %d", len(hierarchical))
		}

		// GetDimension should return false for any query
		_, exists := emptyConfig.GetDimension("anything")
		if exists {
			t.Error("expected not to find dimension in empty config")
		}
	})

	t.Run("DimensionPrefixes", func(t *testing.T) {
		statusDim, _ := config.GetDimension("status")

		// Check prefix mapping
		if prefix, ok := statusDim.Prefixes["done"]; !ok || prefix != "d" {
			t.Errorf("expected prefix 'd' for 'done' status, got %s", prefix)
		}

		priorityDim, _ := config.GetDimension("priority")
		if prefix, ok := priorityDim.Prefixes["high"]; !ok || prefix != "h" {
			t.Errorf("expected prefix 'h' for 'high' priority, got %s", prefix)
		}

		// Category has no prefixes defined
		categoryDim, _ := config.GetDimension("category")
		if len(categoryDim.Prefixes) != 0 {
			t.Errorf("expected no prefixes for category, got %d", len(categoryDim.Prefixes))
		}
	})

	t.Run("types.DimensionValues", func(t *testing.T) {
		statusDim, _ := config.GetDimension("status")

		expectedValues := []string{"pending", "active", "done"}
		if len(statusDim.Values) != len(expectedValues) {
			t.Errorf("expected %d values for status, got %d", len(expectedValues), len(statusDim.Values))
		}

		// Check all expected values are present
		valueMap := make(map[string]bool)
		for _, v := range statusDim.Values {
			valueMap[v] = true
		}
		for _, expected := range expectedValues {
			if !valueMap[expected] {
				t.Errorf("expected value %s not found in status dimension", expected)
			}
		}
	})
}

func TestNewListOptions(t *testing.T) {
	opts := nanostore.NewListOptions()

	// Should have initialized Filters map
	if opts.Filters == nil {
		t.Error("expected Filters map to be initialized, got nil")
	}

	// Filters map should be empty
	if len(opts.Filters) != 0 {
		t.Errorf("expected empty Filters map, got %d entries", len(opts.Filters))
	}

	// Other fields should have zero values
	if opts.FilterBySearch != "" {
		t.Errorf("expected empty FilterBySearch, got %s", opts.FilterBySearch)
	}

	if len(opts.OrderBy) != 0 {
		t.Errorf("expected empty OrderBy, got %d clauses", len(opts.OrderBy))
	}

	if opts.Limit != nil {
		t.Errorf("expected nil Limit, got %d", *opts.Limit)
	}

	if opts.Offset != nil {
		t.Errorf("expected nil Offset, got %d", *opts.Offset)
	}

	// Should be able to add filters without panic
	opts.Filters["status"] = "active"
	if opts.Filters["status"] != "active" {
		t.Error("failed to add filter to initialized map")
	}
}
