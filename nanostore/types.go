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
