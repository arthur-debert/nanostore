package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// addTypeSpecificFlags dynamically adds flags based on the current type
func (cli *ViperCLI) addTypeSpecificFlags(cmd *cobra.Command, commandName string) error {
	typeName := cli.viperInst.GetString("type")
	if typeName == "" {
		availableTypes := cli.registry.ListTypes()
		return NewTypeError(commandName, "", availableTypes)
	}

	typeDef, exists := cli.registry.GetTypeDefinition(typeName)
	if !exists {
		availableTypes := cli.registry.ListTypes()
		return NewTypeError(commandName, typeName, availableTypes)
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

		if cmd.Flags().Lookup(flagName) == nil {
			cmd.Flags().String(flagName, "", description)
			_ = cli.viperInst.BindPFlag(flagName, cmd.Flags().Lookup(flagName))
			_ = cli.viperInst.BindEnv(flagName, "NANOSTORE_"+strings.ToUpper(strings.ReplaceAll(flagName, "-", "_")))
		}
	}

	// Add field flags
	for fieldName, fieldSchema := range typeDef.Schema.Fields {
		flagName := strings.ReplaceAll(fieldName, "_", "-")
		description := fieldSchema.Description
		if description == "" {
			description = fmt.Sprintf("Value for %s field (%s)", fieldName, fieldSchema.Type)
		}

		if cmd.Flags().Lookup(flagName) == nil {
			cmd.Flags().String(flagName, "", description)
			_ = cli.viperInst.BindPFlag(flagName, cmd.Flags().Lookup(flagName))
			_ = cli.viperInst.BindEnv(flagName, "NANOSTORE_"+strings.ToUpper(strings.ReplaceAll(flagName, "-", "_")))
		}
	}

	return nil
}

// executeTypesCommand executes the types command
func (cli *ViperCLI) executeTypesCommand(args []string) error {
	types := cli.registry.ListTypes()

	if len(args) == 0 {
		// List all types
		if len(types) == 0 {
			fmt.Println("No types registered.")
			fmt.Println("You can register types using JSON schema files or use built-in types.")
			return nil
		}

		fmt.Println("Available document types:")
		for _, typeName := range types {
			fmt.Printf("  - %s\n", typeName)
		}
		return nil
	}

	// Show schema for specific type
	typeName := args[0]
	schema, err := cli.registry.GetSchemaJSON(typeName)
	if err != nil {
		return WrapError("get schema for type "+typeName, err,
			CommonSuggestions.CheckType,
			"Verify type is properly registered")
	}

	fmt.Printf("Schema for %s:\n%s\n", typeName, schema)
	return nil
}

// executeCreateCommand executes the create command
func (cli *ViperCLI) executeCreateCommand(title string, cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("create"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("create", map[string]interface{}{
			"title": title,
			"type":  typeName,
			"db":    dbPath,
		})
	}

	// Collect field values from Viper configuration
	data := cli.collectFieldValues(typeName)

	// Execute actual create operation
	result, err := cli.reflectionExec.ExecuteCreate(typeName, dbPath, title, data)
	if err != nil {
		return WrapError("create document", err,
			CommonSuggestions.CheckDB,
			CommonSuggestions.CheckType,
			CommonSuggestions.TryDryRun)
	}

	return cli.outputResult(result)
}

// executeGetCommand executes the get command
func (cli *ViperCLI) executeGetCommand(id string) error {
	if err := cli.validateRequiredFlags("get"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("get", map[string]interface{}{
			"id":   id,
			"type": typeName,
			"db":   dbPath,
		})
	}

	// Execute actual get operation
	result, err := cli.reflectionExec.ExecuteGet(typeName, dbPath, id)
	if err != nil {
		return WrapError("get document", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			"Try 'list' command to see available documents")
	}

	return cli.outputResult(result)
}

// executeUpdateCommand executes the update command
func (cli *ViperCLI) executeUpdateCommand(id string, cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("update"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		updates := cli.collectFieldValues(typeName)
		return cli.showDryRun("update", map[string]interface{}{
			"id":      id,
			"type":    typeName,
			"db":      dbPath,
			"updates": updates,
		})
	}

	updates := cli.collectFieldValues(typeName)

	// Execute actual update operation
	result, err := cli.reflectionExec.ExecuteUpdate(typeName, dbPath, id, updates)
	if err != nil {
		return WrapError("update document", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			CommonSuggestions.CheckType,
			CommonSuggestions.TryDryRun)
	}

	return cli.outputResult(result)
}

// executeDeleteCommand executes the delete command
func (cli *ViperCLI) executeDeleteCommand(id string, cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("delete"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")
	cascade := cli.viperInst.GetBool("cascade")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("delete", map[string]interface{}{
			"id":      id,
			"type":    typeName,
			"db":      dbPath,
			"cascade": cascade,
		})
	}

	// Execute actual delete operation
	err := cli.reflectionExec.ExecuteDelete(typeName, dbPath, id, cascade)
	if err != nil {
		return WrapError("delete document", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			"Use --cascade flag if document has children")
	}

	result := map[string]interface{}{
		"command":     "delete",
		"document_id": id,
		"type":        typeName,
		"database":    dbPath,
		"cascade":     cascade,
		"message":     "Document deleted successfully",
	}

	return cli.outputResult(result)
}

// executeListCommand executes the list command
func (cli *ViperCLI) executeListCommand(cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("list"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")
	filters := cli.viperInst.GetStringSlice("filter")
	whereClause := cli.viperInst.GetString("where")
	whereArgsStr := cli.viperInst.GetStringSlice("where-args")

	// Date range flags
	createdAfter := cli.viperInst.GetString("created-after")
	createdBefore := cli.viperInst.GetString("created-before")
	updatedAfter := cli.viperInst.GetString("updated-after")
	updatedBefore := cli.viperInst.GetString("updated-before")

	// NULL handling flags
	nullFields := cli.viperInst.GetStringSlice("null-fields")
	notNullFields := cli.viperInst.GetStringSlice("not-null-fields")

	// Text search flags
	searchText := cli.viperInst.GetString("search")
	titleContains := cli.viperInst.GetString("title-contains")
	bodyContains := cli.viperInst.GetString("body-contains")
	caseSensitive := cli.viperInst.GetBool("search-case-sensitive")

	// Enhanced filter flags
	filterEq := cli.viperInst.GetStringSlice("filter-eq")
	filterNe := cli.viperInst.GetStringSlice("filter-ne")
	filterGt := cli.viperInst.GetStringSlice("filter-gt")
	filterLt := cli.viperInst.GetStringSlice("filter-lt")
	filterGte := cli.viperInst.GetStringSlice("filter-gte")
	filterLte := cli.viperInst.GetStringSlice("filter-lte")
	filterLike := cli.viperInst.GetStringSlice("filter-like")
	// filterIn not supported - removed to avoid confusion

	// Convenience flags
	status := cli.viperInst.GetString("status")
	priority := cli.viperInst.GetString("priority")
	// statusIn and priorityIn not supported - removed to avoid confusion

	sort := cli.viperInst.GetString("sort")
	limit := cli.viperInst.GetInt("limit")
	offset := cli.viperInst.GetInt("offset")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("list", map[string]interface{}{
			"type":       typeName,
			"db":         dbPath,
			"filters":    filters,
			"where":      whereClause,
			"where_args": whereArgsStr,
			"sort":       sort,
			"limit":      limit,
			"offset":     offset,
		})
	}

	var documents interface{}
	var err error

	// Build comprehensive filter WHERE clauses
	filterWhere, filterArgs, err := cli.reflectionExec.buildFilterWhere(
		createdAfter, createdBefore, updatedAfter, updatedBefore,
		nullFields, notNullFields,
		searchText, titleContains, bodyContains, caseSensitive,
		filterEq, filterNe, filterGt, filterLt, filterGte, filterLte, filterLike,
		status, priority)
	if err != nil {
		return WrapError("build query filters", err,
			"Check filter format: field=value",
			"Use RFC3339 dates: 2024-01-01T00:00:00Z",
			"Verify field names match document schema",
			CommonSuggestions.RunHelp)
	}

	// Check if we need to use complex querying (WHERE clauses, dates, NULL checks, or text search)
	needsComplexQuery := whereClause != "" || filterWhere != ""

	if needsComplexQuery {
		// Convert string args to interface{} slice for explicit WHERE clause
		explicitArgs := make([]interface{}, len(whereArgsStr))
		for i, arg := range whereArgsStr {
			explicitArgs[i] = arg
		}

		// Combine explicit WHERE clause with all filters
		finalWhere, finalArgs := cli.reflectionExec.combineWhereClauses(
			whereClause, explicitArgs, filterWhere, filterArgs)

		// Execute complex query
		documents, err = cli.reflectionExec.ExecuteQuery(typeName, dbPath, finalWhere, finalArgs, sort, limit, offset)
		if err != nil {
			return WrapError("execute complex query", err,
				"Check WHERE clause syntax and parameters",
				"Verify field names match document schema",
				CommonSuggestions.CheckDB,
				CommonSuggestions.TryDryRun)
		}
	} else {
		// Use basic List for simple filtering (just --filter flags)
		options := cli.reflectionExec.parseListOptions(filters, sort, limit, offset)

		documents, err = cli.reflectionExec.ExecuteList(typeName, dbPath, options)
		if err != nil {
			return WrapError("list documents", err,
				CommonSuggestions.CheckDB,
				CommonSuggestions.CheckType,
				"Verify filter values match document fields")
		}
	}

	result := map[string]interface{}{
		"command":   "list",
		"type":      typeName,
		"database":  dbPath,
		"filters":   filters,
		"where":     whereClause,
		"sort":      sort,
		"limit":     limit,
		"offset":    offset,
		"documents": documents,
	}

	return cli.outputResult(result)
}

// executeBulkUpdateByDimension executes bulk update by dimension
func (cli *ViperCLI) executeBulkUpdateByDimension(cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("bulk update"); err != nil {
		return err
	}

	filters := cli.viperInst.GetStringSlice("filter")
	setValues := cli.viperInst.GetStringSlice("set")

	result := map[string]interface{}{
		"command":    "update-by-dimension",
		"filters":    filters,
		"set_values": setValues,
		"affected":   5, // Mock count
		"note":       "Actual bulk update not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeBulkUpdateByUUIDs executes bulk update by UUIDs
func (cli *ViperCLI) executeBulkUpdateByUUIDs(uuidsStr string, cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("bulk update"); err != nil {
		return err
	}

	uuids := strings.Split(uuidsStr, ",")
	setValues := cli.viperInst.GetStringSlice("set")

	result := map[string]interface{}{
		"command":    "update-by-uuids",
		"uuids":      uuids,
		"set_values": setValues,
		"affected":   len(uuids),
		"note":       "Actual bulk update not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeBulkDeleteByDimension executes bulk delete by dimension
func (cli *ViperCLI) executeBulkDeleteByDimension(cmd *cobra.Command) error {
	if err := cli.validateRequiredFlags("bulk delete"); err != nil {
		return err
	}

	filters := cli.viperInst.GetStringSlice("filter")

	result := map[string]interface{}{
		"command": "delete-by-dimension",
		"filters": filters,
		"deleted": 3, // Mock count
		"note":    "Actual bulk delete not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeGetRaw executes the get-raw command
func (cli *ViperCLI) executeGetRaw(id string) error {
	if err := cli.validateRequiredFlags("get-raw"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("get-raw", map[string]interface{}{
			"id":   id,
			"type": typeName,
			"db":   dbPath,
		})
	}

	// Execute actual get-raw operation
	result, err := cli.reflectionExec.ExecuteGetRaw(typeName, dbPath, id)
	if err != nil {
		return WrapError("get raw document", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			"Try 'list' command to see available documents")
	}

	return cli.outputResult(result)
}

// executeGetDimensions executes the get-dimensions command
func (cli *ViperCLI) executeGetDimensions(id string) error {
	if err := cli.validateRequiredFlags("get-dimensions"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("get-dimensions", map[string]interface{}{
			"id":   id,
			"type": typeName,
			"db":   dbPath,
		})
	}

	// Execute actual get-dimensions operation
	result, err := cli.reflectionExec.ExecuteGetDimensions(typeName, dbPath, id)
	if err != nil {
		return WrapError("get document dimensions", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			"Try 'list' command to see available documents")
	}

	return cli.outputResult(result)
}

// executeGetMetadata executes the get-metadata command
func (cli *ViperCLI) executeGetMetadata(id string) error {
	if err := cli.validateRequiredFlags("get-metadata"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	if cli.viperInst.GetBool("dry-run") {
		return cli.showDryRun("get-metadata", map[string]interface{}{
			"id":   id,
			"type": typeName,
			"db":   dbPath,
		})
	}

	// Execute actual get-metadata operation
	result, err := cli.reflectionExec.ExecuteGetMetadata(typeName, dbPath, id)
	if err != nil {
		return WrapError("get document metadata", err,
			CommonSuggestions.CheckID,
			CommonSuggestions.CheckDB,
			"Try 'list' command to see available documents")
	}

	return cli.outputResult(result)
}

// executeResolveUUID executes the resolve-uuid command
func (cli *ViperCLI) executeResolveUUID(simpleID string) error {
	if err := cli.validateRequiredFlags("resolve-uuid"); err != nil {
		return err
	}

	result := map[string]interface{}{
		"command":   "resolve-uuid",
		"simple_id": simpleID,
		"uuid":      "f47ac10b-58cc-4372-a567-" + simpleID,
		"note":      "Actual UUID resolution not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeDebug executes the debug command
func (cli *ViperCLI) executeDebug() error {
	if err := cli.validateRequiredFlags("debug"); err != nil {
		return err
	}

	typeName := cli.viperInst.GetString("type")
	typeDef, _ := cli.registry.GetTypeDefinition(typeName)

	result := map[string]interface{}{
		"command": "debug",
		"type":    typeName,
		"schema":  typeDef.Schema,
		"config":  cli.viperInst.AllSettings(),
		"note":    "Actual debug info not yet implemented - this shows current config",
	}

	return cli.outputResult(result)
}

// executeStats executes the stats command
func (cli *ViperCLI) executeStats() error {
	if err := cli.validateRequiredFlags("stats"); err != nil {
		return err
	}

	result := map[string]interface{}{
		"command":         "stats",
		"total_documents": 42,
		"dimensions":      3,
		"last_updated":    "2024-01-01T10:00:00Z",
		"note":            "Actual stats not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeValidate executes the validate command
func (cli *ViperCLI) executeValidate() error {
	if err := cli.validateRequiredFlags("validate"); err != nil {
		return err
	}

	result := map[string]interface{}{
		"command": "validate",
		"valid":   true,
		"errors":  []string{},
		"note":    "Actual validation not yet implemented - this is a simulation",
	}

	return cli.outputResult(result)
}

// executeShowConfig executes the config command
func (cli *ViperCLI) executeShowConfig() error {
	config := cli.viperInst.AllSettings()

	// Add information about configuration sources
	configFile := cli.viperInst.ConfigFileUsed()
	if configFile != "" {
		config["_config_file"] = configFile
	}

	config["_available_types"] = cli.registry.ListTypes()

	return cli.outputResult(config)
}

// executeGenerateConfig executes the generate-config command
func (cli *ViperCLI) executeGenerateConfig(typeName, outputFile string) error {
	typeDef, exists := cli.registry.GetTypeDefinition(typeName)
	if !exists {
		availableTypes := cli.registry.ListTypes()
		return NewTypeError("generate config", typeName, availableTypes)
	}

	config := map[string]interface{}{
		"type":   typeName,
		"db":     fmt.Sprintf("%s.db", strings.ToLower(typeName)),
		"format": "json",
	}

	// Add dimension defaults
	for dimName, dimSchema := range typeDef.Schema.Dimensions {
		if dimSchema.Default != "" {
			config[dimName] = dimSchema.Default
		}
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return WrapError("marshal configuration", err,
			"Configuration data may be invalid",
			"Try with a different document type")
	}

	if outputFile == "" {
		fmt.Println(string(configJSON))
	} else {
		if err := os.WriteFile(outputFile, configJSON, 0644); err != nil {
			return WrapError("write config file", err,
				CommonSuggestions.CheckPerms,
				"Verify output directory exists and is writable")
		}
		fmt.Printf("Configuration written to %s\n", outputFile)
	}

	return nil
}

// Helper methods

// validateRequiredFlags validates that required flags are present
func (cli *ViperCLI) validateRequiredFlags(command string) error {
	typeName := cli.viperInst.GetString("type")
	dbPath := cli.viperInst.GetString("db")

	// Some commands don't require type/db
	metaCommands := map[string]bool{
		"types":           true,
		"config":          true,
		"generate-config": true,
	}

	if metaCommands[command] {
		return nil
	}

	if typeName == "" {
		availableTypes := cli.registry.ListTypes()
		return NewTypeError(command, "", availableTypes)
	}

	if dbPath == "" {
		return NewConfigError(command, "missing database path",
			"Set --db flag to specify database file path",
			"Use environment variable: export NANOSTORE_DB=path/to/db.json",
			"Add \"db\": \"path/to/db.json\" to your config file")
	}

	return nil
}

// collectFieldValues collects field values from Viper configuration
func (cli *ViperCLI) collectFieldValues(typeName string) map[string]interface{} {
	data := make(map[string]interface{})

	typeDef, exists := cli.registry.GetTypeDefinition(typeName)
	if !exists {
		return data
	}

	// Collect dimension values
	for dimName := range typeDef.Schema.Dimensions {
		flagName := strings.ReplaceAll(dimName, "_", "-")
		if value := cli.viperInst.GetString(flagName); value != "" {
			data[dimName] = value
		}
	}

	// Collect field values
	for fieldName := range typeDef.Schema.Fields {
		flagName := strings.ReplaceAll(fieldName, "_", "-")
		if value := cli.viperInst.GetString(flagName); value != "" {
			data[fieldName] = value
		}
	}

	return data
}

// showDryRun displays what would be executed in dry-run mode
func (cli *ViperCLI) showDryRun(command string, params map[string]interface{}) error {
	result := map[string]interface{}{
		"dry_run":    true,
		"command":    command,
		"parameters": params,
		"config":     cli.viperInst.AllSettings(),
		"message":    "Dry run - no actual changes would be made",
	}

	return cli.outputResult(result)
}

// outputResult formats and outputs the result based on the configured format
func (cli *ViperCLI) outputResult(result interface{}) error {
	format := cli.viperInst.GetString("format")
	quiet := cli.viperInst.GetBool("quiet")

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "yaml":
		// TODO: Implement YAML output
		return cli.outputJSON(result)
	case "csv":
		// TODO: Implement CSV output
		return cli.outputJSON(result)
	case "table":
		return cli.outputTable(result, quiet)
	default:
		return cli.outputJSON(result)
	}
}

// outputJSON outputs result as JSON
func (cli *ViperCLI) outputJSON(result interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputTable outputs result as a human-readable table
func (cli *ViperCLI) outputTable(result interface{}, quiet bool) error {
	// For now, just output as formatted JSON
	// TODO: Implement proper table formatting
	return cli.outputJSON(result)
}
