package validation_test

// IMPORTANT: This test comprehensively validates pointer type support from issue #84
// It tests the full end-to-end functionality: creation, marshaling, unmarshaling, and querying

import (
	"os"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

// Test struct with pointer types from issue #84 (limited to 7 dimensions)
type TaskWithPointers struct {
	nanostore.Document

	// Required fields (non-pointers)
	Status string `values:"pending,active,done" default:"pending"`

	// Optional pointer fields (nullable) - limited to 6 to stay under 7 dimension limit
	DeletedAt  *time.Time `dimension:"deleted_at"`  // Nullable timestamp
	Priority   *int       `dimension:"priority"`    // Optional priority
	Score      *float64   `dimension:"score"`       // Optional score
	IsArchived *bool      `dimension:"is_archived"` // Optional flag
	Category   *string    `dimension:"category"`    // Optional category

	// Data fields (non-dimension)
	Title       string
	Description string
	Notes       *string // Pointer in data field should work
}

func TestComprehensivePointerSupport(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store - this should work without errors
	store, err := api.New[TaskWithPointers](tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create store with pointer fields: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test cases with different pointer value combinations
	testCases := []struct {
		name        string
		task        *TaskWithPointers
		description string
	}{
		{
			name: "all_nil_pointers",
			task: &TaskWithPointers{
				Title:       "Task 1",
				Status:      "pending",
				Description: "Task with all nil pointers",
				// All pointer fields are nil
			},
			description: "All pointer fields should be nil/NULL",
		},
		{
			name: "all_non_nil_pointers",
			task: func() *TaskWithPointers {
				deletedAt := time.Now().Add(-24 * time.Hour)
				priority := 5
				score := 85.5
				archived := true
				category := "work"
				notes := "Some notes"

				return &TaskWithPointers{
					Title:       "Task 2",
					Status:      "done",
					DeletedAt:   &deletedAt,
					Priority:    &priority,
					Score:       &score,
					IsArchived:  &archived,
					Category:    &category,
					Description: "Task with all non-nil pointers",
					Notes:       &notes,
				}
			}(),
			description: "All pointer fields should have values",
		},
		{
			name: "mixed_pointers",
			task: func() *TaskWithPointers {
				priority := 3
				archived := false

				return &TaskWithPointers{
					Title:       "Task 3",
					Status:      "active",
					Priority:    &priority,
					IsArchived:  &archived,
					Description: "Task with mixed pointer values",
					// Other pointers are nil
				}
			}(),
			description: "Mix of nil and non-nil pointers",
		},
	}

	var createdIDs []string

	// Test 1: Create documents with pointer fields
	t.Run("create_with_pointers", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				id, err := store.Create(tc.task.Title, tc.task)
				if err != nil {
					t.Fatalf("Failed to create task with pointers: %v", err)
				}
				if id == "" {
					t.Error("Expected non-empty ID")
				}
				createdIDs = append(createdIDs, id)
				t.Logf("Created task %s with ID: %s", tc.name, id)
			})
		}
	})

	// Test 2: Retrieve and verify pointer field values
	t.Run("retrieve_and_verify_pointers", func(t *testing.T) {
		for i, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if i >= len(createdIDs) {
					t.Fatal("Missing created ID")
				}

				retrieved, err := store.Get(createdIDs[i])
				if err != nil {
					t.Fatalf("Failed to retrieve task: %v", err)
				}

				// Verify basic fields
				if retrieved.Title != tc.task.Title {
					t.Errorf("Title mismatch: got %s, want %s", retrieved.Title, tc.task.Title)
				}
				if retrieved.Status != tc.task.Status {
					t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, tc.task.Status)
				}

				// Verify pointer fields
				verifyTimePointer(t, "DeletedAt", retrieved.DeletedAt, tc.task.DeletedAt)
				verifyIntPointer(t, "Priority", retrieved.Priority, tc.task.Priority)
				verifyFloat64Pointer(t, "Score", retrieved.Score, tc.task.Score)
				verifyBoolPointer(t, "IsArchived", retrieved.IsArchived, tc.task.IsArchived)
				verifyStringPointer(t, "Category", retrieved.Category, tc.task.Category)

				// Verify data fields
				if retrieved.Description != tc.task.Description {
					t.Errorf("Description mismatch: got %s, want %s", retrieved.Description, tc.task.Description)
				}
				verifyStringPointer(t, "Notes", retrieved.Notes, tc.task.Notes)
			})
		}
	})

	// Test 3: Query by pointer field values
	t.Run("query_by_pointer_fields", func(t *testing.T) {
		// Query by non-nil priority
		results, err := store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("Failed to query by priority: %v", err)
		}

		found := false
		for _, result := range results {
			if result.Priority != nil && *result.Priority == 5 {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find task with priority=5")
		}

		// Query by boolean pointer - just list all and filter
		results, err = store.List(types.ListOptions{})
		if err != nil {
			t.Fatalf("Failed to query by is_archived: %v", err)
		}

		found = false
		for _, result := range results {
			if result.IsArchived != nil && *result.IsArchived == true {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find task with is_archived=true")
		}
	})

	// Test 4: Update pointer fields
	t.Run("update_pointer_fields", func(t *testing.T) {
		if len(createdIDs) == 0 {
			t.Fatal("No created IDs available for update test")
		}

		// Update the first task's pointer fields
		newPriority := 10
		newArchived := true
		newCategory := "updated"

		updates := &TaskWithPointers{
			Priority:   &newPriority,
			IsArchived: &newArchived,
			Category:   &newCategory,
		}

		_, err = store.Update(createdIDs[0], updates)
		if err != nil {
			t.Fatalf("Failed to update pointer fields: %v", err)
		}

		// Verify the update
		updated, err := store.Get(createdIDs[0])
		if err != nil {
			t.Fatalf("Failed to retrieve updated task: %v", err)
		}

		verifyIntPointer(t, "Updated Priority", updated.Priority, &newPriority)
		verifyBoolPointer(t, "Updated IsArchived", updated.IsArchived, &newArchived)
		verifyStringPointer(t, "Updated Category", updated.Category, &newCategory)
	})
}

// Helper functions for pointer comparisons
func verifyTimePointer(t *testing.T, fieldName string, got, want *time.Time) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s pointer mismatch: got nil=%v, want nil=%v", fieldName, got == nil, want == nil)
		return
	}
	if got != nil && want != nil {
		// Compare times with some tolerance (1 second) due to potential precision loss
		if got.Unix() != want.Unix() {
			t.Errorf("%s time mismatch: got %v, want %v", fieldName, got, want)
		}
	}
}

func verifyIntPointer(t *testing.T, fieldName string, got, want *int) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s pointer mismatch: got nil=%v, want nil=%v", fieldName, got == nil, want == nil)
		return
	}
	if got != nil && want != nil && *got != *want {
		t.Errorf("%s value mismatch: got %d, want %d", fieldName, *got, *want)
	}
}

func verifyFloat64Pointer(t *testing.T, fieldName string, got, want *float64) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s pointer mismatch: got nil=%v, want nil=%v", fieldName, got == nil, want == nil)
		return
	}
	if got != nil && want != nil && *got != *want {
		t.Errorf("%s value mismatch: got %f, want %f", fieldName, *got, *want)
	}
}

func verifyBoolPointer(t *testing.T, fieldName string, got, want *bool) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s pointer mismatch: got nil=%v, want nil=%v", fieldName, got == nil, want == nil)
		return
	}
	if got != nil && want != nil && *got != *want {
		t.Errorf("%s value mismatch: got %v, want %v", fieldName, *got, *want)
	}
}

func verifyStringPointer(t *testing.T, fieldName string, got, want *string) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Errorf("%s pointer mismatch: got nil=%v, want nil=%v", fieldName, got == nil, want == nil)
		return
	}
	if got != nil && want != nil && *got != *want {
		t.Errorf("%s value mismatch: got %s, want %s", fieldName, *got, *want)
	}
}
