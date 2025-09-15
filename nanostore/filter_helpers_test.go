package nanostore

import (
	"reflect"
	"testing"
)

// TODO: Remove this test - WithStatusFilter has been removed
/*
func TestWithStatusFilter(t *testing.T) {
	tests := []struct {
		name         string
		initialOpts  ListOptions
		statuses     []string
		expectedOpts ListOptions
	}{
		{
			name:        "single status on empty options",
			initialOpts: ListOptions{},
			statuses:    []string{"pending"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": "pending",
				},
			},
		},
		{
			name:        "multiple statuses on empty options",
			initialOpts: ListOptions{},
			statuses:    []string{"pending", "completed", "archived"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": []string{"pending", "completed", "archived"},
				},
			},
		},
		{
			name: "single status with nil Filters map",
			initialOpts: ListOptions{
				Filters: nil,
			},
			statuses: []string{"active"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": "active",
				},
			},
		},
		{
			name: "add status to existing filters",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"priority": "high",
				},
			},
			statuses: []string{"pending"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"priority": "high",
					"status":   "pending",
				},
			},
		},
		{
			name: "override existing status filter",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": "old-status",
				},
			},
			statuses: []string{"new-status"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": "new-status",
				},
			},
		},
		{
			name: "test method chaining",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"tag": "important",
				},
			},
			statuses: []string{"pending"},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"tag":    "important",
					"status": "pending",
				},
			},
		},
		{
			name:        "empty status slice",
			initialOpts: ListOptions{},
			statuses:    []string{},
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": []string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.initialOpts.WithStatusFilter(tt.statuses...)

			if !reflect.DeepEqual(got, tt.expectedOpts) {
				t.Errorf("WithStatusFilter() = %+v, want %+v", got, tt.expectedOpts)
			}

			// Test method chaining by applying filter twice
			if tt.name == "test method chaining" {
				chained := tt.initialOpts.WithStatusFilter("first").WithStatusFilter("second")
				if chained.Filters["status"] != "second" {
					t.Errorf("Method chaining failed: expected status 'second', got %v", chained.Filters["status"])
				}
			}
		})
	}
}
*/

func TestWithParentFilter(t *testing.T) {
	parentID1 := "parent-uuid-1"
	parentID2 := "parent-uuid-2"

	tests := []struct {
		name         string
		initialOpts  ListOptions
		parentUUID   *string
		expectedOpts ListOptions
	}{
		{
			name:        "non-nil parent UUID on empty options",
			initialOpts: ListOptions{},
			parentUUID:  &parentID1,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": parentID1,
				},
			},
		},
		{
			name:        "nil parent UUID (root documents)",
			initialOpts: ListOptions{},
			parentUUID:  nil,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": nil,
				},
			},
		},
		{
			name: "parent UUID with nil Filters map",
			initialOpts: ListOptions{
				Filters: nil,
			},
			parentUUID: &parentID1,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": parentID1,
				},
			},
		},
		{
			name: "add parent to existing filters",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"status": "pending",
				},
			},
			parentUUID: &parentID1,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"status":      "pending",
					"parent_uuid": parentID1,
				},
			},
		},
		{
			name: "override existing parent filter",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": "old-parent",
				},
			},
			parentUUID: &parentID2,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": parentID2,
				},
			},
		},
		{
			name: "nil parent overwrites existing parent",
			initialOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": "existing-parent",
				},
			},
			parentUUID: nil,
			expectedOpts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.initialOpts.WithParentFilter(tt.parentUUID)

			if !reflect.DeepEqual(got, tt.expectedOpts) {
				t.Errorf("WithParentFilter() = %+v, want %+v", got, tt.expectedOpts)
			}
		})
	}
}

// TODO: Remove this test - WithStatusFilter has been removed
/*
func TestFilterMethodChaining(t *testing.T) {
	parentID := "test-parent-uuid"

	// Test chaining both filter methods
	opts := NewListOptions().
		WithStatusFilter("pending", "active").
		WithParentFilter(&parentID)

	expected := ListOptions{
		Filters: map[string]interface{}{
			"status":      []string{"pending", "active"},
			"parent_uuid": parentID,
		},
	}

	if !reflect.DeepEqual(opts, expected) {
		t.Errorf("Method chaining = %+v, want %+v", opts, expected)
	}

	// Test chaining in reverse order
	opts2 := NewListOptions().
		WithParentFilter(nil).
		WithStatusFilter("completed")

	expected2 := ListOptions{
		Filters: map[string]interface{}{
			"parent_uuid": nil,
			"status":      "completed",
		},
	}

	if !reflect.DeepEqual(opts2, expected2) {
		t.Errorf("Reverse method chaining = %+v, want %+v", opts2, expected2)
	}

	// Test complex chaining with multiple applications
	opts3 := ListOptions{}.
		WithStatusFilter("first").
		WithParentFilter(&parentID).
		WithStatusFilter("second", "third").
		WithParentFilter(nil)

	expected3 := ListOptions{
		Filters: map[string]interface{}{
			"status":      []string{"second", "third"},
			"parent_uuid": nil,
		},
	}

	if !reflect.DeepEqual(opts3, expected3) {
		t.Errorf("Complex chaining = %+v, want %+v", opts3, expected3)
	}
}
*/
