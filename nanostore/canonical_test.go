package nanostore

import (
	"testing"
)

func TestCanonicalView(t *testing.T) {
	// Create test dimension set
	dims := []Dimension{
		{
			Name:         "status",
			Type:         Enumerated,
			Values:       []string{"pending", "active", "done"},
			DefaultValue: "pending",
		},
		{
			Name:     "parent",
			Type:     Hierarchical,
			RefField: "parent_uuid",
		},
		{
			Name:         "priority",
			Type:         Enumerated,
			Values:       []string{"low", "medium", "high"},
			DefaultValue: "medium",
		},
	}
	ds := NewDimensionSet(dims)

	t.Run("String representation", func(t *testing.T) {
		tests := []struct {
			name     string
			cv       *CanonicalView
			expected string
		}{
			{
				name:     "nil canonical view",
				cv:       nil,
				expected: "canonical:*",
			},
			{
				name:     "empty filters",
				cv:       NewCanonicalView(),
				expected: "canonical:*",
			},
			{
				name: "single filter",
				cv: NewCanonicalView(
					CanonicalFilter{Dimension: "status", Value: "pending"},
				),
				expected: "canonical:status:pending",
			},
			{
				name: "multiple filters",
				cv: NewCanonicalView(
					CanonicalFilter{Dimension: "status", Value: "pending"},
					CanonicalFilter{Dimension: "priority", Value: "medium"},
				),
				expected: "canonical:status:pending,priority:medium",
			},
			{
				name: "with wildcard",
				cv: NewCanonicalView(
					CanonicalFilter{Dimension: "status", Value: "pending"},
					CanonicalFilter{Dimension: "parent", Value: "*"},
				),
				expected: "canonical:status:pending,parent:*",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.cv.String()
				if got != tt.expected {
					t.Errorf("String(): got %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("Matches", func(t *testing.T) {
		cv := NewCanonicalView(
			CanonicalFilter{Dimension: "status", Value: "pending"},
			CanonicalFilter{Dimension: "priority", Value: "medium"},
			CanonicalFilter{Dimension: "parent", Value: "*"},
		)

		tests := []struct {
			name    string
			doc     Document
			matches bool
		}{
			{
				name: "matches all filters",
				doc: Document{
					Dimensions: map[string]interface{}{
						"status":      "pending",
						"priority":    "medium",
						"parent_uuid": "some-parent",
					},
				},
				matches: true,
			},
			{
				name: "matches with defaults",
				doc: Document{
					Dimensions: map[string]interface{}{
						"parent_uuid": "some-parent",
					},
				},
				matches: true, // status and priority use defaults
			},
			{
				name: "different status",
				doc: Document{
					Dimensions: map[string]interface{}{
						"status":   "done",
						"priority": "medium",
					},
				},
				matches: false,
			},
			{
				name: "different priority",
				doc: Document{
					Dimensions: map[string]interface{}{
						"status":   "pending",
						"priority": "high",
					},
				},
				matches: false,
			},
			{
				name: "empty dimensions",
				doc: Document{
					Dimensions: map[string]interface{}{},
				},
				matches: true, // Uses all defaults
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := cv.Matches(tt.doc, ds)
				if got != tt.matches {
					t.Errorf("Matches(): got %v, want %v", got, tt.matches)
				}
			})
		}
	})

	t.Run("Filter operations", func(t *testing.T) {
		cv := NewCanonicalView(
			CanonicalFilter{Dimension: "status", Value: "pending"},
			CanonicalFilter{Dimension: "priority", Value: "medium"},
		)

		// Test GetFilterValue
		if val, ok := cv.GetFilterValue("status"); !ok || val != "pending" {
			t.Errorf("GetFilterValue(status): got (%q, %v), want (\"pending\", true)", val, ok)
		}
		if val, ok := cv.GetFilterValue("nonexistent"); ok {
			t.Errorf("GetFilterValue(nonexistent): got (%q, %v), want (\"\", false)", val, ok)
		}

		// Test HasFilter
		if !cv.HasFilter("status") {
			t.Error("HasFilter(status) should return true")
		}
		if cv.HasFilter("nonexistent") {
			t.Error("HasFilter(nonexistent) should return false")
		}

		// Test IsCanonicalValue
		if !cv.IsCanonicalValue("status", "pending") {
			t.Error("IsCanonicalValue(status, pending) should return true")
		}
		if cv.IsCanonicalValue("status", "done") {
			t.Error("IsCanonicalValue(status, done) should return false")
		}
		if !cv.IsCanonicalValue("parent", "anything") {
			t.Error("IsCanonicalValue(parent, anything) should return true (no filter)")
		}
	})

	t.Run("ExtractFromPartition", func(t *testing.T) {
		cv := NewCanonicalView(
			CanonicalFilter{Dimension: "status", Value: "pending"},
			CanonicalFilter{Dimension: "priority", Value: "medium"},
		)

		partition := Partition{
			Values: []DimensionValue{
				{Dimension: "parent", Value: "1"},
				{Dimension: "status", Value: "pending"},
				{Dimension: "priority", Value: "high"},
			},
			Position: 1,
		}

		canonical := cv.ExtractFromPartition(partition)

		// Should only extract status:pending (matches canonical)
		// priority:high doesn't match canonical filter
		// parent:1 has no canonical filter
		if len(canonical) != 1 {
			t.Fatalf("ExtractFromPartition: expected 1 canonical value, got %d", len(canonical))
		}
		if canonical[0].Dimension != "status" || canonical[0].Value != "pending" {
			t.Errorf("ExtractFromPartition: expected status:pending, got %s:%s",
				canonical[0].Dimension, canonical[0].Value)
		}
	})

	t.Run("ConfigWithCanonicalView", func(t *testing.T) {
		config := Config{
			Dimensions: []DimensionConfig{
				{
					Name:         "status",
					Type:         Enumerated,
					Values:       []string{"pending", "active", "done"},
					DefaultValue: "pending",
				},
				{
					Name:     "parent",
					Type:     Hierarchical,
					RefField: "parent_uuid",
				},
				{
					Name:         "priority",
					Type:         Enumerated,
					Values:       []string{"low", "medium", "high"},
					DefaultValue: "medium",
				},
			},
		}

		cwcv := &ConfigWithCanonicalView{Config: config}

		// Test GetCanonicalView creates default
		cv := cwcv.GetCanonicalView()
		if cv == nil {
			t.Fatal("GetCanonicalView should create default view")
		}

		// Check default filters
		if !cv.HasFilter("status") {
			t.Error("Default canonical view should have status filter")
		}
		if !cv.HasFilter("priority") {
			t.Error("Default canonical view should have priority filter")
		}
		if !cv.HasFilter("parent") {
			t.Error("Default canonical view should have parent filter")
		}

		// Check filter values
		if val, _ := cv.GetFilterValue("status"); val != "pending" {
			t.Errorf("Default status filter should be 'pending', got %q", val)
		}
		if val, _ := cv.GetFilterValue("priority"); val != "medium" {
			t.Errorf("Default priority filter should be 'medium', got %q", val)
		}
		if val, _ := cv.GetFilterValue("parent"); val != "*" {
			t.Errorf("Default parent filter should be '*', got %q", val)
		}
	})
}
