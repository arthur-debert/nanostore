package nanostore

import (
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

// ValidateSimpleType is now in the validation package
// This is kept for backward compatibility
func ValidateSimpleType(value interface{}, dimensionName string) error {
	return validation.ValidateSimpleType(value, dimensionName)
}
