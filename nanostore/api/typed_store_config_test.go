package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// TestTypedStoreConfiguration holds all configuration-related tests
func TestTypedStoreConfiguration(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("GetDimensionConfig", func(t *testing.T) {
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatalf("failed to get dimension config: %v", err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		// Verify we have the expected dimensions from TodoItem
		foundDimensions := make(map[string]bool)
		for _, dim := range config.Dimensions {
			foundDimensions[dim.Name] = true

			t.Logf("Found dimension: %s (type: %v)", dim.Name, dim.Type)

			if dim.Type == nanostore.Enumerated {
				t.Logf("  Values: %v", dim.Values)
				t.Logf("  Prefixes: %v", dim.Prefixes)
				t.Logf("  Default: %s", dim.DefaultValue)
			} else if dim.Type == nanostore.Hierarchical {
				t.Logf("  RefField: %s", dim.RefField)
			}
		}

		// Verify expected dimensions exist (based on TodoItem struct)
		expectedDimensions := []string{"status", "priority", "activity"}
		for _, expected := range expectedDimensions {
			if !foundDimensions[expected] {
				t.Errorf("expected dimension '%s' not found in config", expected)
			}
		}
	})

	t.Run("ValidateConfiguration", func(t *testing.T) {
		err := store.ValidateConfiguration()
		if err != nil {
			t.Errorf("configuration validation failed: %v", err)
		}
	})

	t.Run("ConfigurationConsistency", func(t *testing.T) {
		// Get config multiple times and ensure consistency
		config1, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatalf("failed to get config first time: %v", err)
		}

		config2, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatalf("failed to get config second time: %v", err)
		}

		// Compare number of dimensions
		if len(config1.Dimensions) != len(config2.Dimensions) {
			t.Errorf("inconsistent dimension count: %d vs %d",
				len(config1.Dimensions), len(config2.Dimensions))
		}

		// Compare dimension details
		for i, dim1 := range config1.Dimensions {
			if i >= len(config2.Dimensions) {
				break
			}
			dim2 := config2.Dimensions[i]

			if dim1.Name != dim2.Name {
				t.Errorf("dimension name mismatch: %s vs %s", dim1.Name, dim2.Name)
			}
			if dim1.Type != dim2.Type {
				t.Errorf("dimension type mismatch for %s: %v vs %v", dim1.Name, dim1.Type, dim2.Type)
			}
		}
	})
}

// Define test types with various configuration scenarios
type ValidConfigItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" prefix:"done=d" default:"pending"`
	Priority string `values:"low,medium,high" prefix:"high=h" default:"medium"`
	Team     string `values:"alpha,beta,gamma" default:"alpha"`
}

type ComplexPrefixItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done,archived" prefix:"done=d,archived=a,active=act" default:"pending"`
	Priority string `values:"low,medium,high,critical" prefix:"high=h,critical=c" default:"medium"`
	Activity string `values:"working,blocked,reviewing" prefix:"blocked=b,reviewing=r"`
}

type InvalidDefaultItem struct {
	nanostore.Document
	Status string `values:"pending,active,done" default:"invalid"`
}

type PrefixConflictItem struct {
	nanostore.Document
	Status   string `values:"pending,active,done" prefix:"done=d"`
	Priority string `values:"low,medium,high" prefix:"high=d"` // Conflict: both use "d"
}

type EmptyValuesItem struct {
	nanostore.Document
	Status string `values:"" default:""`
}

type MissingRefFieldItem struct {
	nanostore.Document
	ParentID string `dimension:"parent_id"` // Missing ",ref"
}

func TestConfigurationValidation(t *testing.T) {
	t.Run("ValidConfiguration", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[ValidConfigItem](tmpfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		err = store.ValidateConfiguration()
		if err != nil {
			t.Errorf("expected valid configuration, got error: %v", err)
		}
	})

	t.Run("ComplexPrefixConfiguration", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[ComplexPrefixItem](tmpfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()

		// Should be valid - no conflicts
		err = store.ValidateConfiguration()
		if err != nil {
			t.Errorf("expected valid complex prefix configuration, got error: %v", err)
		}

		// Verify config details
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatal(err)
		}

		// Check that complex prefixes were parsed correctly
		for _, dim := range config.Dimensions {
			if dim.Name == "status" {
				expectedPrefixes := map[string]string{
					"done":     "d",
					"archived": "a",
					"active":   "act",
				}
				for value, expectedPrefix := range expectedPrefixes {
					if actualPrefix, exists := dim.Prefixes[value]; !exists {
						t.Errorf("missing prefix for value '%s'", value)
					} else if actualPrefix != expectedPrefix {
						t.Errorf("wrong prefix for value '%s': expected '%s', got '%s'",
							value, expectedPrefix, actualPrefix)
					}
				}
			}
		}
	})

	t.Run("InvalidDefaultValue", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		// This should fail during store creation due to invalid default
		_, err = api.NewFromType[InvalidDefaultItem](tmpfile.Name())
		if err == nil {
			t.Error("expected error for invalid default value, got none")
		} else if !strings.Contains(err.Error(), "default value") {
			t.Errorf("expected error about default value, got: %v", err)
		}
	})

	t.Run("PrefixConflictDetection", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		// Create store - might succeed initially
		store, err := api.NewFromType[PrefixConflictItem](tmpfile.Name())
		if err != nil {
			// If creation fails, that's also acceptable for conflict detection
			if strings.Contains(err.Error(), "prefix") || strings.Contains(err.Error(), "conflict") {
				t.Logf("Prefix conflict detected during store creation: %v", err)
				return
			}
			t.Fatalf("unexpected error during store creation: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Validation should detect the prefix conflict
		err = store.ValidateConfiguration()
		if err == nil {
			t.Error("expected prefix conflict error, got none")
		} else if !strings.Contains(err.Error(), "prefix conflict") {
			t.Errorf("expected prefix conflict error, got: %v", err)
		} else {
			t.Logf("Correctly detected prefix conflict: %v", err)
		}
	})

	t.Run("EmptyValuesValidation", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		// Should fail due to empty values
		_, err = api.NewFromType[EmptyValuesItem](tmpfile.Name())
		if err == nil {
			t.Error("expected error for empty values, got none")
		}
	})
}

func TestConfigurationIntrospection(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.NewFromType[ComplexPrefixItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("DimensionEnumeration", func(t *testing.T) {
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatal(err)
		}

		dimensionNames := make([]string, len(config.Dimensions))
		for i, dim := range config.Dimensions {
			dimensionNames[i] = dim.Name
		}

		t.Logf("All dimensions: %v", dimensionNames)

		// Should have our configured dimensions
		expectedCount := 3 // status, priority, activity
		if len(config.Dimensions) != expectedCount {
			t.Errorf("expected %d dimensions, got %d", expectedCount, len(config.Dimensions))
		}
	})

	t.Run("PrefixMapping", func(t *testing.T) {
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatal(err)
		}

		totalPrefixes := 0
		for _, dim := range config.Dimensions {
			if dim.Type == nanostore.Enumerated {
				totalPrefixes += len(dim.Prefixes)
				t.Logf("Dimension '%s' has %d prefix mappings", dim.Name, len(dim.Prefixes))
			}
		}

		t.Logf("Total prefix mappings across all dimensions: %d", totalPrefixes)

		// ComplexPrefixItem should have several prefix mappings
		if totalPrefixes == 0 {
			t.Error("expected some prefix mappings, got none")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatal(err)
		}

		defaultCount := 0
		for _, dim := range config.Dimensions {
			if dim.DefaultValue != "" {
				defaultCount++
				t.Logf("Dimension '%s' has default: '%s'", dim.Name, dim.DefaultValue)
			}
		}

		t.Logf("Dimensions with defaults: %d", defaultCount)

		// ComplexPrefixItem should have defaults for status and priority
		if defaultCount < 2 {
			t.Errorf("expected at least 2 defaults, got %d", defaultCount)
		}
	})
}