package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

// TestUpdateByDimensionCLIParsing tests the CLI parsing for update-by-dimension command
func TestUpdateByDimensionCLIParsing(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Test the parseFilterFlags function directly
	t.Run("parseFilterFlags", func(t *testing.T) {
		filterFlags := []string{"status=pending", "priority=high"}
		expectedFilters := map[string]interface{}{
			"status":   "pending",
			"priority": "high",
		}

		actualFilters := executor.parseFilterFlags(filterFlags)
		if diff := cmp.Diff(expectedFilters, actualFilters); diff != "" {
			t.Errorf("Filter parsing mismatch (-want +got):\n%s", diff)
		}
	})

	// Test the query parsing for update data
	t.Run("query parsing for update data", func(t *testing.T) {
		// Simulate what would be parsed as filter args (non-filter flags)
		filterArgs := []string{"--status=completed", "--assignee=john"}
		query := parseFilters(filterArgs)

		expectedQuery := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "status", Operator: "eq", Value: "completed"},
						{Field: "assignee", Operator: "eq", Value: "john"},
					},
				},
			},
			Operators: []LogicalOperator{},
		}

		if diff := cmp.Diff(expectedQuery, query); diff != "" {
			t.Errorf("Query parsing mismatch (-want +got):\n%s", diff)
		}

		// Test queryToDataMap conversion
		expectedData := map[string]interface{}{
			"status":   "completed",
			"assignee": "john",
		}

		actualData := executor.queryToDataMap(query)
		if diff := cmp.Diff(expectedData, actualData); diff != "" {
			t.Errorf("Data conversion mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestParseFilterFlags tests the parseFilterFlags helper function
func TestParseFilterFlags(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	testCases := []struct {
		name     string
		filters  []string
		expected map[string]interface{}
	}{
		{
			name:     "Empty filters",
			filters:  []string{},
			expected: map[string]interface{}{},
		},
		{
			name:    "Single filter",
			filters: []string{"status=pending"},
			expected: map[string]interface{}{
				"status": "pending",
			},
		},
		{
			name:    "Multiple filters",
			filters: []string{"status=pending", "priority=high", "assignee=john"},
			expected: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
				"assignee": "john",
			},
		},
		{
			name:    "Filter with no equals sign",
			filters: []string{"status=pending", "invalid-filter", "priority=high"},
			expected: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		},
		{
			name:    "Filter with multiple equals signs",
			filters: []string{"description=test=value", "status=pending"},
			expected: map[string]interface{}{
				"description": "test=value", // Should take first split
				"status":      "pending",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.parseFilterFlags(tc.filters)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Test case '%s' failed. Mismatch (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

// TestUpdateByDimensionCommandStructure tests the command structure and flag handling
func TestUpdateByDimensionCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the update-by-dimension command
	var updateByDimCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "update-by-dimension" {
			updateByDimCmd = &cmd
			break
		}
	}

	if updateByDimCmd == nil {
		t.Fatal("update-by-dimension command not found in generated commands")
	}

	// Verify command properties
	if updateByDimCmd.Method != "UpdateByDimension" {
		t.Errorf("Expected method 'UpdateByDimension', got '%s'", updateByDimCmd.Method)
	}

	if updateByDimCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", updateByDimCmd.Category.String())
	}

	// Convert to Cobra command and test
	cobraCmd := updateByDimCmd.ToCobraCommand(generator)

	if cobraCmd.Use != "update-by-dimension" {
		t.Errorf("Expected command use 'update-by-dimension', got '%s'", cobraCmd.Use)
	}

	// Test that common flags are added for data manipulation commands
	expectedFlags := []string{"status", "priority", "category", "parent-id", "description", "assignee", "tags", "content"}
	for _, flagName := range expectedFlags {
		if cobraCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '%s' to be present in update-by-dimension command", flagName)
		}
	}
}

// TestUpdateByDimensionDryRun tests the dry run functionality
func TestUpdateByDimensionDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "update-by-dimension",
		Method:      "UpdateByDimension",
		Description: "Update documents matching dimension filters",
		Category:    CategoryBulk,
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "update-by-dimension",
	}
	cobraCmd.Flags().Bool("x-dry-run", true, "Dry run flag")

	// Create a context with a query
	query := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "pending"},
				},
			},
		},
	}
	ctx := context.Background()
	ctx = withQuery(ctx, query)
	cobraCmd.SetContext(ctx)

	// Test dry run output
	err := executor.showDryRunWithQuery(cmd, "Task", "test.db", []string{}, cobraCmd)
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}
}

// TestQueryToDimensionFiltersEdgeCases tests edge cases for the helper function
func TestQueryToDimensionFiltersEdgeCases(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	testCases := []struct {
		name     string
		query    *Query
		expected map[string]interface{}
	}{
		{
			name:     "Nil query",
			query:    nil,
			expected: map[string]interface{}{},
		},
		{
			name: "Empty query",
			query: &Query{
				Groups:    []FilterGroup{},
				Operators: []LogicalOperator{},
			},
			expected: map[string]interface{}{},
		},
		{
			name: "Query with non-eq operators",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "pending"},
							{Field: "priority", Operator: "gte", Value: "3"},
							{Field: "title", Operator: "contains", Value: "test"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"status": "pending", // Only eq operators should be included
			},
		},
		{
			name: "Query with multiple groups",
			query: &Query{
				Groups: []FilterGroup{
					{
						Conditions: []FilterCondition{
							{Field: "status", Operator: "eq", Value: "pending"},
						},
					},
					{
						Conditions: []FilterCondition{
							{Field: "priority", Operator: "eq", Value: "high"},
						},
					},
				},
				Operators: []LogicalOperator{OpOr},
			},
			expected: map[string]interface{}{
				"status":   "pending",
				"priority": "high",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.queryToDimensionFilters(tc.query)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Test case '%s' failed. Mismatch (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

// TestUpdateByDimensionContextHandling tests that the query is properly passed through context
func TestUpdateByDimensionContextHandling(t *testing.T) {
	// This test verifies that the query parsing and context handling works correctly
	// without actually executing the database operation

	// Simulate filter args that would be parsed for update data
	filterArgs := []string{"--status=completed", "--assignee=john"}
	query := parseFilters(filterArgs)

	// Verify query was parsed correctly
	expectedQuery := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "completed"},
					{Field: "assignee", Operator: "eq", Value: "john"},
				},
			},
		},
		Operators: []LogicalOperator{},
	}

	if diff := cmp.Diff(expectedQuery, query); diff != "" {
		t.Errorf("Context query parsing mismatch (-want +got):\n%s", diff)
	}

	// Test context creation (simulating what happens in main())
	ctx := context.Background()
	ctx = withQuery(ctx, query)

	// Verify context retrieval
	retrievedQuery, ok := fromContext(ctx)
	if !ok {
		t.Fatal("Failed to retrieve query from context")
	}

	if diff := cmp.Diff(query, retrievedQuery); diff != "" {
		t.Errorf("Context retrieval mismatch (-want +got):\n%s", diff)
	}
}
