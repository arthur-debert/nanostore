package matching

import (
	"testing"

	"github.com/arthur-debert/nanostore/types"
)

func TestCanonicalMatcher(t *testing.T) {
	// Create test dimension set
	dims := []types.Dimension{
		{
			Name:         "status",
			Type:         types.Enumerated,
			Values:       []string{"pending", "active", "done"},
			DefaultValue: "pending",
		},
		{
			Name:     "parent",
			Type:     types.Hierarchical,
			RefField: "parent_uuid",
		},
		{
			Name:         "priority",
			Type:         types.Enumerated,
			Values:       []string{"low", "medium", "high"},
			DefaultValue: "medium",
		},
	}
	ds := types.NewDimensionSet(dims)

	t.Run("Matches", func(t *testing.T) {
		cv := types.NewCanonicalView(
			types.CanonicalFilter{Dimension: "status", Value: "pending"},
			types.CanonicalFilter{Dimension: "priority", Value: "medium"},
			types.CanonicalFilter{Dimension: "parent", Value: "*"},
		)
		matcher := NewCanonicalMatcher(cv, ds)

		tests := []struct {
			name    string
			doc     types.Document
			matches bool
		}{
			{
				name: "matches all filters",
				doc: types.Document{
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
				doc: types.Document{
					Dimensions: map[string]interface{}{
						"parent_uuid": "some-parent",
					},
				},
				matches: true, // status and priority use defaults
			},
			{
				name: "different status",
				doc: types.Document{
					Dimensions: map[string]interface{}{
						"status":   "done",
						"priority": "medium",
					},
				},
				matches: false,
			},
			{
				name: "different priority",
				doc: types.Document{
					Dimensions: map[string]interface{}{
						"status":   "pending",
						"priority": "high",
					},
				},
				matches: false,
			},
			{
				name: "empty dimensions",
				doc: types.Document{
					Dimensions: map[string]interface{}{},
				},
				matches: true, // Uses all defaults
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := matcher.Matches(tt.doc)
				if got != tt.matches {
					t.Errorf("Matches(): got %v, want %v", got, tt.matches)
				}
			})
		}
	})

	t.Run("IsCanonicalValue", func(t *testing.T) {
		cv := types.NewCanonicalView(
			types.CanonicalFilter{Dimension: "status", Value: "pending"},
			types.CanonicalFilter{Dimension: "priority", Value: "medium"},
		)
		matcher := NewCanonicalMatcher(cv, ds)

		// Test IsCanonicalValue
		if !matcher.IsCanonicalValue("status", "pending") {
			t.Error("IsCanonicalValue(status, pending) should return true")
		}
		if matcher.IsCanonicalValue("status", "done") {
			t.Error("IsCanonicalValue(status, done) should return false")
		}
		if !matcher.IsCanonicalValue("parent", "anything") {
			t.Error("IsCanonicalValue(parent, anything) should return true (no filter)")
		}
	})

	t.Run("ExtractFromPartition", func(t *testing.T) {
		cv := types.NewCanonicalView(
			types.CanonicalFilter{Dimension: "status", Value: "pending"},
			types.CanonicalFilter{Dimension: "priority", Value: "medium"},
		)
		matcher := NewCanonicalMatcher(cv, ds)

		partition := types.Partition{
			Values: []types.DimensionValue{
				{Dimension: "parent", Value: "1"},
				{Dimension: "status", Value: "pending"},
				{Dimension: "priority", Value: "high"},
			},
			Position: 1,
		}

		canonical := matcher.ExtractFromPartition(partition)

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
}
