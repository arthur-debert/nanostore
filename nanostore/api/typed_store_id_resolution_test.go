package api_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestGetGetRawIDResolutionConsistency(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some test data
	task1 := &TodoItem{
		Status:   "pending",
		Priority: "high",
	}

	task1ID, err := store.Create("Test Task 1", task1)
	if err != nil {
		t.Fatalf("Failed to add test task 1: %v", err)
	}

	task2 := &TodoItem{
		Status:   "active",
		Priority: "medium",
	}

	task2ID, err := store.Create("Test Task 2", task2)
	if err != nil {
		t.Fatalf("Failed to add test task 2: %v", err)
	}

	// Get the raw documents to get simple IDs
	task1Raw, err := store.GetRaw(task1ID)
	if err != nil {
		t.Fatalf("Failed to get raw task1: %v", err)
	}

	// Note: We don't actually need task2Raw for this test, but keeping it for completeness
	_, err = store.GetRaw(task2ID)
	if err != nil {
		t.Fatalf("Failed to get raw task2: %v", err)
	}

	// Test data: documents with their UUIDs and SimpleIDs
	testCases := []struct {
		name        string
		id          string
		shouldExist bool
		description string
	}{
		{
			name:        "ValidSimpleID",
			id:          task1Raw.SimpleID, // Use actual simple ID
			shouldExist: true,
			description: "Valid simple ID that should resolve to UUID",
		},
		{
			name:        "ValidUUID",
			id:          task1Raw.UUID, // Use actual UUID
			shouldExist: true,
			description: "Valid UUID that should work directly",
		},
		{
			name:        "InvalidID",
			id:          "nonexistent",
			shouldExist: false,
			description: "Invalid ID that doesn't exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Get method
			getResult, getErr := store.Get(tc.id)

			// Test GetRaw method
			getRawResult, getRawErr := store.GetRaw(tc.id)

			// Both methods should have consistent behavior
			if tc.shouldExist {
				// Both should succeed
				if getErr != nil {
					t.Errorf("Get(%s) failed: %v", tc.id, getErr)
				}
				if getRawErr != nil {
					t.Errorf("GetRaw(%s) failed: %v", tc.id, getRawErr)
				}

				// If both succeeded, they should return the same document
				if getResult != nil && getRawResult != nil {
					if getResult.UUID != getRawResult.UUID {
						t.Errorf("Get and GetRaw returned different documents for ID %s: Get UUID=%s, GetRaw UUID=%s",
							tc.id, getResult.UUID, getRawResult.UUID)
					}
				}
			} else {
				// Both should fail with similar error types
				if getErr == nil {
					t.Errorf("Get(%s) should have failed but succeeded", tc.id)
				}
				if getRawErr == nil {
					t.Errorf("GetRaw(%s) should have failed but succeeded", tc.id)
				}
			}

			// Log the behavior for manual inspection
			t.Logf("ID: %s | Get: success=%v | GetRaw: success=%v",
				tc.id, getErr == nil, getRawErr == nil)
		})
	}
}

func TestSubtleIDResolutionDifferences(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a test document
	task := &TodoItem{
		Status:   "pending",
		Priority: "high",
	}

	taskID, err := store.Create("Test Task", task)
	if err != nil {
		t.Fatalf("Failed to add test task: %v", err)
	}

	// Get the document details
	taskRaw, err := store.GetRaw(taskID)
	if err != nil {
		t.Fatalf("Failed to get raw task: %v", err)
	}

	t.Logf("Task UUID: %s", taskRaw.UUID)
	t.Logf("Task SimpleID: %s", taskRaw.SimpleID)

	// Test various ID formats
	testIDs := []string{
		taskRaw.UUID,      // Valid UUID
		taskRaw.SimpleID,  // Valid SimpleID
		"fake-uuid-12345", // Invalid UUID format but might be treated as UUID
		"999",             // Numeric that doesn't exist
		"",                // Empty string
	}

	for _, testID := range testIDs {
		t.Run(fmt.Sprintf("ID_%s", testID), func(t *testing.T) {
			// Test Get method
			getResult, getErr := store.Get(testID)
			getSuccess := getErr == nil && getResult != nil

			// Test GetRaw method
			getRawResult, getRawErr := store.GetRaw(testID)
			getRawSuccess := getRawErr == nil && getRawResult != nil

			// Log detailed results
			t.Logf("Testing ID: %q", testID)
			t.Logf("  Get:    success=%v, error=%v", getSuccess, getErr)
			t.Logf("  GetRaw: success=%v, error=%v", getRawSuccess, getRawErr)

			// Both methods should behave consistently
			if getSuccess != getRawSuccess {
				t.Errorf("Inconsistent behavior for ID %q: Get success=%v, GetRaw success=%v",
					testID, getSuccess, getRawSuccess)
			}

			// If both succeed, they should return the same document
			if getSuccess && getRawSuccess {
				if getResult.UUID != getRawResult.UUID {
					t.Errorf("Different documents returned for ID %q: Get UUID=%s, GetRaw UUID=%s",
						testID, getResult.UUID, getRawResult.UUID)
				}
			}

			// Test error message consistency
			if !getSuccess && !getRawSuccess {
				// Both failed - check if error messages are similar
				if getErr == nil || getRawErr == nil {
					t.Errorf("Inconsistent error reporting for ID %q", testID)
				}
			}
		})
	}
}

func TestIDResolutionEdgeCases(t *testing.T) {
	// Create temporary file for the store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create typed store
	store, err := api.NewFromType[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("EmptyString", func(t *testing.T) {
		// Test empty string behavior
		getResult, getErr := store.Get("")
		getRawResult, getRawErr := store.GetRaw("")

		// Both should fail consistently
		if getErr == nil {
			t.Error("Get(\"\") should fail")
		}
		if getRawErr == nil {
			t.Error("GetRaw(\"\") should fail")
		}

		// Results should be nil
		if getResult != nil {
			t.Error("Get(\"\") should return nil result")
		}
		if getRawResult != nil {
			t.Error("GetRaw(\"\") should return nil result")
		}
	})

	t.Run("MalformedUUID", func(t *testing.T) {
		// Test malformed UUID behavior
		malformedUUID := "not-a-uuid-123"

		getResult, getErr := store.Get(malformedUUID)
		getRawResult, getRawErr := store.GetRaw(malformedUUID)

		// Both should fail consistently (since it's neither a valid SimpleID nor UUID)
		if getErr == nil {
			t.Errorf("Get(%s) should fail for malformed UUID", malformedUUID)
		}
		if getRawErr == nil {
			t.Errorf("GetRaw(%s) should fail for malformed UUID", malformedUUID)
		}

		// Results should be nil
		if getResult != nil {
			t.Errorf("Get(%s) should return nil result", malformedUUID)
		}
		if getRawResult != nil {
			t.Errorf("GetRaw(%s) should return nil result", malformedUUID)
		}
	})
}
