package nanostore

import (
	"testing"
)

// Custom string types for testing
type Status string
type Priority string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func TestDeclarativeTypes(t *testing.T) {
	t.Run("Custom string types", func(t *testing.T) {
		type TaskWithTypes struct {
			Document
			Status   Status   `values:"pending,active,completed" default:"pending"`
			Priority Priority `values:"low,medium,high" default:"medium" prefix:"high=h"`
		}

		store, err := NewFromType[TaskWithTypes](":memory:")
		if err != nil {
			t.Fatalf("expected custom string types to work: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Create with custom types
		task := &TaskWithTypes{
			Status:   StatusActive,
			Priority: PriorityHigh,
		}

		uuid, err := store.Create("Test Task", task)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.Get(uuid)
		if err != nil {
			t.Fatalf("failed to get task: %v", err)
		}

		if retrieved.Status != StatusActive {
			t.Errorf("expected status %s, got %s", StatusActive, retrieved.Status)
		}
		if retrieved.Priority != PriorityHigh {
			t.Errorf("expected priority %s, got %s", PriorityHigh, retrieved.Priority)
		}
	})

	t.Run("Non-string types are rejected", func(t *testing.T) {
		type BadTypes struct {
			Document
			Count    int  `dimension:"count"`
			IsActive bool `dimension:"is_active"`
		}

		_, err := NewFromType[BadTypes](":memory:")
		if err == nil {
			t.Fatal("expected error for non-string types")
		}
		if !hasSubstring(err.Error(), "dimensions must be string types") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Pointer types are rejected", func(t *testing.T) {
		type BadPointer struct {
			Document
			Status *string `values:"pending,done"`
		}

		_, err := NewFromType[BadPointer](":memory:")
		if err == nil {
			t.Fatal("expected error for pointer type")
		}
		if !hasSubstring(err.Error(), "dimensions must be string types") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Custom string type without values", func(t *testing.T) {
		type MissingValues struct {
			Document
			Status Status // Missing values tag
		}

		_, err := NewFromType[MissingValues](":memory:")
		if err == nil {
			t.Fatal("expected error for custom type without values")
		}
		if !hasSubstring(err.Error(), "requires 'values' tag") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("String without values requires enumeration", func(t *testing.T) {
		// This test documents that all dimensions must be enumerated
		// This is a fundamental requirement of nanostore for ID generation
		type FlexibleString struct {
			Document
			Category string // No values - will fail
		}

		_, err := NewFromType[FlexibleString](":memory:")
		if err == nil {
			t.Fatal("expected error for dimension without values")
		}
		if !hasSubstring(err.Error(), "enumerated dimensions must have at least one value") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func hasSubstring(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && hasSubstring(s[1:], substr)
}
