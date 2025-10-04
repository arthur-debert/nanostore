package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ViperCLI implements a complete Viper-driven CLI for Nanostore
type ViperCLI struct {
	registry       *EnhancedTypeRegistry
	executor       *MethodExecutor
	reflectionExec *ReflectionExecutor
	rootCmd        *cobra.Command
	viperInst      *viper.Viper
}

// NewViperCLI creates a new Viper-powered CLI
func NewViperCLI() *ViperCLI {
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		fmt.Printf("Warning: failed to load built-in types: %v\n", err)
	}

	executor := NewMethodExecutor(registry)
	reflectionExec := NewReflectionExecutor(registry)
	viperInst := viper.New()

	cli := &ViperCLI{
		registry:       registry,
		executor:       executor,
		reflectionExec: reflectionExec,
		viperInst:      viperInst,
	}

	cli.setupViperConfig()
	cli.createRootCommand()
	cli.addCommands()

	return cli
}

// setupViperConfig configures Viper with environment variables and config files
func (cli *ViperCLI) setupViperConfig() {
	// Check for NANOSTORE_CONFIG environment variable first
	// This allows users to specify a custom config file path
	if configFile := os.Getenv("NANOSTORE_CONFIG"); configFile != "" {
		cli.viperInst.SetConfigFile(configFile)
	} else {
		// Use default config file discovery
		cli.viperInst.SetConfigName("nanostore")
		cli.viperInst.SetConfigType("json")
		cli.viperInst.AddConfigPath(".")
		cli.viperInst.AddConfigPath("$HOME/.nanostore")
		cli.viperInst.AddConfigPath("/etc/nanostore")
	}

	// Enable environment variable support
	cli.viperInst.AutomaticEnv()
	cli.viperInst.SetEnvPrefix("NANOSTORE")

	// Replace dash with underscore in env vars (e.g., --dry-run -> NANOSTORE_DRY_RUN)
	cli.viperInst.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Read config file if it exists (ignore errors)
	_ = cli.viperInst.ReadInConfig()
}

// createRootCommand creates the root Cobra command with Viper integration
func (cli *ViperCLI) createRootCommand() {
	cli.rootCmd = &cobra.Command{
		Use:   "nanostore",
		Short: "Nanostore CLI - Schema-driven document store management",
		Long: `Nanostore CLI provides complete access to the document store API.

Configuration Sources (in order of precedence):
1. Command line flags
2. Environment variables (NANOSTORE_*)
3. Configuration files (custom path or default locations)
4. Default values from type schemas

Configuration File Discovery:
  NANOSTORE_CONFIG=/path/to/config.json  # Custom config file path
  ./nanostore.json                       # Current directory
  ~/.nanostore/nanostore.json            # User directory  
  /etc/nanostore/nanostore.json          # System directory

Examples:
  # Use built-in Task type
  nanostore --type Task --db tasks.db create "New Task" --status active --priority high
  
  # Environment variables
  export NANOSTORE_TYPE=Task NANOSTORE_DB=tasks.db
  nanostore create "New Task" --status active
  
  # Custom config file
  export NANOSTORE_CONFIG=/path/to/project-config.json
  nanostore create "New Task" --status active
  
  # Default config file
  echo '{"type": "Task", "db": "tasks.db", "format": "json"}' > ./nanostore.json
  nanostore create "New Task" --status active`,

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// First attempt to read basic config (type and db) from flags, env, and config
			_ = cli.viperInst.BindPFlags(cmd.Flags())

			// For create and update commands, try to add type-specific flags if type is known
			if (cmd.Name() == "create" || cmd.Name() == "update") && cli.viperInst.GetString("type") != "" {
				_ = cli.addTypeSpecificFlags(cmd, cmd.Name())
				// Don't fail if we can't add type-specific flags, just continue
				// This allows for help commands and basic usage
			}

			return nil
		},
	}

	// Add global flags
	cli.addGlobalFlags()
}

// addGlobalFlags adds persistent flags that apply to all commands
func (cli *ViperCLI) addGlobalFlags() {
	flags := cli.rootCmd.PersistentFlags()

	// Core configuration
	flags.StringP("type", "t", "", "Document type (required for most commands)")
	flags.StringP("db", "d", "", "Database file path (required for most commands)")

	// Output configuration
	flags.StringP("format", "f", "table", "Output format (table|json|yaml|csv)")
	flags.BoolP("quiet", "q", false, "Suppress headers and extra output")
	flags.Bool("no-color", false, "Disable colored output")

	// Execution options
	flags.Bool("dry-run", false, "Show what would happen without executing")
	flags.BoolP("verbose", "v", false, "Enable verbose output")

	// Bind all flags to Viper
	for _, flag := range []string{"type", "db", "format", "quiet", "no-color", "dry-run", "verbose"} {
		_ = cli.viperInst.BindPFlag(flag, flags.Lookup(flag))
	}

	// Set up environment variable bindings
	envVars := map[string]string{
		"type":     "TYPE",
		"db":       "DB",
		"format":   "FORMAT",
		"quiet":    "QUIET",
		"no-color": "NO_COLOR",
		"dry-run":  "DRY_RUN",
		"verbose":  "VERBOSE",
	}

	for key, envVar := range envVars {
		_ = cli.viperInst.BindEnv(key, "NANOSTORE_"+envVar)
	}
}

// addCommands adds all the CLI commands based on the Store API
func (cli *ViperCLI) addCommands() {
	// Meta commands (don't require type/db)
	cli.addTypesCommand()
	cli.addConfigCommand()
	cli.addGenerateConfigCommand()

	// Core CRUD commands
	cli.addCreateCommand()
	cli.addGetCommand()
	cli.addUpdateCommand()
	cli.addDeleteCommand()
	cli.addListCommand()

	// Bulk operations
	cli.addBulkCommands()

	// Metadata and introspection
	cli.addMetadataCommands()

	// Administrative commands
	cli.addAdminCommands()
}

// addTypesCommand adds the types command for type introspection
func (cli *ViperCLI) addTypesCommand() {
	typesCmd := &cobra.Command{
		Use:   "types [type-name]",
		Short: "List available document types or show schema for specific type",
		Long: `List all registered document types or display the complete JSON schema for a specific type.

Examples:
  nanostore types              # List all available types
  nanostore types Task         # Show complete schema for Task type`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeTypesCommand(args)
		},
	}

	cli.rootCmd.AddCommand(typesCmd)
}

// addCreateCommand adds the create command with dynamic type-specific flags
func (cli *ViperCLI) addCreateCommand() {
	createCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new document",
		Long: `Create a new document with the specified title and optional field values.

The command automatically adds type-specific flags based on the document schema.

Examples:
  nanostore --type Task create "New Task" --status active --priority high
  nanostore --type Note create "Meeting Notes" --category work --tags "meeting,q4"`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeCreateCommand(args[0], cmd)
		},
	}

	cli.rootCmd.AddCommand(createCmd)
}

// addGetCommand adds the get command
func (cli *ViperCLI) addGetCommand() {
	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Retrieve a document by ID",
		Long: `Retrieve a document by its Simple ID or UUID.

Examples:
  nanostore --type Task get 1
  nanostore --type Task get h2.1
  nanostore --type Task get f47ac10b-58cc-4372-a567-0e02b2c3d479`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeGetCommand(args[0])
		},
	}

	cli.rootCmd.AddCommand(getCmd)
}

// addUpdateCommand adds the update command with dynamic type-specific flags
func (cli *ViperCLI) addUpdateCommand() {
	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a document by ID",
		Long: `Update a document with new field values.

Zero values will clear the corresponding fields in the document.

Examples:
  nanostore --type Task update 1 --status done --assignee ""
  nanostore --type Task update h2.1 --priority low`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeUpdateCommand(args[0], cmd)
		},
	}

	cli.rootCmd.AddCommand(updateCmd)
}

// addDeleteCommand adds the delete command
func (cli *ViperCLI) addDeleteCommand() {
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a document by ID",
		Long: `Delete a document by its Simple ID or UUID.

Examples:
  nanostore --type Task delete 1
  nanostore --type Task delete 1 --cascade  # Delete with children`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeDeleteCommand(args[0], cmd)
		},
	}

	// Add delete-specific flags
	deleteCmd.Flags().Bool("cascade", false, "Delete children recursively")
	_ = cli.viperInst.BindPFlag("cascade", deleteCmd.Flags().Lookup("cascade"))

	cli.rootCmd.AddCommand(deleteCmd)
}

// addListCommand adds the list command with filtering options
func (cli *ViperCLI) addListCommand() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List documents with optional filtering and sorting",
		Long: `List documents with optional filtering, sorting, and pagination.
Supports both simple filters and complex WHERE clauses with SQL-like syntax.

Examples:
  # Basic listing
  nanostore --type Task list
  
  # Simple filters
  nanostore --type Task list --filter status=active --filter priority=high
  
  # WHERE clauses with placeholders
  nanostore --type Task list --where "status = ?" --where-args active
  nanostore --type Task list --where "status = ? AND priority = ?" --where-args active --where-args high
  
  # Date range queries
  nanostore --type Task list --created-after "2024-01-01T00:00:00Z" --created-before "2024-12-31T23:59:59Z"
  nanostore --type Task list --updated-after "2024-10-01T00:00:00Z"
  
  # NULL/NOT NULL checks
  nanostore --type Task list --null-fields assignee,due_date
  nanostore --type Task list --not-null-fields assignee --created-after "2024-01-01T00:00:00Z"
  
  # Text search in title and body
  nanostore --type Task list --search "urgent"
  nanostore --type Task list --title-contains "meeting" --body-contains "quarterly"
  nanostore --type Task list --search "bug" --search-case-sensitive
  
  # Enhanced filtering with operators
  nanostore --type Task list --filter-eq status=active --filter-ne priority=low
  nanostore --type Task list --filter-in status=active,pending --filter-gt created_at="2024-01-01"
  nanostore --type Task list --status active --priority high
  
  # Complex WHERE clauses with dates
  nanostore --type Task list --where "created_at > ? AND status = ?" --where-args "2024-01-01T00:00:00Z" --where-args active
  
  # Sorting and pagination  
  nanostore --type Task list --where "status != ?" --where-args done --sort created_at --limit 10`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeListCommand(cmd)
		},
	}

	// Add list-specific flags
	listCmd.Flags().StringSlice("filter", []string{}, "Dimension filters (key=value)")
	listCmd.Flags().String("where", "", "WHERE clause with ? placeholders (e.g., \"status = ? AND priority = ?\")")
	listCmd.Flags().StringSlice("where-args", []string{}, "Arguments for WHERE clause placeholders")

	// Enhanced filtering flags with operators
	listCmd.Flags().StringSlice("filter-eq", []string{}, "Equality filters (field=value)")
	listCmd.Flags().StringSlice("filter-ne", []string{}, "Not equal filters (field=value)")
	listCmd.Flags().StringSlice("filter-gt", []string{}, "Greater than filters (field=value)")
	listCmd.Flags().StringSlice("filter-lt", []string{}, "Less than filters (field=value)")
	listCmd.Flags().StringSlice("filter-gte", []string{}, "Greater than or equal filters (field=value)")
	listCmd.Flags().StringSlice("filter-lte", []string{}, "Less than or equal filters (field=value)")
	listCmd.Flags().StringSlice("filter-like", []string{}, "Pattern match filters (field=pattern)")
	listCmd.Flags().StringSlice("filter-in", []string{}, "Value in list filters (field=val1,val2,val3)")

	// Convenience flags for common Task fields
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().String("priority", "", "Filter by priority")
	listCmd.Flags().StringSlice("status-in", []string{}, "Filter by multiple status values")
	listCmd.Flags().StringSlice("priority-in", []string{}, "Filter by multiple priority values")

	// Date range flags
	listCmd.Flags().String("created-after", "", "Find documents created after date (RFC3339 format: 2024-01-01T00:00:00Z)")
	listCmd.Flags().String("created-before", "", "Find documents created before date (RFC3339 format: 2024-01-01T00:00:00Z)")
	listCmd.Flags().String("updated-after", "", "Find documents updated after date (RFC3339 format: 2024-01-01T00:00:00Z)")
	listCmd.Flags().String("updated-before", "", "Find documents updated before date (RFC3339 format: 2024-01-01T00:00:00Z)")

	// NULL handling flags
	listCmd.Flags().StringSlice("null-fields", []string{}, "Find documents where specified fields are NULL")
	listCmd.Flags().StringSlice("not-null-fields", []string{}, "Find documents where specified fields are NOT NULL")

	// Text search flags
	listCmd.Flags().String("search", "", "Search for text in both title and body fields")
	listCmd.Flags().String("title-contains", "", "Find documents where title contains specified text")
	listCmd.Flags().String("body-contains", "", "Find documents where body contains specified text")
	listCmd.Flags().Bool("search-case-sensitive", false, "Make text searches case-sensitive (default: case-insensitive)")

	listCmd.Flags().String("sort", "", "Sort field")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	listCmd.Flags().Int("offset", 0, "Offset for pagination")

	// Bind flags
	for _, flag := range []string{
		"filter", "where", "where-args",
		"filter-eq", "filter-ne", "filter-gt", "filter-lt", "filter-gte", "filter-lte", "filter-like", "filter-in",
		"status", "priority", "status-in", "priority-in",
		"created-after", "created-before", "updated-after", "updated-before",
		"null-fields", "not-null-fields",
		"search", "title-contains", "body-contains", "search-case-sensitive",
		"sort", "limit", "offset"} {
		_ = cli.viperInst.BindPFlag(flag, listCmd.Flags().Lookup(flag))
	}

	cli.rootCmd.AddCommand(listCmd)
}

// addBulkCommands adds bulk operation commands
func (cli *ViperCLI) addBulkCommands() {
	// Update by dimension
	updateByDimCmd := &cobra.Command{
		Use:   "update-by-dimension",
		Short: "Update documents matching dimension filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeBulkUpdateByDimension(cmd)
		},
	}
	updateByDimCmd.Flags().StringSlice("filter", []string{}, "Dimension filters (key=value)")
	updateByDimCmd.Flags().StringSlice("set", []string{}, "Set field values (field=value)")
	cli.rootCmd.AddCommand(updateByDimCmd)

	// Update by UUIDs
	updateByUUIDsCmd := &cobra.Command{
		Use:   "update-by-uuids <uuid1,uuid2,...>",
		Short: "Update documents by list of UUIDs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeBulkUpdateByUUIDs(args[0], cmd)
		},
	}
	updateByUUIDsCmd.Flags().StringSlice("set", []string{}, "Set field values (field=value)")
	cli.rootCmd.AddCommand(updateByUUIDsCmd)

	// Delete operations
	deleteByDimCmd := &cobra.Command{
		Use:   "delete-by-dimension",
		Short: "Delete documents matching dimension filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeBulkDeleteByDimension(cmd)
		},
	}
	deleteByDimCmd.Flags().StringSlice("filter", []string{}, "Dimension filters (key=value)")
	cli.rootCmd.AddCommand(deleteByDimCmd)
}

// addMetadataCommands adds metadata and introspection commands
func (cli *ViperCLI) addMetadataCommands() {
	// Get raw document
	getRawCmd := &cobra.Command{
		Use:   "get-raw <id>",
		Short: "Get raw document data without type unmarshaling",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeGetRaw(args[0])
		},
	}
	cli.rootCmd.AddCommand(getRawCmd)

	// Get dimensions
	getDimensionsCmd := &cobra.Command{
		Use:   "get-dimensions <id>",
		Short: "Get document dimensions map",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeGetDimensions(args[0])
		},
	}
	cli.rootCmd.AddCommand(getDimensionsCmd)

	// Get metadata
	getMetadataCmd := &cobra.Command{
		Use:   "get-metadata <id>",
		Short: "Get document metadata (ID, timestamps, etc.)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeGetMetadata(args[0])
		},
	}
	cli.rootCmd.AddCommand(getMetadataCmd)

	// Resolve UUID
	resolveUUIDCmd := &cobra.Command{
		Use:   "resolve-uuid <simple-id>",
		Short: "Resolve Simple ID to UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeResolveUUID(args[0])
		},
	}
	cli.rootCmd.AddCommand(resolveUUIDCmd)
}

// addAdminCommands adds administrative commands
func (cli *ViperCLI) addAdminCommands() {
	// Debug info
	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Get comprehensive debug information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeDebug()
		},
	}
	cli.rootCmd.AddCommand(debugCmd)

	// Stats
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Get store statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeStats()
		},
	}
	cli.rootCmd.AddCommand(statsCmd)

	// Validate
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate store configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeValidate()
		},
	}
	cli.rootCmd.AddCommand(validateCmd)
}

// addConfigCommand adds the config command to show current configuration
func (cli *ViperCLI) addConfigCommand() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		Long: `Display the current configuration from all sources (flags, env vars, config files).

Examples:
  nanostore config                    # Show all configuration
  nanostore --type Task config        # Show config with type context`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeShowConfig()
		},
	}

	cli.rootCmd.AddCommand(configCmd)
}

// addGenerateConfigCommand adds the generate-config command
func (cli *ViperCLI) addGenerateConfigCommand() {
	genConfigCmd := &cobra.Command{
		Use:   "generate-config <type> [output-file]",
		Short: "Generate a configuration file for a specific type",
		Long: `Generate a configuration file with defaults for the specified type.

Examples:
  nanostore generate-config Task                    # Output to stdout
  nanostore generate-config Task task-config.json  # Save to file`,

		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFile := ""
			if len(args) > 1 {
				outputFile = args[1]
			}
			return cli.executeGenerateConfig(args[0], outputFile)
		},
	}

	cli.rootCmd.AddCommand(genConfigCmd)
}

// Execute runs the CLI
func (cli *ViperCLI) Execute() error {
	return cli.rootCmd.Execute()
}

// GetConfig returns the current Viper configuration
func (cli *ViperCLI) GetConfig(key string) interface{} {
	return cli.viperInst.Get(key)
}

// GetRootCommand returns the root Cobra command for testing
func (cli *ViperCLI) GetRootCommand() *cobra.Command {
	return cli.rootCmd
}
