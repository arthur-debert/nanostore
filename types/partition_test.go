package types

import (
	"testing"
)

func TestDimensionValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DimensionValue
		wantErr  bool
	}{
		{
			name:  "valid dimension:value",
			input: "status:pending",
			expected: DimensionValue{
				Dimension: "status",
				Value:     "pending",
			},
			wantErr: false,
		},
		{
			name:  "dimension:value with spaces",
			input: " status : pending ",
			expected: DimensionValue{
				Dimension: "status",
				Value:     "pending",
			},
			wantErr: false,
		},
		{
			name:  "value with colon",
			input: "url:http://example.com",
			expected: DimensionValue{
				Dimension: "url",
				Value:     "http://example.com",
			},
			wantErr: false,
		},
		{
			name:    "missing colon",
			input:   "statuspending",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv, err := ParseDimensionValue(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dv.Dimension != tt.expected.Dimension {
				t.Errorf("dimension: got %q, want %q", dv.Dimension, tt.expected.Dimension)
			}
			if dv.Value != tt.expected.Value {
				t.Errorf("value: got %q, want %q", dv.Value, tt.expected.Value)
			}

			// Test String() method
			str := dv.String()
			expected := tt.expected.Dimension + ":" + tt.expected.Value
			if str != expected {
				t.Errorf("String(): got %q, want %q", str, expected)
			}
		})
	}
}

func TestPartition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Partition
		wantErr  bool
	}{
		{
			name:  "single dimension partition",
			input: "status:pending|3",
			expected: Partition{
				Values: []DimensionValue{
					{Dimension: "status", Value: "pending"},
				},
				Position: 3,
			},
			wantErr: false,
		},
		{
			name:  "multiple dimension partition",
			input: "parent:1,status:pending,priority:high|5",
			expected: Partition{
				Values: []DimensionValue{
					{Dimension: "parent", Value: "1"},
					{Dimension: "status", Value: "pending"},
					{Dimension: "priority", Value: "high"},
				},
				Position: 5,
			},
			wantErr: false,
		},
		{
			name:  "empty dimensions with position",
			input: "|1",
			expected: Partition{
				Values:   []DimensionValue{},
				Position: 1,
			},
			wantErr: false,
		},
		{
			name:    "missing position",
			input:   "status:pending",
			wantErr: true,
		},
		{
			name:    "invalid position",
			input:   "status:pending|abc",
			wantErr: true,
		},
		{
			name:    "invalid dimension value",
			input:   "status|1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParsePartition(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(p.Values) != len(tt.expected.Values) {
				t.Fatalf("values length: got %d, want %d", len(p.Values), len(tt.expected.Values))
			}
			for i, dv := range p.Values {
				if dv.Dimension != tt.expected.Values[i].Dimension {
					t.Errorf("values[%d].Dimension: got %q, want %q", i, dv.Dimension, tt.expected.Values[i].Dimension)
				}
				if dv.Value != tt.expected.Values[i].Value {
					t.Errorf("values[%d].Value: got %q, want %q", i, dv.Value, tt.expected.Values[i].Value)
				}
			}
			if p.Position != tt.expected.Position {
				t.Errorf("position: got %d, want %d", p.Position, tt.expected.Position)
			}

			// Test String() method round-trip
			str := p.String()
			p2, err := ParsePartition(str)
			if err != nil {
				t.Fatalf("failed to parse String() output: %v", err)
			}
			if p2.String() != str {
				t.Errorf("String() round-trip failed: got %q, want %q", p2.String(), str)
			}
		})
	}
}

func TestPartitionMethods(t *testing.T) {
	p := Partition{
		Values: []DimensionValue{
			{Dimension: "parent", Value: "1"},
			{Dimension: "status", Value: "pending"},
			{Dimension: "priority", Value: "high"},
		},
		Position: 3,
	}

	// Test Key()
	key := p.Key()
	expected := "parent:1,status:pending,priority:high"
	if key != expected {
		t.Errorf("Key(): got %q, want %q", key, expected)
	}

	// Test HasDimension()
	if !p.HasDimension("status") {
		t.Error("HasDimension(\"status\") should return true")
	}
	if p.HasDimension("nonexistent") {
		t.Error("HasDimension(\"nonexistent\") should return false")
	}

	// Test GetValue()
	if val, ok := p.GetValue("status"); !ok || val != "pending" {
		t.Errorf("GetValue(\"status\"): got (%q, %v), want (\"pending\", true)", val, ok)
	}
	if val, ok := p.GetValue("nonexistent"); ok {
		t.Errorf("GetValue(\"nonexistent\"): got (%q, %v), want (\"\", false)", val, ok)
	}
}

func TestPartitionMap(t *testing.T) {
	pm := make(PartitionMap)

	p1 := Partition{
		Values: []DimensionValue{
			{Dimension: "status", Value: "pending"},
		},
		Position: 1,
	}

	p2 := Partition{
		Values: []DimensionValue{
			{Dimension: "status", Value: "pending"},
		},
		Position: 2,
	}

	p3 := Partition{
		Values: []DimensionValue{
			{Dimension: "status", Value: "done"},
		},
		Position: 1,
	}

	doc1 := Document{UUID: "doc1"}
	doc2 := Document{UUID: "doc2"}
	doc3 := Document{UUID: "doc3"}

	// Add documents to partitions
	pm.Add(p1, doc1)
	pm.Add(p2, doc2) // Same partition as p1 (same key)
	pm.Add(p3, doc3)

	// Test Get
	pendingDocs := pm.Get(p1)
	if len(pendingDocs) != 2 {
		t.Errorf("Get(p1): expected 2 documents, got %d", len(pendingDocs))
	}

	doneDocs := pm.Get(p3)
	if len(doneDocs) != 1 {
		t.Errorf("Get(p3): expected 1 document, got %d", len(doneDocs))
	}

	// Test Count
	if count := pm.Count(p1); count != 2 {
		t.Errorf("Count(p1): expected 2, got %d", count)
	}
	if count := pm.Count(p3); count != 1 {
		t.Errorf("Count(p3): expected 1, got %d", count)
	}
}
