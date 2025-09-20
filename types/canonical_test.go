package types

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
