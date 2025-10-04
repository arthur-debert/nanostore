package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestViperIntegration demonstrates how Viper could work with our schema-driven CLI
func TestViperIntegration(t *testing.T) {
	// Create a root command
	rootCmd := &cobra.Command{
		Use:   "nanostore-viper",
		Short: "Nanostore CLI with Viper integration test",
	}

	// Create registry and generator
	registry := NewEnhancedTypeRegistry()
	_ = registry.LoadBuiltinTypes()
	generator := NewViperCliGenerator(registry)

	// Generate Task-specific create command
	baseCreateCmd := ViperCommandConfig{
		Name:        "create",
		Description: "Create a new document",
		Args: []ViperArgSpec{
			{Name: "title", Description: "Document title", Required: true},
		},
		Flags: map[string]ViperFlagSpec{
			"type": {
				Type:        "string",
				Description: "Document type",
				Default:     "Task",
				EnvVar:      "NANOSTORE_TYPE",
			},
			"db": {
				Type:        "string",
				Description: "Database file path",
				Default:     "test.db",
				EnvVar:      "NANOSTORE_DB",
			},
			"format": {
				Type:        "string",
				Description: "Output format",
				Default:     "json",
				EnvVar:      "NANOSTORE_FORMAT",
			},
		},
		Category: "Core Operations",
	}

	// Enhance with Task-specific fields
	enhancedCmd, err := generator.EnhanceCommandWithTypeSchema(baseCreateCmd, "Task")
	if err != nil {
		fmt.Printf("Error enhancing command: %v\n", err)
		return
	}

	// Create the Cobra command
	createCmd := generator.CreateViperCommand(enhancedCmd)
	rootCmd.AddCommand(createCmd)

	// Set up Viper configuration
	viper.SetConfigName("nanostore")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	// Create a sample config file for testing
	configContent, err := generator.GenerateConfigFile("Task")
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
		return
	}

	fmt.Println("Generated config file content:")
	fmt.Println(configContent)

	// Write config to file for testing
	if err := os.WriteFile("nanostore.json", []byte(configContent), 0644); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		return
	}

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config: %v\n", err)
	} else {
		fmt.Printf("Loaded config from: %s\n", viper.ConfigFileUsed())
	}

	// Test command execution
	fmt.Println("\nTesting command execution:")

	// Simulate CLI arguments
	os.Args = []string{"nanostore-viper", "create", "Test Task", "--status", "active", "--priority", "high"}

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error executing command: %v\n", err)
	}

	// Clean up
	_ = os.Remove("nanostore.json")
}

// CreateViperRootCommand creates a Viper-enabled root command for the actual CLI
func CreateViperRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "nanostore",
		Short: "Nanostore CLI with Viper configuration management",
		Long: `Nanostore CLI with Viper integration for advanced configuration management.

Supports configuration via:
- Command line flags
- Environment variables  
- Configuration files (JSON, YAML, TOML)
- Default values from type schemas

Configuration files are loaded from:
- ./nanostore.json
- ~/.nanostore/config.json
- /etc/nanostore/config.json`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set up Viper configuration for all commands
			return initViperConfig()
		},
	}

	// Set up global configuration
	setupGlobalViperFlags(rootCmd)

	return rootCmd
}

// initViperConfig initializes Viper configuration
func initViperConfig() error {
	// Set config file locations and types
	viper.SetConfigName("nanostore")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.nanostore")
	viper.AddConfigPath("/etc/nanostore")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NANOSTORE")

	// Read config file (ignore if not found)
	_ = viper.ReadInConfig()

	return nil
}

// setupGlobalViperFlags adds global flags that all commands use
func setupGlobalViperFlags(cmd *cobra.Command) {
	// Global flags
	cmd.PersistentFlags().StringP("type", "t", "", "Document type")
	cmd.PersistentFlags().StringP("db", "d", "", "Database file path")
	cmd.PersistentFlags().StringP("format", "f", "table", "Output format (table|json|yaml|csv)")
	cmd.PersistentFlags().Bool("no-color", false, "Disable colors")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress headers")
	cmd.PersistentFlags().Bool("dry-run", false, "Show what would happen without executing")
	cmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	// Bind flags to Viper
	_ = viper.BindPFlag("type", cmd.PersistentFlags().Lookup("type"))
	_ = viper.BindPFlag("db", cmd.PersistentFlags().Lookup("db"))
	_ = viper.BindPFlag("format", cmd.PersistentFlags().Lookup("format"))
	_ = viper.BindPFlag("no-color", cmd.PersistentFlags().Lookup("no-color"))
	_ = viper.BindPFlag("quiet", cmd.PersistentFlags().Lookup("quiet"))
	_ = viper.BindPFlag("dry-run", cmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))

	// Bind environment variables
	_ = viper.BindEnv("type", "NANOSTORE_TYPE")
	_ = viper.BindEnv("db", "NANOSTORE_DB")
	_ = viper.BindEnv("format", "NANOSTORE_FORMAT")
}
