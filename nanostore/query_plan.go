package nanostore

import "strings"

// QueryPlan represents a fully analyzed query ready for SQL generation.
// It serves as an intermediate representation between the high-level query API
// and the low-level SQL generation.
type QueryPlan struct {
	// Type determines the query execution strategy
	Type QueryType

	// Core query components
	Filters    []Filter      // WHERE conditions
	OrderBy    []OrderClause // ORDER BY clauses
	Limit      *int          // LIMIT value
	Offset     *int          // OFFSET value
	TextSearch string        // Full-text search term

	// Dimension information from store config
	DimensionConfigs []DimensionConfig

	// ID generation requirements
	RequiresUserIDs bool // Whether to generate user-facing IDs

	// Parent filtering
	ParentFilter *ParentFilter
}

// QueryType determines the SQL generation strategy
type QueryType int

const (
	// FlatQuery uses simple SELECT for filtered queries
	FlatQuery QueryType = iota
	// HierarchicalQuery uses recursive CTE for tree traversal
	HierarchicalQuery
)

// Filter represents a single WHERE condition
type Filter struct {
	Type   FilterType
	Column string
	Value  interface{}
	Values []interface{} // For IN queries
}

// FilterType defines the type of filter operation
type FilterType int

const (
	FilterEquals FilterType = iota
	FilterNotEquals
	FilterIn
	FilterNotIn
	FilterIsNull
	FilterIsNotNull
	FilterLike
	FilterExists    // For dimension__exists filters
	FilterNotExists // For dimension__not_exists filters
)

// ParentFilter represents filtering by parent relationship
type ParentFilter struct {
	ParentUUID string
	Exists     *bool // true = must have parent, false = must not have parent
}

// QueryAnalyzer converts high-level query options into a QueryPlan
type QueryAnalyzer struct {
	config Config
}

// NewQueryAnalyzer creates a new query analyzer with the given store configuration
func NewQueryAnalyzer(config Config) *QueryAnalyzer {
	return &QueryAnalyzer{
		config: config,
	}
}

// Analyze converts ListOptions into a QueryPlan
func (qa *QueryAnalyzer) Analyze(opts ListOptions) (*QueryPlan, error) {
	plan := &QueryPlan{
		DimensionConfigs: qa.config.Dimensions,
		RequiresUserIDs:  true, // Always generate IDs for List operations
		TextSearch:       opts.FilterBySearch,
		OrderBy:          opts.OrderBy,
		Limit:            opts.Limit,
		Offset:           opts.Offset,
	}

	// Analyze filters
	plan.Filters = qa.analyzeFilters(opts.Filters)

	// Check for parent filter
	if parentFilter := qa.extractParentFilter(opts.Filters); parentFilter != nil {
		plan.ParentFilter = parentFilter
	}

	// Determine query type based on filters
	plan.Type = qa.determineQueryType(plan)

	return plan, nil
}

// analyzeFilters converts generic filter map to typed Filter structs
func (qa *QueryAnalyzer) analyzeFilters(filters map[string]interface{}) []Filter {
	var result []Filter

	// Find the hierarchical dimension RefField to skip it
	var refField string
	for _, dim := range qa.config.Dimensions {
		if dim.Type == Hierarchical {
			refField = dim.RefField
			break
		}
	}

	for key, value := range filters {
		// Skip special filters (parent field and search)
		if key == refField || key == "parent_uuid" || key == "search" {
			continue
		}

		// Parse filter key for suffixes
		column, filterType := qa.parseFilterKey(key)

		// Check if this is a valid column (dimension or uuid)
		isValidColumn := column == "uuid"
		for _, dim := range qa.config.Dimensions {
			if dim.Name == column {
				isValidColumn = true
				break
			}
		}

		if !isValidColumn {
			continue // Skip unknown columns
		}

		// Handle different value types and filter types
		switch filterType {
		case FilterExists:
			result = append(result, Filter{
				Type:   FilterIsNotNull,
				Column: column,
			})
		case FilterNotExists:
			result = append(result, Filter{
				Type:   FilterIsNull,
				Column: column,
			})
		case FilterNotEquals:
			result = append(result, Filter{
				Type:   FilterNotEquals,
				Column: column,
				Value:  value,
			})
		default: // FilterEquals or others
			// Handle value type for equals/in filters
			switch v := value.(type) {
			case []string:
				// Convert []string to []interface{}
				values := make([]interface{}, len(v))
				for i, s := range v {
					values[i] = s
				}
				result = append(result, Filter{
					Type:   FilterIn,
					Column: column,
					Values: values,
				})
			case []interface{}:
				result = append(result, Filter{
					Type:   FilterIn,
					Column: column,
					Values: v,
				})
			default:
				result = append(result, Filter{
					Type:   FilterEquals,
					Column: column,
					Value:  value,
				})
			}
		}
	}

	return result
}

// parseFilterKey parses filter keys with suffixes like "__not", "__exists", etc.
func (qa *QueryAnalyzer) parseFilterKey(key string) (column string, filterType FilterType) {
	if strings.HasSuffix(key, "__not") {
		return key[:len(key)-5], FilterNotEquals
	} else if strings.HasSuffix(key, "__exists") {
		return key[:len(key)-8], FilterExists
	} else if strings.HasSuffix(key, "__not_exists") {
		return key[:len(key)-12], FilterNotExists
	}
	return key, FilterEquals
}

// extractParentFilter checks for parent-related filters
func (qa *QueryAnalyzer) extractParentFilter(filters map[string]interface{}) *ParentFilter {
	// Find the hierarchical dimension to get the RefField name
	var refField string
	for _, dim := range qa.config.Dimensions {
		if dim.Type == Hierarchical {
			refField = dim.RefField
			break
		}
	}

	if refField == "" {
		// No hierarchical dimension, check for standard parent_uuid
		refField = "parent_uuid"
	}

	if parentValue, ok := filters[refField]; ok {
		if parentUUID, isString := parentValue.(string); isString {
			// Handle empty string as "root documents" filter
			if parentUUID == "" {
				exists := false
				return &ParentFilter{
					Exists: &exists,
				}
			}
			return &ParentFilter{
				ParentUUID: parentUUID,
			}
		}
	}
	return nil
}

// determineQueryType decides whether to use flat or hierarchical query
func (qa *QueryAnalyzer) determineQueryType(plan *QueryPlan) QueryType {
	// Use flat query when we have specific filters, ordering, or pagination
	// This is more efficient as it doesn't need recursive CTE
	if len(plan.Filters) > 0 || plan.TextSearch != "" || plan.ParentFilter != nil ||
		len(plan.OrderBy) > 0 || plan.Limit != nil || plan.Offset != nil {
		return FlatQuery
	}

	// Use hierarchical query for unfiltered listing without ordering/pagination
	// This maintains the tree structure and proper ordering
	return HierarchicalQuery
}

// QueryOptimizer optimizes a QueryPlan before SQL generation
type QueryOptimizer struct {
	config Config
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(config Config) *QueryOptimizer {
	return &QueryOptimizer{
		config: config,
	}
}

// Optimize applies optimization rules to the query plan
func (qo *QueryOptimizer) Optimize(plan *QueryPlan) *QueryPlan {
	// For now, just return the plan as-is
	// Future optimizations:
	// - Reorder filters by selectivity
	// - Push down limits when possible
	// - Choose optimal indexes
	// - Convert NOT queries to more efficient forms
	return plan
}
