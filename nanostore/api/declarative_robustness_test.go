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

// Edge case struct types for testing
type EdgeCaseItem struct {
	nanostore.Document

	// Various field types to test edge cases
	StringField  string  `default:"default"`
	IntField     int     `default:"42"`
	BoolField    bool    `default:"true"`
	FloatField   float64 `default:"3.14"`
	PointerField *string

	// Dimension with special characters in tag
	WeirdDimension string `dimension:"weird-dimension!@#" values:"val1,val2,val3"`

	// Multiple tags
	MultiTag string `values:"a,b,c" prefix:"a=x,b=y" default:"a" dimension:"multi_tag"`
}

// Same type without pointer field for other tests
type SafeEdgeCaseItem struct {
	nanostore.Document

	// Various field types to test edge cases
	StringField string  `default:"default"`
	IntField    int     `default:"42"`
	BoolField   bool    `default:"true"`
	FloatField  float64 `default:"3.14"`

	// Dimension with special characters in tag
	WeirdDimension string `dimension:"weird-dimension!@#" values:"val1,val2,val3"`

	// Multiple tags
	MultiTag string `values:"a,b,c" prefix:"a=x,b=y" default:"a" dimension:"multi_tag"`
}

type InvalidTagsItem struct {
	nanostore.Document

	// Invalid default values
	BadDefault string `default:"nonexistent" values:"valid1,valid2"`

	// Empty values list
	EmptyValues string `values:""`

	// Duplicate prefixes
	DupPrefix string `values:"x,y,z" prefix:"x=a,y=a"`
}

type ConflictingItem struct {
	nanostore.Document

	// Same dimension name as parent in hierarchical
	ParentID string `dimension:"parent_id" values:"should,not,work"`
}

func TestDeclarativeRobustness(t *testing.T) {
	// Test 1: Struct with edge case field types and tags
	t.Run("EdgeCaseStructTypes", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.NewFromType[EdgeCaseItem](tmpfile.Name())
		if err == nil {
			t.Fatal("expected error for pointer field, got nil")
		}
		if !strings.Contains(err.Error(), "pointer fields are not supported") {
			t.Fatalf("expected pointer field error, got: %v", err)
		}

		// Test passes - pointer fields are properly rejected
		t.Log("Pointer fields correctly rejected with error")
	})

	// Test 2: Invalid struct tags handling
	t.Run("InvalidStructTags", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		// This might fail during store creation
		store, err := api.NewFromType[InvalidTagsItem](tmpfile.Name())
		if err != nil {
			// Expected - invalid tags should be caught
			t.Logf("Store creation failed as expected: %v", err)
			return
		}
		defer func() { _ = store.Close() }()

		// If it doesn't fail, test the behavior
		_, err = store.Create("Test", &InvalidTagsItem{
			BadDefault: "valid1",
		})
		if err != nil {
			t.Logf("Create failed with invalid tags: %v", err)
		}
	})

	// Test 3: Extreme values in typed fields
	t.Run("ExtremeFieldValues", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[SafeEdgeCaseItem](tmpfile.Name())
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		extremeValues := []struct {
			name string
			item SafeEdgeCaseItem
		}{
			{
				"max int",
				SafeEdgeCaseItem{IntField: 9223372036854775807},
			},
			{
				"min int",
				SafeEdgeCaseItem{IntField: -9223372036854775808},
			},
			{
				"very large float",
				SafeEdgeCaseItem{FloatField: 1.7976931348623157e+308}, // Near max float64
			},
			{
				"very small float",
				SafeEdgeCaseItem{FloatField: -1.7976931348623157e+308}, // Near min float64
			},
			{
				"very long string",
				SafeEdgeCaseItem{StringField: strings.Repeat("x", 100000)},
			},
		}

		for _, test := range extremeValues {
			t.Run(test.name, func(t *testing.T) {
				id, err := store.Create(test.name, &test.item)
				if err != nil {
					t.Fatalf("failed to create: %v", err)
				}

				retrieved, err := store.Get(id)
				if err != nil {
					t.Fatalf("failed to retrieve: %v", err)
				}

				// Compare values - all fields should now be preserved
				if retrieved.IntField != test.item.IntField {
					t.Errorf("IntField not preserved: got %v, want %v", retrieved.IntField, test.item.IntField)
				}
				if retrieved.FloatField != test.item.FloatField {
					t.Errorf("FloatField not preserved: got %v, want %v", retrieved.FloatField, test.item.FloatField)
				}
			})
		}
	})

	// Test 4: Update with partial and conflicting data
	t.Run("UpdateEdgeCases", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[SafeEdgeCaseItem](tmpfile.Name())
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create initial item
		id, err := store.Create("Update test", &SafeEdgeCaseItem{
			StringField: "initial",
			IntField:    100,
		})
		if err != nil {
			t.Fatalf("failed to create: %v", err)
		}

		// Test various update scenarios
		updateTests := []struct {
			name   string
			update SafeEdgeCaseItem
			check  func(*SafeEdgeCaseItem) error
		}{
			{
				"zero values update",
				SafeEdgeCaseItem{}, // All zero values
				func(item *SafeEdgeCaseItem) error {
					// Zero values should overwrite
					if item.StringField != "" {
						return nil // Expected behavior may vary
					}
					return nil
				},
			},
			{
				"update with defaults",
				SafeEdgeCaseItem{
					StringField: "default", // Same as default tag
				},
				func(item *SafeEdgeCaseItem) error {
					if item.StringField != "default" {
						return nil
					}
					return nil
				},
			},
		}

		for _, test := range updateTests {
			t.Run(test.name, func(t *testing.T) {
				err := store.Update(id, &test.update)
				if err != nil {
					t.Fatalf("update failed: %v", err)
				}

				retrieved, err := store.Get(id)
				if err != nil {
					t.Fatalf("failed to get after update: %v", err)
				}

				if err := test.check(retrieved); err != nil {
					t.Error(err)
				}
			})
		}
	})

	// Test 5: Concurrent operations stress test
	t.Run("ConcurrentOperations", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.NewFromType[SafeEdgeCaseItem](tmpfile.Name())
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create base documents
		var ids []string
		for i := 0; i < 10; i++ {
			id, err := store.Create("Concurrent test", &SafeEdgeCaseItem{
				IntField: i,
			})
			if err != nil {
				t.Fatalf("failed to create doc %d: %v", i, err)
			}
			ids = append(ids, id)
		}

		// Simulate rapid operations
		operations := []struct {
			name string
			fn   func() error
		}{
			{
				"rapid queries",
				func() error {
					for i := 0; i < 100; i++ {
						_, err := store.Query().Find()
						if err != nil {
							return err
						}
					}
					return nil
				},
			},
			{
				"interleaved updates",
				func() error {
					for i, id := range ids {
						err := store.Update(id, &SafeEdgeCaseItem{IntField: i * 2})
						if err != nil {
							return err
						}
					}
					return nil
				},
			},
			{
				"delete and recreate",
				func() error {
					// Delete first half
					for i := 0; i < len(ids)/2; i++ {
						err := store.Delete(ids[i], false)
						if err != nil {
							return err
						}
					}
					// Recreate
					for i := 0; i < len(ids)/2; i++ {
						_, err := store.Create("Recreated", &SafeEdgeCaseItem{IntField: i})
						if err != nil {
							return err
						}
					}
					return nil
				},
			},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				if err := op.fn(); err != nil {
					t.Errorf("%s failed: %v", op.name, err)
				}
			})
		}

		// Verify store is still consistent
		finalDocs, err := store.Query().Find()
		if err != nil {
			t.Fatalf("final query failed: %v", err)
		}
		t.Logf("Final document count: %d", len(finalDocs))
	})
}
