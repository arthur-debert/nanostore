package migration

import (
	"github.com/arthur-debert/nanostore/types"
)

// API provides the public interface for migrations
type API struct{}

// NewAPI creates a new migration API instance
func NewAPI() *API {
	return &API{}
}

// RenameField renames a field across all documents
func (a *API) RenameField(docs []types.Document, config types.Config, oldName, newName string, opts Options) *Result {
	ctx := &MigrationContext{
		Documents: docs,
		Config:    config,
		DryRun:    opts.DryRun,
	}

	cmd := &RenameField{
		OldName: oldName,
		NewName: newName,
	}

	return cmd.Execute(ctx)
}

// RemoveField removes a field from all documents
func (a *API) RemoveField(docs []types.Document, config types.Config, fieldName string, opts Options) *Result {
	// TODO: Implement
	return &Result{
		Success: false,
		Code:    CodeExecutionError,
		Messages: []Message{
			{Level: LevelError, Text: "RemoveField not implemented"},
		},
	}
}

// AddField adds a field with a default value to all documents
func (a *API) AddField(docs []types.Document, config types.Config, fieldName string, defaultValue interface{}, opts Options) *Result {
	// TODO: Implement
	return &Result{
		Success: false,
		Code:    CodeExecutionError,
		Messages: []Message{
			{Level: LevelError, Text: "AddField not implemented"},
		},
	}
}

// TransformField applies a transformation to a field across all documents
func (a *API) TransformField(docs []types.Document, config types.Config, fieldName string, transformer string, opts Options) *Result {
	// TODO: Implement
	return &Result{
		Success: false,
		Code:    CodeExecutionError,
		Messages: []Message{
			{Level: LevelError, Text: "TransformField not implemented"},
		},
	}
}

// ValidateSchema validates that all documents conform to the current schema
func (a *API) ValidateSchema(docs []types.Document, config types.Config, opts Options) *Result {
	// TODO: Implement
	return &Result{
		Success: false,
		Code:    CodeExecutionError,
		Messages: []Message{
			{Level: LevelError, Text: "ValidateSchema not implemented"},
		},
	}
}
