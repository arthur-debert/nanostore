package migration

import (
	"fmt"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/internal/validation"
)

// AddField adds a field with a default value to all documents
type AddField struct {
	FieldName    string
	DefaultValue interface{}
	IsDataField  bool // If true, add as _data.fieldName
}

// Description returns a human-readable description of the command
func (a *AddField) Description() string {
	fieldType := "dimension"
	if a.IsDataField {
		fieldType = "data"
	}
	return fmt.Sprintf("Add %s field '%s' with default value", fieldType, a.FieldName)
}

// Validate checks if the add can be executed
func (a *AddField) Validate(ctx *MigrationContext) []Message {
	var messages []Message

	// Validate field name
	if strings.TrimSpace(a.FieldName) == "" {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Field name cannot be empty",
		})
		return messages
	}

	// Check if field name contains invalid characters
	if strings.HasPrefix(a.FieldName, "_data.") {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Field name cannot start with '_data.' prefix",
		})
	}

	// Validate the default value for dimension fields
	if !a.IsDataField {
		if err := validation.ValidateSimpleType(a.DefaultValue, a.FieldName); err != nil {
			messages = append(messages, Message{
				Level: LevelError,
				Text:  fmt.Sprintf("Invalid default value for dimension field: %v", err),
			})
		}
	}

	// Check if field already exists in any document
	conflicts := 0
	conflictDocs := []string{}
	fieldKey := a.FieldName
	if a.IsDataField {
		fieldKey = "_data." + a.FieldName
	}

	for _, doc := range ctx.Documents {
		if _, exists := doc.Dimensions[fieldKey]; exists {
			conflicts++
			if len(conflictDocs) < 5 { // Collect first 5 for details
				conflictDocs = append(conflictDocs, doc.UUID)
			}
		}
	}

	if conflicts > 0 {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  fmt.Sprintf("Field '%s' already exists in %d documents", a.FieldName, conflicts),
			Details: map[string]interface{}{
				"document_ids": conflictDocs,
				"total":        conflicts,
			},
		})
	}

	// Info about what will be added
	fieldType := "dimension"
	if a.IsDataField {
		fieldType = "data"
	}
	messages = append(messages, Message{
		Level: LevelInfo,
		Text:  fmt.Sprintf("Will add %s field '%s' to %d documents", fieldType, a.FieldName, len(ctx.Documents)),
		Details: map[string]interface{}{
			"default_value": a.DefaultValue,
			"field_type":    fieldType,
		},
	})

	return messages
}

// Execute performs the add operation
func (a *AddField) Execute(ctx *MigrationContext) *Result {
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
	validationMessages := a.Validate(ctx)
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

	// Perform the addition
	fieldKey := a.FieldName
	if a.IsDataField {
		fieldKey = "_data." + a.FieldName
	}

	for i := range ctx.Documents {
		doc := &ctx.Documents[i]

		// Add the field
		if !ctx.DryRun {
			doc.Dimensions[fieldKey] = a.DefaultValue
		}

		result.ModifiedDocs = append(result.ModifiedDocs, doc.UUID)
		result.Stats.ModifiedDocs++
	}

	result.Stats.Duration = time.Since(startTime)

	// Success message
	fieldType := "dimension"
	if a.IsDataField {
		fieldType = "data"
	}

	if result.Stats.ModifiedDocs > 0 {
		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text: fmt.Sprintf("Added %s field '%s' with default value to %d documents",
				fieldType, a.FieldName, result.Stats.ModifiedDocs),
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
