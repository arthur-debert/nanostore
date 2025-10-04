package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/spf13/cobra"
)

// Command represents a generic CLI command mapped to a Go API method
type Command struct {
	Name        string          // CLI command name (e.g., "create", "get")
	Method      string          // Go method name (e.g., "Create", "Get")
	Description string          // Short description for help
	Args        []ArgSpec       // Required arguments
	Flags       []FlagSpec      // Optional flags
	Returns     ReturnSpec      // Return type/format
	Category    CommandCategory // Command category for organization
}

// ArgSpec defines a required command argument
type ArgSpec struct {
	Name        string       // Argument name
	Type        reflect.Type // Expected Go type
	Description string       // Help description
	Required    bool         // Whether argument is required
}

// FlagSpec defines an optional command flag
type FlagSpec struct {
	Name        string       // Flag name (e.g., "cascade")
	Short       string       // Short flag (e.g., "c")
	Type        reflect.Type // Expected Go type
	Description string       // Help description
	Default     interface{}  // Default value
}

// ReturnSpec defines the command return format
type ReturnSpec struct {
	Type        reflect.Type // Return type
	Description string       // Description of what's returned
	IsList      bool         // Whether return is a list/array
}

// CommandCategory categorizes commands for better CLI organization
type CommandCategory int

const (
	CategoryCRUD CommandCategory = iota
	CategoryBulk
	CategoryQuery
	CategoryMetadata
	CategoryConfig
	CategoryAdmin
)

func (c CommandCategory) String() string {
	switch c {
	case CategoryCRUD:
		return "Core Operations"
	case CategoryBulk:
		return "Bulk Operations"
	case CategoryQuery:
		return "Query Operations"
	case CategoryMetadata:
		return "Metadata & Introspection"
	case CategoryConfig:
		return "Configuration & Debug"
	case CategoryAdmin:
		return "Administrative Operations"
	default:
		return "Other"
	}
}

// TypeRegistry manages type definitions for CLI operations
type TypeRegistry struct {
	types map[string]reflect.Type
}

// NewTypeRegistry creates a new type registry
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[string]reflect.Type),
	}
}

// RegisterType registers a type for CLI use
func (tr *TypeRegistry) RegisterType(name string, typ reflect.Type) {
	tr.types[name] = typ
}

// GetType retrieves a registered type
func (tr *TypeRegistry) GetType(name string) (reflect.Type, bool) {
	typ, exists := tr.types[name]
	return typ, exists
}

// ListTypes returns all registered type names
func (tr *TypeRegistry) ListTypes() []string {
	var names []string
	for name := range tr.types {
		names = append(names, name)
	}
	return names
}

// CommandGenerator generates Cobra commands from Go API methods using reflection
type CommandGenerator struct {
	registry *EnhancedTypeRegistry
}

// NewCommandGenerator creates a new command generator
func NewCommandGenerator() *CommandGenerator {
	registry := NewEnhancedTypeRegistry()

	// Load built-in types
	if err := registry.LoadBuiltinTypes(); err != nil {
		fmt.Printf("Warning: failed to load built-in types: %v\n", err)
	}

	return &CommandGenerator{
		registry: registry,
	}
}

// GenerateCommands analyzes the Store[T] interface and generates CLI commands
func (cg *CommandGenerator) GenerateCommands() []Command {
	var commands []Command

	// Define the complete API mapping based on the Store interface
	apiMethods := []Command{
		// Core CRUD Operations
		{
			Name:        "create",
			Method:      "Create",
			Description: "Create a new document with title and data",
			Args: []ArgSpec{
				{Name: "title", Type: reflect.TypeOf(""), Description: "Document title", Required: true},
			},
			Flags: []FlagSpec{
				// Dynamic flags based on struct fields will be added
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(""), Description: "Simple ID of created document"},
			Category: CategoryCRUD,
		},
		{
			Name:        "get",
			Method:      "Get",
			Description: "Retrieve a document by ID",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID (Simple ID or UUID)", Required: true},
			},
			Returns:  ReturnSpec{Type: nil, Description: "Document data", IsList: false}, // Type set dynamically
			Category: CategoryCRUD,
		},
		{
			Name:        "update",
			Method:      "Update",
			Description: "Update a document with new data",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID to update", Required: true},
			},
			Flags: []FlagSpec{
				// Dynamic flags based on struct fields will be added
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents updated"},
			Category: CategoryCRUD,
		},
		{
			Name:        "delete",
			Method:      "Delete",
			Description: "Delete a document by ID",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID to delete", Required: true},
			},
			Flags: []FlagSpec{
				{Name: "cascade", Short: "c", Type: reflect.TypeOf(false), Description: "Delete children recursively", Default: false},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(nil), Description: "Success confirmation"},
			Category: CategoryCRUD,
		},

		// Query Operations
		{
			Name:        "list",
			Method:      "List",
			Description: "List documents with optional filtering and sorting",
			Flags: []FlagSpec{
				{Name: "filter", Type: reflect.TypeOf([]string{}), Description: "Dimension filters (key=value)", Default: []string{}},
				{Name: "sort", Type: reflect.TypeOf(""), Description: "Sort field", Default: ""},
				{Name: "limit", Type: reflect.TypeOf(0), Description: "Limit number of results", Default: 0},
			},
			Returns:  ReturnSpec{Type: nil, Description: "List of matching documents", IsList: true},
			Category: CategoryQuery,
		},

		// Bulk Operations
		{
			Name:        "update-by-dimension",
			Method:      "UpdateByDimension",
			Description: "Update documents matching dimension filters",
			Flags: []FlagSpec{
				{Name: "filter", Type: reflect.TypeOf([]string{}), Description: "Dimension filters (key=value)", Default: []string{}},
				// Dynamic flags for update fields
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents updated"},
			Category: CategoryBulk,
		},
		{
			Name:        "update-where",
			Method:      "UpdateWhere",
			Description: "Update documents matching WHERE clause",
			Args: []ArgSpec{
				{Name: "where", Type: reflect.TypeOf(""), Description: "SQL WHERE clause", Required: true},
			},
			Flags: []FlagSpec{
				{Name: "args", Type: reflect.TypeOf([]string{}), Description: "WHERE clause arguments", Default: []string{}},
				// Dynamic flags for update fields
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents updated"},
			Category: CategoryBulk,
		},
		{
			Name:        "update-by-uuids",
			Method:      "UpdateByUUIDs",
			Description: "Update documents by list of UUIDs",
			Args: []ArgSpec{
				{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
			},
			Flags: []FlagSpec{
				// Dynamic flags for update fields
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents updated"},
			Category: CategoryBulk,
		},
		{
			Name:        "delete-by-dimension",
			Method:      "DeleteByDimension",
			Description: "Delete documents matching dimension filters",
			Flags: []FlagSpec{
				{Name: "filter", Type: reflect.TypeOf([]string{}), Description: "Dimension filters (key=value)", Default: []string{}},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents deleted"},
			Category: CategoryBulk,
		},
		{
			Name:        "delete-where",
			Method:      "DeleteWhere",
			Description: "Delete documents matching WHERE clause",
			Args: []ArgSpec{
				{Name: "where", Type: reflect.TypeOf(""), Description: "SQL WHERE clause", Required: true},
			},
			Flags: []FlagSpec{
				{Name: "args", Type: reflect.TypeOf([]string{}), Description: "WHERE clause arguments", Default: []string{}},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents deleted"},
			Category: CategoryBulk,
		},
		{
			Name:        "delete-by-uuids",
			Method:      "DeleteByUUIDs",
			Description: "Delete documents by list of UUIDs",
			Args: []ArgSpec{
				{Name: "uuids", Type: reflect.TypeOf([]string{}), Description: "Comma-separated UUIDs", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(0), Description: "Number of documents deleted"},
			Category: CategoryBulk,
		},

		// Metadata Operations
		{
			Name:        "get-raw",
			Method:      "GetRaw",
			Description: "Get raw document data without type unmarshaling",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(&nanostore.Document{}), Description: "Raw document data"},
			Category: CategoryMetadata,
		},
		{
			Name:        "get-dimensions",
			Method:      "GetDimensions",
			Description: "Get document dimensions map",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(map[string]interface{}{}), Description: "Dimensions map"},
			Category: CategoryMetadata,
		},
		{
			Name:        "get-metadata",
			Method:      "GetMetadata",
			Description: "Get document metadata (ID, timestamps, etc.)",
			Args: []ArgSpec{
				{Name: "id", Type: reflect.TypeOf(""), Description: "Document ID", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(&api.DocumentMetadata{}), Description: "Document metadata"},
			Category: CategoryMetadata,
		},
		{
			Name:        "resolve-uuid",
			Method:      "ResolveUUID",
			Description: "Resolve Simple ID to UUID",
			Args: []ArgSpec{
				{Name: "simple-id", Type: reflect.TypeOf(""), Description: "Simple ID to resolve", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(""), Description: "UUID"},
			Category: CategoryMetadata,
		},

		// Configuration & Debug Operations
		{
			Name:        "config",
			Method:      "GetDimensionConfig",
			Description: "Get store dimension configuration",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&nanostore.Config{}), Description: "Dimension configuration"},
			Category:    CategoryConfig,
		},
		{
			Name:        "debug",
			Method:      "GetDebugInfo",
			Description: "Get comprehensive debug information",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&api.DebugInfo{}), Description: "Debug information"},
			Category:    CategoryConfig,
		},
		{
			Name:        "stats",
			Method:      "GetStoreStats",
			Description: "Get store statistics",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&api.StoreStats{}), Description: "Store statistics"},
			Category:    CategoryConfig,
		},
		{
			Name:        "validate",
			Method:      "ValidateConfiguration",
			Description: "Validate store configuration",
			Returns:     ReturnSpec{Type: reflect.TypeOf(nil), Description: "Validation result"},
			Category:    CategoryConfig,
		},
		{
			Name:        "integrity",
			Method:      "ValidateStoreIntegrity",
			Description: "Validate store data integrity",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&api.IntegrityReport{}), Description: "Integrity report"},
			Category:    CategoryConfig,
		},
		{
			Name:        "field-stats",
			Method:      "GetFieldUsageStats",
			Description: "Get field usage statistics",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&api.FieldUsageStats{}), Description: "Field usage statistics"},
			Category:    CategoryConfig,
		},
		{
			Name:        "schema",
			Method:      "GetTypeSchema",
			Description: "Get type schema information",
			Returns:     ReturnSpec{Type: reflect.TypeOf(&api.TypeSchema{}), Description: "Type schema"},
			Category:    CategoryConfig,
		},

		// Administrative Operations
		{
			Name:        "add-raw",
			Method:      "AddRaw",
			Description: "Add raw document with explicit dimensions",
			Args: []ArgSpec{
				{Name: "title", Type: reflect.TypeOf(""), Description: "Document title", Required: true},
				{Name: "dimensions", Type: reflect.TypeOf(""), Description: "Dimensions JSON", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(""), Description: "Simple ID of created document"},
			Category: CategoryAdmin,
		},
		{
			Name:        "add-dimension-value",
			Method:      "AddDimensionValue",
			Description: "Add new value to dimension configuration",
			Args: []ArgSpec{
				{Name: "dimension", Type: reflect.TypeOf(""), Description: "Dimension name", Required: true},
				{Name: "value", Type: reflect.TypeOf(""), Description: "New value", Required: true},
				{Name: "prefix", Type: reflect.TypeOf(""), Description: "Value prefix", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(nil), Description: "Success confirmation"},
			Category: CategoryAdmin,
		},
		{
			Name:        "modify-dimension-default",
			Method:      "ModifyDimensionDefault",
			Description: "Modify default value for dimension",
			Args: []ArgSpec{
				{Name: "dimension", Type: reflect.TypeOf(""), Description: "Dimension name", Required: true},
				{Name: "default", Type: reflect.TypeOf(""), Description: "New default value", Required: true},
			},
			Returns:  ReturnSpec{Type: reflect.TypeOf(nil), Description: "Success confirmation"},
			Category: CategoryAdmin,
		},
	}

	commands = append(commands, apiMethods...)
	return commands
}

// ToCobraCommand converts a Command to a cobra.Command with dynamic type handling
func (cmd *Command) ToCobraCommand(generator *CommandGenerator) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   cmd.GenerateUsage(),
		Short: cmd.Description,
		Long:  cmd.GenerateLongDescription(),
		RunE:  cmd.GenerateRunFunc(generator),
	}

	// Add basic flags
	for _, flag := range cmd.Flags {
		cmd.addFlagToCommand(cobraCmd, flag)
	}

	// For data manipulation commands, add common type-agnostic flags
	if cmd.Name == "create" || cmd.Name == "update" || strings.HasPrefix(cmd.Name, "update-") {
		// Add generic data flags that work with any type
		cobraCmd.Flags().String("status", "", "Status dimension value")
		cobraCmd.Flags().String("priority", "", "Priority dimension value")
		cobraCmd.Flags().String("category", "", "Category dimension value")
		cobraCmd.Flags().String("parent-id", "", "Parent ID for hierarchical relationships")
		cobraCmd.Flags().String("description", "", "Description field")
		cobraCmd.Flags().String("assignee", "", "Assignee field")
		cobraCmd.Flags().String("tags", "", "Tags field")
		cobraCmd.Flags().String("content", "", "Content field")
	}

	return cobraCmd
}

// GenerateUsage creates the usage string for the command
func (cmd *Command) GenerateUsage() string {
	usage := cmd.Name

	// Add required arguments
	for _, arg := range cmd.Args {
		if arg.Required {
			usage += fmt.Sprintf(" <%s>", arg.Name)
		} else {
			usage += fmt.Sprintf(" [%s]", arg.Name)
		}
	}

	return usage
}

// GenerateLongDescription creates detailed help text for the command
func (cmd *Command) GenerateLongDescription() string {
	desc := cmd.Description + "\n\n"

	if len(cmd.Args) > 0 {
		desc += "Arguments:\n"
		for _, arg := range cmd.Args {
			required := ""
			if arg.Required {
				required = " (required)"
			}
			desc += fmt.Sprintf("  %s: %s%s\n", arg.Name, arg.Description, required)
		}
		desc += "\n"
	}

	if len(cmd.Flags) > 0 {
		desc += "Flags:\n"
		for _, flag := range cmd.Flags {
			desc += fmt.Sprintf("  --%s: %s\n", flag.Name, flag.Description)
		}
		desc += "\n"
	}

	desc += fmt.Sprintf("Returns: %s\n", cmd.Returns.Description)
	desc += fmt.Sprintf("Category: %s\n", cmd.Category.String())

	return desc
}

// GenerateRunFunc creates the execution function for the command
func (cmd *Command) GenerateRunFunc(generator *CommandGenerator) func(*cobra.Command, []string) error {
	return func(cobraCmd *cobra.Command, args []string) error {
		// Validate arguments
		if len(args) < len(cmd.Args) {
			var requiredArgs []string
			for _, arg := range cmd.Args {
				if arg.Required {
					requiredArgs = append(requiredArgs, arg.Name)
				}
			}
			return fmt.Errorf("missing required arguments: %v", requiredArgs)
		}

		// Use the method executor to handle the command
		executor := NewMethodExecutor(generator.registry)
		return executor.ExecuteCommand(cmd, cobraCmd, args)
	}
}

// addFlagToCommand adds a flag specification to a cobra command
func (cmd *Command) addFlagToCommand(cobraCmd *cobra.Command, flag FlagSpec) {
	switch flag.Type.Kind() {
	case reflect.String:
		defaultVal := ""
		if flag.Default != nil {
			defaultVal = flag.Default.(string)
		}
		if flag.Short != "" {
			cobraCmd.Flags().StringP(flag.Name, flag.Short, defaultVal, flag.Description)
		} else {
			cobraCmd.Flags().String(flag.Name, defaultVal, flag.Description)
		}

	case reflect.Bool:
		defaultVal := false
		if flag.Default != nil {
			defaultVal = flag.Default.(bool)
		}
		if flag.Short != "" {
			cobraCmd.Flags().BoolP(flag.Name, flag.Short, defaultVal, flag.Description)
		} else {
			cobraCmd.Flags().Bool(flag.Name, defaultVal, flag.Description)
		}

	case reflect.Int:
		defaultVal := 0
		if flag.Default != nil {
			defaultVal = flag.Default.(int)
		}
		if flag.Short != "" {
			cobraCmd.Flags().IntP(flag.Name, flag.Short, defaultVal, flag.Description)
		} else {
			cobraCmd.Flags().Int(flag.Name, defaultVal, flag.Description)
		}

	case reflect.Slice:
		// Handle string slices
		if flag.Type.Elem().Kind() == reflect.String {
			defaultVal := []string{}
			if flag.Default != nil {
				defaultVal = flag.Default.([]string)
			}
			if flag.Short != "" {
				cobraCmd.Flags().StringSliceP(flag.Name, flag.Short, defaultVal, flag.Description)
			} else {
				cobraCmd.Flags().StringSlice(flag.Name, defaultVal, flag.Description)
			}
		}

	default:
		// For other types, treat as string and handle conversion later
		if flag.Short != "" {
			cobraCmd.Flags().StringP(flag.Name, flag.Short, "", flag.Description)
		} else {
			cobraCmd.Flags().String(flag.Name, "", flag.Description)
		}
	}
}

// OutputFormatter handles formatting command results for different output formats
type OutputFormatter struct {
	format string
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter(format string) *OutputFormatter {
	return &OutputFormatter{format: format}
}

// Format formats the given data according to the specified format
func (of *OutputFormatter) Format(data interface{}) (string, error) {
	switch of.format {
	case "json":
		return of.formatJSON(data)
	case "yaml":
		return of.formatYAML(data)
	case "csv":
		return of.formatCSV(data)
	case "table":
		return of.formatTable(data)
	default:
		return of.formatTable(data) // Default to table format
	}
}

func (of *OutputFormatter) formatJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (of *OutputFormatter) formatYAML(data interface{}) (string, error) {
	// TODO: Implement YAML formatting (requires yaml package)
	return of.formatJSON(data)
}

func (of *OutputFormatter) formatCSV(data interface{}) (string, error) {
	// TODO: Implement CSV formatting
	return of.formatJSON(data)
}

func (of *OutputFormatter) formatTable(data interface{}) (string, error) {
	// TODO: Implement table formatting
	return of.formatJSON(data)
}
