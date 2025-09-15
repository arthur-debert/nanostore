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

// buildDelete builds a safe DELETE query with a single condition
func (b *sqlBuilder) buildDelete(table string, condition squirrel.Eq) (string, []interface{}, error) {
	if len(condition) == 0 {
		return "", nil, fmt.Errorf("no condition specified for delete")
	}

	deleteQuery := b.sq.Delete(table).Where(condition)
	return deleteQuery.ToSql()
}

// buildDeleteWhere builds a safe DELETE query with a custom where clause
func (b *sqlBuilder) buildDeleteWhere(table string, whereClause interface{}, args ...interface{}) (string, []interface{}, error) {
	if whereClause == nil {
		return "", nil, fmt.Errorf("no where clause specified for delete")
	}

	deleteQuery := b.sq.Delete(table).Where(whereClause, args...)
	return deleteQuery.ToSql()
}

// buildUpdateByCondition builds a safe UPDATE query with a condition
func (b *sqlBuilder) buildUpdateByCondition(table string, setColumns []string, setValues []interface{}, condition squirrel.Eq) (string, []interface{}, error) {
	if len(setColumns) == 0 {
		return "", nil, fmt.Errorf("no columns specified for update")
	}
	if len(setColumns) != len(setValues) {
		return "", nil, fmt.Errorf("column count (%d) does not match value count (%d)", len(setColumns), len(setValues))
	}
	if len(condition) == 0 {
		return "", nil, fmt.Errorf("no condition specified for update")
	}

	update := b.sq.Update(table)
	for i, col := range setColumns {
		update = update.Set(col, setValues[i])
	}
	update = update.Where(condition)

	return update.ToSql()
}

// buildUpdateWhere builds a safe UPDATE query with a custom where clause
func (b *sqlBuilder) buildUpdateWhere(table string, setColumns []string, setValues []interface{}, whereClause interface{}, args ...interface{}) (string, []interface{}, error) {
	if len(setColumns) == 0 {
		return "", nil, fmt.Errorf("no columns specified for update")
	}
	if len(setColumns) != len(setValues) {
		return "", nil, fmt.Errorf("column count (%d) does not match value count (%d)", len(setColumns), len(setValues))
	}
	if whereClause == nil {
		return "", nil, fmt.Errorf("no where clause specified for update")
	}

	update := b.sq.Update(table)
	for i, col := range setColumns {
		update = update.Set(col, setValues[i])
	}
	update = update.Where(whereClause, args...)

	return update.ToSql()
}
