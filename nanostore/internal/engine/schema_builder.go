package engine

import (
	"fmt"
	"strings"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

// SchemaBuilder generates SQL DDL statements based on dimension configuration
type SchemaBuilder struct {
	config types.Config
}

// NewSchemaBuilder creates a new schema builder for the given configuration
func NewSchemaBuilder(config types.Config) *SchemaBuilder {
	return &SchemaBuilder{config: config}
}

// GenerateBaseSchema creates the base documents table with core fields
func (sb *SchemaBuilder) GenerateBaseSchema() string {
	return `-- Base schema for document store
CREATE TABLE IF NOT EXISTS documents (
    uuid TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT DEFAULT '',
    created_at INTEGER NOT NULL,  -- Unix timestamp for consistent ordering
    updated_at INTEGER NOT NULL   -- Unix timestamp, updated on modifications
);`
}

// GenerateDimensionColumns creates ALTER TABLE statements for dimension columns
func (sb *SchemaBuilder) GenerateDimensionColumns() []string {
	var statements []string

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case types.Enumerated:
			stmt := sb.generateEnumeratedColumn(dim)
			statements = append(statements, stmt)
		case types.Hierarchical:
			stmt := sb.generateHierarchicalColumn(dim)
			statements = append(statements, stmt)
		}
	}

	return statements
}

// generateEnumeratedColumn creates an ALTER TABLE statement for an enumerated dimension
func (sb *SchemaBuilder) generateEnumeratedColumn(dim types.DimensionConfig) string {
	// Build CHECK constraint for valid values
	quotedValues := make([]string, len(dim.Values))
	for i, value := range dim.Values {
		quotedValues[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "''"))
	}
	checkConstraint := fmt.Sprintf("CHECK (%s IN (%s))", dim.Name, strings.Join(quotedValues, ", "))

	// Determine default value
	defaultValue := dim.DefaultValue
	if defaultValue == "" && len(dim.Values) > 0 {
		defaultValue = dim.Values[0]
	}

	return fmt.Sprintf(
		"ALTER TABLE documents ADD COLUMN %s TEXT DEFAULT '%s' %s;",
		dim.Name,
		strings.ReplaceAll(defaultValue, "'", "''"),
		checkConstraint,
	)
}

// generateHierarchicalColumn creates an ALTER TABLE statement for a hierarchical dimension
func (sb *SchemaBuilder) generateHierarchicalColumn(dim types.DimensionConfig) string {
	return fmt.Sprintf(
		"ALTER TABLE documents ADD COLUMN %s TEXT REFERENCES documents(uuid) ON DELETE CASCADE;",
		dim.RefField,
	)
}

// GenerateIndexes creates index statements for optimal query performance
func (sb *SchemaBuilder) GenerateIndexes() []string {
	var statements []string

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case types.Enumerated:
			// Index on dimension + created_at for efficient partitioned ordering
			indexName := fmt.Sprintf("idx_documents_%s", dim.Name)
			stmt := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS %s ON documents(%s, created_at);",
				indexName, dim.Name,
			)
			statements = append(statements, stmt)
		case types.Hierarchical:
			// Index on reference field + created_at for parent-child queries
			indexName := fmt.Sprintf("idx_documents_%s", strings.TrimSuffix(dim.RefField, "_uuid"))
			stmt := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS %s ON documents(%s, created_at);",
				indexName, dim.RefField,
			)
			statements = append(statements, stmt)
		}
	}

	// General search index on title and body
	statements = append(statements,
		"CREATE INDEX IF NOT EXISTS idx_documents_search ON documents(title, body);")

	return statements
}

// GenerateFullSchema creates the complete schema including base table, columns, and indexes
func (sb *SchemaBuilder) GenerateFullSchema() []string {
	var statements []string

	// Base table
	statements = append(statements, sb.GenerateBaseSchema())

	// Dimension columns
	statements = append(statements, sb.GenerateDimensionColumns()...)

	// Indexes
	statements = append(statements, sb.GenerateIndexes()...)

	// Schema version tracking table
	statements = append(statements, `-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);`)

	return statements
}

// GenerateMigrationSQL creates SQL for migrating from one schema to another
// This handles adding new dimensions to existing databases
func (sb *SchemaBuilder) GenerateMigrationSQL(existingDimensions []string) []string {
	var statements []string

	// Track which dimensions already exist
	existingMap := make(map[string]bool)
	for _, dim := range existingDimensions {
		existingMap[dim] = true
	}

	// Add new dimensions that don't exist yet
	for _, dim := range sb.config.Dimensions {
		var columnName string
		switch dim.Type {
		case types.Enumerated:
			columnName = dim.Name
		case types.Hierarchical:
			columnName = dim.RefField
		}

		if !existingMap[columnName] {
			switch dim.Type {
			case types.Enumerated:
				statements = append(statements, sb.generateEnumeratedColumn(dim))
			case types.Hierarchical:
				statements = append(statements, sb.generateHierarchicalColumn(dim))
			}
		}
	}

	// Add new indexes (CREATE INDEX IF NOT EXISTS handles duplicates)
	statements = append(statements, sb.GenerateIndexes()...)

	return statements
}

// ValidateSchemaCompatibility checks if the current config is compatible with existing schema
func (sb *SchemaBuilder) ValidateSchemaCompatibility(existingColumns map[string]string) error {
	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case types.Enumerated:
			if existingType, exists := existingColumns[dim.Name]; exists {
				if existingType != "TEXT" {
					return fmt.Errorf("dimension '%s' exists with incompatible type '%s', expected TEXT",
						dim.Name, existingType)
				}
			}
		case types.Hierarchical:
			if existingType, exists := existingColumns[dim.RefField]; exists {
				if existingType != "TEXT" {
					return fmt.Errorf("hierarchical dimension '%s' field '%s' exists with incompatible type '%s', expected TEXT",
						dim.Name, dim.RefField, existingType)
				}
			}
		}
	}

	return nil
}

// GetExpectedColumns returns the set of columns that should exist for this configuration
func (sb *SchemaBuilder) GetExpectedColumns() map[string]string {
	columns := map[string]string{
		"uuid":       "TEXT",
		"title":      "TEXT",
		"body":       "TEXT",
		"created_at": "INTEGER",
		"updated_at": "INTEGER",
	}

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case types.Enumerated:
			columns[dim.Name] = "TEXT"
		case types.Hierarchical:
			columns[dim.RefField] = "TEXT"
		}
	}

	return columns
}
