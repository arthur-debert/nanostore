package nanostore

import (
	"fmt"
	"reflect"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/types"
)

// Re-export types from the types package for convenience
type DimensionType = types.DimensionType

const (
	Enumerated   = types.Enumerated
	Hierarchical = types.Hierarchical
)

// Document is an alias for the types.Document
type Document = types.Document

// ListOptions is an alias for types.ListOptions
type ListOptions = types.ListOptions

// OrderClause is an alias for types.OrderClause
type OrderClause = types.OrderClause

// UpdateRequest is an alias for types.UpdateRequest
type UpdateRequest = types.UpdateRequest

// DimensionConfig is an alias for types.DimensionConfig
type DimensionConfig = types.DimensionConfig

// Config is an alias for types.Config
type Config = types.Config

// ValidateConfig validates a configuration
func ValidateConfig(config Config) error {
	return validation.Validate(config.GetDimensionSet())
}

// Store defines the public interface for the document store
type Store interface {
	// List returns documents based on the provided options
	// The returned documents include generated user-facing IDs
	List(opts ListOptions) ([]Document, error)

	// Add creates a new document with the given title and dimension values
	// The dimensions map allows setting any dimension values, including:
	// - Enumerated dimensions (e.g., "status": "pending")
	// - Hierarchical dimensions (e.g., "parent_uuid": "parent-id")
	// Dimensions not specified will use their default values from the configuration
	// Returns the UUID of the created document
	Add(title string, dimensions map[string]interface{}) (string, error)

	// Update modifies an existing document
	Update(id string, updates UpdateRequest) error

	// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(simpleID string) (string, error)

	// Delete removes a document
	// If cascade is true and the document has children (via hierarchical dimensions),
	// all descendant documents are also deleted
	Delete(id string, cascade bool) error

	// UpdateWhere updates all documents matching the where clause
	// The where clause uses SQL syntax and can reference dimension names
	// Example: UpdateWhere("status = ?", UpdateRequest{Title: &newTitle}, "active")
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents updated
	UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error)

	// DeleteByDimension removes all documents matching dimension filters
	DeleteByDimension(filters map[string]interface{}) (int, error)

	// DeleteWhere removes documents matching a custom WHERE clause
	DeleteWhere(whereClause string, args ...interface{}) (int, error)

	// UpdateByDimension updates all documents matching dimension filters
	UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error)

	// Close releases any resources held by the store
	Close() error
}

// ValidateSimpleType ensures a dimension value is a simple type (string, number, bool)
func ValidateSimpleType(value interface{}, dimensionName string) error {
	if value == nil {
		return nil
	}

	// Check the type using reflection
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return nil
	case reflect.Slice, reflect.Array:
		return fmt.Errorf("dimension '%s' cannot be an array/slice type, got %T", dimensionName, value)
	case reflect.Map:
		return fmt.Errorf("dimension '%s' cannot be a map type, got %T", dimensionName, value)
	case reflect.Ptr:
		// Dereference the pointer and check again
		if v.IsNil() {
			return nil // nil pointer is OK
		}
		return ValidateSimpleType(v.Elem().Interface(), dimensionName)
	case reflect.Struct:
		return fmt.Errorf("dimension '%s' cannot be a struct type, got %T", dimensionName, value)
	default:
		return fmt.Errorf("dimension '%s' must be a simple type (string, number, or bool), got %T", dimensionName, value)
	}
}
