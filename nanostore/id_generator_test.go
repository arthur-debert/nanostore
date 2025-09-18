package nanostore

import (
	"sort"
	"testing"
	"time"
)

func TestIDGenerator(t *testing.T) {
	// Set up test dimensions
	dims := []Dimension{
		{
			Name:     "parent",
			Type:     Hierarchical,
			RefField: "parent_uuid",
			Meta:     DimensionMetadata{Order: 0},
		},
		{
			Name:         "status",
			Type:         Enumerated,
			Values:       []string{"pending", "active", "done"},
			Prefixes:     map[string]string{"done": "d", "active": "a"},
			DefaultValue: "pending",
			Meta:         DimensionMetadata{Order: 1},
		},
		{
			Name:         "priority",
			Type:         Enumerated,
			Values:       []string{"low", "medium", "high"},
			Prefixes:     map[string]string{"high": "h", "low": "l"},
			DefaultValue: "medium",
			Meta:         DimensionMetadata{Order: 2},
		},
	}
	ds := NewDimensionSet(dims)

	// Set up canonical view (pending status, medium priority)
	cv := NewCanonicalView(
		CanonicalFilter{Dimension: "status", Value: "pending"},
		CanonicalFilter{Dimension: "priority", Value: "medium"},
		CanonicalFilter{Dimension: "parent", Value: "*"},
	)

	generator := NewIDGenerator(ds, cv)

	t.Run("GenerateIDs", func(t *testing.T) {
		baseTime := time.Now()
		documents := []Document{
			{
				UUID:      "11111111-1111-1111-1111-111111111111",
				Title:     "First",
				CreatedAt: baseTime,
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "medium",
				},
			},
			{
				UUID:      "22222222-2222-2222-2222-222222222222",
				Title:     "Second",
				CreatedAt: baseTime.Add(time.Minute),
				Dimensions: map[string]interface{}{
					"status":   "done",
					"priority": "medium",
				},
			},
			{
				UUID:      "33333333-3333-3333-3333-333333333333",
				Title:     "Third",
				CreatedAt: baseTime.Add(2 * time.Minute),
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "high",
				},
			},
			{
				UUID:      "44444444-4444-4444-4444-444444444444",
				Title:     "Fourth",
				CreatedAt: baseTime.Add(3 * time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "11111111-1111-1111-1111-111111111111",
					"status":      "pending",
					"priority":    "medium",
				},
			},
			{
				UUID:      "55555555-5555-5555-5555-555555555555",
				Title:     "Fifth", 
				CreatedAt: baseTime.Add(4 * time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "11111111-1111-1111-1111-111111111111",
					"status":      "done",
					"priority":    "medium",
				},
			},
		}

		idMap := generator.GenerateIDs(documents)

		// Check expected IDs
		expected := map[string]string{
			"1":    "11111111-1111-1111-1111-111111111111", // First in pending/medium partition
			"d1":   "22222222-2222-2222-2222-222222222222", // First in done/medium partition
			"h1":   "33333333-3333-3333-3333-333333333333", // First in pending/high partition
			"1.1":  "44444444-4444-4444-4444-444444444444", // First child of doc1 in pending/medium
			"1.d1": "55555555-5555-5555-5555-555555555555", // First child of doc1 in done/medium
		}

		for simpleID, expectedUUID := range expected {
			if actualUUID, exists := idMap[simpleID]; !exists {
				t.Errorf("Expected ID %q not found in map", simpleID)
			} else if actualUUID != expectedUUID {
				t.Errorf("ID %q: expected UUID %q, got %q", simpleID, expectedUUID, actualUUID)
			}
		}

		// Check we have the right number of IDs
		if len(idMap) != len(expected) {
			t.Errorf("Expected %d IDs, got %d", len(expected), len(idMap))
			// Print actual IDs for debugging
			var ids []string
			for id := range idMap {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			t.Errorf("Actual IDs: %v", ids)
		}
	})

	t.Run("IDStability", func(t *testing.T) {
		// Test that IDs remain stable when documents change
		baseTime := time.Now()
		documents := []Document{
			{
				UUID:      "groceries",
				Title:     "Groceries",
				CreatedAt: baseTime,
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "medium",
				},
			},
			{
				UUID:      "milk",
				Title:     "Milk",
				CreatedAt: baseTime.Add(time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "groceries",
					"status":      "pending",
					"priority":    "medium",
				},
			},
			{
				UUID:      "bread",
				Title:     "Bread",
				CreatedAt: baseTime.Add(2 * time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "groceries",
					"status":      "pending",
					"priority":    "medium",
				},
			},
			{
				UUID:      "eggs",
				Title:     "Eggs",
				CreatedAt: baseTime.Add(3 * time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "groceries",
					"status":      "pending",
					"priority":    "medium",
				},
			},
		}

		// Generate initial IDs
		idMap1 := generator.GenerateIDs(documents)

		// Verify initial state
		if idMap1["1"] != "groceries" {
			t.Errorf("Expected groceries to be '1', got %v", idMap1)
		}
		if idMap1["1.1"] != "milk" {
			t.Errorf("Expected milk to be '1.1', got %v", idMap1)
		}
		if idMap1["1.2"] != "bread" {
			t.Errorf("Expected bread to be '1.2', got %v", idMap1)
		}
		if idMap1["1.3"] != "eggs" {
			t.Errorf("Expected eggs to be '1.3', got %v", idMap1)
		}

		// Mark bread as done
		documents[2].Dimensions["status"] = "done"

		// Generate IDs again
		idMap2 := generator.GenerateIDs(documents)

		// Verify stability - milk and eggs should keep their IDs
		if idMap2["1.1"] != "milk" {
			t.Errorf("Expected milk to remain '1.1', got %v", idMap2)
		}
		if idMap2["1.3"] != "eggs" {
			t.Errorf("Expected eggs to remain '1.3' (not renumbered), got %v", idMap2)
		}
		
		// Bread should now be in done partition
		if idMap2["1.d1"] != "bread" {
			t.Errorf("Expected bread to be '1.d1', got %v", idMap2)
		}
	})

	t.Run("ResolveID", func(t *testing.T) {
		baseTime := time.Now()
		documents := []Document{
			{
				UUID:      "550e8400-e29b-41d4-a716-446655440001",
				Title:     "First",
				CreatedAt: baseTime,
				Dimensions: map[string]interface{}{
					"status":   "pending",
					"priority": "medium",
				},
			},
			{
				UUID:      "550e8400-e29b-41d4-a716-446655440002",
				Title:     "Child",
				CreatedAt: baseTime.Add(time.Minute),
				Dimensions: map[string]interface{}{
					"parent_uuid": "550e8400-e29b-41d4-a716-446655440001",
					"status":      "done",
					"priority":    "medium",
				},
			},
		}

		tests := []struct {
			simpleID     string
			expectedUUID string
			expectError  bool
		}{
			{"1", "550e8400-e29b-41d4-a716-446655440001", false},
			{"1.d1", "550e8400-e29b-41d4-a716-446655440002", false},
			{"550e8400-e29b-41d4-a716-446655440001", "550e8400-e29b-41d4-a716-446655440001", false}, // UUID passthrough
			{"invalid", "", true},
			{"999", "", true}, // Non-existent position
		}

		for _, tt := range tests {
			uuid, err := generator.ResolveID(tt.simpleID, documents)
			if tt.expectError {
				if err == nil {
					t.Errorf("ResolveID(%q): expected error but got none", tt.simpleID)
				}
			} else {
				if err != nil {
					t.Errorf("ResolveID(%q): unexpected error: %v", tt.simpleID, err)
				} else if uuid != tt.expectedUUID {
					t.Errorf("ResolveID(%q): expected %q, got %q", tt.simpleID, tt.expectedUUID, uuid)
				}
			}
		}
	})

	t.Run("GetFullyQualifiedPartition", func(t *testing.T) {
		baseTime := time.Now()
		doc := Document{
			UUID:      "test-doc",
			Title:     "Test",
			CreatedAt: baseTime,
			Dimensions: map[string]interface{}{
				"parent_uuid": "parent-123",
				"status":      "done",
				"priority":    "high",
			},
		}

		partition := generator.getFullyQualifiedPartition(doc, 5)

		// Check position
		if partition.Position != 5 {
			t.Errorf("Expected position 5, got %d", partition.Position)
		}

		// Check dimension values
		expectedValues := map[string]string{
			"parent":   "parent-123",
			"status":   "done",
			"priority": "high",
		}

		for _, dv := range partition.Values {
			if expected, ok := expectedValues[dv.Dimension]; ok {
				if dv.Value != expected {
					t.Errorf("Dimension %s: expected value %q, got %q", dv.Dimension, expected, dv.Value)
				}
				delete(expectedValues, dv.Dimension)
			}
		}

		// Check all expected dimensions were found
		if len(expectedValues) > 0 {
			var missing []string
			for dim := range expectedValues {
				missing = append(missing, dim)
			}
			t.Errorf("Missing dimensions: %v", missing)
		}

		// Test string representation
		partitionStr := partition.String()
		expected := "parent:parent-123,status:done,priority:high|5"
		if partitionStr != expected {
			t.Errorf("String representation: expected %q, got %q", expected, partitionStr)
		}
	})
}