package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFilters(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected *Query
	}{
		{
			name: "Simple AND query",
			args: []string{"--status=active", "--user=alice"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
							{Field: "user", Operator: "eq", Value: "alice"},
						},
					},
				},
				Operators: []LogicalOperator{},
			},
		},
		{
			name: "Query with operators",
			args: []string{"--priority__gte=5", "--user__ne=guest"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "priority", Operator: "gte", Value: "5"},
							{Field: "user", Operator: "ne", Value: "guest"},
						},
					},
				},
				Operators: []LogicalOperator{},
			},
		},
		{
			name: "Single OR query",
			args: []string{"--status=active", "--or", "--status=pending"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "pending"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr},
			},
		},
		{
			name: "Left-to-right precedence test",
			args: []string{"--user=alice", "--or", "--user=bob", "--and", "--status=active"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "user", Operator: "eq", Value: "alice"},
						},
					},
					{
						Conditions: []FilterCondition{
							{Field: "user", Operator: "eq", Value: "bob"},
						},
					},
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr, OpAnd},
			},
		},
		{
			name: "Empty args",
			args: []string{},
			expected: &Query{
				Groups:    []FilterGroup{{Conditions: []FilterCondition{}}},
				Operators: []LogicalOperator{},
			},
		},
		{
			name: "Edge case - trailing operator",
			args: []string{"--status=active", "--or"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
					{Conditions: []FilterCondition{}},
				},
				Operators: []LogicalOperator{OpOr},
			},
		},
		{
			name: "Edge case - leading operator",
			args: []string{"--or", "--status=active"},
			expected: &Query{
				Groups: []FilterGroup{
					{Conditions: []FilterCondition{}},
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr},
			},
		},
		{
			name: "Edge case - consecutive operators",
			args: []string{"--status=active", "--or", "--and", "--user=bob"},
			expected: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
					{Conditions: []FilterCondition{}},
					{
						Conditions: []FilterCondition{
							{Field: "user", Operator: "eq", Value: "bob"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr, OpAnd},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseFilters(tc.args)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Test case '%s' failed. Mismatch (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

func TestBuildWhereFromQuery(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewReflectionExecutor(registry)

	testCases := []struct {
		name           string
		query          *Query
		expectedClause string
		expectedArgs   []interface{}
	}{
		{
			name: "Single condition",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
				},
			},
			expectedClause: "status = ?",
			expectedArgs:   []interface{}{"active"},
		},
		{
			name: "Multiple conditions in single group",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
							{Field: "priority", Operator: "gte", Value: "5"},
						},
					},
				},
			},
			expectedClause: "(status = ? AND priority >= ?)",
			expectedArgs:   []interface{}{"active", "5"},
		},
		{
			name: "Multiple groups with OR",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "active"},
						},
					},
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "pending"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr},
			},
			expectedClause: "(status = ?) OR (status = ?)",
			expectedArgs:   []interface{}{"active", "pending"},
		},
		{
			name: "Contains operator",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "title", Operator: "contains", Value: "test"},
						},
					},
				},
			},
			expectedClause: "title LIKE ?",
			expectedArgs:   []interface{}{"%test%"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clause, args := executor.BuildWhereFromQuery(tc.query)
			if clause != tc.expectedClause {
				t.Errorf("Expected clause '%s', got '%s'", tc.expectedClause, clause)
			}
			if len(args) != len(tc.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tc.expectedArgs), len(args))
			}
			for i, expectedArg := range tc.expectedArgs {
				if i < len(args) && args[i] != expectedArg {
					t.Errorf("Expected arg[%d] '%v', got '%v'", i, expectedArg, args[i])
				}
			}
		})
	}
}
