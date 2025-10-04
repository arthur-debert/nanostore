package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func runViperDemo() {
	fmt.Println("=== Viper Integration Demo ===")

	// Create registry and generator
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		fmt.Printf("Error loading types: %v\n", err)
		return
	}

	generator := NewViperCliGenerator(registry)

	// Show available types
	types := registry.ListTypes()
	fmt.Printf("Available types: %v\n", types)

	// Generate config for Task type
	configContent, err := generator.GenerateConfigFile("Task")
	if err != nil {
		fmt.Printf("Error generating config: %v\n", err)
		return
	}

	fmt.Println("\nGenerated config file for Task type:")
	fmt.Println(configContent)

	// Create enhanced command config
	baseCmd := ViperCommandConfig{
		Name:        "create",
		Description: "Create a new Task document",
		Args: []ViperArgSpec{
			{Name: "title", Description: "Task title", Required: true},
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
				Default:     "tasks.db",
				EnvVar:      "NANOSTORE_DB",
			},
		},
	}

	// Enhance with Task-specific flags
	enhancedCmd, err := generator.EnhanceCommandWithTypeSchema(baseCmd, "Task")
	if err != nil {
		fmt.Printf("Error enhancing command: %v\n", err)
		return
	}

	fmt.Println("\nEnhanced command configuration:")
	cmdJSON := fmt.Sprintf("%+v", enhancedCmd.Flags)
	fmt.Printf("Command flags: %s\n", cmdJSON)

	// Create Cobra command
	cobraCmd := generator.CreateViperCommand(enhancedCmd)

	// Set up Viper
	viper.SetConfigType("json")
	viper.SetConfigFile("nanostore.json")

	// Write and read config file
	if err := os.WriteFile("nanostore.json", []byte(configContent), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
	} else {
		fmt.Println("\nConfig file written to nanostore.json")

		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Error reading config: %v\n", err)
		} else {
			fmt.Printf("Config loaded from: %s\n", viper.ConfigFileUsed())
		}
	}

	// Simulate command usage
	fmt.Println("\nSimulating command execution:")
	fmt.Printf("Command: %s\n", cobraCmd.Use)
	fmt.Printf("Description: %s\n", cobraCmd.Short)

	// Show what flags are available
	fmt.Println("\nAvailable flags:")
	cobraCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		fmt.Printf("  --%s: %s (default: %s)\n", flag.Name, flag.Usage, flag.DefValue)
	})

	// Clean up
	_ = os.Remove("nanostore.json")

	fmt.Println("\n=== Demo Complete ===")
}
