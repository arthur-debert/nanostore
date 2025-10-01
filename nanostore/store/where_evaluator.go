package store

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/types"
)

// WhereEvaluator provides safe evaluation of WHERE clauses against documents.
//
// This implementation focuses on security by:
// 1. Parsing WHERE clauses BEFORE parameter substitution to prevent injection attacks
// 2. Using parameterized queries with safe parameter binding
// 3. Avoiding arbitrary code execution or SQL evaluation
// 4. Supporting only a limited, safe set of operators
//
// Security Design:
// - The clause structure is parsed first, establishing the query plan
// - Parameters are bound safely during evaluation, not during parsing
// - This prevents injection attacks where malicious parameters could alter the query structure
//
// Supported operators: =, !=, >, >=, <, <=, LIKE, NOT LIKE, IS NULL, IS NOT NULL
// Supported logic: AND (OR is not supported for security simplicity)
type WhereEvaluator struct {
	whereClause string        // The WHERE clause template with ? placeholders
	args        []interface{} // Parameter values to bind to ? placeholders
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

// evaluateClause safely evaluates a WHERE clause against a document.
//
// Security-first approach:
// 1. Parse the clause structure first to establish the query plan
// 2. Validate parameter count matches placeholder count
// 3. Evaluate conditions with safe parameter binding
//
// This prevents injection attacks by ensuring the query structure
// cannot be modified by parameter values.
func (we *WhereEvaluator) evaluateClause(doc *types.Document) (bool, error) {
	// Parse the clause into conditions FIRST, before parameter substitution
	// This prevents injection attacks where parameters could modify the clause structure
	conditions, err := we.parseConditions(we.whereClause)
	if err != nil {
		return false, fmt.Errorf("clause parsing failed: %w", err)
	}

	// Evaluate all conditions with safe parameter binding
	return we.evaluateConditions(doc, conditions)
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

// Condition represents a single condition in the WHERE clause.
//
// Each condition represents one comparison operation like "status = ?" or "priority = 'high'".
// Conditions are connected by AND operations (OR is not supported for security simplicity).
type Condition struct {
	Field       string // Document field name (e.g., "status", "_data.assignee")
	Operator    string // Comparison operator (=, !=, >, >=, <, <=, like, not like, etc.)
	Value       string // Expected value (literal) or placeholder for parameter binding
	IsData      bool   // true if this is a _data.* field (custom user data)
	IsParameter bool   // true if value is a ? parameter that needs binding
	ParamIndex  int    // index into args array if IsParameter is true
}

// parseConditions parses a WHERE clause into individual conditions.
//
// Parsing Strategy:
// 1. Normalize whitespace while preserving field name case
// 2. Validate parameter count matches placeholder count (security check)
// 3. Split by AND keywords (case-insensitive)
// 4. Parse each condition individually with parameter tracking
//
// This approach ensures the query structure is established before any parameter
// values are considered, preventing injection attacks.
func (we *WhereEvaluator) parseConditions(clause string) ([]Condition, error) {
	// This is a simplified parser that handles basic conditions
	// Supports: field = value, field != value, field > value, etc.
	// Also supports AND operations (OR is not implemented for safety)

	// Normalize whitespace but preserve case for field names
	clause = regexp.MustCompile(`\s+`).ReplaceAllString(clause, " ")
	clause = strings.TrimSpace(clause)

	// Validate parameter count first
	placeholderCount := strings.Count(clause, "?")

	// Filter out nil arguments at the end (common pattern when people add nil as safety)
	filteredArgs := we.args
	for len(filteredArgs) > 0 && filteredArgs[len(filteredArgs)-1] == nil && placeholderCount < len(filteredArgs) {
		filteredArgs = filteredArgs[:len(filteredArgs)-1]
	}

	if placeholderCount != len(filteredArgs) {
		return nil, fmt.Errorf("placeholder count (%d) doesn't match argument count (%d)", placeholderCount, len(filteredArgs))
	}

	// Update args to the filtered version
	we.args = filteredArgs

	// Split by AND (case insensitive)
	// We need to handle this carefully to preserve field name case
	parts := we.splitByAND(clause)
	conditions := make([]Condition, 0, len(parts))
	currentParamIndex := 0

	for _, part := range parts {
		condition, paramCount, err := we.parseConditionWithParamTracking(strings.TrimSpace(part), currentParamIndex)
		if err != nil {
			return nil, fmt.Errorf("parsing condition '%s': %w", part, err)
		}
		conditions = append(conditions, condition)
		currentParamIndex += paramCount
	}

	return conditions, nil
}

// splitByAND splits a clause by AND keyword (case insensitive) while preserving field name case
func (we *WhereEvaluator) splitByAND(clause string) []string {
	// Use a case-insensitive regex to split by " AND "
	re := regexp.MustCompile(`(?i)\s+and\s+`)
	return re.Split(clause, -1)
}

// parseConditionWithParamTracking parses a single condition and tracks parameter usage
func (we *WhereEvaluator) parseConditionWithParamTracking(condition string, startParamIndex int) (Condition, int, error) {
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

	paramCount := 0 // Track how many parameters this condition uses

	for _, opInfo := range operators {
		// Search for operator case-insensitively
		lowerCondition := strings.ToLower(condition)
		if idx := strings.Index(lowerCondition, opInfo.searchOp); idx > 0 {
			field := strings.TrimSpace(condition[:idx])
			value := strings.TrimSpace(condition[idx+len(opInfo.searchOp):])

			isParameter := false
			paramIndex := startParamIndex

			// Check if value is a parameter placeholder
			if value == "?" {
				isParameter = true
				paramCount = 1
			} else {
				// Remove quotes from literal value if present
				if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
					value = value[1 : len(value)-1]
				}
			}

			isData := strings.HasPrefix(field, "_data.")

			return Condition{
				Field:       field,
				Operator:    strings.TrimSpace(opInfo.searchOp), // Use lowercase operator for consistency
				Value:       value,
				IsData:      isData,
				IsParameter: isParameter,
				ParamIndex:  paramIndex,
			}, paramCount, nil
		}
	}

	return Condition{}, 0, fmt.Errorf("no valid operator found in condition: %s", condition)
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

	// Get the expected value (either literal or from parameters)
	expectedValue := condition.Value
	if condition.IsParameter {
		// Safely bind parameter value
		if condition.ParamIndex >= len(we.args) {
			return false, fmt.Errorf("parameter index %d out of range (have %d args)", condition.ParamIndex, len(we.args))
		}

		// Format the parameter value safely
		formattedValue, err := we.formatArgument(we.args[condition.ParamIndex])
		if err != nil {
			return false, fmt.Errorf("failed to format parameter %d: %w", condition.ParamIndex, err)
		}

		// Remove quotes that formatArgument adds for string comparison
		if strings.HasPrefix(formattedValue, "'") && strings.HasSuffix(formattedValue, "'") {
			expectedValue = formattedValue[1 : len(formattedValue)-1]
		} else {
			expectedValue = formattedValue
		}
	}

	// Compare based on operator
	return we.compareValues(actualValue, condition.Operator, expectedValue)
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

// compareNumerically attempts numeric comparison, falls back to lexicographic string comparison
func (we *WhereEvaluator) compareNumerically(actual, expected string, cmp func(float64, float64) bool) (bool, error) {
	actualNum, err1 := strconv.ParseFloat(actual, 64)
	expectedNum, err2 := strconv.ParseFloat(expected, 64)

	// If both values can be parsed as numbers, do numeric comparison
	if err1 == nil && err2 == nil {
		return cmp(actualNum, expectedNum), nil
	}

	// Fall back to lexicographic string comparison
	// Convert string comparison result to numeric form for the comparison function
	stringCmp := strings.Compare(actual, expected)

	// Map string comparison result to float values that make sense for the comparison
	switch {
	case stringCmp > 0:
		// actual > expected in lexicographic order
		return cmp(1.0, 0.0), nil
	case stringCmp < 0:
		// actual < expected in lexicographic order
		return cmp(-1.0, 0.0), nil
	default:
		// actual == expected
		return cmp(0.0, 0.0), nil
	}
}

// matchLike implements simple LIKE pattern matching with proper regex escaping
func (we *WhereEvaluator) matchLike(actual, pattern string) (bool, error) {
	// Convert SQL LIKE pattern to regex
	// % matches any sequence of characters
	// _ matches any single character
	// We need to escape all regex special characters except % and _

	// Replace % and _ with temporary placeholders first
	tempPattern := strings.ReplaceAll(pattern, "%", "\x00PERCENT\x00")
	tempPattern = strings.ReplaceAll(tempPattern, "_", "\x00UNDERSCORE\x00")

	// Escape all regex special characters
	escaped := regexp.QuoteMeta(tempPattern)

	// Replace our placeholders with the correct regex equivalents
	regexPattern := strings.ReplaceAll(escaped, "\x00PERCENT\x00", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "\x00UNDERSCORE\x00", ".")
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, actual)
	if err != nil {
		return false, fmt.Errorf("invalid LIKE pattern '%s': %w", pattern, err)
	}

	return matched, nil
}
