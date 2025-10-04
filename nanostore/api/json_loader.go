package api

import (
	"encoding/json"
	"fmt"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/types"
)

// LoadConfigFromJSON parses a JSON configuration and validates it for use with nanostore
func LoadConfigFromJSON(jsonData []byte) (types.Config, error) {
	var config types.Config

	// Parse JSON into Config struct
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return config, fmt.Errorf("failed to parse JSON configuration: %w", err)
	}

	// Validate the configuration using existing validation logic
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		return config, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// LoadConfigFromJSONWithDetails parses JSON configuration and returns detailed validation results
func LoadConfigFromJSONWithDetails(jsonData []byte) (types.Config, []ValidationResult, error) {
	var config types.Config
	var results []ValidationResult

	// Parse JSON into Config struct
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return config, results, fmt.Errorf("failed to parse JSON configuration: %w", err)
	}

	// Validate the configuration and collect detailed results
	if err := validation.Validate(config.GetDimensionSet()); err != nil {
		results = append(results, ValidationResult{
			Type:    "error",
			Message: err.Error(),
		})
		return config, results, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Add success message
	results = append(results, ValidationResult{
		Type:    "success",
		Message: fmt.Sprintf("Configuration is valid with %d dimensions", len(config.Dimensions)),
	})

	return config, results, nil
}

// ValidationResult represents a validation result with type and message
type ValidationResult struct {
	Type    string `json:"type"`    // "error", "warning", "success"
	Message string `json:"message"` // Descriptive message
}

// CreateStoreFromJSON creates a new nanostore database from JSON configuration
func CreateStoreFromJSON(filePath string, jsonData []byte) (nanostore.Store, error) {
	// Load and validate configuration
	config, err := LoadConfigFromJSON(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create the store
	store, err := nanostore.New(filePath, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return store, nil
}

// ValidateJSONConfig validates a JSON configuration without creating a store
func ValidateJSONConfig(jsonData []byte) error {
	_, err := LoadConfigFromJSON(jsonData)
	return err
}

// ValidateJSONConfigWithDetails validates JSON config and returns detailed results
func ValidateJSONConfigWithDetails(jsonData []byte) []ValidationResult {
	_, results, err := LoadConfigFromJSONWithDetails(jsonData)
	if err != nil && len(results) == 0 {
		// If we have an error but no results, add the error as a result
		results = append(results, ValidationResult{
			Type:    "error",
			Message: err.Error(),
		})
	}
	return results
}
