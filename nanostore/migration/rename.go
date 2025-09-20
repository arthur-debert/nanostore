package migration

import (
	"fmt"
	"strings"
	"time"
)

// RenameField renames a field across all documents
type RenameField struct {
	OldName string
	NewName string
}

// Description returns a human-readable description of the command
func (r *RenameField) Description() string {
	return fmt.Sprintf("Rename field '%s' to '%s'", r.OldName, r.NewName)
}

// Validate checks if the rename can be executed
func (r *RenameField) Validate(ctx *MigrationContext) []Message {
	var messages []Message

	// Validate field names first
	if strings.TrimSpace(r.OldName) == "" {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Old field name cannot be empty",
		})
	}

	if strings.TrimSpace(r.NewName) == "" {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "New field name cannot be empty",
		})
	}

	// Ensure not trying to rename to same name
	if r.OldName == r.NewName {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Old and new field names are the same",
		})
	}

	// Check if old field exists in any document
	found := false
	fieldType := ""
	count := 0

	for _, doc := range ctx.Documents {
		// Check dimensions
		if _, exists := doc.Dimensions[r.OldName]; exists {
			found = true
			fieldType = "dimension"
			count++
		}
		// Check data fields (_data. prefix)
		dataKey := "_data." + r.OldName
		if _, exists := doc.Dimensions[dataKey]; exists {
			found = true
			fieldType = "data"
			count++
		}
	}

	if !found {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  fmt.Sprintf("Field '%s' not found in any document", r.OldName),
		})
		return messages
	}

	// Report what we found
	messages = append(messages, Message{
		Level: LevelInfo,
		Text:  fmt.Sprintf("Found field '%s' (%s field) in %d documents", r.OldName, fieldType, count),
	})

	// Check if new field already exists
	conflicts := 0
	conflictDocs := []string{}

	for _, doc := range ctx.Documents {
		// Check both dimension and data fields
		if _, exists := doc.Dimensions[r.NewName]; exists {
			conflicts++
			conflictDocs = append(conflictDocs, doc.UUID)
		}
		dataKey := "_data." + r.NewName
		if _, exists := doc.Dimensions[dataKey]; exists {
			conflicts++
			conflictDocs = append(conflictDocs, doc.UUID)
		}
	}

	if conflicts > 0 {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  fmt.Sprintf("Field '%s' already exists in %d documents", r.NewName, conflicts),
			Details: map[string]interface{}{
				"document_ids": conflictDocs[:min(5, len(conflictDocs))], // Show first 5
				"total":        conflicts,
			},
		})
	}

	return messages
}

// Execute performs the rename operation
func (r *RenameField) Execute(ctx *MigrationContext) *Result {
	result := &Result{
		Success:  true,
		Code:     CodeSuccess,
		Messages: []Message{},
		Stats: Stats{
			TotalDocs: len(ctx.Documents),
		},
	}

	startTime := time.Now()

	// Validation first
	validationMessages := r.Validate(ctx)
	result.Messages = append(result.Messages, validationMessages...)

	// Check for errors in validation
	hasError := false
	for _, msg := range validationMessages {
		if msg.Level == LevelError {
			hasError = true
		}
	}

	if hasError {
		result.Success = false
		result.Code = CodeValidationError
		return result
	}

	// Perform the rename
	isDataField := false
	for i := range ctx.Documents {
		doc := &ctx.Documents[i]
		modified := false

		// Check if it's a regular dimension
		if val, exists := doc.Dimensions[r.OldName]; exists {
			if !ctx.DryRun {
				doc.Dimensions[r.NewName] = val
				delete(doc.Dimensions, r.OldName)
			}
			modified = true
		}

		// Check if it's a data field
		oldDataKey := "_data." + r.OldName
		newDataKey := "_data." + r.NewName
		if val, exists := doc.Dimensions[oldDataKey]; exists {
			if !ctx.DryRun {
				doc.Dimensions[newDataKey] = val
				delete(doc.Dimensions, oldDataKey)
			}
			modified = true
			isDataField = true
		}

		if modified {
			result.ModifiedDocs = append(result.ModifiedDocs, doc.UUID)
			result.Stats.ModifiedDocs++
		}
	}

	result.Stats.Duration = time.Since(startTime)

	// Success message
	fieldTypeStr := "dimension"
	if isDataField {
		fieldTypeStr = "data"
	}

	if result.Stats.ModifiedDocs > 0 {
		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text: fmt.Sprintf("Renamed %s field '%s' to '%s' in %d documents",
				fieldTypeStr, r.OldName, r.NewName, result.Stats.ModifiedDocs),
		})
	}

	if ctx.DryRun {
		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text:  "(DRY RUN - no changes applied)",
		})
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
