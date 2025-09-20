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
func (a *API) RenameField(docs []types.Document, config types.Config, oldName, newName string, opts Options) ([]types.Document, *Result) {
	// Make a deep copy of documents to avoid modifying the input slice
	docsCopy := make([]types.Document, len(docs))
	for i, doc := range docs {
		docsCopy[i] = doc
		docsCopy[i].Dimensions = make(map[string]interface{})
		for k, v := range doc.Dimensions {
			docsCopy[i].Dimensions[k] = v
		}
	}

	ctx := &MigrationContext{
		Documents: docsCopy,
		Config:    config,
		DryRun:    opts.DryRun,
	}

	cmd := &RenameField{
		OldName:   oldName,
		NewName:   newName,
		FieldType: opts.FieldType,
	}

	result := cmd.Execute(ctx)
	return ctx.Documents, result
}

// RemoveField removes a field from all documents
func (a *API) RemoveField(docs []types.Document, config types.Config, fieldName string, opts Options) ([]types.Document, *Result) {
	// Make a deep copy of documents to avoid modifying the input slice
	docsCopy := make([]types.Document, len(docs))
	for i, doc := range docs {
		docsCopy[i] = doc
		docsCopy[i].Dimensions = make(map[string]interface{})
		for k, v := range doc.Dimensions {
			docsCopy[i].Dimensions[k] = v
		}
	}

	ctx := &MigrationContext{
		Documents: docsCopy,
		Config:    config,
		DryRun:    opts.DryRun,
	}

	cmd := &RemoveField{
		FieldName: fieldName,
		FieldType: opts.FieldType,
	}

	result := cmd.Execute(ctx)
	return ctx.Documents, result
}

// AddField adds a field with a default value to all documents
func (a *API) AddField(docs []types.Document, config types.Config, fieldName string, defaultValue interface{}, opts Options) ([]types.Document, *Result) {
	// Make a deep copy of documents to avoid modifying the input slice
	docsCopy := make([]types.Document, len(docs))
	for i, doc := range docs {
		docsCopy[i] = doc
		docsCopy[i].Dimensions = make(map[string]interface{})
		for k, v := range doc.Dimensions {
			docsCopy[i].Dimensions[k] = v
		}
	}

	ctx := &MigrationContext{
		Documents: docsCopy,
		Config:    config,
		DryRun:    opts.DryRun,
	}

	cmd := &AddField{
		FieldName:    fieldName,
		DefaultValue: defaultValue,
		IsDataField:  opts.IsDataField,
	}

	result := cmd.Execute(ctx)
	return ctx.Documents, result
}

// TransformField applies a transformation to a field across all documents
func (a *API) TransformField(docs []types.Document, config types.Config, fieldName string, transformer string, opts Options) ([]types.Document, *Result) {
	// Make a deep copy of documents to avoid modifying the input slice
	docsCopy := make([]types.Document, len(docs))
	for i, doc := range docs {
		docsCopy[i] = doc
		docsCopy[i].Dimensions = make(map[string]interface{})
		for k, v := range doc.Dimensions {
			docsCopy[i].Dimensions[k] = v
		}
	}

	ctx := &MigrationContext{
		Documents: docsCopy,
		Config:    config,
		DryRun:    opts.DryRun,
	}

	cmd := &TransformField{
		FieldName:       fieldName,
		TransformerName: transformer,
	}

	result := cmd.Execute(ctx)
	return ctx.Documents, result
}

// ValidateSchema validates that all documents conform to the current schema
func (a *API) ValidateSchema(docs []types.Document, config types.Config, opts Options) ([]types.Document, *Result) {
	// TODO: Implement
	return docs, &Result{
		Success: false,
		Code:    CodeExecutionError,
		Messages: []Message{
			{Level: LevelError, Text: "ValidateSchema not implemented"},
		},
	}
}
