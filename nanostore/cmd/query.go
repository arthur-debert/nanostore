package main

import "strings"

// LogicalOperator defines the operator connecting filter groups (e.g., AND, OR, SQL, DATA).
type LogicalOperator string

const (
	OpAnd  LogicalOperator = "AND"
	OpOr   LogicalOperator = "OR"
	OpSQL  LogicalOperator = "SQL"
	OpData LogicalOperator = "DATA"
)

// FilterCondition represents a single filter condition, like 'status = "active"'.
type FilterCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// FilterGroup represents a set of conditions that are implicitly joined by AND.
type FilterGroup struct {
	Conditions []FilterCondition
}

// Query represents a parsed CLI query with support for logical grouping.
type Query struct {
	// A list of filter groups.
	Groups []FilterGroup
	// A list of logical operators that connect the groups.
	// Example: Groups[0] Operators[0] Groups[1] Operators[1] Groups[2]
	Operators []LogicalOperator
}

// parseFilters takes a slice of filter arguments and parses them into a Query object,
// handling logical operators for grouping.
func parseFilters(filterArgs []string) *Query {
	query := &Query{
		// Initialize with empty, non-nil slices
		Groups:    []FilterGroup{},
		Operators: []LogicalOperator{},
	}

	// Start with the first group, ensuring its Conditions slice is also non-nil
	currentGroup := FilterGroup{Conditions: []FilterCondition{}}

	for _, arg := range filterArgs {
		cleanArg := strings.TrimPrefix(arg, "--")

		// Check for logical operators
		if cleanArg == "and" {
			query.Groups = append(query.Groups, currentGroup)
			query.Operators = append(query.Operators, OpAnd)
			currentGroup = FilterGroup{Conditions: []FilterCondition{}} // Start a new group
			continue
		}
		if cleanArg == "or" {
			query.Groups = append(query.Groups, currentGroup)
			query.Operators = append(query.Operators, OpOr)
			currentGroup = FilterGroup{Conditions: []FilterCondition{}} // Start a new group
			continue
		}
		if cleanArg == "sql" {
			query.Groups = append(query.Groups, currentGroup)
			query.Operators = append(query.Operators, OpSQL)
			currentGroup = FilterGroup{Conditions: []FilterCondition{}} // Start a new group
			continue
		}
		if cleanArg == "data" {
			query.Groups = append(query.Groups, currentGroup)
			query.Operators = append(query.Operators, OpData)
			currentGroup = FilterGroup{Conditions: []FilterCondition{}}
			continue
		}

		// It's a filter condition
		parts := strings.SplitN(arg, "=", 2)
		flag := strings.TrimPrefix(parts[0], "--")
		var value string
		if len(parts) == 2 {
			value = parts[1]
		}

		fieldParts := strings.SplitN(flag, "__", 2)
		condition := FilterCondition{Value: value}
		if len(fieldParts) == 2 {
			condition.Field = fieldParts[0]
			condition.Operator = fieldParts[1]
		} else {
			condition.Field = fieldParts[0]
			condition.Operator = "eq"
		}

		currentGroup.Conditions = append(currentGroup.Conditions, condition)
	}

	// Add the final group to the query
	query.Groups = append(query.Groups, currentGroup)

	return query
}
