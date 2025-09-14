package nanostore

import (
	"fmt"
	"github.com/Masterminds/squirrel"
)

// sqlBuilder wraps squirrel to provide safe SQL generation
type sqlBuilder struct {
	sq squirrel.StatementBuilderType
}

// newSQLBuilder creates a new SQL builder
func newSQLBuilder() *sqlBuilder {
	return &sqlBuilder{
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

// buildInsert builds a safe INSERT query
func (b *sqlBuilder) buildInsert(table string, columns []string, values []interface{}) (string, []interface{}, error) {
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

// buildSelectCount builds a safe SELECT COUNT query
func (b *sqlBuilder) buildSelectCount(table string, whereClause squirrel.Eq) (string, []interface{}, error) {
	selectQuery := b.sq.Select("COUNT(*)").From(table)
	if len(whereClause) > 0 {
		selectQuery = selectQuery.Where(whereClause)
	}

	return selectQuery.ToSql()
}

// buildDynamicUpdate builds a safe UPDATE query with dynamic SET clauses
func (b *sqlBuilder) buildDynamicUpdate(columns []string, values []interface{}, id string) (string, []interface{}, error) {
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no columns specified for update")
	}
	if len(columns) != len(values) {
		return "", nil, fmt.Errorf("column count (%d) does not match value count (%d)", len(columns), len(values))
	}

	update := b.sq.Update("documents")
	for i, col := range columns {
		update = update.Set(col, values[i])
	}
	update = update.Where(squirrel.Eq{"uuid": id})

	return update.ToSql()
}
