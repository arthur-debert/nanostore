package ids

import (
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestBuildPartitionForDocument(t *testing.T) {
	// Create test dimension set
	dims := []types.Dimension{
		{
			Name:         "status",
			Type:         types.Enumerated,
			Values:       []string{"pending", "active", "done"},
			DefaultValue: "pending",
			Meta:         types.DimensionMetadata{Order: 0},
		},
		{
			Name:     "parent",
			Type:     types.Hierarchical,
			RefField: "parent_uuid",
			Meta:     types.DimensionMetadata{Order: 1},
		},
		{
			Name:         "priority",
			Type:         types.Enumerated,
			Values:       []string{"low", "medium", "high"},
			DefaultValue: "medium",
			Meta:         types.DimensionMetadata{Order: 2},
		},
	}
	ds := types.NewDimensionSet(dims)

	tests := []struct {
		name     string
		doc      types.Document
		expected []types.DimensionValue
	}{
		{
			name: "document with all dimensions",
			doc: types.Document{
				Dimensions: map[string]interface{}{
					"status":      "active",
					"parent_uuid": "parent-123",
					"priority":    "high",
				},
			},
			expected: []types.DimensionValue{
				{Dimension: "status", Value: "active"},
				{Dimension: "parent", Value: "parent-123"},
				{Dimension: "priority", Value: "high"},
			},
		},
		{
			name: "document with defaults",
			doc: types.Document{
				Dimensions: map[string]interface{}{
					"parent_uuid": "parent-456",
				},
			},
			expected: []types.DimensionValue{
				{Dimension: "status", Value: "pending"}, // default
				{Dimension: "parent", Value: "parent-456"},
				{Dimension: "priority", Value: "medium"}, // default
			},
		},
		{
			name: "document without hierarchical",
			doc: types.Document{
				Dimensions: map[string]interface{}{
					"status":   "done",
					"priority": "low",
				},
			},
			expected: []types.DimensionValue{
				{Dimension: "status", Value: "done"},
				// no parent dimension value
				{Dimension: "priority", Value: "low"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildPartitionForDocument(tt.doc, ds)

			if len(p.Values) != len(tt.expected) {
				t.Fatalf("values length: got %d, want %d", len(p.Values), len(tt.expected))
			}

			for i, dv := range p.Values {
				if dv.Dimension != tt.expected[i].Dimension {
					t.Errorf("values[%d].Dimension: got %q, want %q", i, dv.Dimension, tt.expected[i].Dimension)
				}
				if dv.Value != tt.expected[i].Value {
					t.Errorf("values[%d].Value: got %q, want %q", i, dv.Value, tt.expected[i].Value)
				}
			}

			if p.Position != 0 {
				t.Errorf("position should be 0, got %d", p.Position)
			}
		})
	}
}
