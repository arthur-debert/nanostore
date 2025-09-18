package nanostore

import (
	"testing"
)

func TestIDTransformer(t *testing.T) {
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

	transformer := NewIDTransformer(ds, cv)

	t.Run("ToShortForm", func(t *testing.T) {
		tests := []struct {
			name      string
			partition Partition
			expected  string
		}{
			{
				name: "root with canonical values",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 1,
				},
				expected: "1",
			},
			{
				name: "root with done status",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 2,
				},
				expected: "d2",
			},
			{
				name: "root with high priority",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "high"},
					},
					Position: 3,
				},
				expected: "h3",
			},
			{
				name: "root with done and high",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "high"},
					},
					Position: 4,
				},
				expected: "dh4",
			},
			{
				name: "child with canonical values",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "parent", Value: "1"},
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 2,
				},
				expected: "1.2",
			},
			{
				name: "child with done status",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "parent", Value: "1"},
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 3,
				},
				expected: "1.d3",
			},
			{
				name: "grandchild with high priority",
				partition: Partition{
					Values: []DimensionValue{
						{Dimension: "parent", Value: "1.2"},
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "high"},
					},
					Position: 1,
				},
				expected: "1.2.h1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := transformer.ToShortForm(tt.partition)
				if got != tt.expected {
					t.Errorf("ToShortForm(): got %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("FromShortForm", func(t *testing.T) {
		tests := []struct {
			name     string
			shortID  string
			expected Partition
			wantErr  bool
		}{
			{
				name:    "simple root",
				shortID: "1",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 1,
				},
			},
			{
				name:    "root with done prefix",
				shortID: "d2",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 2,
				},
			},
			{
				name:    "root with high prefix",
				shortID: "h3",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "priority", Value: "high"},
						{Dimension: "status", Value: "pending"},
					},
					Position: 3,
				},
			},
			{
				name:    "root with multiple prefixes",
				shortID: "dh4",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "high"},
					},
					Position: 4,
				},
			},
			{
				name:    "child",
				shortID: "1.2",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "parent", Value: "1"},
						{Dimension: "status", Value: "pending"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 2,
				},
			},
			{
				name:    "child with prefix",
				shortID: "1.d3",
				expected: Partition{
					Values: []DimensionValue{
						{Dimension: "parent", Value: "1"},
						{Dimension: "status", Value: "done"},
						{Dimension: "priority", Value: "medium"},
					},
					Position: 3,
				},
			},
			{
				name:    "empty ID",
				shortID: "",
				wantErr: true,
			},
			{
				name:    "invalid prefix",
				shortID: "x1",
				wantErr: true,
			},
			{
				name:    "no position",
				shortID: "d",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := transformer.FromShortForm(tt.shortID)
				if tt.wantErr {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if got.Position != tt.expected.Position {
					t.Errorf("Position: got %d, want %d", got.Position, tt.expected.Position)
				}

				// Check dimension values (order may vary)
				expectedMap := make(map[string]string)
				for _, dv := range tt.expected.Values {
					expectedMap[dv.Dimension] = dv.Value
				}

				gotMap := make(map[string]string)
				for _, dv := range got.Values {
					gotMap[dv.Dimension] = dv.Value
				}

				if len(gotMap) != len(expectedMap) {
					t.Errorf("dimension count: got %d, want %d", len(gotMap), len(expectedMap))
				}

				for dim, expectedVal := range expectedMap {
					if gotVal, exists := gotMap[dim]; !exists || gotVal != expectedVal {
						t.Errorf("dimension %q: got %q, want %q", dim, gotVal, expectedVal)
					}
				}
			})
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Test that ToShortForm and FromShortForm are inverses
		partitions := []Partition{
			{
				Values: []DimensionValue{
					{Dimension: "status", Value: "pending"},
					{Dimension: "priority", Value: "medium"},
				},
				Position: 1,
			},
			{
				Values: []DimensionValue{
					{Dimension: "parent", Value: "1"},
					{Dimension: "status", Value: "done"},
					{Dimension: "priority", Value: "high"},
				},
				Position: 5,
			},
			{
				Values: []DimensionValue{
					{Dimension: "parent", Value: "2.3"},
					{Dimension: "status", Value: "active"},
					{Dimension: "priority", Value: "low"},
				},
				Position: 7,
			},
		}

		for _, original := range partitions {
			shortForm := transformer.ToShortForm(original)
			reconstructed, err := transformer.FromShortForm(shortForm)
			if err != nil {
				t.Fatalf("FromShortForm(%q) failed: %v", shortForm, err)
			}

			// Verify position matches
			if reconstructed.Position != original.Position {
				t.Errorf("Position mismatch for %q: got %d, want %d",
					shortForm, reconstructed.Position, original.Position)
			}

			// Verify all original values are present
			originalMap := make(map[string]string)
			for _, dv := range original.Values {
				originalMap[dv.Dimension] = dv.Value
			}

			for _, dv := range reconstructed.Values {
				if originalVal, exists := originalMap[dv.Dimension]; exists {
					if dv.Value != originalVal {
						t.Errorf("Value mismatch for dimension %q: got %q, want %q",
							dv.Dimension, dv.Value, originalVal)
					}
				}
			}
		}
	})

	t.Run("extractPrefixesAndPosition", func(t *testing.T) {
		tests := []struct {
			name         string
			segment      string
			wantPrefixes map[string]string
			wantPosition int
			wantErr      bool
		}{
			{
				name:         "position only",
				segment:      "123",
				wantPrefixes: map[string]string{},
				wantPosition: 123,
			},
			{
				name:         "single prefix",
				segment:      "d1",
				wantPrefixes: map[string]string{"d": "status"},
				wantPosition: 1,
			},
			{
				name:         "multiple prefixes",
				segment:      "dh42",
				wantPrefixes: map[string]string{"d": "status", "h": "priority"},
				wantPosition: 42,
			},
			{
				name:    "no position",
				segment: "dh",
				wantErr: true,
			},
			{
				name:    "invalid prefix",
				segment: "xyz1",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				prefixes, position, err := transformer.extractPrefixesAndPosition(tt.segment)
				if tt.wantErr {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if position != tt.wantPosition {
					t.Errorf("position: got %d, want %d", position, tt.wantPosition)
				}

				if len(prefixes) != len(tt.wantPrefixes) {
					t.Errorf("prefix count: got %d, want %d", len(prefixes), len(tt.wantPrefixes))
				}

				for prefix, dimName := range tt.wantPrefixes {
					if got, exists := prefixes[prefix]; !exists || got != dimName {
						t.Errorf("prefix %q: got %q, want %q", prefix, got, dimName)
					}
				}
			})
		}
	})
}
