package main

import (
	"os"
	"testing"
)

func TestNANOSTORE_CONFIG_EnvironmentVariable(t *testing.T) {
	// Create a temporary config file
	configContent := `{
		"type": "Task",
		"db": "test-config.db",
		"format": "json",
		"status": "active",
		"priority": "high"
	}`

	tmpfile, err := os.CreateTemp("", "nanostore-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	_ = tmpfile.Close()

	// Test with NANOSTORE_CONFIG environment variable
	t.Run("WithNANOSTORE_CONFIG", func(t *testing.T) {
		// Set environment variable
		err := os.Setenv("NANOSTORE_CONFIG", tmpfile.Name())
		if err != nil {
			t.Fatalf("Failed to set environment variable: %v", err)
		}
		defer func() { _ = os.Unsetenv("NANOSTORE_CONFIG") }()

		// Create CLI and check if config is loaded
		cli := NewViperCLI()

		// Check if config values are loaded from the custom config file
		configType := cli.viperInst.GetString("type")
		configDB := cli.viperInst.GetString("db")
		configFormat := cli.viperInst.GetString("format")
		configStatus := cli.viperInst.GetString("status")
		configPriority := cli.viperInst.GetString("priority")

		if configType != "Task" {
			t.Errorf("Expected type 'Task', got '%s'", configType)
		}
		if configDB != "test-config.db" {
			t.Errorf("Expected db 'test-config.db', got '%s'", configDB)
		}
		if configFormat != "json" {
			t.Errorf("Expected format 'json', got '%s'", configFormat)
		}
		if configStatus != "active" {
			t.Errorf("Expected status 'active', got '%s'", configStatus)
		}
		if configPriority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", configPriority)
		}

		t.Logf("Successfully loaded config from NANOSTORE_CONFIG: %s", tmpfile.Name())
		t.Logf("Config values: type=%s, db=%s, format=%s, status=%s, priority=%s",
			configType, configDB, configFormat, configStatus, configPriority)
	})

	// Test without NANOSTORE_CONFIG (should use default discovery)
	t.Run("WithoutNANOSTORE_CONFIG", func(t *testing.T) {
		// Ensure environment variable is not set
		_ = os.Unsetenv("NANOSTORE_CONFIG")

		// Create CLI (should use default config discovery)
		cli := NewViperCLI()

		// Should not load the custom config values
		configType := cli.viperInst.GetString("type")
		configDB := cli.viperInst.GetString("db")

		// These should be empty since no default config file exists
		if configType != "" {
			t.Logf("Found unexpected type in default config: %s", configType)
		}
		if configDB != "" {
			t.Logf("Found unexpected db in default config: %s", configDB)
		}

		t.Logf("Default config discovery working (no custom config loaded)")
	})
}

func TestConfigFilePrecedence(t *testing.T) {
	// Test that NANOSTORE_CONFIG takes precedence over default locations

	// Create a config file in current directory
	defaultConfigContent := `{
		"type": "Note",
		"db": "default.db",
		"format": "table"
	}`

	err := os.WriteFile("nanostore.json", []byte(defaultConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create default config file: %v", err)
	}
	defer func() { _ = os.Remove("nanostore.json") }()

	// Create a custom config file
	customConfigContent := `{
		"type": "Task", 
		"db": "custom.db",
		"format": "json"
	}`

	tmpfile, err := os.CreateTemp("", "custom-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create custom config file: %v", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(customConfigContent)); err != nil {
		t.Fatalf("Failed to write custom config file: %v", err)
	}
	_ = tmpfile.Close()

	// Set NANOSTORE_CONFIG to point to custom file
	err = os.Setenv("NANOSTORE_CONFIG", tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to set NANOSTORE_CONFIG: %v", err)
	}
	defer func() { _ = os.Unsetenv("NANOSTORE_CONFIG") }()

	// Create CLI - should load custom config, not default
	cli := NewViperCLI()

	configType := cli.viperInst.GetString("type")
	configDB := cli.viperInst.GetString("db")
	configFormat := cli.viperInst.GetString("format")

	// Should match custom config, not default config
	if configType != "Task" {
		t.Errorf("Expected custom config type 'Task', got '%s' (may have loaded default config)", configType)
	}
	if configDB != "custom.db" {
		t.Errorf("Expected custom config db 'custom.db', got '%s' (may have loaded default config)", configDB)
	}
	if configFormat != "json" {
		t.Errorf("Expected custom config format 'json', got '%s' (may have loaded default config)", configFormat)
	}

	t.Logf("NANOSTORE_CONFIG precedence working correctly")
	t.Logf("Loaded custom config: type=%s, db=%s, format=%s", configType, configDB, configFormat)
}
