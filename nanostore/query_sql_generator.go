package nanostore

import (
	"fmt"
	"strings"
)

// QuerySQLGenerator converts QueryPlan to SQL
type QuerySQLGenerator struct {
	config Config
	qb     *queryBuilder // Reuse existing ID generation logic
}

// NewQuerySQLGenerator creates a new SQL generator
func NewQuerySQLGenerator(config Config) *QuerySQLGenerator {
	return &QuerySQLGenerator{
		config: config,
		qb:     newQueryBuilder(config),
	}
}

// GenerateSQL converts a QueryPlan into SQL query and args
func (qsg *QuerySQLGenerator) GenerateSQL(plan *QueryPlan) (string, []interface{}, error) {
	switch plan.Type {
	case FlatQuery:
		return qsg.generateFlatQuery(plan)
	case HierarchicalQuery:
		// For now, delegate to existing implementation
		// TODO: refactor hierarchical query to use QueryPlan
		return qsg.qb.GenerateListQuery(nil)
	default:
		return "", nil, fmt.Errorf("unsupported query type: %v", plan.Type)
	}
}

// generateFlatQuery generates SQL for a flat (non-hierarchical) query
func (qsg *QuerySQLGenerator) generateFlatQuery(plan *QueryPlan) (string, []interface{}, error) {
	var args []interface{}
	argCounter := 1

	// Build WHERE clause first to get filters and args
	whereClause, whereArgs, _ := qsg.buildWhereClause(plan, argCounter)
	args = append(args, whereArgs...)

	// Build the query with filters applied before ID generation
	query := qsg.buildFilteredQuery(plan, whereClause)

	// Add ORDER BY if specified
	if len(plan.OrderBy) > 0 {
		query += " ORDER BY " + qsg.buildOrderByClause(plan.OrderBy)
	} else {
		// Default ordering by created_at
		query += " ORDER BY created_at ASC"
	}

	// Add LIMIT and OFFSET
	if plan.Limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *plan.Limit)
		if plan.Offset != nil {
			query += fmt.Sprintf(" OFFSET %d", *plan.Offset)
		}
	} else if plan.Offset != nil {
		// SQLite requires LIMIT when using OFFSET, so use a very large limit
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", 9223372036854775807, *plan.Offset)
	}

	return query, args, nil
}

// buildFilteredQuery builds a query that filters first, then generates IDs
func (qsg *QuerySQLGenerator) buildFilteredQuery(plan *QueryPlan, whereClause string) string {
	if !plan.RequiresUserIDs {
		// Simple case without ID generation
		query := `SELECT uuid, ` + qsg.buildDimensionColumns() + `, 
		        title, body, created_at, updated_at 
		        FROM documents`
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		return query
	}

	// Build query that filters first, then generates IDs on filtered results
	enumDims := qsg.config.GetEnumeratedDimensions()
	idExpression := qsg.qb.generateIDExpression(enumDims, true)

	query := fmt.Sprintf(`SELECT 
		%s AS user_facing_id,
		uuid,
		%s,
		title,
		body,
		created_at,
		updated_at
	FROM documents`,
		idExpression,
		qsg.buildDimensionColumns())

	if whereClause != "" {
		query += "\nWHERE " + whereClause
	}

	return query
}

// buildDimensionColumns returns the dimension column list
func (qsg *QuerySQLGenerator) buildDimensionColumns() string {
	var columns []string
	for _, dim := range qsg.config.Dimensions {
		if dim.Type == Hierarchical {
			// For hierarchical dimensions, use the RefField as the column name
			columns = append(columns, dim.RefField)
		} else {
			// For other dimensions, use the dimension name
			columns = append(columns, dim.Name)
		}
	}
	return strings.Join(columns, ", ")
}

// buildWhereClause builds the WHERE clause from filters
func (qsg *QuerySQLGenerator) buildWhereClause(plan *QueryPlan, startArgNum int) (string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argNum := startArgNum

	// Add filter conditions
	for _, filter := range plan.Filters {
		condition, filterArgs := qsg.buildFilterCondition(filter, argNum)
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, filterArgs...)
			argNum += len(filterArgs)
		}
	}

	// Add text search condition
	if plan.TextSearch != "" {
		searchCondition := qsg.buildTextSearchCondition(plan.TextSearch, argNum)
		conditions = append(conditions, searchCondition)
		args = append(args, plan.TextSearch)
		argNum++
	}

	// Add parent filter condition
	if plan.ParentFilter != nil {
		condition, parentArgs := qsg.buildParentCondition(plan.ParentFilter, argNum)
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, parentArgs...)
			argNum += len(parentArgs)
		}
	}

	whereClause := strings.Join(conditions, " AND ")
	return whereClause, args, argNum
}

// buildFilterCondition builds SQL condition for a single filter
func (qsg *QuerySQLGenerator) buildFilterCondition(filter Filter, argNum int) (string, []interface{}) {
	// Column name is the same as the dimension name in the database
	dbColumn := filter.Column

	switch filter.Type {
	case FilterEquals:
		return fmt.Sprintf("%s = $%d", dbColumn, argNum), []interface{}{filter.Value}
	case FilterNotEquals:
		return fmt.Sprintf("%s != $%d", dbColumn, argNum), []interface{}{filter.Value}
	case FilterIn:
		placeholders := make([]string, len(filter.Values))
		for i := range filter.Values {
			placeholders[i] = fmt.Sprintf("$%d", argNum+i)
		}
		return fmt.Sprintf("%s IN (%s)", dbColumn, strings.Join(placeholders, ", ")), filter.Values
	case FilterNotIn:
		placeholders := make([]string, len(filter.Values))
		for i := range filter.Values {
			placeholders[i] = fmt.Sprintf("$%d", argNum+i)
		}
		return fmt.Sprintf("%s NOT IN (%s)", dbColumn, strings.Join(placeholders, ", ")), filter.Values
	case FilterIsNull:
		return fmt.Sprintf("%s IS NULL", dbColumn), nil
	case FilterIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", dbColumn), nil
	case FilterExists:
		// EXISTS means: IS NOT NULL AND != ''
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", dbColumn, dbColumn), nil
	case FilterNotExists:
		// NOT EXISTS means: IS NULL OR = ''
		return fmt.Sprintf("(%s IS NULL OR %s = '')", dbColumn, dbColumn), nil
	case FilterLike:
		return fmt.Sprintf("%s LIKE $%d", dbColumn, argNum), []interface{}{filter.Value}
	default:
		// Unsupported filter type, ignore
		return "", nil
	}
}

// buildTextSearchCondition builds the text search WHERE clause
func (qsg *QuerySQLGenerator) buildTextSearchCondition(searchTerm string, argNum int) string {
	// Search in title and body columns
	return fmt.Sprintf("(title LIKE '%%' || $%d || '%%' OR body LIKE '%%' || $%d || '%%')", argNum, argNum)
}

// buildParentCondition builds the parent filter condition
func (qsg *QuerySQLGenerator) buildParentCondition(pf *ParentFilter, argNum int) (string, []interface{}) {
	// Find the hierarchical dimension to get the RefField name
	refField := "parent_uuid" // default
	for _, dim := range qsg.config.Dimensions {
		if dim.Type == Hierarchical {
			refField = dim.RefField
			break
		}
	}

	if pf.ParentUUID != "" {
		return fmt.Sprintf("%s = $%d", refField, argNum), []interface{}{pf.ParentUUID}
	}

	if pf.Exists != nil {
		if *pf.Exists {
			return fmt.Sprintf("%s IS NOT NULL", refField), nil
		} else {
			return fmt.Sprintf("%s IS NULL", refField), nil
		}
	}

	return "", nil
}

// buildOrderByClause builds the ORDER BY clause
func (qsg *QuerySQLGenerator) buildOrderByClause(orderBy []OrderClause) string {
	if len(orderBy) == 0 {
		return ""
	}

	var clauses []string
	for _, oc := range orderBy {
		// Check if this is an enumerated dimension
		var dimConfig *DimensionConfig
		for _, dim := range qsg.config.Dimensions {
			if dim.Name == oc.Column {
				dimConfig = &dim
				break
			}
		}

		var orderExpression string
		if dimConfig != nil && dimConfig.Type == Enumerated {
			// For enumerated dimensions, use CASE to map values to their index order
			orderExpression = qsg.buildEnumeratedOrderExpression(oc.Column, dimConfig.Values, oc.Descending)
		} else {
			// For other columns, use direct ordering
			direction := "ASC"
			if oc.Descending {
				direction = "DESC"
			}
			orderExpression = fmt.Sprintf("%s %s", oc.Column, direction)
		}

		clauses = append(clauses, orderExpression)
	}

	return strings.Join(clauses, ", ")
}

// buildEnumeratedOrderExpression creates a CASE statement for enumerated dimension ordering
func (qsg *QuerySQLGenerator) buildEnumeratedOrderExpression(column string, values []string, descending bool) string {
	// Build CASE statement that maps enum values to their positional index
	var caseStmt strings.Builder
	caseStmt.WriteString("CASE ")
	caseStmt.WriteString(column)

	for i, value := range values {
		caseStmt.WriteString(fmt.Sprintf(" WHEN '%s' THEN %d", value, i))
	}
	caseStmt.WriteString(" ELSE 999") // Unknown values go to the end
	caseStmt.WriteString(" END")

	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	return caseStmt.String() + " " + direction
}
