package main

import (
	"fmt"
	"strings"
)

// CLIError represents a user-friendly CLI error with context and suggestions
type CLIError struct {
	Operation   string   // The operation that failed (e.g., "create", "list", "get")
	Cause       string   // The underlying cause (e.g., "document not found")
	Details     string   // Additional technical details
	Suggestions []string // Helpful suggestions for the user
	Underlying  error    // Original error for debugging
}

// Error implements the error interface
func (e *CLIError) Error() string {
	var msg strings.Builder

	// Start with operation context
	if e.Operation != "" {
		msg.WriteString(fmt.Sprintf("Failed to %s", e.Operation))
	} else {
		msg.WriteString("Operation failed")
	}

	// Add the main cause
	if e.Cause != "" {
		msg.WriteString(fmt.Sprintf(": %s", e.Cause))
	}

	// Add technical details if available
	if e.Details != "" {
		msg.WriteString(fmt.Sprintf(" (%s)", e.Details))
	}

	// Add suggestions
	if len(e.Suggestions) > 0 {
		msg.WriteString("\n\nSuggestions:")
		for i, suggestion := range e.Suggestions {
			msg.WriteString(fmt.Sprintf("\n  %d. %s", i+1, suggestion))
		}
	}

	return msg.String()
}

// Unwrap returns the underlying error for error chain compatibility
func (e *CLIError) Unwrap() error {
	return e.Underlying
}

// Error constructors for common CLI error scenarios

// NewValidationError creates an error for validation failures
func NewValidationError(operation, field, value string, suggestions ...string) *CLIError {
	return &CLIError{
		Operation:   operation,
		Cause:       fmt.Sprintf("invalid %s: %q", field, value),
		Suggestions: suggestions,
	}
}

// NewNotFoundError creates an error for missing resources
func NewNotFoundError(operation, resource, id string, suggestions ...string) *CLIError {
	return &CLIError{
		Operation:   operation,
		Cause:       fmt.Sprintf("%s with ID %q not found", resource, id),
		Suggestions: suggestions,
	}
}

// NewConfigError creates an error for configuration issues
func NewConfigError(operation, issue string, suggestions ...string) *CLIError {
	return &CLIError{
		Operation:   operation,
		Cause:       fmt.Sprintf("configuration error: %s", issue),
		Suggestions: suggestions,
	}
}

// NewStoreError creates an error for store-related issues
func NewStoreError(operation string, underlying error, suggestions ...string) *CLIError {
	cause := "store operation failed"
	details := ""

	if underlying != nil {
		details = underlying.Error()

		// Provide more user-friendly descriptions for common errors
		errStr := strings.ToLower(underlying.Error())
		switch {
		case strings.Contains(errStr, "no such file"):
			cause = "database file not found"
		case strings.Contains(errStr, "permission denied"):
			cause = "insufficient permissions to access database"
		case strings.Contains(errStr, "database is locked"):
			cause = "database is currently locked by another process"
		case strings.Contains(errStr, "not found"):
			cause = "resource not found"
		case strings.Contains(errStr, "invalid"):
			cause = "invalid data provided"
		}
	}

	return &CLIError{
		Operation:   operation,
		Cause:       cause,
		Details:     details,
		Suggestions: suggestions,
		Underlying:  underlying,
	}
}

// NewTypeError creates an error for type-related issues
func NewTypeError(operation, typeName string, availableTypes []string) *CLIError {
	suggestions := []string{
		"Use --type flag to specify a valid type",
		"Run 'nano-db types' to see available types",
	}

	if len(availableTypes) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Available types: %s", strings.Join(availableTypes, ", ")))
	}

	return &CLIError{
		Operation:   operation,
		Cause:       fmt.Sprintf("invalid or missing document type: %q", typeName),
		Suggestions: suggestions,
	}
}

// NewFilterError creates an error for filtering issues
func NewFilterError(operation, filter, issue string) *CLIError {
	suggestions := []string{
		"Use format: --field=value or --field__operator=value",
		"Available operators: eq, ne, gt, lt, gte, lte, contains, startswith, endswith",
		"Use --or to combine conditions with OR logic",
		"Check field names match your document schema",
	}

	return &CLIError{
		Operation:   operation,
		Cause:       fmt.Sprintf("invalid filter %q: %s", filter, issue),
		Suggestions: suggestions,
	}
}

// WrapError wraps an existing error with CLI-friendly context
func WrapError(operation string, err error, suggestions ...string) error {
	if err == nil {
		return nil
	}

	// If it's already a CLIError, just update the operation
	if cliErr, ok := err.(*CLIError); ok {
		if cliErr.Operation == "" {
			cliErr.Operation = operation
		}
		return cliErr
	}

	return NewStoreError(operation, err, suggestions...)
}

// Common error messages and suggestions
var (
	CommonSuggestions = struct {
		CheckType   string
		CheckDB     string
		CheckID     string
		CheckConfig string
		CheckFlags  string
		RunHelp     string
		CheckPerms  string
		TryDryRun   string
	}{
		CheckType:   "Verify --type flag matches your document type",
		CheckDB:     "Verify --db flag points to a valid database file",
		CheckID:     "Verify the document ID exists (try 'list' command first)",
		CheckConfig: "Check your configuration file or environment variables",
		CheckFlags:  "Check command line flags and their values",
		RunHelp:     "Run command with --help for usage information",
		CheckPerms:  "Check file permissions and directory access",
		TryDryRun:   "Use --dry-run to preview the operation",
	}
)
