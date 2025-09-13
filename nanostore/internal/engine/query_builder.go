package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

// QueryBuilder generates dynamic SQL queries based on dimension configuration
type QueryBuilder struct {
	config types.Config
}

// NewQueryBuilder creates a new query builder for the given configuration
func NewQueryBuilder(config types.Config) *QueryBuilder {
	return &QueryBuilder{config: config}
}

// GenerateListQuery creates a SQL query for listing documents with generated IDs
// This is the core of the configurable ID generation system
func (qb *QueryBuilder) GenerateListQuery(filters map[string]interface{}) (string, []interface{}, error) {
	// Get dimension information
	enumDims := GetEnumeratedDimensions(qb.config)
	hierDim := qb.findHierarchicalDimension()

	// Check if we should use flat listing (when filters are present)
	_, hasStatusFilter := filters["status"]
	_, hasParentFilter := filters["parent"]
	_, hasSearchFilter := filters["search"]
	hasFilters := hasStatusFilter || hasParentFilter || hasSearchFilter

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

// generateRootQuery creates the query for root-level documents with ID generation
func (qb *QueryBuilder) generateRootQuery(enumDims []types.DimensionConfig, filters map[string]interface{}) string {
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
func (qb *QueryBuilder) generateChildQuery(enumDims []types.DimensionConfig, hierDim *types.DimensionConfig) string {
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
func (qb *QueryBuilder) generateTreeQuery(hierDim *types.DimensionConfig) string {
	var query strings.Builder

	// Base case: root documents
	query.WriteString("    -- Base case: root documents\n")
	query.WriteString("    SELECT\n")
	query.WriteString("        uuid, title, body, created_at, updated_at,\n")

	// Include all dimension columns in tree
	enumDims := GetEnumeratedDimensions(qb.config)
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

// generateIDExpression creates the CASE/ROW_NUMBER expression for ID generation
func (qb *QueryBuilder) generateIDExpression(enumDims []types.DimensionConfig, isRoot bool) string {
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
func (qb *QueryBuilder) generateSimpleRowNumber(isRoot bool) string {
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
func (qb *QueryBuilder) generateDimensionCombinations(enumDims []types.DimensionConfig) []dimensionCombination {
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
func (qb *QueryBuilder) findHierarchicalDimension() *types.DimensionConfig {
	for _, dim := range qb.config.Dimensions {
		if dim.Type == types.Hierarchical {
			return &dim
		}
	}
	return nil
}

// buildWhereClausesAndArgs constructs WHERE clauses and arguments from filters
func (qb *QueryBuilder) buildWhereClausesAndArgs(filters map[string]interface{}, hierDim *types.DimensionConfig) ([]string, []interface{}) {
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
		case "parent":
			if hierDim != nil {
				if parentID, ok := value.(*string); ok {
					if parentID == nil || *parentID == "" {
						// Empty string or nil means root documents (NULL parent)
						whereClauses = append(whereClauses, hierDim.RefField+" IS NULL")
					} else {
						whereClauses = append(whereClauses, hierDim.RefField+" = ?")
						args = append(args, *parentID)
					}
				}
			}
		default:
			// Check if it's a dimension filter
			if dim, found := GetDimension(qb.config, key); found {
				if dim.Type == types.Enumerated {
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
			}
		}
	}

	return whereClauses, args
}

// generateFlatListQuery creates a simple query for filtered results with flat numbering
func (qb *QueryBuilder) generateFlatListQuery(enumDims []types.DimensionConfig, hierDim *types.DimensionConfig, filters map[string]interface{}) (string, []interface{}, error) {
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
