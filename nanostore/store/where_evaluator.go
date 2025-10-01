package store

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// WhereEvaluator provides safe evaluation of WHERE clauses against documents
// This implementation focuses on security by using parameterized queries
// and avoiding arbitrary code execution
type WhereEvaluator struct {
	whereClause string
	args        []interface{}
}

// NewWhereEvaluator creates a new WHERE clause evaluator
func NewWhereEvaluator(whereClause string, args ...interface{}) *WhereEvaluator {
	return &WhereEvaluator{
		whereClause: strings.TrimSpace(whereClause),
		args:        args,
	}
}

// EvaluateDocument checks if a document matches the WHERE clause
func (we *WhereEvaluator) EvaluateDocument(doc *types.Document) (bool, error) {
	if we.whereClause == "" {
		return true, nil // Empty clause matches everything
	}

	// Parse and evaluate the WHERE clause safely
	return we.evaluateClause(doc)
}

// evaluateClause safely evaluates a WHERE clause against a document
func (we *WhereEvaluator) evaluateClause(doc *types.Document) (bool, error) {
	// Replace parameters with actual values first
	processedClause, err := we.substituteParameters()
	if err != nil {
		return false, fmt.Errorf("parameter substitution failed: %w", err)
	}

	// Parse the clause into conditions
	conditions, err := we.parseConditions(processedClause)
	if err != nil {
		return false, fmt.Errorf("clause parsing failed: %w", err)
	}

	// Evaluate all conditions
	return we.evaluateConditions(doc, conditions)
}

// substituteParameters safely replaces ? placeholders with actual values
func (we *WhereEvaluator) substituteParameters() (string, error) {
	if len(we.args) == 0 {
		return we.whereClause, nil
	}

	// Count placeholders
	placeholderCount := strings.Count(we.whereClause, "?")

	// If no placeholders, ignore the arguments (they might be nil or unused)
	if placeholderCount == 0 {
		return we.whereClause, nil
	}

	// Filter out nil arguments at the end (common pattern when people add nil as safety)
	filteredArgs := we.args
	for len(filteredArgs) > 0 && filteredArgs[len(filteredArgs)-1] == nil && placeholderCount < len(filteredArgs) {
		filteredArgs = filteredArgs[:len(filteredArgs)-1]
	}

	if placeholderCount != len(filteredArgs) {
		return "", fmt.Errorf("placeholder count (%d) doesn't match argument count (%d)",
			placeholderCount, len(filteredArgs))
	}

	clause := we.whereClause
	for i, arg := range filteredArgs {
		// Convert argument to safe string representation
		value, err := we.formatArgument(arg)
		if err != nil {
			return "", fmt.Errorf("formatting argument %d failed: %w", i, err)
		}

		// Replace first occurrence of ?
		clause = strings.Replace(clause, "?", value, 1)
	}

	return clause, nil
}

// formatArgument safely formats an argument for inclusion in the clause
func (we *WhereEvaluator) formatArgument(arg interface{}) (string, error) {
	if arg == nil {
		return "NULL", nil
	}

	switch v := arg.(type) {
	case string:
		// Escape single quotes and wrap in quotes
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case bool:
		if v {
			return "TRUE", nil
		}
		return "FALSE", nil
	case time.Time:
		return fmt.Sprintf("'%s'", v.Format(time.RFC3339)), nil
	default:
		// For any other type, convert to string and treat as string
		return fmt.Sprintf("'%s'", fmt.Sprintf("%v", v)), nil
	}
}

// Condition represents a single condition in the WHERE clause
type Condition struct {
	Field    string
	Operator string
	Value    string
	IsData   bool // true if this is a _data.* field
}

// parseConditions parses a WHERE clause into individual conditions
func (we *WhereEvaluator) parseConditions(clause string) ([]Condition, error) {
	// This is a simplified parser that handles basic conditions
	// Supports: field = value, field != value, field > value, etc.
	// Also supports AND operations (OR is not implemented for safety)

	// Normalize whitespace but preserve case for field names
	clause = regexp.MustCompile(`\s+`).ReplaceAllString(clause, " ")
	clause = strings.TrimSpace(clause)

	// Split by AND (case insensitive)
	// We need to handle this carefully to preserve field name case
	parts := we.splitByAND(clause)
	conditions := make([]Condition, 0, len(parts))

	for _, part := range parts {
		condition, err := we.parseCondition(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("parsing condition '%s': %w", part, err)
		}
		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// splitByAND splits a clause by AND keyword (case insensitive) while preserving field name case
func (we *WhereEvaluator) splitByAND(clause string) []string {
	// Use a case-insensitive regex to split by " AND "
	re := regexp.MustCompile(`(?i)\s+and\s+`)
	return re.Split(clause, -1)
}

// parseCondition parses a single condition
func (we *WhereEvaluator) parseCondition(condition string) (Condition, error) {
	// Supported operators in order of precedence (longest first)
	// We need to handle case insensitive operators
	operators := []struct {
		op       string
		searchOp string // case-insensitive version for searching
	}{
		{" IS NOT NULL", " is not null"}, // Must come before " IS "
		{" IS NULL", " is null"},
		{" NOT LIKE ", " not like "}, // Must come before " LIKE "
		{"!=", "!="},
		{"<=", "<="},
		{">=", ">="},
		{"=", "="},
		{"<", "<"},
		{">", ">"},
		{" LIKE ", " like "},
	}

	for _, opInfo := range operators {
		// Search for operator case-insensitively
		lowerCondition := strings.ToLower(condition)
		if idx := strings.Index(lowerCondition, opInfo.searchOp); idx > 0 {
			field := strings.TrimSpace(condition[:idx])
			value := strings.TrimSpace(condition[idx+len(opInfo.searchOp):])

			// Remove quotes from value if present
			if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
				value = value[1 : len(value)-1]
			}

			isData := strings.HasPrefix(field, "_data.")

			return Condition{
				Field:    field,
				Operator: strings.TrimSpace(opInfo.searchOp), // Use lowercase operator for consistency
				Value:    value,
				IsData:   isData,
			}, nil
		}
	}

	return Condition{}, fmt.Errorf("no valid operator found in condition: %s", condition)
}

// evaluateConditions evaluates all conditions against a document
func (we *WhereEvaluator) evaluateConditions(doc *types.Document, conditions []Condition) (bool, error) {
	for _, condition := range conditions {
		match, err := we.evaluateCondition(doc, condition)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil // AND logic: all must match
		}
	}
	return true, nil
}

// evaluateCondition evaluates a single condition against a document
func (we *WhereEvaluator) evaluateCondition(doc *types.Document, condition Condition) (bool, error) {
	// Get the actual value from the document
	actualValue, err := we.getDocumentValue(doc, condition.Field)
	if err != nil {
		return false, err
	}

	// Compare based on operator
	return we.compareValues(actualValue, condition.Operator, condition.Value)
}

// getDocumentValue extracts a field value from a document
func (we *WhereEvaluator) getDocumentValue(doc *types.Document, field string) (interface{}, error) {
	switch field {
	case "uuid":
		return doc.UUID, nil
	case "simple_id":
		return doc.SimpleID, nil
	case "title":
		return doc.Title, nil
	case "body":
		return doc.Body, nil
	case "created_at":
		return doc.CreatedAt, nil
	case "updated_at":
		return doc.UpdatedAt, nil
	default:
		// Check dimensions and data fields
		if value, exists := doc.Dimensions[field]; exists {
			return value, nil
		}
		// Field not found
		return nil, nil
	}
}

// compareValues compares two values using the specified operator
func (we *WhereEvaluator) compareValues(actual interface{}, operator, expected string) (bool, error) {
	// Handle NULL comparisons
	if actual == nil {
		switch operator {
		case "=":
			return expected == "null", nil
		case "!=":
			return expected != "null", nil
		default:
			return false, nil // NULL comparisons with other operators are false
		}
	}

	// Convert actual value to string for comparison
	actualStr := fmt.Sprintf("%v", actual)

	switch operator {
	case "=":
		return we.compareStrings(actualStr, expected), nil
	case "!=":
		return !we.compareStrings(actualStr, expected), nil
	case ">":
		return we.compareNumerically(actualStr, expected, func(a, b float64) bool { return a > b })
	case ">=":
		return we.compareNumerically(actualStr, expected, func(a, b float64) bool { return a >= b })
	case "<":
		return we.compareNumerically(actualStr, expected, func(a, b float64) bool { return a < b })
	case "<=":
		return we.compareNumerically(actualStr, expected, func(a, b float64) bool { return a <= b })
	case "like":
		return we.matchLike(actualStr, expected)
	case "not like":
		match, err := we.matchLike(actualStr, expected)
		return !match, err
	case "is null":
		return actual == nil, nil
	case "is not null":
		return actual != nil, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareStrings compares two strings, handling boolean values case-insensitively
func (we *WhereEvaluator) compareStrings(actual, expected string) bool {
	// Handle boolean comparisons case-insensitively
	if (strings.ToLower(actual) == "true" || strings.ToLower(actual) == "false") &&
		(strings.ToLower(expected) == "true" || strings.ToLower(expected) == "false") {
		return strings.EqualFold(actual, expected)
	}

	// Regular string comparison
	return actual == expected
}

// compareNumerically attempts numeric comparison, falls back to string comparison
func (we *WhereEvaluator) compareNumerically(actual, expected string, cmp func(float64, float64) bool) (bool, error) {
	actualNum, err1 := strconv.ParseFloat(actual, 64)
	expectedNum, err2 := strconv.ParseFloat(expected, 64)

	if err1 == nil && err2 == nil {
		return cmp(actualNum, expectedNum), nil
	}

	// Fall back to string comparison
	if actual > expected {
		return cmp(1, 0), nil
	} else if actual < expected {
		return cmp(-1, 0), nil
	} else {
		return cmp(0, 0), nil
	}
}

// matchLike implements simple LIKE pattern matching
func (we *WhereEvaluator) matchLike(actual, pattern string) (bool, error) {
	// Convert SQL LIKE pattern to regex
	// % matches any sequence of characters
	// _ matches any single character
	regexPattern := strings.ReplaceAll(pattern, "%", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "_", ".")
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, actual)
	if err != nil {
		return false, fmt.Errorf("invalid LIKE pattern '%s': %w", pattern, err)
	}

	return matched, nil
}
