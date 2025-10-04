package api_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// ComplexTodoItem with multiple dimensions to make config generation more expensive
type ComplexTodoItem struct {
	nanostore.Document
	Status    string `values:"draft,review,approved,rejected,archived" default:"draft"`
	Priority  string `values:"urgent,high,normal,low,defer" default:"normal"`
	Activity  string `values:"new,working,blocked,testing,complete" default:"new"`
	Category  string `values:"bug,feature,task,research,documentation" default:"task"`
	Severity  string `values:"critical,major,minor,trivial" default:"minor"`
	Component string `values:"frontend,backend,database,infrastructure,docs" default:"backend"`
	Team      string `values:"engineering,design,product,qa,devops" default:"engineering"`
}

func TestGetDimensionConfigPerformance(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store
	store, err := api.New[ComplexTodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Measure time for first call
	start := time.Now()
	config1, err := store.GetDimensionConfig()
	if err != nil {
		t.Fatalf("First GetDimensionConfig failed: %v", err)
	}
	firstCallDuration := time.Since(start)

	// Measure time for subsequent calls
	const numCalls = 100
	start = time.Now()

	for i := 0; i < numCalls; i++ {
		config, err := store.GetDimensionConfig()
		if err != nil {
			t.Fatalf("GetDimensionConfig call %d failed: %v", i, err)
		}

		// Verify configs are equivalent (they should be identical)
		if len(config.Dimensions) != len(config1.Dimensions) {
			t.Errorf("Config dimensions count changed between calls: first=%d, call %d=%d",
				len(config1.Dimensions), i, len(config.Dimensions))
		}
	}

	totalDuration := time.Since(start)
	avgDuration := totalDuration / numCalls

	t.Logf("Performance Analysis:")
	t.Logf("  First call: %v", firstCallDuration)
	t.Logf("  %d subsequent calls: %v total, %v average", numCalls, totalDuration, avgDuration)
	t.Logf("  Dimensions found: %d", len(config1.Dimensions))

	// If caching was working properly, subsequent calls should be much faster
	// Currently they're not because config is regenerated each time
	if avgDuration > firstCallDuration/2 {
		t.Logf("PERFORMANCE ISSUE: Subsequent calls are not significantly faster than first call")
		t.Logf("This suggests GetDimensionConfig is regenerating config each time instead of caching")
	} else {
		t.Logf("GOOD: Subsequent calls are faster, suggesting caching is working")
	}
}

func TestGetDimensionConfigConsistency(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store
	store, err := api.New[ComplexTodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get config multiple times and verify they're identical
	config1, err := store.GetDimensionConfig()
	if err != nil {
		t.Fatalf("First GetDimensionConfig failed: %v", err)
	}

	config2, err := store.GetDimensionConfig()
	if err != nil {
		t.Fatalf("Second GetDimensionConfig failed: %v", err)
	}

	// Verify basic properties are identical
	if len(config1.Dimensions) != len(config2.Dimensions) {
		t.Errorf("Dimension count differs: first=%d, second=%d",
			len(config1.Dimensions), len(config2.Dimensions))
	}

	// Verify each dimension is identical
	for i, dim1 := range config1.Dimensions {
		if i >= len(config2.Dimensions) {
			t.Errorf("Second config has fewer dimensions than first")
			break
		}

		dim2 := config2.Dimensions[i]

		if dim1.Name != dim2.Name {
			t.Errorf("Dimension %d name differs: %s vs %s", i, dim1.Name, dim2.Name)
		}

		if dim1.Type != dim2.Type {
			t.Errorf("Dimension %s type differs: %v vs %v", dim1.Name, dim1.Type, dim2.Type)
		}

		if dim1.DefaultValue != dim2.DefaultValue {
			t.Errorf("Dimension %s default differs: %s vs %s", dim1.Name, dim1.DefaultValue, dim2.DefaultValue)
		}

		if len(dim1.Values) != len(dim2.Values) {
			t.Errorf("Dimension %s values count differs: %d vs %d",
				dim1.Name, len(dim1.Values), len(dim2.Values))
		}
	}

	t.Logf("Configuration consistency verified across multiple calls")
}
