package migration

import (
	"fmt"
	"strings"
	"time"
)

// RemoveField removes a field from all documents
type RemoveField struct {
	FieldName string
}

// Description returns a human-readable description of the command
func (r *RemoveField) Description() string {
	return fmt.Sprintf("Remove field '%s'", r.FieldName)
}

// Validate checks if the remove can be executed
func (r *RemoveField) Validate(ctx *MigrationContext) []Message {
	var messages []Message

	// Validate field name
	if strings.TrimSpace(r.FieldName) == "" {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Field name cannot be empty",
		})
		return messages
	}

	// Check if field exists in any document
	found := false
	fieldType := ""
	count := 0
	dimensionCount := 0
	dataCount := 0

	for _, doc := range ctx.Documents {
		// Check dimensions
		if _, exists := doc.Dimensions[r.FieldName]; exists {
			found = true
			dimensionCount++
			count++
		}
		// Check data fields (_data. prefix)
		dataKey := "_data." + r.FieldName
		if _, exists := doc.Dimensions[dataKey]; exists {
			found = true
			dataCount++
			count++
		}
	}

	// Determine field type
	if dimensionCount > 0 && dataCount > 0 {
		fieldType = "mixed (both dimension and data)"
	} else if dimensionCount > 0 {
		fieldType = "dimension"
	} else if dataCount > 0 {
		fieldType = "data"
	}

	if !found {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  fmt.Sprintf("Field '%s' not found in any document", r.FieldName),
		})
	} else {
		messages = append(messages, Message{
			Level: LevelInfo,
			Text:  fmt.Sprintf("Found field '%s' (%s field) in %d documents", r.FieldName, fieldType, count),
			Details: map[string]interface{}{
				"dimension_occurrences": dimensionCount,
				"data_occurrences":      dataCount,
			},
		})
	}

	// Warning if removing dimension field
	if dimensionCount > 0 {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  fmt.Sprintf("Removing dimension field '%s' may affect ID generation and filtering", r.FieldName),
		})
	}

	return messages
}

// Execute performs the remove operation
func (r *RemoveField) Execute(ctx *MigrationContext) *Result {
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

	// Perform the removal
	dimensionRemovals := 0
	dataRemovals := 0
	modifiedDocs := make(map[string]bool)

	for i := range ctx.Documents {
		doc := &ctx.Documents[i]
		modified := false

		// Check if it's a regular dimension
		if _, exists := doc.Dimensions[r.FieldName]; exists {
			if !ctx.DryRun {
				delete(doc.Dimensions, r.FieldName)
			}
			dimensionRemovals++
			modified = true
		}

		// Check if it's a data field
		dataKey := "_data." + r.FieldName
		if _, exists := doc.Dimensions[dataKey]; exists {
			if !ctx.DryRun {
				delete(doc.Dimensions, dataKey)
			}
			dataRemovals++
			modified = true
		}

		if modified {
			modifiedDocs[doc.UUID] = true
		}
	}

	// Convert map to slice
	for uuid := range modifiedDocs {
		result.ModifiedDocs = append(result.ModifiedDocs, uuid)
		result.Stats.ModifiedDocs++
	}

	result.Stats.Duration = time.Since(startTime)

	// Success message
	if result.Stats.ModifiedDocs > 0 {
		details := ""
		if dimensionRemovals > 0 && dataRemovals > 0 {
			details = fmt.Sprintf(" (%d dimension, %d data)", dimensionRemovals, dataRemovals)
		} else if dimensionRemovals > 0 {
			details = " (dimension field)"
		} else if dataRemovals > 0 {
			details = " (data field)"
		}

		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text:  fmt.Sprintf("Removed field '%s' from %d documents%s", r.FieldName, result.Stats.ModifiedDocs, details),
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
