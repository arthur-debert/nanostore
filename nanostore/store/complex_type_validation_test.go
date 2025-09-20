package store_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestComplexTypeValidation(t *testing.T) {
	// Create test store
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
				Values:       []string{"active", "inactive"},
				DefaultValue: "active",
			},
			{
				Name:     "parent_uuid",
				Type:     nanostore.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	store, err := nanostore.New(tmpfile.Name(), config)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test cases for Add method
	t.Run("Add with complex types", func(t *testing.T) {
		testCases := []struct {
			name       string
			dimensions map[string]interface{}
			expectErr  bool
			errMsg     string
		}{
			{
				name: "map type",
				dimensions: map[string]interface{}{
					"status": "active",
					"data":   map[string]interface{}{"nested": "value"},
				},
				expectErr: true,
				errMsg:    "cannot be a map type",
			},
			{
				name: "slice type",
				dimensions: map[string]interface{}{
					"status": "active",
					"tags":   []string{"tag1", "tag2"},
				},
				expectErr: true,
				errMsg:    "cannot be an array/slice type",
			},
			{
				name: "struct type",
				dimensions: map[string]interface{}{
					"status": "active",
					"config": struct{ Field string }{Field: "value"},
				},
				expectErr: true,
				errMsg:    "cannot be a struct type",
			},
			{
				name: "simple types only",
				dimensions: map[string]interface{}{
					"status": "active",
					"count":  42,
					"ratio":  3.14,
					"flag":   true,
				},
				expectErr: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := store.Add("Test doc", tc.dimensions)
				if tc.expectErr {
					if err == nil {
						t.Fatal("expected error, got nil")
					}
					if !strings.Contains(err.Error(), tc.errMsg) {
						t.Errorf("expected error containing '%s', got: %v", tc.errMsg, err)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		}
	})

	// Test cases for Update method
	t.Run("Update with complex types", func(t *testing.T) {
		// Create a document to update
		id, err := store.Add("Update test", map[string]interface{}{
			"status": "active",
		})
		if err != nil {
			t.Fatalf("failed to create document: %v", err)
		}

		testCases := []struct {
			name      string
			update    nanostore.UpdateRequest
			expectErr bool
			errMsg    string
		}{
			{
				name: "map in dimensions",
				update: nanostore.UpdateRequest{
					Dimensions: map[string]interface{}{
						"metadata": map[string]string{"key": "value"},
					},
				},
				expectErr: true,
				errMsg:    "cannot be a map type",
			},
			{
				name: "array in dimensions",
				update: nanostore.UpdateRequest{
					Dimensions: map[string]interface{}{
						"items": []int{1, 2, 3},
					},
				},
				expectErr: true,
				errMsg:    "cannot be an array/slice type",
			},
			{
				name: "valid update",
				update: nanostore.UpdateRequest{
					Dimensions: map[string]interface{}{
						"status": "inactive",
					},
				},
				expectErr: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := store.Update(id, tc.update)
				if tc.expectErr {
					if err == nil {
						t.Fatal("expected error, got nil")
					}
					if !strings.Contains(err.Error(), tc.errMsg) {
						t.Errorf("expected error containing '%s', got: %v", tc.errMsg, err)
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		}
	})
}

func TestUnmarshalDimensionsComplexTypes(t *testing.T) {
	// Test that UnmarshalDimensions properly handles complex types
	testCases := []struct {
		name      string
		doc       nanostore.Document
		expectErr bool
		errMsg    string
	}{
		{
			name: "map dimension to string field",
			doc: nanostore.Document{
				Title: "Test",
				Dimensions: map[string]interface{}{
					"data": map[string]interface{}{"nested": "value"},
				},
			},
			expectErr: true,
			errMsg:    "cannot convert",
		},
		{
			name: "array dimension to string field",
			doc: nanostore.Document{
				Title: "Test",
				Dimensions: map[string]interface{}{
					"tags": []string{"tag1", "tag2"},
				},
			},
			expectErr: true,
			errMsg:    "cannot convert",
		},
	}

	type TestStruct struct {
		nanostore.Document
		Data string `dimension:"data"`
		Tags string `dimension:"tags"`
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result TestStruct
			err := api.UnmarshalDimensions(tc.doc, &result)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("expected error containing '%s', got: %v", tc.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
