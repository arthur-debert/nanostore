package api_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/api"
	"github.com/arthur-debert/nanostore/types"
)

func TestLoadConfigFromJSON(t *testing.T) {
	t.Run("Valid enumerated dimension configuration", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "active", "done"],
					"default_value": "pending",
					"prefixes": {
						"done": "d"
					}
				}
			]
		}`

		config, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err != nil {
			t.Fatalf("Failed to load valid config: %v", err)
		}

		if len(config.Dimensions) != 1 {
			t.Errorf("Expected 1 dimension, got %d", len(config.Dimensions))
		}

		dim := config.Dimensions[0]
		if dim.Name != "status" {
			t.Errorf("Expected dimension name 'status', got '%s'", dim.Name)
		}
		if dim.Type != types.Enumerated {
			t.Errorf("Expected type Enumerated, got %v", dim.Type)
		}
		if len(dim.Values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(dim.Values))
		}
		if dim.DefaultValue != "pending" {
			t.Errorf("Expected default 'pending', got '%s'", dim.DefaultValue)
		}
		if dim.Prefixes["done"] != "d" {
			t.Errorf("Expected prefix for 'done' to be 'd', got '%s'", dim.Prefixes["done"])
		}
	})

	t.Run("Valid hierarchical dimension configuration", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "parent_hierarchy",
					"type": "hierarchical",
					"ref_field": "parent_id"
				}
			]
		}`

		config, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err != nil {
			t.Fatalf("Failed to load valid hierarchical config: %v", err)
		}

		dim := config.Dimensions[0]
		if dim.Type != types.Hierarchical {
			t.Errorf("Expected type Hierarchical, got %v", dim.Type)
		}
		if dim.RefField != "parent_id" {
			t.Errorf("Expected ref_field 'parent_id', got '%s'", dim.RefField)
		}
	})

	t.Run("Complex configuration with multiple dimensions", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "active", "done"],
					"default_value": "pending",
					"prefixes": {
						"done": "d",
						"active": "a"
					}
				},
				{
					"name": "priority",
					"type": "enumerated",
					"values": ["low", "medium", "high"],
					"default_value": "medium",
					"prefixes": {
						"high": "h"
					}
				},
				{
					"name": "parent_hierarchy",
					"type": "hierarchical",
					"ref_field": "parent_id"
				}
			]
		}`

		config, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err != nil {
			t.Fatalf("Failed to load complex config: %v", err)
		}

		if len(config.Dimensions) != 3 {
			t.Errorf("Expected 3 dimensions, got %d", len(config.Dimensions))
		}

		// Verify each dimension
		statusFound := false
		priorityFound := false
		hierarchicalFound := false

		for _, dim := range config.Dimensions {
			switch dim.Name {
			case "status":
				statusFound = true
				if len(dim.Prefixes) != 2 {
					t.Errorf("Expected 2 prefixes for status, got %d", len(dim.Prefixes))
				}
			case "priority":
				priorityFound = true
				if dim.Prefixes["high"] != "h" {
					t.Errorf("Expected priority prefix for 'high' to be 'h'")
				}
			case "parent_hierarchy":
				hierarchicalFound = true
				if dim.Type != types.Hierarchical {
					t.Errorf("Expected hierarchical type for parent_hierarchy")
				}
			}
		}

		if !statusFound || !priorityFound || !hierarchicalFound {
			t.Error("Not all expected dimensions were found")
		}
	})

	t.Run("Invalid JSON syntax", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated"
					// Missing comma
					"values": ["pending"]
				}
			]
		}`

		_, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected error for invalid JSON syntax")
		}
		if !strings.Contains(err.Error(), "failed to parse JSON") {
			t.Errorf("Expected parse error, got: %v", err)
		}
	})

	t.Run("Invalid dimension type", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "invalid_type",
					"values": ["pending", "done"]
				}
			]
		}`

		_, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected error for invalid dimension type")
		}
		if !strings.Contains(err.Error(), "invalid dimension type") {
			t.Errorf("Expected dimension type error, got: %v", err)
		}
	})

	t.Run("Validation errors - duplicate dimension names", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"]
				},
				{
					"name": "status",
					"type": "enumerated", 
					"values": ["low", "high"]
				}
			]
		}`

		_, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected validation error for duplicate dimension names")
		}
		if !strings.Contains(err.Error(), "configuration validation failed") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("Validation errors - conflicting prefixes", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"],
					"prefixes": {"done": "d"}
				},
				{
					"name": "priority",
					"type": "enumerated",
					"values": ["low", "high"],
					"prefixes": {"high": "d"}
				}
			]
		}`

		_, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected validation error for conflicting prefixes")
		}
	})

	t.Run("Validation errors - invalid default value", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"],
					"default_value": "invalid"
				}
			]
		}`

		_, err := api.LoadConfigFromJSON([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected validation error for invalid default value")
		}
	})
}

func TestLoadConfigFromJSONWithDetails(t *testing.T) {
	t.Run("Valid configuration returns success result", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"],
					"default_value": "pending"
				}
			]
		}`

		config, results, err := api.LoadConfigFromJSONWithDetails([]byte(jsonConfig))
		if err != nil {
			t.Fatalf("Expected no error for valid config, got: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		result := results[0]
		if result.Type != "success" {
			t.Errorf("Expected success result, got type: %s", result.Type)
		}
		if !strings.Contains(result.Message, "Configuration is valid") {
			t.Errorf("Expected success message, got: %s", result.Message)
		}

		if len(config.Dimensions) != 1 {
			t.Errorf("Expected 1 dimension in returned config")
		}
	})

	t.Run("Invalid configuration returns error result", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"],
					"default_value": "invalid"
				}
			]
		}`

		_, results, err := api.LoadConfigFromJSONWithDetails([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected error for invalid config")
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		result := results[0]
		if result.Type != "error" {
			t.Errorf("Expected error result, got type: %s", result.Type)
		}
	})
}

func TestCreateStoreFromJSON(t *testing.T) {
	t.Run("Successfully create store from valid JSON", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"],
					"default_value": "pending"
				}
			]
		}`

		tmpfile, err := os.CreateTemp("", "test_json_store*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		store, err := api.CreateStoreFromJSON(tmpfile.Name(), []byte(jsonConfig))
		if err != nil {
			t.Fatalf("Failed to create store from JSON: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Verify store was created by attempting a basic operation
		_, err = store.Add("Test Document", map[string]interface{}{
			"status": "pending",
		})
		if err != nil {
			t.Errorf("Failed to add document to new store: %v", err)
		}
	})

	t.Run("Fail to create store from invalid JSON", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "invalid_type"
				}
			]
		}`

		tmpfile, err := os.CreateTemp("", "test_invalid_store*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(tmpfile.Name()) }()
		_ = tmpfile.Close()

		_, err = api.CreateStoreFromJSON(tmpfile.Name(), []byte(jsonConfig))
		if err == nil {
			t.Error("Expected error when creating store from invalid JSON")
		}
	})
}

func TestValidateJSONConfig(t *testing.T) {
	t.Run("Valid config passes validation", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"]
				}
			]
		}`

		err := api.ValidateJSONConfig([]byte(jsonConfig))
		if err != nil {
			t.Errorf("Expected valid config to pass validation, got: %v", err)
		}
	})

	t.Run("Invalid config fails validation", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": []
		}`

		err := api.ValidateJSONConfig([]byte(jsonConfig))
		if err == nil {
			t.Error("Expected invalid config to fail validation")
		}
	})
}

func TestValidateJSONConfigWithDetails(t *testing.T) {
	t.Run("Returns detailed validation results", func(t *testing.T) {
		jsonConfig := `{
			"dimensions": [
				{
					"name": "status",
					"type": "enumerated",
					"values": ["pending", "done"]
				}
			]
		}`

		results := api.ValidateJSONConfigWithDetails([]byte(jsonConfig))
		if len(results) == 0 {
			t.Error("Expected validation results")
		}

		found := false
		for _, result := range results {
			if result.Type == "success" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected success result in validation results")
		}
	})
}

func TestDimensionTypeJSONMarshaling(t *testing.T) {
	t.Run("Marshal and unmarshal DimensionType", func(t *testing.T) {
		tests := []struct {
			dt       types.DimensionType
			expected string
		}{
			{types.Enumerated, `"enumerated"`},
			{types.Hierarchical, `"hierarchical"`},
		}

		for _, test := range tests {
			// Test marshaling
			data, err := json.Marshal(test.dt)
			if err != nil {
				t.Errorf("Failed to marshal %v: %v", test.dt, err)
			}
			if string(data) != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, string(data))
			}

			// Test unmarshaling
			var dt types.DimensionType
			err = json.Unmarshal(data, &dt)
			if err != nil {
				t.Errorf("Failed to unmarshal %s: %v", test.expected, err)
			}
			if dt != test.dt {
				t.Errorf("Expected %v, got %v", test.dt, dt)
			}
		}
	})

	t.Run("Unmarshal invalid DimensionType", func(t *testing.T) {
		var dt types.DimensionType
		err := json.Unmarshal([]byte(`"invalid"`), &dt)
		if err == nil {
			t.Error("Expected error for invalid dimension type")
		}
		if !strings.Contains(err.Error(), "invalid dimension type") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
}
