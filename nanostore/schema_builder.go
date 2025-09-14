package nanostore

import (
	"fmt"
	"strings"
)

// SchemaBuilder generates SQL DDL statements based on dimension configuration
type schemaBuilder struct {
	config Config
}

// newSchemaBuilder creates a new schema builder for the given configuration
func newSchemaBuilder(config Config) *schemaBuilder {
	return &schemaBuilder{config: config}
}


// generateDimensionColumns creates ALTER TABLE statements for dimension columns
func (sb *schemaBuilder) generateDimensionColumns() []string {
	var statements []string

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case Enumerated:
			stmt := sb.generateEnumeratedColumn(dim)
			statements = append(statements, stmt)
		case Hierarchical:
			stmt := sb.generateHierarchicalColumn(dim)
			statements = append(statements, stmt)
		}
	}

	return statements
}

// generateEnumeratedColumn creates an ALTER TABLE statement for an enumerated dimension
func (sb *schemaBuilder) generateEnumeratedColumn(dim DimensionConfig) string {
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
func (sb *schemaBuilder) generateHierarchicalColumn(dim DimensionConfig) string {
	return fmt.Sprintf(
		"ALTER TABLE documents ADD COLUMN %s TEXT REFERENCES documents(uuid) ON DELETE CASCADE;",
		dim.RefField,
	)
}

// generateIndexes creates index statements for optimal query performance
func (sb *schemaBuilder) generateIndexes() []string {
	var statements []string

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case Enumerated:
			// Index on dimension + created_at for efficient partitioned ordering
			indexName := fmt.Sprintf("idx_documents_%s", dim.Name)
			stmt := fmt.Sprintf(
				"CREATE INDEX IF NOT EXISTS %s ON documents(%s, created_at);",
				indexName, dim.Name,
			)
			statements = append(statements, stmt)
		case Hierarchical:
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



// ValidateSchemaCompatibility checks if the current config is compatible with existing schema
func (sb *schemaBuilder) ValidateSchemaCompatibility(existingColumns map[string]string) error {
	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case Enumerated:
			if existingType, exists := existingColumns[dim.Name]; exists {
				if existingType != "TEXT" {
					return fmt.Errorf("dimension '%s' exists with incompatible type '%s', expected TEXT",
						dim.Name, existingType)
				}
			}
		case Hierarchical:
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
func (sb *schemaBuilder) GetExpectedColumns() map[string]string {
	columns := map[string]string{
		"uuid":       "TEXT",
		"title":      "TEXT",
		"body":       "TEXT",
		"created_at": "INTEGER",
		"updated_at": "INTEGER",
	}

	for _, dim := range sb.config.Dimensions {
		switch dim.Type {
		case Enumerated:
			columns[dim.Name] = "TEXT"
		case Hierarchical:
			columns[dim.RefField] = "TEXT"
		}
	}

	return columns
}
