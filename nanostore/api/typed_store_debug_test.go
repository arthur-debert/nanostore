package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestDebuggingUtilities(t *testing.T) {
	// Create a temporary file for typed store
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create some test data for debugging analysis
	_, err = store.Create("Test Task 1", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Create("Test Task 2", &TodoItem{
		Status:   "pending",
		Priority: "medium",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddRaw("Custom Task", map[string]interface{}{
		"status":         "done",
		"priority":       "low",
		"activity":       "archived",
		"_data.assignee": "alice",
		"_data.estimate": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("GetDebugInfo", func(t *testing.T) {
		debugInfo, err := store.GetDebugInfo()
		if err != nil {
			t.Fatalf("failed to get debug info: %v", err)
		}

		if debugInfo == nil {
			t.Fatal("expected debug info, got nil")
		}

		// Verify basic structure
		if debugInfo.StoreType == "" {
			t.Error("expected store type to be set")
		}

		if debugInfo.DocumentCount < 3 {
			t.Errorf("expected at least 3 documents, got %d", debugInfo.DocumentCount)
		}

		if debugInfo.Configuration == nil {
			t.Error("expected configuration to be set")
		}

		// Verify type information
		if debugInfo.TypeInfo.TypeName == "" {
			t.Error("expected type name to be set")
		}

		if !debugInfo.TypeInfo.HasDocument {
			t.Error("expected TodoItem to embed nanostore.Document")
		}

		// Verify runtime stats
		if debugInfo.RuntimeStats.TotalDimensions == 0 {
			t.Error("expected some dimensions to be configured")
		}

		t.Logf("Debug Info: %+v", debugInfo)
		t.Logf("Type Info: %+v", debugInfo.TypeInfo)
		t.Logf("Runtime Stats: %+v", debugInfo.RuntimeStats)
	})

	t.Run("GetStoreStats", func(t *testing.T) {
		stats, err := store.GetStoreStats()
		if err != nil {
			t.Fatalf("failed to get store stats: %v", err)
		}

		if stats == nil {
			t.Fatal("expected stats, got nil")
		}

		// Verify document count
		if stats.TotalDocuments < 3 {
			t.Errorf("expected at least 3 documents, got %d", stats.TotalDocuments)
		}

		// Verify dimension distribution
		if len(stats.DimensionDistribution) == 0 {
			t.Error("expected dimension distribution data")
		}

		// Check for specific dimension data
		if statusDist, exists := stats.DimensionDistribution["status"]; exists {
			t.Logf("Status distribution: %+v", statusDist)
			if statusDist["active"] < 1 {
				t.Error("expected at least one active status document")
			}
		} else {
			t.Error("expected status dimension in distribution")
		}

		// Verify data field coverage
		if len(stats.DataFieldCoverage) == 0 {
			t.Error("expected data field coverage information")
		}

		// Check for specific data fields
		if assigneeCoverage, exists := stats.DataFieldCoverage["assignee"]; exists {
			t.Logf("Assignee coverage: %.2f", assigneeCoverage)
			if assigneeCoverage <= 0 || assigneeCoverage > 1 {
				t.Errorf("expected coverage between 0 and 1, got %.2f", assigneeCoverage)
			}
		}

		t.Logf("Store Stats: %+v", stats)
	})

	t.Run("ValidateStoreIntegrity", func(t *testing.T) {
		report, err := store.ValidateStoreIntegrity()
		if err != nil {
			t.Fatalf("failed to validate store integrity: %v", err)
		}

		if report == nil {
			t.Fatal("expected integrity report, got nil")
		}

		// Verify report structure
		if report.TotalDocuments < 3 {
			t.Errorf("expected at least 3 documents in report, got %d", report.TotalDocuments)
		}

		// With our valid test data, we should have a valid store
		if !report.IsValid && report.ErrorCount > 0 {
			t.Errorf("expected valid store, but got %d errors: %v", report.ErrorCount, report.Errors)
		}

		if report.Summary == "" {
			t.Error("expected summary to be set")
		}

		t.Logf("Integrity Report: IsValid=%v, Errors=%d, Warnings=%d",
			report.IsValid, report.ErrorCount, report.WarningCount)
		t.Logf("Summary: %s", report.Summary)

		if len(report.Errors) > 0 {
			for _, err := range report.Errors {
				t.Logf("Error: %s - %s", err.Type, err.Message)
			}
		}

		if len(report.Warnings) > 0 {
			for _, warning := range report.Warnings {
				t.Logf("Warning: %s - %s", warning.Type, warning.Message)
			}
		}
	})
}

func TestDebuggingUtilitiesWithProblematicData(t *testing.T) {
	// For this test, we'll verify that valid data produces no errors
	// The underlying store validates data on creation, so invalid dimension values
	// are rejected before they can be stored.

	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Create valid data
	_, err = store.Create("Valid Task", &TodoItem{
		Status:   "active",
		Priority: "high",
		Activity: "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("IntegrityValidationWithValidData", func(t *testing.T) {
		report, err := store.ValidateStoreIntegrity()
		if err != nil {
			t.Fatalf("failed to validate store integrity: %v", err)
		}

		// With valid data, we should have no errors
		if report.ErrorCount > 0 {
			t.Errorf("expected no integrity errors with valid data, got %d", report.ErrorCount)
			for _, err := range report.Errors {
				t.Logf("Unexpected error: %s - %s", err.Type, err.Message)
			}
		}

		if !report.IsValid {
			t.Error("expected store to be valid with clean data")
		}

		t.Logf("Integrity validation passed: %s", report.Summary)
	})

	t.Run("ValidateInvalidDimensionValueHandling", func(t *testing.T) {
		// Test that the store properly rejects invalid dimension values
		_, err := store.AddRaw("Invalid Status Task", map[string]interface{}{
			"status":   "invalid_status", // This should be rejected
			"priority": "high",
			"activity": "active",
		})

		// The store should reject this during creation
		if err == nil {
			t.Error("expected error when adding document with invalid dimension value")
		} else {
			t.Logf("Store correctly rejected invalid data: %v", err)
		}
	})
}

func TestTypeInfoExtraction(t *testing.T) {
	// This tests the extractTypeInfo helper function indirectly through GetDebugInfo
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	t.Run("TypeInfoDetails", func(t *testing.T) {
		debugInfo, err := store.GetDebugInfo()
		if err != nil {
			t.Fatalf("failed to get debug info: %v", err)
		}

		typeInfo := debugInfo.TypeInfo

		// Verify type name contains TodoItem
		if !strings.Contains(typeInfo.TypeName, "TodoItem") {
			t.Errorf("expected type name to contain 'TodoItem', got %s", typeInfo.TypeName)
		}

		// Verify we have expected fields
		if typeInfo.FieldCount < 4 { // Document + Status + Priority + Activity
			t.Errorf("expected at least 4 fields, got %d", typeInfo.FieldCount)
		}

		// Check for embedded Document
		if !typeInfo.HasDocument {
			t.Error("expected TodoItem to embed nanostore.Document")
		}

		// Verify embedded list contains Document
		foundDocument := false
		for _, embed := range typeInfo.EmbedsList {
			if strings.Contains(embed, "Document") {
				foundDocument = true
				break
			}
		}
		if !foundDocument {
			t.Error("expected Document to be in embeds list")
		}

		// Check field details
		dimensionFieldCount := 0
		for _, field := range typeInfo.Fields {
			if field.IsDimension {
				dimensionFieldCount++
				t.Logf("Dimension field: %s (tag: %s)", field.Name, field.DimensionTag)
			}
		}

		if dimensionFieldCount < 3 { // Status, Priority, Activity
			t.Errorf("expected at least 3 dimension fields, got %d", dimensionFieldCount)
		}

		t.Logf("Type analysis complete: %d total fields, %d dimension fields",
			typeInfo.FieldCount, dimensionFieldCount)
	})
}
