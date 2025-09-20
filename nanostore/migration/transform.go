package migration

import (
	"fmt"
	"strings"
	"time"
)

// TransformField transforms field values across all documents
type TransformField struct {
	FieldName       string
	TransformerName string
}

// Description returns a human-readable description of the command
func (t *TransformField) Description() string {
	return fmt.Sprintf("Transform field '%s' using '%s' transformer", t.FieldName, t.TransformerName)
}

// Validate checks if the transform can be executed
func (t *TransformField) Validate(ctx *MigrationContext) []Message {
	var messages []Message

	// Validate field name
	if strings.TrimSpace(t.FieldName) == "" {
		messages = append(messages, Message{
			Level: LevelError,
			Text:  "Field name cannot be empty",
		})
		return messages
	}

	// Validate transformer exists
	transformer, exists := TransformerRegistry[t.TransformerName]
	if !exists {
		available := make([]string, 0, len(TransformerRegistry))
		for name := range TransformerRegistry {
			available = append(available, name)
		}
		messages = append(messages, Message{
			Level: LevelError,
			Text:  fmt.Sprintf("Unknown transformer '%s'", t.TransformerName),
			Details: map[string]interface{}{
				"available_transformers": available,
			},
		})
		return messages
	}

	// Check if field exists and test transformation
	found := false
	fieldType := ""
	count := 0
	errors := 0
	sampleErrors := []string{}

	for _, doc := range ctx.Documents {
		// Check dimensions
		if val, exists := doc.Dimensions[t.FieldName]; exists {
			found = true
			fieldType = "dimension"
			count++

			// Test transformation
			if _, err := transformer(val); err != nil {
				errors++
				if len(sampleErrors) < 3 {
					sampleErrors = append(sampleErrors,
						fmt.Sprintf("doc %s: %v", doc.UUID, err))
				}
			}
		}

		// Check data fields (_data. prefix)
		dataKey := "_data." + t.FieldName
		if val, exists := doc.Dimensions[dataKey]; exists {
			found = true
			fieldType = "data"
			count++

			// Test transformation
			if _, err := transformer(val); err != nil {
				errors++
				if len(sampleErrors) < 3 {
					sampleErrors = append(sampleErrors,
						fmt.Sprintf("doc %s: %v", doc.UUID, err))
				}
			}
		}
	}

	if !found {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  fmt.Sprintf("Field '%s' not found in any document", t.FieldName),
		})
		return messages
	}

	// Report what we found
	messages = append(messages, Message{
		Level: LevelInfo,
		Text:  fmt.Sprintf("Found field '%s' (%s field) in %d documents", t.FieldName, fieldType, count),
	})

	// Report transformation errors as warning during validation
	if errors > 0 {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  fmt.Sprintf("Transformation will fail for %d values", errors),
			Details: map[string]interface{}{
				"sample_errors": sampleErrors,
				"total_errors":  errors,
			},
		})
	}

	return messages
}

// Execute performs the transform operation
func (t *TransformField) Execute(ctx *MigrationContext) *Result {
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
	validationMessages := t.Validate(ctx)
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

	// Get transformer
	transformer := TransformerRegistry[t.TransformerName]

	// Perform the transformation
	isDataField := false
	transformErrors := 0
	errorDetails := []map[string]interface{}{}

	for i := range ctx.Documents {
		doc := &ctx.Documents[i]
		modified := false

		// Check if it's a regular dimension
		if val, exists := doc.Dimensions[t.FieldName]; exists {
			newVal, err := transformer(val)
			if err != nil {
				transformErrors++
				if len(errorDetails) < 5 {
					errorDetails = append(errorDetails, map[string]interface{}{
						"document_id": doc.UUID,
						"old_value":   val,
						"error":       err.Error(),
					})
				}
			} else if !ctx.DryRun {
				doc.Dimensions[t.FieldName] = newVal
				modified = true
			}
		}

		// Check if it's a data field
		dataKey := "_data." + t.FieldName
		if val, exists := doc.Dimensions[dataKey]; exists {
			isDataField = true
			newVal, err := transformer(val)
			if err != nil {
				transformErrors++
				if len(errorDetails) < 5 {
					errorDetails = append(errorDetails, map[string]interface{}{
						"document_id": doc.UUID,
						"old_value":   val,
						"error":       err.Error(),
					})
				}
			} else if !ctx.DryRun {
				doc.Dimensions[dataKey] = newVal
				modified = true
			}
		}

		if modified {
			result.ModifiedDocs = append(result.ModifiedDocs, doc.UUID)
			result.Stats.ModifiedDocs++
		}
	}

	result.Stats.Duration = time.Since(startTime)

	// Handle errors
	if transformErrors > 0 {
		result.Success = false
		result.Code = CodePartialFailure
		result.Messages = append(result.Messages, Message{
			Level: LevelError,
			Text:  fmt.Sprintf("Failed to transform %d values", transformErrors),
			Details: map[string]interface{}{
				"errors": errorDetails,
			},
		})
	}

	// Success message
	fieldTypeStr := "dimension"
	if isDataField {
		fieldTypeStr = "data"
	}

	if result.Stats.ModifiedDocs > 0 {
		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text: fmt.Sprintf("Transformed %s field '%s' in %d documents using '%s' transformer",
				fieldTypeStr, t.FieldName, result.Stats.ModifiedDocs, t.TransformerName),
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
