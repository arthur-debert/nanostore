package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// MethodExecutor handles the execution of Store methods using reflection
type MethodExecutor struct {
	registry *EnhancedTypeRegistry
}

// NewMethodExecutor creates a new method executor
func NewMethodExecutor(registry *EnhancedTypeRegistry) *MethodExecutor {
	return &MethodExecutor{registry: registry}
}

// ExecuteCommand executes a command using reflection to call the appropriate Store method
func (me *MethodExecutor) ExecuteCommand(cmd *Command, cobraCmd *cobra.Command, args []string) error {
	// Use global flag variables (set by root.go)
	typeName := x_typeName
	dbPath := x_dbPath
	format := x_format
	dryRun := x_dryRun

	if typeName == "" {
		return fmt.Errorf("--x-type flag is required")
	}

	if dbPath == "" {
		return fmt.Errorf("--x-db flag is required")
	}

	// Get type definition
	typeDef, exists := me.registry.GetTypeDefinition(typeName)
	if !exists {
		return fmt.Errorf("type %s not registered. Available types: %v", typeName, me.registry.ListTypes())
	}

	if dryRun {
		return me.showDryRunWithQuery(cmd, typeName, dbPath, args, cobraCmd)
	}

	// Use ReflectionExecutor for actual command execution
	reflectionExec := NewReflectionExecutor(me.registry)

	// Get the query from context
	query, _ := fromContext(cobraCmd.Context())

	switch cmd.Name {
	case "list":
		sort, _ := cobraCmd.Flags().GetString("sort")
		limit, _ := cobraCmd.Flags().GetInt("limit")

		result, err := reflectionExec.ExecuteList(typeName, dbPath, query, sort, limit, 0)
		if err != nil {
			return fmt.Errorf("failed to execute list: %w", err)
		}

		return me.outputResult(result, format)

	case "create":
		if len(args) == 0 {
			return fmt.Errorf("create command requires a title argument")
		}
		title := args[0]

		// Convert query conditions to data map for create
		data := me.queryToDataMap(query)

		result, err := reflectionExec.ExecuteCreate(typeName, dbPath, title, data)
		if err != nil {
			return fmt.Errorf("failed to execute create: %w", err)
		}

		return me.outputResult(result, format)

	case "update-by-dimension":
		// Get filter flags for dimension matching
		filterFlags, _ := cobraCmd.Flags().GetStringSlice("filter")
		filters := me.parseFilterFlags(filterFlags)
		// Use query conditions as update data
		updateData := me.queryToDataMap(query)

		result, err := reflectionExec.ExecuteUpdateByDimension(typeName, dbPath, filters, updateData)
		if err != nil {
			return fmt.Errorf("failed to execute update-by-dimension: %w", err)
		}

		return me.outputResult(result, format)

	case "update-where":
		// Get WHERE clause from arguments
		if len(args) == 0 {
			return fmt.Errorf("update-where command requires a WHERE clause argument")
		}
		whereClause := args[0]

		// Get WHERE clause arguments from flags
		whereArgs, _ := cobraCmd.Flags().GetStringSlice("args")

		// Convert WHERE args to []interface{}
		interfaceArgs := make([]interface{}, len(whereArgs))
		for i, arg := range whereArgs {
			interfaceArgs[i] = arg
		}

		// Use query conditions as update data
		updateData := me.queryToDataMap(query)

		result, err := reflectionExec.ExecuteUpdateWhere(typeName, dbPath, whereClause, updateData, interfaceArgs)
		if err != nil {
			return fmt.Errorf("failed to execute update-where: %w", err)
		}

		return me.outputResult(result, format)

	case "update-by-uuids":
		// Get UUID list from arguments
		if len(args) == 0 {
			return fmt.Errorf("update-by-uuids command requires a UUID list argument")
		}
		uuidsStr := args[0]
		uuids := strings.Split(uuidsStr, ",")

		// Use query conditions as update data
		updateData := me.queryToDataMap(query)

		result, err := reflectionExec.ExecuteUpdateByUUIDs(typeName, dbPath, uuids, updateData)
		if err != nil {
			return fmt.Errorf("failed to execute update-by-uuids: %w", err)
		}

		return me.outputResult(result, format)

	case "delete-by-dimension":
		// Get filter flags for dimension matching
		filterFlags, _ := cobraCmd.Flags().GetStringSlice("filter")
		filters := me.parseFilterFlags(filterFlags)

		result, err := reflectionExec.ExecuteDeleteByDimension(typeName, dbPath, filters)
		if err != nil {
			return fmt.Errorf("failed to execute delete-by-dimension: %w", err)
		}

		return me.outputResult(result, format)

	case "delete-where":
		// Get WHERE clause from arguments
		if len(args) == 0 {
			return fmt.Errorf("delete-where command requires a WHERE clause argument")
		}
		whereClause := args[0]

		// Get WHERE clause arguments from flags
		whereArgs, _ := cobraCmd.Flags().GetStringSlice("args")

		// Convert WHERE args to []interface{}
		interfaceArgs := make([]interface{}, len(whereArgs))
		for i, arg := range whereArgs {
			interfaceArgs[i] = arg
		}

		result, err := reflectionExec.ExecuteDeleteWhere(typeName, dbPath, whereClause, interfaceArgs)
		if err != nil {
			return fmt.Errorf("failed to execute delete-where: %w", err)
		}

		return me.outputResult(result, format)

	case "delete-by-uuids":
		// Get UUID list from arguments
		if len(args) == 0 {
			return fmt.Errorf("delete-by-uuids command requires a UUID list argument")
		}
		uuidsStr := args[0]
		uuids := strings.Split(uuidsStr, ",")

		result, err := reflectionExec.ExecuteDeleteByUUIDs(typeName, dbPath, uuids)
		if err != nil {
			return fmt.Errorf("failed to execute delete-by-uuids: %w", err)
		}

		return me.outputResult(result, format)

	default:
		// For unimplemented commands, simulate for now
		formatter := NewOutputFormatter(format)
		result, err := me.simulateCommandExecution(cmd, typeDef, args, cobraCmd)
		if err != nil {
			return err
		}

		output, err := formatter.Format(result)
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}

		fmt.Println(output)
		return nil
	}
}

// showDryRunWithQuery displays what would be executed including query information from context
func (me *MethodExecutor) showDryRunWithQuery(cmd *Command, typeName, dbPath string, args []string, cobraCmd *cobra.Command) error {
	fmt.Printf("DRY RUN: Would execute command '%s'\n", cmd.Name)
	fmt.Printf("  Type: %s\n", typeName)
	fmt.Printf("  Database: %s\n", dbPath)
	fmt.Printf("  Method: %s\n", cmd.Method)

	if len(args) > 0 {
		fmt.Printf("  Arguments: %v\n", args)
	}

	// Show the parsed query from context
	if query, ok := fromContext(cobraCmd.Context()); ok && query != nil {
		fmt.Printf("  Parsed Query:\n")
		if len(query.Groups) > 0 {
			for i, group := range query.Groups {
				if i > 0 && i-1 < len(query.Operators) {
					fmt.Printf("    %s\n", query.Operators[i-1])
				}
				fmt.Printf("    Group %d:\n", i+1)
				for _, condition := range group.Conditions {
					fmt.Printf("      %s %s %v\n", condition.Field, condition.Operator, condition.Value)
				}
			}
		} else {
			fmt.Printf("    No filter conditions\n")
		}
	} else {
		fmt.Printf("  Query: No query found in context\n")
	}

	// Show flags that would be used
	fmt.Printf("  Global Flags:\n")
	cobraCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed && strings.HasPrefix(flag.Name, "x-") {
			fmt.Printf("    --%s: %s\n", flag.Name, flag.Value.String())
		}
	})

	return nil
}

// outputResult formats and outputs a result
func (me *MethodExecutor) outputResult(result interface{}, format string) error {
	formatter := NewOutputFormatter(format)
	output, err := formatter.Format(result)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}
	fmt.Print(output)
	return nil
}

// queryToDataMap converts query conditions to a data map for create/update operations
func (me *MethodExecutor) queryToDataMap(query *Query) map[string]interface{} {
	data := make(map[string]interface{})
	if query == nil {
		return data
	}

	// Convert filter conditions to data fields
	for _, group := range query.Groups {
		for _, condition := range group.Conditions {
			if condition.Operator == "eq" {
				data[condition.Field] = condition.Value
			}
		}
	}

	return data
}

// queryToDimensionFilters converts query conditions to dimension filters map
func (me *MethodExecutor) queryToDimensionFilters(query *Query) map[string]interface{} {
	filters := make(map[string]interface{})
	if query == nil {
		return filters
	}

	// Convert filter conditions to dimension filters (same logic as queryToDataMap)
	for _, group := range query.Groups {
		for _, condition := range group.Conditions {
			if condition.Operator == "eq" {
				filters[condition.Field] = condition.Value
			}
		}
	}

	return filters
}

// parseFilterFlags converts CLI filter flags to dimension filters map
func (me *MethodExecutor) parseFilterFlags(filterFlags []string) map[string]interface{} {
	filters := make(map[string]interface{})
	for _, filter := range filterFlags {
		if parts := strings.SplitN(filter, "=", 2); len(parts) == 2 {
			filters[parts[0]] = parts[1]
		}
	}
	return filters
}

// simulateCommandExecution simulates command execution and returns mock results
// This is a placeholder until full reflection-based execution is implemented
func (me *MethodExecutor) simulateCommandExecution(cmd *Command, typeDef *TypeDefinition, args []string, cobraCmd *cobra.Command) (interface{}, error) {
	switch cmd.Name {
	case "create":
		if len(args) == 0 {
			return nil, fmt.Errorf("create command requires a title argument")
		}
		title := args[0]

		// Collect field values from flags
		data := make(map[string]interface{})
		data["title"] = title

		// Add dimension and field values from flags
		for dimName := range typeDef.Schema.Dimensions {
			if flagValue, err := cobraCmd.Flags().GetString(dimName); err == nil && flagValue != "" {
				data[dimName] = flagValue
			}
		}

		for fieldName := range typeDef.Schema.Fields {
			if flagValue, err := cobraCmd.Flags().GetString(fieldName); err == nil && flagValue != "" {
				data[fieldName] = flagValue
			}
		}

		return map[string]interface{}{
			"message":   "Document created successfully",
			"simple_id": "1", // Mock ID
			"title":     title,
			"data":      data,
			"note":      "This is a simulation - actual creation not implemented yet",
		}, nil

	case "get":
		if len(args) == 0 {
			return nil, fmt.Errorf("get command requires an ID argument")
		}
		id := args[0]

		// Return mock document data
		result := map[string]interface{}{
			"uuid":       "mock-uuid-" + id,
			"simple_id":  id,
			"title":      "Mock Document " + id,
			"created_at": "2024-01-01T10:00:00Z",
			"updated_at": "2024-01-01T10:00:00Z",
		}

		// Add mock dimension values
		for dimName, dimSchema := range typeDef.Schema.Dimensions {
			if len(dimSchema.Values) > 0 {
				result[dimName] = dimSchema.Values[0] // Use first value as mock
			} else {
				result[dimName] = "mock-" + dimName
			}
		}

		// Add mock field values
		for fieldName, fieldSchema := range typeDef.Schema.Fields {
			switch fieldSchema.Type {
			case "string":
				result[fieldName] = "mock-" + fieldName + "-value"
			case "int":
				result[fieldName] = 42
			case "bool":
				result[fieldName] = true
			default:
				result[fieldName] = nil
			}
		}

		result["note"] = "This is a simulation - actual retrieval not implemented yet"
		return result, nil

	case "list":
		// Parse filter flags
		filters, _ := cobraCmd.Flags().GetStringSlice("filter")
		sort, _ := cobraCmd.Flags().GetString("sort")
		limit, _ := cobraCmd.Flags().GetInt("limit")

		// Return mock list
		mockDocs := []map[string]interface{}{
			{
				"simple_id": "1",
				"title":     "Mock Document 1",
				"status":    "active",
			},
			{
				"simple_id": "2",
				"title":     "Mock Document 2",
				"status":    "pending",
			},
		}

		result := map[string]interface{}{
			"documents": mockDocs,
			"count":     len(mockDocs),
			"filters":   filters,
			"sort":      sort,
			"limit":     limit,
			"note":      "This is a simulation - actual listing not implemented yet",
		}

		return result, nil

	case "update":
		if len(args) == 0 {
			return nil, fmt.Errorf("update command requires an ID argument")
		}
		id := args[0]

		// Collect update values from flags
		updates := make(map[string]interface{})
		for dimName := range typeDef.Schema.Dimensions {
			if flagValue, err := cobraCmd.Flags().GetString(dimName); err == nil && flagValue != "" {
				updates[dimName] = flagValue
			}
		}

		for fieldName := range typeDef.Schema.Fields {
			if flagValue, err := cobraCmd.Flags().GetString(fieldName); err == nil && flagValue != "" {
				updates[fieldName] = flagValue
			}
		}

		return map[string]interface{}{
			"message":         "Document updated successfully",
			"updated_id":      id,
			"updates_applied": updates,
			"note":            "This is a simulation - actual update not implemented yet",
		}, nil

	case "delete":
		if len(args) == 0 {
			return nil, fmt.Errorf("delete command requires an ID argument")
		}
		id := args[0]
		cascade, _ := cobraCmd.Flags().GetBool("cascade")

		return map[string]interface{}{
			"message":    "Document deleted successfully",
			"deleted_id": id,
			"cascade":    cascade,
			"note":       "This is a simulation - actual deletion not implemented yet",
		}, nil

	case "config":
		// Return mock configuration
		config := map[string]interface{}{
			"type":       typeDef.Name,
			"dimensions": typeDef.Schema.Dimensions,
			"fields":     typeDef.Schema.Fields,
			"note":       "This is the registered type schema",
		}
		return config, nil

	case "schema":
		// Return the type schema
		return typeDef.Schema, nil

	default:
		return map[string]interface{}{
			"message":     fmt.Sprintf("Command '%s' execution simulated", cmd.Name),
			"method":      cmd.Method,
			"category":    cmd.Category.String(),
			"description": cmd.Description,
			"note":        "This is a simulation - actual execution not implemented yet",
		}, nil
	}
}

// UpdateCommandWithTypeFlags updates a command with type-specific flags
func (me *MethodExecutor) UpdateCommandWithTypeFlags(cmd *Command, typeName string, cobraCmd *cobra.Command) error {
	typeDef, exists := me.registry.GetTypeDefinition(typeName)
	if !exists {
		return fmt.Errorf("type %s not registered", typeName)
	}

	// Add dimension flags
	for dimName, dimSchema := range typeDef.Schema.Dimensions {
		flagName := strings.ReplaceAll(dimName, "_", "-")
		description := fmt.Sprintf("Value for %s dimension", dimName)

		if dimSchema.Type == "enumerated" && len(dimSchema.Values) > 0 {
			description += fmt.Sprintf(" (values: %s)", strings.Join(dimSchema.Values, ", "))
			if dimSchema.Default != "" {
				description += fmt.Sprintf(" (default: %s)", dimSchema.Default)
			}
		}

		cobraCmd.Flags().String(flagName, "", description)
	}

	// Add field flags
	for fieldName, fieldSchema := range typeDef.Schema.Fields {
		flagName := strings.ReplaceAll(fieldName, "_", "-")
		description := fieldSchema.Description
		if description == "" {
			description = fmt.Sprintf("Value for %s field (%s)", fieldName, fieldSchema.Type)
		}

		cobraCmd.Flags().String(flagName, "", description)
	}

	return nil
}

// ParseFlagValue parses a flag value according to its expected type
func (me *MethodExecutor) ParseFlagValue(value string, expectedType reflect.Type) (interface{}, error) {
	if value == "" {
		return nil, nil
	}

	switch expectedType.Kind() {
	case reflect.String:
		return value, nil

	case reflect.Int, reflect.Int64:
		return strconv.ParseInt(value, 10, 64)

	case reflect.Float64:
		return strconv.ParseFloat(value, 64)

	case reflect.Bool:
		return strconv.ParseBool(value)

	case reflect.Slice:
		if expectedType.Elem().Kind() == reflect.String {
			return strings.Split(value, ","), nil
		}

	case reflect.Ptr:
		// Handle pointer types by parsing the underlying type
		elemType := expectedType.Elem()
		parsed, err := me.ParseFlagValue(value, elemType)
		if err != nil {
			return nil, err
		}

		// Create pointer to the parsed value
		ptrValue := reflect.New(elemType)
		if parsed != nil {
			ptrValue.Elem().Set(reflect.ValueOf(parsed))
		}
		return ptrValue.Interface(), nil
	}

	// For complex types, try JSON parsing
	var result interface{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("unable to parse value %q as %s: %w", value, expectedType.String(), err)
	}

	return result, nil
}
