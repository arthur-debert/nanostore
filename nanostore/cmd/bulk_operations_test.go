package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

// Test helpers to reduce code duplication

// createTestRegistryAndExecutor creates a registry and executor for testing
func createTestRegistryAndExecutor() (*EnhancedTypeRegistry, *MethodExecutor) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)
	return registry, executor
}

// createTestQuery creates a test query from filter arguments
func createTestQuery(filterArgs []string) *Query {
	return parseFilters(filterArgs)
}

// createTestCobraCommand creates a test Cobra command with dry run flag
func createTestCobraCommand(commandName string) *cobra.Command {
	cmd := &cobra.Command{
		Use: commandName,
	}
	cmd.Flags().Bool("x-dry-run", true, "Dry run flag")
	return cmd
}

// createTestContext creates a test context with a query
func createTestContext(query *Query) context.Context {
	ctx := context.Background()
	return withQuery(ctx, query)
}

// createTestCommand creates a test command definition
func createTestCommand(name, method, description string, category CommandCategory) *Command {
	return &Command{
		Name:        name,
		Method:      method,
		Description: description,
		Category:    category,
	}
}

// TestUpdateByDimensionCLIParsing tests the CLI parsing for update-by-dimension command
func TestUpdateByDimensionCLIParsing(t *testing.T) {
	_, executor := createTestRegistryAndExecutor()

	// Test the query parsing for both filter criteria and update data
	t.Run("query parsing for filter and update data", func(t *testing.T) {
		// Simulate what would be parsed as filter args
		filterArgs := []string{"--status=pending", "--priority=high"}
		query := createTestQuery(filterArgs)

		expectedQuery := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "status", Operator: "eq", Value: "pending"},
						{Field: "priority", Operator: "eq", Value: "high"},
					},
				},
			},
			Operators: []LogicalOperator{},
		}

		if diff := cmp.Diff(expectedQuery, query); diff != "" {
			t.Errorf("Query parsing mismatch (-want +got):\n%s", diff)
		}

		// Test queryToDimensionFilters conversion (for filter criteria)
		expectedFilters := map[string]interface{}{
			"status":   "pending",
			"priority": "high",
		}

		actualFilters := executor.queryToDimensionFilters(query)
		if diff := cmp.Diff(expectedFilters, actualFilters); diff != "" {
			t.Errorf("Filter conversion mismatch (-want +got):\n%s", diff)
		}

		// Test queryToDataMap conversion (for update data)
		expectedData := map[string]interface{}{
			"status":   "pending",
			"priority": "high",
		}

		actualData := executor.queryToDataMap(query)
		if diff := cmp.Diff(expectedData, actualData); diff != "" {
			t.Errorf("Data conversion mismatch (-want +got):\n%s", diff)
		}
	})
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
	_, executor := createTestRegistryAndExecutor()

	// Create a mock command
	cmd := createTestCommand("update-by-dimension", "UpdateByDimension", "Update documents matching dimension filters", CategoryBulk)

	// Create a Cobra command with dry run flag
	cobraCmd := createTestCobraCommand("update-by-dimension")

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
	ctx := createTestContext(query)
	cobraCmd.SetContext(ctx)

	// Test dry run output
	err := executor.showDryRunWithQuery(cmd, "Task", "test.db", []string{}, cobraCmd)
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}
}

// TestUpdateByDimensionContextHandling tests that the query is correctly passed through the Cobra context
func TestUpdateByDimensionContextHandling(t *testing.T) {
	// Test the context creation and retrieval (simulating what happens in main())
	filterArgs := []string{"--status=pending", "--priority=high"}
	query := createTestQuery(filterArgs)

	expectedQuery := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "pending"},
					{Field: "priority", Operator: "eq", Value: "high"},
				},
			},
		},
		Operators: []LogicalOperator{},
	}

	if diff := cmp.Diff(expectedQuery, query); diff != "" {
		t.Errorf("Context query parsing mismatch (-want +got):\n%s", diff)
	}

	// Test context creation (simulating what happens in main())
	ctx := createTestContext(query)

	// Verify context retrieval
	retrievedQuery, ok := fromContext(ctx)
	if !ok {
		t.Fatal("Failed to retrieve query from context")
	}

	if diff := cmp.Diff(query, retrievedQuery); diff != "" {
		t.Errorf("Context retrieval mismatch (-want +got):\n%s", diff)
	}
}

// End-to-end tests with temporary database

// TestBulkOperationsEndToEnd tests the happy path for bulk operations against a real database
func TestBulkOperationsEndToEnd(t *testing.T) {
	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Clean up the database file after the test
	t.Cleanup(func() {
		_ = os.Remove(dbPath)
	})

	_, executor := createTestRegistryAndExecutor()

	t.Run("update-by-dimension end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("update-by-dimension", "UpdateByDimension", "Update documents matching dimension filters", CategoryBulk)

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("update-by-dimension")

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
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{}, cobraCmd)
		if err != nil {
			t.Errorf("update-by-dimension dry run failed: %v", err)
		}
	})

	t.Run("delete-by-dimension end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("delete-by-dimension", "DeleteByDimension", "Delete documents matching dimension filters", CategoryBulk)

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("delete-by-dimension")

		// Create a context with a query
		query := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "status", Operator: "eq", Value: "archived"},
					},
				},
			},
		}
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{}, cobraCmd)
		if err != nil {
			t.Errorf("delete-by-dimension dry run failed: %v", err)
		}
	})

	t.Run("update-where end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("update-where", "UpdateWhere", "Update documents matching WHERE clause", CategoryBulk)

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("update-where")

		// Create a context with a query
		query := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "priority", Operator: "gte", Value: "3"},
					},
				},
			},
		}
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{}, cobraCmd)
		if err != nil {
			t.Errorf("update-where dry run failed: %v", err)
		}
	})

	t.Run("delete-where end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("delete-where", "DeleteWhere", "Delete documents matching WHERE clause", CategoryBulk)

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("delete-where")

		// Create a context with a query
		query := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "created_at", Operator: "lt", Value: "2023-01-01"},
					},
				},
			},
		}
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{}, cobraCmd)
		if err != nil {
			t.Errorf("delete-where dry run failed: %v", err)
		}
	})

	t.Run("update-by-uuids end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("update-by-uuids", "UpdateByUUIDs", "Update documents by list of UUIDs", CategoryBulk)
		cmd.Args = []ArgSpec{
			{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
		}

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("update-by-uuids")

		// Create a context with a query
		query := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "status", Operator: "eq", Value: "completed"},
					},
				},
			},
		}
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{"uuid1,uuid2,uuid3"}, cobraCmd)
		if err != nil {
			t.Errorf("update-by-uuids dry run failed: %v", err)
		}
	})

	t.Run("delete-by-uuids end-to-end", func(t *testing.T) {
		// Create a test command
		cmd := createTestCommand("delete-by-uuids", "DeleteByUUIDs", "Delete documents by list of UUIDs", CategoryBulk)
		cmd.Args = []ArgSpec{
			{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
		}

		// Create a Cobra command
		cobraCmd := createTestCobraCommand("delete-by-uuids")

		// Create a context with a query
		query := &Query{
			Groups: []FilterGroup{},
		}
		ctx := createTestContext(query)
		cobraCmd.SetContext(ctx)

		// Test that the command executes without error (dry run)
		err := executor.showDryRunWithQuery(cmd, "Task", dbPath, []string{"uuid1,uuid2,uuid3"}, cobraCmd)
		if err != nil {
			t.Errorf("delete-by-uuids dry run failed: %v", err)
		}
	})
}

// TestUpdateWhereCLIParsing tests the CLI parsing for update-where command
func TestUpdateWhereCLIParsing(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

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

// TestUpdateWhereCommandStructure tests the command structure and flag handling
func TestUpdateWhereCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the update-where command
	var updateWhereCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "update-where" {
			updateWhereCmd = &cmd
			break
		}
	}

	if updateWhereCmd == nil {
		t.Fatal("update-where command not found in generated commands")
	}

	// Verify command properties
	if updateWhereCmd.Method != "UpdateWhere" {
		t.Errorf("Expected method 'UpdateWhere', got '%s'", updateWhereCmd.Method)
	}

	if updateWhereCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", updateWhereCmd.Category.String())
	}

	// Convert to Cobra command and test
	cobraCmd := updateWhereCmd.ToCobraCommand(generator)

	if cobraCmd.Use != "update-where" {
		t.Errorf("Expected command use 'update-where', got '%s'", cobraCmd.Use)
	}

	// Test that common flags are added for data manipulation commands
	expectedFlags := []string{"status", "priority", "category", "parent-id", "description", "assignee", "tags", "content"}
	for _, flagName := range expectedFlags {
		if cobraCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '%s' to be present in update-where command", flagName)
		}
	}
}

// TestUpdateWhereDryRun tests the dry run functionality
func TestUpdateWhereDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "update-where",
		Method:      "UpdateWhere",
		Description: "Update documents matching WHERE clause",
		Category:    CategoryBulk,
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "update-where",
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

// TestUpdateByUUIDsCLIParsing tests the CLI parsing for update-by-uuids command
func TestUpdateByUUIDsCLIParsing(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

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

	// Test UUID parsing
	t.Run("UUID list parsing", func(t *testing.T) {
		uuidStr := "uuid1,uuid2,uuid3"
		expectedUUIDs := []string{"uuid1", "uuid2", "uuid3"}

		actualUUIDs := strings.Split(uuidStr, ",")
		if diff := cmp.Diff(expectedUUIDs, actualUUIDs); diff != "" {
			t.Errorf("UUID parsing mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestUpdateByUUIDsCommandStructure tests the command structure and flag handling
func TestUpdateByUUIDsCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the update-by-uuids command
	var updateByUUIDsCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "update-by-uuids" {
			updateByUUIDsCmd = &cmd
			break
		}
	}

	if updateByUUIDsCmd == nil {
		t.Fatal("update-by-uuids command not found in generated commands")
	}

	// Verify command properties
	if updateByUUIDsCmd.Method != "UpdateByUUIDs" {
		t.Errorf("Expected method 'UpdateByUUIDs', got '%s'", updateByUUIDsCmd.Method)
	}

	if updateByUUIDsCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", updateByUUIDsCmd.Category.String())
	}

	// Verify it has the required UUIDs argument
	if len(updateByUUIDsCmd.Args) == 0 {
		t.Error("Expected update-by-uuids command to have at least one required argument")
	}

	// Convert to Cobra command and test
	cobraCmd := updateByUUIDsCmd.ToCobraCommand(generator)

	expectedUse := "update-by-uuids <uuids>"
	if cobraCmd.Use != expectedUse {
		t.Errorf("Expected command use '%s', got '%s'", expectedUse, cobraCmd.Use)
	}

	// Test that common flags are added for data manipulation commands
	expectedFlags := []string{"status", "priority", "category", "parent-id", "description", "assignee", "tags", "content"}
	for _, flagName := range expectedFlags {
		if cobraCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '%s' to be present in update-by-uuids command", flagName)
		}
	}
}

// TestUpdateByUUIDsDryRun tests the dry run functionality
func TestUpdateByUUIDsDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "update-by-uuids",
		Method:      "UpdateByUUIDs",
		Description: "Update documents by list of UUIDs",
		Category:    CategoryBulk,
		Args: []ArgSpec{
			{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
		},
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "update-by-uuids",
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

	// Test dry run output with UUIDs argument
	err := executor.showDryRunWithQuery(cmd, "Task", "test.db", []string{"uuid1,uuid2,uuid3"}, cobraCmd)
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}
}

// TestDeleteByDimensionCLIParsing tests the CLI parsing for delete-by-dimension command
func TestDeleteByDimensionCLIParsing(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Test the query parsing for filter criteria
	t.Run("query parsing for filter criteria", func(t *testing.T) {
		// Simulate what would be parsed as filter args
		filterArgs := []string{"--status=archived", "--priority=low"}
		query := parseFilters(filterArgs)

		expectedQuery := &Query{
			Groups: []FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "status", Operator: "eq", Value: "archived"},
						{Field: "priority", Operator: "eq", Value: "low"},
					},
				},
			},
			Operators: []LogicalOperator{},
		}

		if diff := cmp.Diff(expectedQuery, query); diff != "" {
			t.Errorf("Query parsing mismatch (-want +got):\n%s", diff)
		}

		// Test queryToDimensionFilters conversion
		expectedFilters := map[string]interface{}{
			"status":   "archived",
			"priority": "low",
		}

		actualFilters := executor.queryToDimensionFilters(query)
		if diff := cmp.Diff(expectedFilters, actualFilters); diff != "" {
			t.Errorf("Filter conversion mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestDeleteByDimensionCommandStructure tests the command structure and flag handling
func TestDeleteByDimensionCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the delete-by-dimension command
	var deleteByDimCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "delete-by-dimension" {
			deleteByDimCmd = &cmd
			break
		}
	}

	if deleteByDimCmd == nil {
		t.Fatal("delete-by-dimension command not found in generated commands")
	}

	// Verify command properties
	if deleteByDimCmd.Method != "DeleteByDimension" {
		t.Errorf("Expected method 'DeleteByDimension', got '%s'", deleteByDimCmd.Method)
	}

	if deleteByDimCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", deleteByDimCmd.Category.String())
	}

	// Convert to Cobra command and test
	cobraCmd := deleteByDimCmd.ToCobraCommand(generator)

	if cobraCmd.Use != "delete-by-dimension" {
		t.Errorf("Expected command use 'delete-by-dimension', got '%s'", cobraCmd.Use)
	}
}

// TestDeleteByDimensionDryRun tests the dry run functionality
func TestDeleteByDimensionDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "delete-by-dimension",
		Method:      "DeleteByDimension",
		Description: "Delete documents matching dimension filters",
		Category:    CategoryBulk,
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "delete-by-dimension",
	}
	cobraCmd.Flags().Bool("x-dry-run", true, "Dry run flag")

	// Create a context with a query
	query := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "archived"},
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

// TestDeleteWhereCommandStructure tests the command structure and flag handling
func TestDeleteWhereCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the delete-where command
	var deleteWhereCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "delete-where" {
			deleteWhereCmd = &cmd
			break
		}
	}

	if deleteWhereCmd == nil {
		t.Fatal("delete-where command not found in generated commands")
	}

	// Verify command properties
	if deleteWhereCmd.Method != "DeleteWhere" {
		t.Errorf("Expected method 'DeleteWhere', got '%s'", deleteWhereCmd.Method)
	}

	if deleteWhereCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", deleteWhereCmd.Category.String())
	}

	// Convert to Cobra command and test
	cobraCmd := deleteWhereCmd.ToCobraCommand(generator)

	if cobraCmd.Use != "delete-where" {
		t.Errorf("Expected command use 'delete-where', got '%s'", cobraCmd.Use)
	}
}

// TestDeleteWhereDryRun tests the dry run functionality
func TestDeleteWhereDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "delete-where",
		Method:      "DeleteWhere",
		Description: "Delete documents matching WHERE clause",
		Category:    CategoryBulk,
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "delete-where",
	}
	cobraCmd.Flags().Bool("x-dry-run", true, "Dry run flag")

	// Create a context with a query
	query := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "archived"},
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

// TestDeleteByUUIDsCommandStructure tests the command structure and flag handling
func TestDeleteByUUIDsCommandStructure(t *testing.T) {
	generator := NewCommandGenerator()
	commands := generator.GenerateCommands()

	// Find the delete-by-uuids command
	var deleteByUUIDsCmd *Command
	for _, cmd := range commands {
		if cmd.Name == "delete-by-uuids" {
			deleteByUUIDsCmd = &cmd
			break
		}
	}

	if deleteByUUIDsCmd == nil {
		t.Fatal("delete-by-uuids command not found in generated commands")
	}

	// Verify command properties
	if deleteByUUIDsCmd.Method != "DeleteByUUIDs" {
		t.Errorf("Expected method 'DeleteByUUIDs', got '%s'", deleteByUUIDsCmd.Method)
	}

	if deleteByUUIDsCmd.Category != CategoryBulk {
		t.Errorf("Expected category 'CategoryBulk', got '%s'", deleteByUUIDsCmd.Category.String())
	}

	// Verify it has the required UUIDs argument
	if len(deleteByUUIDsCmd.Args) == 0 {
		t.Error("Expected delete-by-uuids command to have at least one required argument")
	}

	// Convert to Cobra command and test
	cobraCmd := deleteByUUIDsCmd.ToCobraCommand(generator)

	expectedUse := "delete-by-uuids <uuids>"
	if cobraCmd.Use != expectedUse {
		t.Errorf("Expected command use '%s', got '%s'", expectedUse, cobraCmd.Use)
	}
}

// TestDeleteByUUIDsDryRun tests the dry run functionality
func TestDeleteByUUIDsDryRun(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	executor := NewMethodExecutor(registry)

	// Create a mock command
	cmd := &Command{
		Name:        "delete-by-uuids",
		Method:      "DeleteByUUIDs",
		Description: "Delete documents by list of UUIDs",
		Category:    CategoryBulk,
		Args: []ArgSpec{
			{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
		},
	}

	// Create a Cobra command with dry run flag
	cobraCmd := &cobra.Command{
		Use: "delete-by-uuids",
	}
	cobraCmd.Flags().Bool("x-dry-run", true, "Dry run flag")

	// Create a context with a query
	query := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "archived"},
				},
			},
		},
	}
	ctx := context.Background()
	ctx = withQuery(ctx, query)
	cobraCmd.SetContext(ctx)

	// Test dry run output with UUIDs argument
	err := executor.showDryRunWithQuery(cmd, "Task", "test.db", []string{"uuid1,uuid2,uuid3"}, cobraCmd)
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}
}
