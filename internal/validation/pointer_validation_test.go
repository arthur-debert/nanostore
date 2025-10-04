package validation_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// EXCEPTION: This test validates error conditions and configuration failures.
// It creates fresh stores to test specific validation scenarios.

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
)

// Test structs with various pointer configurations
type StructWithStringPointer struct {
	nanostore.Document
	Name  string
	Value *string `dimension:"value"`
}

type StructWithIntPointer struct {
	nanostore.Document
	Count *int `dimension:"count"`
}

type StructWithNestedPointer struct {
	nanostore.Document
	Normal string `dimension:"normal"`
	Ptr    *struct {
		Field string
	}
}

type StructWithPointerNoTag struct {
	nanostore.Document
	Name     string  `dimension:"name"`
	PtrField *string // No tag, but still a pointer
}

func TestPointerFieldValidation(t *testing.T) {
	testCases := []struct {
		name       string
		createFunc func(string) error
		expectErr  bool
		errMsg     string
	}{
		{
			name: "string pointer with dimension tag",
			createFunc: func(filename string) error {
				_, err := api.New[StructWithStringPointer](filename)
				return err
			},
			expectErr: false, // Now supported
			errMsg:    "",
		},
		{
			name: "int pointer with dimension tag",
			createFunc: func(filename string) error {
				_, err := api.New[StructWithIntPointer](filename)
				return err
			},
			expectErr: false, // Now supported
			errMsg:    "",
		},
		{
			name: "nested struct pointer",
			createFunc: func(filename string) error {
				_, err := api.New[StructWithNestedPointer](filename)
				return err
			},
			expectErr: true, // Still not supported
			errMsg:    "pointer type",
		},
		{
			name: "pointer field without tag",
			createFunc: func(filename string) error {
				_, err := api.New[StructWithPointerNoTag](filename)
				return err
			},
			expectErr: false, // Data fields with pointers are fine
			errMsg:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a non-existent file - we should fail before trying to open it
			err := tc.createFunc("/tmp/test_pointer_validation.json")

			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error for pointer field, got nil")
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

// Verify that structs without pointers work fine
func TestValidStructsWithoutPointers(t *testing.T) {
	type ValidStruct struct {
		nanostore.Document
		Status string `values:"pending,active,done" default:"pending"`
	}

	// This should work fine
	store, err := api.New[ValidStruct]("/tmp/test_valid_struct.json")
	if err != nil {
		t.Fatalf("expected valid struct to work, got error: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Should be able to create documents
	id, err := store.Create("Test", &ValidStruct{
		Status: "active",
	})
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}
