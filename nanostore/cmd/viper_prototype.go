package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ViperCommandConfig represents a Viper-compatible command configuration
type ViperCommandConfig struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Flags       map[string]ViperFlagSpec `json:"flags"`
	Args        []ViperArgSpec           `json:"args"`
	Category    string                   `json:"category"`
}

// ViperFlagSpec represents a flag configuration for Viper
type ViperFlagSpec struct {
	Type        string      `json:"type"` // "string", "int", "bool", "stringSlice"
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Short       string      `json:"short,omitempty"`
	EnvVar      string      `json:"env_var,omitempty"`
}

// ViperArgSpec represents an argument configuration
type ViperArgSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ViperCliGenerator generates Viper-compatible CLI configurations from type schemas
type ViperCliGenerator struct {
	registry *EnhancedTypeRegistry
}

// NewViperCliGenerator creates a new Viper CLI generator
func NewViperCliGenerator(registry *EnhancedTypeRegistry) *ViperCliGenerator {
	return &ViperCliGenerator{registry: registry}
}

// GenerateCliConfig generates a complete CLI configuration that Viper can use
func (vcg *ViperCliGenerator) GenerateCliConfig() (map[string]ViperCommandConfig, error) {
	commands := make(map[string]ViperCommandConfig)

	// Generate base commands
	baseCommands := vcg.generateBaseCommands()
	for _, cmd := range baseCommands {
		commands[cmd.Name] = cmd
	}

	return commands, nil
}

// generateBaseCommands creates the core CRUD and utility commands
func (vcg *ViperCliGenerator) generateBaseCommands() []ViperCommandConfig {
	return []ViperCommandConfig{
		{
			Name:        "create",
			Description: "Create a new document with title and data",
			Args: []ViperArgSpec{
				{Name: "title", Description: "Document title", Required: true},
			},
			Flags: map[string]ViperFlagSpec{
				"type": {
					Type:        "string",
					Description: "Document type",
					Required:    true,
					EnvVar:      "NANOSTORE_TYPE",
				},
				"db": {
					Type:        "string",
					Description: "Database file path",
					Required:    true,
					EnvVar:      "NANOSTORE_DB",
				},
				"format": {
					Type:        "string",
					Description: "Output format",
					Default:     "table",
					EnvVar:      "NANOSTORE_FORMAT",
				},
				"dry-run": {
					Type:        "bool",
					Description: "Show what would happen without executing",
					Default:     false,
				},
				// Dynamic type-specific flags will be added based on schema
			},
			Category: "Core Operations",
		},
		{
			Name:        "get",
			Description: "Retrieve a document by ID",
			Args: []ViperArgSpec{
				{Name: "id", Description: "Document ID", Required: true},
			},
			Flags: map[string]ViperFlagSpec{
				"type": {
					Type:        "string",
					Description: "Document type",
					Required:    true,
					EnvVar:      "NANOSTORE_TYPE",
				},
				"db": {
					Type:        "string",
					Description: "Database file path",
					Required:    true,
					EnvVar:      "NANOSTORE_DB",
				},
				"format": {
					Type:        "string",
					Description: "Output format",
					Default:     "table",
					EnvVar:      "NANOSTORE_FORMAT",
				},
			},
			Category: "Core Operations",
		},
		{
			Name:        "list",
			Description: "List documents with optional filtering and sorting",
			Flags: map[string]ViperFlagSpec{
				"type": {
					Type:        "string",
					Description: "Document type",
					Required:    true,
					EnvVar:      "NANOSTORE_TYPE",
				},
				"db": {
					Type:        "string",
					Description: "Database file path",
					Required:    true,
					EnvVar:      "NANOSTORE_DB",
				},
				"format": {
					Type:        "string",
					Description: "Output format",
					Default:     "table",
					EnvVar:      "NANOSTORE_FORMAT",
				},
				"filter": {
					Type:        "stringSlice",
					Description: "Dimension filters (key=value)",
					Default:     []string{},
				},
				"sort": {
					Type:        "string",
					Description: "Sort field",
					Default:     "",
				},
				"limit": {
					Type:        "int",
					Description: "Limit number of results",
					Default:     0,
				},
			},
			Category: "Query Operations",
		},
	}
}

// EnhanceCommandWithTypeSchema adds type-specific flags to a command configuration
func (vcg *ViperCliGenerator) EnhanceCommandWithTypeSchema(cmd ViperCommandConfig, typeName string) (ViperCommandConfig, error) {
	typeDef, exists := vcg.registry.GetTypeDefinition(typeName)
	if !exists {
		return cmd, fmt.Errorf("type %s not found", typeName)
	}

	// Add dimension flags
	for dimName, dimSchema := range typeDef.Schema.Dimensions {
		flagName := strings.ReplaceAll(dimName, "_", "-")

		description := fmt.Sprintf("Value for %s dimension", dimName)
		if dimSchema.Type == "enumerated" && len(dimSchema.Values) > 0 {
			description += fmt.Sprintf(" (values: %s)", strings.Join(dimSchema.Values, ", "))
		}

		flagSpec := ViperFlagSpec{
			Type:        "string",
			Description: description,
			EnvVar:      fmt.Sprintf("NANOSTORE_%s", strings.ToUpper(dimName)),
		}

		if dimSchema.Default != "" {
			flagSpec.Default = dimSchema.Default
		}

		cmd.Flags[flagName] = flagSpec
	}

	// Add field flags
	for fieldName, fieldSchema := range typeDef.Schema.Fields {
		flagName := strings.ReplaceAll(fieldName, "_", "-")

		description := fieldSchema.Description
		if description == "" {
			description = fmt.Sprintf("Value for %s field", fieldName)
		}

		flagType := vcg.mapGoTypeToViperType(fieldSchema.Type)

		cmd.Flags[flagName] = ViperFlagSpec{
			Type:        flagType,
			Description: description,
			EnvVar:      fmt.Sprintf("NANOSTORE_%s", strings.ToUpper(fieldName)),
		}
	}

	return cmd, nil
}

// mapGoTypeToViperType converts Go type strings to Viper flag types
func (vcg *ViperCliGenerator) mapGoTypeToViperType(goType string) string {
	switch goType {
	case "string", "*string":
		return "string"
	case "int", "*int", "int64", "*int64":
		return "int"
	case "float64", "*float64":
		return "float64"
	case "bool", "*bool":
		return "bool"
	case "[]string":
		return "stringSlice"
	default:
		return "string" // Default to string for complex types
	}
}

// CreateViperCommand creates a Cobra command that uses Viper for configuration
func (vcg *ViperCliGenerator) CreateViperCommand(config ViperCommandConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   config.Name,
		Short: config.Description,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to Viper
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return vcg.executeViperCommand(config, args)
		},
	}

	// Add flags from configuration
	for flagName, flagSpec := range config.Flags {
		vcg.addViperFlag(cmd, flagName, flagSpec)
	}

	return cmd
}

// addViperFlag adds a flag to a Cobra command based on Viper flag spec
func (vcg *ViperCliGenerator) addViperFlag(cmd *cobra.Command, name string, spec ViperFlagSpec) {
	switch spec.Type {
	case "string":
		defaultVal := ""
		if spec.Default != nil {
			defaultVal = spec.Default.(string)
		}
		if spec.Short != "" {
			cmd.Flags().StringP(name, spec.Short, defaultVal, spec.Description)
		} else {
			cmd.Flags().String(name, defaultVal, spec.Description)
		}

	case "int":
		defaultVal := 0
		if spec.Default != nil {
			defaultVal = spec.Default.(int)
		}
		if spec.Short != "" {
			cmd.Flags().IntP(name, spec.Short, defaultVal, spec.Description)
		} else {
			cmd.Flags().Int(name, defaultVal, spec.Description)
		}

	case "bool":
		defaultVal := false
		if spec.Default != nil {
			defaultVal = spec.Default.(bool)
		}
		if spec.Short != "" {
			cmd.Flags().BoolP(name, spec.Short, defaultVal, spec.Description)
		} else {
			cmd.Flags().Bool(name, defaultVal, spec.Description)
		}

	case "stringSlice":
		defaultVal := []string{}
		if spec.Default != nil {
			defaultVal = spec.Default.([]string)
		}
		if spec.Short != "" {
			cmd.Flags().StringSliceP(name, spec.Short, defaultVal, spec.Description)
		} else {
			cmd.Flags().StringSlice(name, defaultVal, spec.Description)
		}
	}

	// Mark as required if specified
	if spec.Required {
		_ = cmd.MarkFlagRequired(name)
	}

	// Set up environment variable binding
	if spec.EnvVar != "" {
		_ = viper.BindEnv(name, spec.EnvVar)
	}
}

// executeViperCommand executes a command using Viper for configuration
func (vcg *ViperCliGenerator) executeViperCommand(config ViperCommandConfig, args []string) error {
	// Get all configuration values from Viper (combines flags, env vars, config files)
	values := viper.AllSettings()

	// Show what we would execute
	fmt.Printf("Executing %s command with Viper configuration:\n", config.Name)

	configJSON, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Printf("Configuration: %s\n", string(configJSON))

	if len(args) > 0 {
		fmt.Printf("Arguments: %v\n", args)
	}

	// TODO: Use the executor to actually run the command
	return nil
}

// GenerateConfigFile generates a configuration file that can be used with Viper
func (vcg *ViperCliGenerator) GenerateConfigFile(typeName string) (string, error) {
	config := map[string]interface{}{
		"type":   typeName,
		"format": "json",
		"db":     fmt.Sprintf("%s.db", strings.ToLower(typeName)),
	}

	// Add type-specific defaults
	if typeDef, exists := vcg.registry.GetTypeDefinition(typeName); exists {
		for dimName, dimSchema := range typeDef.Schema.Dimensions {
			if dimSchema.Default != "" {
				config[dimName] = dimSchema.Default
			}
		}
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(configJSON), nil
}

// Example of how this would be used:
func ExampleViperUsage() {
	// Create registry and load types
	registry := NewEnhancedTypeRegistry()
	_ = registry.LoadBuiltinTypes()

	// Create Viper generator
	generator := NewViperCliGenerator(registry)

	// Generate base CLI config
	commands, _ := generator.GenerateCliConfig()

	// Enhance commands with type-specific configuration for Task type
	createCmd := commands["create"]
	enhancedCreateCmd, _ := generator.EnhanceCommandWithTypeSchema(createCmd, "Task")

	// Create the actual Cobra command
	cobraCmd := generator.CreateViperCommand(enhancedCreateCmd)

	// Set up Viper to read from config files
	viper.SetConfigName("nanostore")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.nanostore")

	// Read config file if it exists
	_ = viper.ReadInConfig()

	fmt.Printf("Command created: %s\n", cobraCmd.Name())
}
