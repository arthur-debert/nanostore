package migration

import (
	"fmt"
	"time"

	"github.com/arthur-debert/nanostore/internal/validation"
	"github.com/arthur-debert/nanostore/types"
)

// ValidateSchema validates that all documents conform to the schema
type ValidateSchema struct{}

// Description returns a human-readable description of the command
func (v *ValidateSchema) Description() string {
	return "Validate all documents against schema"
}

// Validate checks if validation can be executed
func (v *ValidateSchema) Validate(ctx *MigrationContext) []Message {
	var messages []Message

	// Check if we have dimensions defined in config
	if len(ctx.Config.Dimensions) == 0 {
		messages = append(messages, Message{
			Level: LevelWarning,
			Text:  "No dimensions defined in config - all fields will be treated as data fields",
		})
	}

	messages = append(messages, Message{
		Level: LevelInfo,
		Text:  fmt.Sprintf("Will validate %d documents against schema", len(ctx.Documents)),
		Details: map[string]interface{}{
			"dimension_count": len(ctx.Config.Dimensions),
		},
	})

	return messages
}

// Execute performs the validation
func (v *ValidateSchema) Execute(ctx *MigrationContext) *Result {
	result := &Result{
		Success:  true,
		Code:     CodeSuccess,
		Messages: []Message{},
		Stats: Stats{
			TotalDocs: len(ctx.Documents),
		},
	}

	startTime := time.Now()

	// Run validation messages first
	validationMessages := v.Validate(ctx)
	result.Messages = append(result.Messages, validationMessages...)

	// Track validation errors
	totalErrors := 0
	docErrors := make(map[string][]string)

	// Create a map of dimension names for faster lookup
	dimensionMap := make(map[string]types.DimensionConfig)
	for _, dim := range ctx.Config.Dimensions {
		dimensionMap[dim.Name] = dim
	}

	// Validate each document
	for i := range ctx.Documents {
		doc := &ctx.Documents[i]
		docErrorList := []string{}

		// Check for unknown dimensions (fields not in config that aren't data fields)
		for fieldName := range doc.Dimensions {
			// Skip data field keys
			if len(fieldName) >= 6 && fieldName[:6] == "_data." {
				continue
			}

			if dimConfig, isDimension := dimensionMap[fieldName]; isDimension {
				// Validate dimension value
				val := doc.Dimensions[fieldName]

				// Check if it's a valid dimension type (simple type)
				if err := validation.ValidateSimpleType(val, fieldName); err != nil {
					docErrorList = append(docErrorList, fmt.Sprintf("%s: %v", fieldName, err))
					totalErrors++
				}

				// If it's an enumerated dimension, validate against allowed values
				if dimConfig.Type == types.Enumerated && len(dimConfig.Values) > 0 {
					found := false
					valStr := fmt.Sprintf("%v", val)
					for _, allowed := range dimConfig.Values {
						if valStr == allowed {
							found = true
							break
						}
					}
					if !found {
						docErrorList = append(docErrorList, fmt.Sprintf("%s: value '%v' not in allowed values %v", fieldName, val, dimConfig.Values))
						totalErrors++
					}
				}
			} else {
				// Unknown dimension
				docErrorList = append(docErrorList, fmt.Sprintf("%s: unknown dimension (not defined in config)", fieldName))
				totalErrors++
			}
		}

		// Check for missing required dimensions with defaults
		for _, dimConfig := range ctx.Config.Dimensions {
			if _, exists := doc.Dimensions[dimConfig.Name]; !exists {
				// For enumerated dimensions with defaults, this might be ok
				// but we'll warn about it
				if dimConfig.Type == types.Enumerated && dimConfig.DefaultValue != "" {
					// This is ok, default will be used
				} else if dimConfig.Type == types.Hierarchical {
					// Hierarchical dimensions are typically optional
				} else {
					// No default and missing - this could be a problem
					docErrorList = append(docErrorList, fmt.Sprintf("%s: dimension is missing (no default value)", dimConfig.Name))
					totalErrors++
				}
			}
		}

		if len(docErrorList) > 0 {
			docErrors[doc.UUID] = docErrorList
			result.Stats.SkippedDocs++ // Count as "skipped" since they have errors
		}
	}

	result.Stats.Duration = time.Since(startTime)
	result.Stats.ModifiedDocs = len(ctx.Documents) - result.Stats.SkippedDocs // Valid docs

	// Report results
	if totalErrors > 0 {
		result.Success = false
		result.Code = CodeValidationError

		// Create error details
		errorSamples := []map[string]interface{}{}
		count := 0
		for uuid, errors := range docErrors {
			if count < 5 { // Show first 5 documents with errors
				errorSamples = append(errorSamples, map[string]interface{}{
					"document_id": uuid,
					"errors":      errors,
				})
				count++
			}
		}

		result.Messages = append(result.Messages, Message{
			Level: LevelError,
			Text:  fmt.Sprintf("Found %d validation errors in %d documents", totalErrors, len(docErrors)),
			Details: map[string]interface{}{
				"total_errors":  totalErrors,
				"failed_docs":   len(docErrors),
				"error_samples": errorSamples,
			},
		})
	} else {
		result.Messages = append(result.Messages, Message{
			Level: LevelInfo,
			Text:  fmt.Sprintf("All %d documents passed validation", len(ctx.Documents)),
		})
	}

	return result
}
