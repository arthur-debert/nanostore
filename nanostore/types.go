package nanostore

import (
	"github.com/arthur-debert/nanostore/nanostore/types"
)

// Re-export types from the shared types package for public API compatibility.
// This maintains the existing API surface while using the shared definitions.

// Status represents the status of a document
type Status = types.Status

// Status constants
const (
	StatusPending   = types.StatusPending
	StatusCompleted = types.StatusCompleted
)

// Document represents a document in the store with its generated ID
type Document = types.Document

// ListOptions configures how documents are listed
type ListOptions = types.ListOptions

// UpdateRequest specifies fields to update on a document
type UpdateRequest = types.UpdateRequest

// Config defines the dimension configuration for the store
type Config = types.Config

// DimensionConfig defines a single dimension for ID generation
type DimensionConfig = types.DimensionConfig

// DimensionType represents the type of dimension
type DimensionType = types.DimensionType

// Dimension type constants
const (
	Enumerated   = types.Enumerated
	Hierarchical = types.Hierarchical
)
