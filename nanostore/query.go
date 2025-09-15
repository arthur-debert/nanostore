package nanostore

import (
	"fmt"
	"sort"
	"strings"
)

// queryBuilder generates dynamic SQL queries based on dimension configuration
type queryBuilder struct {
	config Config
}

// newQueryBuilder creates a new query builder for the given configuration
func newQueryBuilder(config Config) *queryBuilder {
	return &queryBuilder{config: config}
}

// GenerateListQuery creates a SQL query for listing documents with generated IDs.
// This is the core of the configurable ID generation system.
//
// The function generates different query strategies based on the presence of filters:
// 1. Flat Query (with filters): Simple query with WHERE clauses for performance
// 2. Hierarchical Query (no filters): Complex recursive CTE that generates tree structure
//
// ID Generation Strategy:
// - Uses ROW_NUMBER() OVER (PARTITION BY dimensions ORDER BY created_at)
// - Partitioning ensures contiguous IDs within each dimension combination
// - Example: status="done" documents get IDs d1, d2, d3... regardless of creation gaps
// - Hierarchical documents get IDs like 1.1, 1.2, 2.1 based on their parent context
//
// Performance Considerations:
// - Flat queries are O(log n) with proper indexing on dimension columns
// - Hierarchical queries are O(n log n) due to recursive CTE traversal
// - Query complexity scales with number of configured dimensions
func (qb *queryBuilder) GenerateListQuery(filters map[string]interface{}) (string, []interface{}, error) {
	// Get dimension information
	enumDims := qb.config.GetEnumeratedDimensions()
	hierDim := qb.findHierarchicalDimension()

	// Check if we should use flat listing (when filters are present)
	// Any filters present means we should use flat listing instead of hierarchical
	hasFilters := len(filters) > 0

	// When we have filters, use a simpler query without hierarchy
	if hasFilters {
		return qb.generateFlatListQuery(enumDims, hierDim, filters)
	}

	// Build the WITH RECURSIVE clause for hierarchical + enumerated dimensions
	var cteBuilder strings.Builder
	cteBuilder.WriteString("WITH RECURSIVE ")

	// Generate base query for root documents
	rootQuery := qb.generateRootQuery(enumDims, filters)
	cteBuilder.WriteString("root_docs AS (\n")
	cteBuilder.WriteString(rootQuery)
	cteBuilder.WriteString("\n),\n")

	// Generate child documents query if we have hierarchical dimension
	if hierDim != nil {
		childQuery := qb.generateChildQuery(enumDims, hierDim)
		cteBuilder.WriteString("child_docs AS (\n")
		cteBuilder.WriteString(childQuery)
		cteBuilder.WriteString("\n),\n")

		// Generate recursive tree builder
		treeQuery := qb.generateTreeQuery(hierDim)
		cteBuilder.WriteString("id_tree AS (\n")
		cteBuilder.WriteString(treeQuery)
		cteBuilder.WriteString("\n)\n")
	} else {
		// No hierarchy, just use root docs
		cteBuilder.WriteString("id_tree AS (\n")
		cteBuilder.WriteString("    SELECT * FROM root_docs\n")
		cteBuilder.WriteString(")\n")
	}

	// Final SELECT from the tree
	cteBuilder.WriteString("SELECT uuid, user_facing_id, title, body, ")

	// Add dimension columns
	for _, dim := range enumDims {
		cteBuilder.WriteString(dim.Name)
		cteBuilder.WriteString(", ")
	}
	if hierDim != nil {
		cteBuilder.WriteString(hierDim.RefField)
		cteBuilder.WriteString(", ")
	}

	cteBuilder.WriteString("created_at, updated_at\n")
	cteBuilder.WriteString("FROM id_tree\n")

	// Add WHERE clauses for filters
	whereClauses, args := qb.buildWhereClausesAndArgs(filters, hierDim)
	if len(whereClauses) > 0 {
		cteBuilder.WriteString("WHERE ")
		cteBuilder.WriteString(strings.Join(whereClauses, " AND "))
		cteBuilder.WriteString("\n")
	}

	// Order by depth for hierarchical display
	if hierDim != nil {
		cteBuilder.WriteString("ORDER BY depth, created_at")
	} else {
		cteBuilder.WriteString("ORDER BY created_at")
	}

	return cteBuilder.String(), args, nil
}

// generateRootQuery creates the base query for root documents (documents with no parent).
// This query generates contiguous IDs starting from 1 within each dimension partition.
//
// ID Generation Logic:
// - Uses ROW_NUMBER() OVER (PARTITION BY dimensions ORDER BY created_at) for contiguous numbering
// - Applies prefixes based on dimension values (e.g., status="done" gets "d" prefix)
// - Multiple dimensions are sorted alphabetically (e.g., "high priority + done" = "hd1")
//
// Example SQL output for status+priority dimensions:
//
//	CASE
//	  WHEN status = 'done' AND priority = 'high' THEN 'hd' || ROW_NUMBER() OVER (...)
//	  WHEN status = 'done' THEN 'd' || ROW_NUMBER() OVER (...)
//	  WHEN priority = 'high' THEN 'h' || ROW_NUMBER() OVER (...)
//	  ELSE CAST(ROW_NUMBER() OVER (...) AS TEXT)
//	END
func (qb *queryBuilder) generateRootQuery(enumDims []DimensionConfig, filters map[string]interface{}) string {
	var query strings.Builder

	query.WriteString("    SELECT\n")
	query.WriteString("        uuid, title, body, created_at, updated_at,\n")

	// Add dimension columns
	for _, dim := range enumDims {
		query.WriteString("        ")
		query.WriteString(dim.Name)
		query.WriteString(",\n")
	}

	// Generate CASE statement for user-facing ID with configurable prefixes
	query.WriteString("        ")
	query.WriteString(qb.generateIDExpression(enumDims, true))
	query.WriteString(" as user_facing_id\n")

	query.WriteString("    FROM documents\n")

	// For root documents, check if hierarchical dimension is NULL
	hierDim := qb.findHierarchicalDimension()
	if hierDim != nil {
		query.WriteString("    WHERE ")
		query.WriteString(hierDim.RefField)
		query.WriteString(" IS NULL")
	}

	return query.String()
}

// generateChildQuery creates the query for child documents with local ID generation
func (qb *queryBuilder) generateChildQuery(enumDims []DimensionConfig, hierDim *DimensionConfig) string {
	var query strings.Builder

	query.WriteString("    SELECT\n")
	query.WriteString("        uuid, title, body, created_at, updated_at,\n")
	query.WriteString("        ")
	query.WriteString(hierDim.RefField)
	query.WriteString(",\n")

	// Add dimension columns
	for _, dim := range enumDims {
		query.WriteString("        ")
		query.WriteString(dim.Name)
		query.WriteString(",\n")
	}

	// Generate local ID for children
	query.WriteString("        ")
	query.WriteString(qb.generateIDExpression(enumDims, false))
	query.WriteString(" as local_id\n")

	query.WriteString("    FROM documents\n")
	query.WriteString("    WHERE ")
	query.WriteString(hierDim.RefField)
	query.WriteString(" IS NOT NULL")

	return query.String()
}

// generateTreeQuery creates the recursive CTE for building the hierarchical tree
func (qb *queryBuilder) generateTreeQuery(hierDim *DimensionConfig) string {
	var query strings.Builder

	// Base case: root documents
	query.WriteString("    -- Base case: root documents\n")
	query.WriteString("    SELECT\n")
	query.WriteString("        uuid, title, body, created_at, updated_at,\n")

	// Include all dimension columns in tree
	enumDims := qb.config.GetEnumeratedDimensions()
	for _, dim := range enumDims {
		query.WriteString("        ")
		query.WriteString(dim.Name)
		query.WriteString(",\n")
	}

	query.WriteString("        NULL as ")
	query.WriteString(hierDim.RefField)
	query.WriteString(",\n")
	query.WriteString("        0 as depth,\n")
	query.WriteString("        user_facing_id\n")
	query.WriteString("    FROM root_docs\n")

	query.WriteString("    \n    UNION ALL\n    \n")

	// Recursive case: children
	query.WriteString("    -- Recursive case: children with concatenated IDs\n")
	query.WriteString("    SELECT\n")
	query.WriteString("        c.uuid, c.title, c.body, c.created_at, c.updated_at,\n")

	for _, dim := range enumDims {
		query.WriteString("        c.")
		query.WriteString(dim.Name)
		query.WriteString(",\n")
	}

	query.WriteString("        c.")
	query.WriteString(hierDim.RefField)
	query.WriteString(",\n")
	query.WriteString("        p.depth + 1,\n")
	query.WriteString("        p.user_facing_id || '.' || c.local_id as user_facing_id\n")
	query.WriteString("    FROM child_docs c\n")
	query.WriteString("    INNER JOIN id_tree p ON c.")
	query.WriteString(hierDim.RefField)
	query.WriteString(" = p.uuid")

	return query.String()
}

// generateIDExpression creates the complex CASE statement for dynamic ID generation.
// This is the heart of the configurable ID system - it generates SQL that produces
// user-facing IDs with prefixes based on dimension values.
//
// Algorithm:
// 1. Generate all possible combinations of dimension values that have prefixes
// 2. Create CASE WHEN conditions for each combination (sorted alphabetically)
// 3. Apply appropriate prefixes based on the combination
// 4. Use ROW_NUMBER() with proper partitioning to ensure contiguous numbering
//
// Example for dimensions [status, priority]:
// - status: ["pending", "done"] with prefix "d" for done
// - priority: ["low", "high"] with prefix "h" for high
//
// Generated SQL:
//
//	CASE
//	  WHEN priority = 'high' AND status = 'done' THEN 'hd' || ROW_NUMBER() OVER (...)
//	  WHEN status = 'done' THEN 'd' || ROW_NUMBER() OVER (...)
//	  WHEN priority = 'high' THEN 'h' || ROW_NUMBER() OVER (...)
//	  ELSE CAST(ROW_NUMBER() OVER (...) AS TEXT)
//	END
//
// Partitioning Strategy:
// - Root documents: PARTITION BY all_dimensions ORDER BY created_at
// - Child documents: PARTITION BY all_dimensions, parent_id ORDER BY created_at
// - Ensures IDs 1,2,3... within each dimension combination and parent context
func (qb *queryBuilder) generateIDExpression(enumDims []DimensionConfig, isRoot bool) string {
	if len(enumDims) == 0 {
		// No enumerated dimensions, just use row number
		return qb.generateSimpleRowNumber(isRoot)
	}

	// Build CASE statement for prefix generation
	var expr strings.Builder
	expr.WriteString("CASE\n")

	// Generate all combinations of dimension values
	combinations := qb.generateDimensionCombinations(enumDims)

	for _, combo := range combinations {
		expr.WriteString("        WHEN ")

		// Build condition for this combination
		conditions := make([]string, 0, len(combo.values))
		for dimName, value := range combo.values {
			conditions = append(conditions, fmt.Sprintf("%s = '%s'", dimName, value))
		}
		sort.Strings(conditions) // Ensure consistent ordering
		expr.WriteString(strings.Join(conditions, " AND "))

		expr.WriteString(" THEN\n")
		expr.WriteString("            '")
		expr.WriteString(combo.prefix)
		expr.WriteString("' || CAST(ROW_NUMBER() OVER (\n")
		expr.WriteString("                PARTITION BY ")

		// Partition by all dimensions (maintains separate numbering)
		partitions := make([]string, 0, len(enumDims))
		for _, dim := range enumDims {
			partitions = append(partitions, dim.Name)
		}
		if !isRoot {
			// For children, also partition by parent
			hierDim := qb.findHierarchicalDimension()
			if hierDim != nil {
				partitions = append(partitions, hierDim.RefField)
			}
		}
		expr.WriteString(strings.Join(partitions, ", "))

		expr.WriteString("\n                ORDER BY created_at\n")
		expr.WriteString("            ) AS TEXT)\n")
	}

	// Default case (shouldn't happen with proper constraints)
	expr.WriteString("        ELSE CAST(ROW_NUMBER() OVER (ORDER BY created_at) AS TEXT)\n")
	expr.WriteString("    END")

	return expr.String()
}

// generateSimpleRowNumber creates a basic ROW_NUMBER expression without prefixes
func (qb *queryBuilder) generateSimpleRowNumber(isRoot bool) string {
	var partition string
	if !isRoot {
		hierDim := qb.findHierarchicalDimension()
		if hierDim != nil {
			partition = fmt.Sprintf("PARTITION BY %s ", hierDim.RefField)
		}
	}

	return fmt.Sprintf("CAST(ROW_NUMBER() OVER (%sORDER BY created_at) AS TEXT)", partition)
}

// dimensionCombination represents a combination of dimension values and resulting prefix
type dimensionCombination struct {
	values map[string]string // dimension name -> value
	prefix string            // resulting prefix (alphabetically ordered)
}

// generateDimensionCombinations creates all possible combinations of dimension values
func (qb *queryBuilder) generateDimensionCombinations(enumDims []DimensionConfig) []dimensionCombination {
	if len(enumDims) == 0 {
		return nil
	}

	// Generate all possible combinations of dimension values
	var combinations []dimensionCombination

	// Helper to recursively generate combinations
	var generate func(dimIndex int, current map[string]string)
	generate = func(dimIndex int, current map[string]string) {
		if dimIndex == len(enumDims) {
			// We've assigned values to all dimensions
			// Now build the prefix string in alphabetical order by dimension name
			prefixParts := make([]struct {
				dimName string
				prefix  string
			}, 0)

			for dimName, value := range current {
				// Find the dimension config
				for _, dim := range enumDims {
					if dim.Name == dimName {
						if prefix, hasPrefix := dim.Prefixes[value]; hasPrefix && prefix != "" {
							prefixParts = append(prefixParts, struct {
								dimName string
								prefix  string
							}{dimName: dimName, prefix: prefix})
						}
						break
					}
				}
			}

			// Sort by dimension name for consistent ordering
			sort.Slice(prefixParts, func(i, j int) bool {
				return prefixParts[i].dimName < prefixParts[j].dimName
			})

			// Build prefix string
			var prefix strings.Builder
			for _, part := range prefixParts {
				prefix.WriteString(part.prefix)
			}

			// Create combination
			combo := dimensionCombination{
				values: make(map[string]string),
				prefix: prefix.String(),
			}
			for k, v := range current {
				combo.values[k] = v
			}
			combinations = append(combinations, combo)
			return
		}

		// Try each value for the current dimension
		dim := enumDims[dimIndex]
		for _, value := range dim.Values {
			newCurrent := make(map[string]string)
			for k, v := range current {
				newCurrent[k] = v
			}
			newCurrent[dim.Name] = value
			generate(dimIndex+1, newCurrent)
		}
	}

	// Start generation
	generate(0, make(map[string]string))

	return combinations
}

// findHierarchicalDimension returns the first hierarchical dimension (if any)
func (qb *queryBuilder) findHierarchicalDimension() *DimensionConfig {
	for _, dim := range qb.config.Dimensions {
		if dim.Type == Hierarchical {
			return &dim
		}
	}
	return nil
}

// buildWhereClausesAndArgs constructs WHERE clauses and arguments from filters
func (qb *queryBuilder) buildWhereClausesAndArgs(filters map[string]interface{}, hierDim *DimensionConfig) ([]string, []interface{}) {
	var whereClauses []string
	var args []interface{}

	for key, value := range filters {
		switch key {
		case "search":
			if searchTerm, ok := value.(string); ok && searchTerm != "" {
				whereClauses = append(whereClauses, "(title LIKE ? OR body LIKE ?)")
				searchPattern := "%" + searchTerm + "%"
				args = append(args, searchPattern, searchPattern)
			}
		default:
			// Check if it's a dimension filter
			if dim, found := qb.config.GetDimension(key); found {
				if dim.Type == Enumerated {
					// Handle both single value and slice of values
					switch v := value.(type) {
					case string:
						whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
						args = append(args, v)
					case []string:
						if len(v) > 0 {
							placeholders := make([]string, len(v))
							for i := range v {
								placeholders[i] = "?"
								args = append(args, v[i])
							}
							whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", key, strings.Join(placeholders, ",")))
						}
					}
				}
			} else if hierDim != nil && key == hierDim.RefField {
				// Handle filtering by custom hierarchical dimension RefField (e.g., parent_id)
				// Handle both string and *string values
				if parentID, ok := value.(string); ok {
					if parentID == "" {
						// Empty string means root documents (NULL parent)
						whereClauses = append(whereClauses, hierDim.RefField+" IS NULL")
					} else {
						whereClauses = append(whereClauses, hierDim.RefField+" = ?")
						args = append(args, parentID)
					}
				} else if parentID, ok := value.(*string); ok {
					if parentID == nil || *parentID == "" {
						// Empty string or nil means root documents (NULL parent)
						whereClauses = append(whereClauses, hierDim.RefField+" IS NULL")
					} else {
						whereClauses = append(whereClauses, hierDim.RefField+" = ?")
						args = append(args, *parentID)
					}
				}
			}
		}
	}

	return whereClauses, args
}

// generateFlatListQuery creates a simple query for filtered results with flat numbering
func (qb *queryBuilder) generateFlatListQuery(enumDims []DimensionConfig, hierDim *DimensionConfig, filters map[string]interface{}) (string, []interface{}, error) {
	var query strings.Builder
	query.WriteString("SELECT uuid, title, body, created_at, updated_at,\n")

	// Add dimension columns
	for _, dim := range enumDims {
		query.WriteString("    ")
		query.WriteString(dim.Name)
		query.WriteString(",\n")
	}

	// Add hierarchical dimension column if it exists
	if hierDim != nil {
		query.WriteString("    ")
		query.WriteString(hierDim.RefField)
		query.WriteString(" as ")
		query.WriteString(hierDim.RefField)
		query.WriteString(",\n")
	}

	// Generate ID expression with ROW_NUMBER
	query.WriteString("    ")
	query.WriteString(qb.generateIDExpression(enumDims, true))
	query.WriteString(" as user_facing_id\n")

	query.WriteString("FROM documents\n")

	// Build WHERE clauses
	whereClauses, args := qb.buildWhereClausesAndArgs(filters, hierDim)
	if len(whereClauses) > 0 {
		query.WriteString("WHERE ")
		query.WriteString(strings.Join(whereClauses, " AND "))
		query.WriteString("\n")
	}

	query.WriteString("ORDER BY created_at")

	return query.String(), args, nil
}
