package engine

import (
	"fmt"
	"github.com/Masterminds/squirrel"
)

// SQLBuilder wraps squirrel to provide safe SQL generation
type SQLBuilder struct {
	sq squirrel.StatementBuilderType
}

// NewSQLBuilder creates a new SQL builder
func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

// BuildInsert builds a safe INSERT query
func (b *SQLBuilder) BuildInsert(table string, columns []string, values []interface{}) (string, []interface{}, error) {
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified for insert")
	}
	if len(columns) != len(values) {
		return "", nil, fmt.Errorf("column count (%d) does not match value count (%d)", len(columns), len(values))
	}

	insert := b.sq.Insert(table).Columns(columns...)
	insert = insert.Values(values...)

	return insert.ToSql()
}

// BuildUpdate builds a safe UPDATE query with dynamic SET clauses
func (b *SQLBuilder) BuildUpdate(table string, setClauses map[string]interface{}, whereClause squirrel.Eq) (string, []interface{}, error) {
	if len(setClauses) == 0 {
		return "", nil, fmt.Errorf("no SET clauses specified for update")
	}

	update := b.sq.Update(table)
	for col, val := range setClauses {
		update = update.Set(col, val)
	}
	update = update.Where(whereClause)

	return update.ToSql()
}

// BuildDeleteWithCTE builds a DELETE query with recursive CTE for cascade delete
func (b *SQLBuilder) BuildDeleteWithCTE(hierField string, uuid string) (string, []interface{}, error) {
	// Build the recursive CTE query safely
	cte := fmt.Sprintf(`
WITH RECURSIVE descendants AS (
    SELECT uuid FROM documents WHERE uuid = ?
    UNION ALL
    SELECT d.uuid 
    FROM documents d
    INNER JOIN descendants desc ON d.%s = desc.uuid
)
DELETE FROM documents WHERE uuid IN (SELECT uuid FROM descendants)`, hierField)

	return cte, []interface{}{uuid}, nil
}

// BuildSelectCount builds a COUNT query
func (b *SQLBuilder) BuildSelectCount(table string, whereClause squirrel.Eq) (string, []interface{}, error) {
	return b.sq.Select("COUNT(*)").From(table).Where(whereClause).ToSql()
}

// BuildDynamicUpdate builds an UPDATE with dynamic columns
func (b *SQLBuilder) BuildDynamicUpdate(columns []string, values []interface{}, uuid string) (string, []interface{}, error) {
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no columns to update")
	}
	if len(columns) != len(values) {
		return "", nil, fmt.Errorf("column count (%d) does not match value count (%d)", len(columns), len(values))
	}

	update := b.sq.Update("documents")
	for i, col := range columns {
		update = update.Set(col, values[i])
	}
	update = update.Where(squirrel.Eq{"uuid": uuid})

	return update.ToSql()
}
