package nanostore

import (
	"reflect"
	"testing"
)

func TestQueryAnalyzer(t *testing.T) {
	config := Config{
		Dimensions: []DimensionConfig{
			{
				Name:     "status",
				Type:     Enumerated,
				Values:   []string{"pending", "done", "archived"},
				Prefixes: map[string]string{"pending": "p", "done": "d", "archived": "a"},
			},
			{
				Name:     "priority",
				Type:     Enumerated,
				Values:   []string{"low", "medium", "high"},
				Prefixes: map[string]string{"low": "l", "medium": "m", "high": "h"},
			},
			{
				Name:     "parent",
				Type:     Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	analyzer := NewQueryAnalyzer(config)

	tests := []struct {
		name     string
		opts     ListOptions
		expected *QueryPlan
	}{
		{
			name: "empty filters uses hierarchical query",
			opts: ListOptions{},
			expected: &QueryPlan{
				Type:             HierarchicalQuery,
				Filters:          []Filter{},
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
		{
			name: "simple dimension filter",
			opts: ListOptions{
				Filters: map[string]interface{}{
					"status": "done",
				},
			},
			expected: &QueryPlan{
				Type: FlatQuery,
				Filters: []Filter{
					{Type: FilterEquals, Column: "status", Value: "done"},
				},
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
		{
			name: "multiple dimension filters",
			opts: ListOptions{
				Filters: map[string]interface{}{
					"status":   "done",
					"priority": "high",
				},
			},
			expected: &QueryPlan{
				Type: FlatQuery,
				Filters: []Filter{
					{Type: FilterEquals, Column: "status", Value: "done"},
					{Type: FilterEquals, Column: "priority", Value: "high"},
				},
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
		{
			name: "filter with IN query",
			opts: ListOptions{
				Filters: map[string]interface{}{
					"status": []string{"pending", "done"},
				},
			},
			expected: &QueryPlan{
				Type: FlatQuery,
				Filters: []Filter{
					{Type: FilterIn, Column: "status", Values: []interface{}{"pending", "done"}},
				},
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
		{
			name: "parent filter",
			opts: ListOptions{
				Filters: map[string]interface{}{
					"parent_uuid": "parent-123",
				},
			},
			expected: &QueryPlan{
				Type:             FlatQuery,
				Filters:          []Filter{},
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
				ParentFilter:     &ParentFilter{ParentUUID: "parent-123"},
			},
		},
		{
			name: "text search",
			opts: ListOptions{
				FilterBySearch: "important task",
			},
			expected: &QueryPlan{
				Type:             FlatQuery,
				Filters:          []Filter{},
				TextSearch:       "important task",
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
		{
			name: "combined filters with search",
			opts: ListOptions{
				Filters: map[string]interface{}{
					"status": "pending",
				},
				FilterBySearch: "urgent",
			},
			expected: &QueryPlan{
				Type: FlatQuery,
				Filters: []Filter{
					{Type: FilterEquals, Column: "status", Value: "pending"},
				},
				TextSearch:       "urgent",
				DimensionConfigs: config.Dimensions,
				RequiresUserIDs:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := analyzer.Analyze(tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check query type
			if plan.Type != tt.expected.Type {
				t.Errorf("expected query type %v, got %v", tt.expected.Type, plan.Type)
			}

			// Check filters (order doesn't matter)
			if !filtersEqual(plan.Filters, tt.expected.Filters) {
				t.Errorf("expected filters %+v, got %+v", tt.expected.Filters, plan.Filters)
			}

			// Check text search
			if plan.TextSearch != tt.expected.TextSearch {
				t.Errorf("expected text search %q, got %q", tt.expected.TextSearch, plan.TextSearch)
			}

			// Check parent filter
			if !parentFilterEqual(plan.ParentFilter, tt.expected.ParentFilter) {
				t.Errorf("expected parent filter %+v, got %+v", tt.expected.ParentFilter, plan.ParentFilter)
			}

			// Check other fields
			if plan.RequiresUserIDs != tt.expected.RequiresUserIDs {
				t.Errorf("expected RequiresUserIDs %v, got %v", tt.expected.RequiresUserIDs, plan.RequiresUserIDs)
			}
		})
	}
}

func TestParseFilterKey(t *testing.T) {
	analyzer := &QueryAnalyzer{}

	tests := []struct {
		key            string
		expectedColumn string
		expectedType   FilterType
	}{
		{"status", "status", FilterEquals},
		{"status__not", "status", FilterNotEquals},
		{"priority__exists", "priority", FilterExists},
		{"priority__not_exists", "priority", FilterNotExists},
		{"parent_uuid", "parent_uuid", FilterEquals},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			column, filterType := analyzer.parseFilterKey(tt.key)
			if column != tt.expectedColumn {
				t.Errorf("expected column %q, got %q", tt.expectedColumn, column)
			}
			if filterType != tt.expectedType {
				t.Errorf("expected filter type %v, got %v", tt.expectedType, filterType)
			}
		})
	}
}

func TestAnalyzeFiltersWithSuffixes(t *testing.T) {
	config := Config{
		Dimensions: []DimensionConfig{
			{
				Name:   "status",
				Type:   Enumerated,
				Values: []string{"pending", "done"},
			},
			{
				Name:   "assignee",
				Type:   Enumerated,
				Values: []string{"alice", "bob", "charlie"},
			},
		},
	}

	analyzer := NewQueryAnalyzer(config)

	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected []Filter
	}{
		{
			name: "not suffix",
			filters: map[string]interface{}{
				"status__not": "pending",
			},
			expected: []Filter{
				{Type: FilterNotEquals, Column: "status", Value: "pending"},
			},
		},
		{
			name: "exists suffix",
			filters: map[string]interface{}{
				"assignee__exists": true,
			},
			expected: []Filter{
				{Type: FilterExists, Column: "assignee"},
			},
		},
		{
			name: "not_exists suffix",
			filters: map[string]interface{}{
				"assignee__not_exists": true,
			},
			expected: []Filter{
				{Type: FilterNotExists, Column: "assignee"},
			},
		},
		{
			name: "mixed filters",
			filters: map[string]interface{}{
				"status":               "done",
				"assignee__not_exists": true,
			},
			expected: []Filter{
				{Type: FilterEquals, Column: "status", Value: "done"},
				{Type: FilterNotExists, Column: "assignee"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := analyzer.analyzeFilters(tt.filters)

			if !filtersEqual(filters, tt.expected) {
				t.Errorf("expected filters %+v, got %+v", tt.expected, filters)
			}
		})
	}
}

// Helper functions for comparing filters

func filtersEqual(a, b []Filter) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for easy comparison (order doesn't matter)
	aMap := make(map[string]Filter)
	bMap := make(map[string]Filter)

	for _, f := range a {
		aMap[f.Column] = f
	}
	for _, f := range b {
		bMap[f.Column] = f
	}

	for col, af := range aMap {
		bf, ok := bMap[col]
		if !ok || !filterEqual(af, bf) {
			return false
		}
	}

	return true
}

func filterEqual(a, b Filter) bool {
	if a.Type != b.Type || a.Column != b.Column {
		return false
	}

	if a.Values != nil || b.Values != nil {
		return reflect.DeepEqual(a.Values, b.Values)
	}

	return a.Value == b.Value
}

func parentFilterEqual(a, b *ParentFilter) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ParentUUID == b.ParentUUID &&
		((a.Exists == nil && b.Exists == nil) ||
			(a.Exists != nil && b.Exists != nil && *a.Exists == *b.Exists))
}
