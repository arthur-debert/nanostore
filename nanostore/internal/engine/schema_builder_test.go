package engine

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

func TestSchemaBuilder_GenerateBaseSchema(t *testing.T) {
	config := types.Config{}
	builder := NewSchemaBuilder(config)

	schema := builder.GenerateBaseSchema()

	// Should contain core table definition
	if !strings.Contains(schema, "CREATE TABLE IF NOT EXISTS documents") {
		t.Error("schema should contain documents table creation")
	}

	// Should contain all core columns
	coreColumns := []string{"uuid TEXT PRIMARY KEY", "title TEXT NOT NULL", "body TEXT DEFAULT", "created_at INTEGER", "updated_at INTEGER"}
	for _, col := range coreColumns {
		if !strings.Contains(schema, col) {
			t.Errorf("schema should contain core column: %s", col)
		}
	}
}

func TestSchemaBuilder_GenerateDimensionColumns(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed", "blocked"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	builder := NewSchemaBuilder(config)
	statements := builder.GenerateDimensionColumns()

	if len(statements) != 2 {
		t.Errorf("expected 2 dimension column statements, got %d", len(statements))
	}

	// Check enumerated dimension
	statusStmt := statements[0]
	expectedParts := []string{
		"ALTER TABLE documents ADD COLUMN status TEXT DEFAULT 'pending'",
		"CHECK (status IN ('pending', 'completed', 'blocked'))",
	}
	for _, part := range expectedParts {
		if !strings.Contains(statusStmt, part) {
			t.Errorf("status statement should contain: %s\nGot: %s", part, statusStmt)
		}
	}

	// Check hierarchical dimension
	parentStmt := statements[1]
	expectedParts = []string{
		"ALTER TABLE documents ADD COLUMN parent_uuid TEXT",
		"REFERENCES documents(uuid) ON DELETE CASCADE",
	}
	for _, part := range expectedParts {
		if !strings.Contains(parentStmt, part) {
			t.Errorf("parent statement should contain: %s\nGot: %s", part, parentStmt)
		}
	}
}

func TestSchemaBuilder_GenerateEnumeratedColumn(t *testing.T) {
	tests := []struct {
		name     string
		dim      types.DimensionConfig
		expected []string
	}{
		{
			name: "with explicit default",
			dim: types.DimensionConfig{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed"},
				DefaultValue: "pending",
			},
			expected: []string{
				"ALTER TABLE documents ADD COLUMN status TEXT DEFAULT 'pending'",
				"CHECK (status IN ('pending', 'completed'))",
			},
		},
		{
			name: "without explicit default",
			dim: types.DimensionConfig{
				Name:   "priority",
				Type:   types.Enumerated,
				Values: []string{"low", "high"},
			},
			expected: []string{
				"ALTER TABLE documents ADD COLUMN priority TEXT DEFAULT 'low'",
				"CHECK (priority IN ('low', 'high'))",
			},
		},
		{
			name: "with single quotes in values",
			dim: types.DimensionConfig{
				Name:   "category",
				Type:   types.Enumerated,
				Values: []string{"user's choice", "admin"},
			},
			expected: []string{
				"ALTER TABLE documents ADD COLUMN category TEXT DEFAULT 'user''s choice'",
				"CHECK (category IN ('user''s choice', 'admin'))",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &SchemaBuilder{}
			stmt := builder.generateEnumeratedColumn(tt.dim)

			for _, expected := range tt.expected {
				if !strings.Contains(stmt, expected) {
					t.Errorf("statement should contain: %s\nGot: %s", expected, stmt)
				}
			}
		})
	}
}

func TestSchemaBuilder_GenerateIndexes(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name: "status",
				Type: types.Enumerated,
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	builder := NewSchemaBuilder(config)
	statements := builder.GenerateIndexes()

	// Should have 3 indexes: status, parent, and search
	if len(statements) != 3 {
		t.Errorf("expected 3 index statements, got %d", len(statements))
	}

	// Check status index
	statusIndex := statements[0]
	if !strings.Contains(statusIndex, "CREATE INDEX IF NOT EXISTS idx_documents_status") {
		t.Errorf("should create status index, got: %s", statusIndex)
	}
	if !strings.Contains(statusIndex, "ON documents(status, created_at)") {
		t.Errorf("status index should include created_at, got: %s", statusIndex)
	}

	// Check parent index
	parentIndex := statements[1]
	if !strings.Contains(parentIndex, "CREATE INDEX IF NOT EXISTS idx_documents_parent") {
		t.Errorf("should create parent index, got: %s", parentIndex)
	}
	if !strings.Contains(parentIndex, "ON documents(parent_uuid, created_at)") {
		t.Errorf("parent index should include created_at, got: %s", parentIndex)
	}

	// Check search index
	searchIndex := statements[2]
	if !strings.Contains(searchIndex, "CREATE INDEX IF NOT EXISTS idx_documents_search") {
		t.Errorf("should create search index, got: %s", searchIndex)
	}
	if !strings.Contains(searchIndex, "ON documents(title, body)") {
		t.Errorf("search index should include title and body, got: %s", searchIndex)
	}
}

func TestSchemaBuilder_GenerateFullSchema(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending", "completed"},
			},
		},
	}

	builder := NewSchemaBuilder(config)
	statements := builder.GenerateFullSchema()

	// Should have: base table + 1 dimension column + 2 indexes + version table = 5 statements
	if len(statements) != 5 {
		t.Errorf("expected 5 statements for full schema, got %d", len(statements))
	}

	// Check that base table comes first
	if !strings.Contains(statements[0], "CREATE TABLE IF NOT EXISTS documents") {
		t.Error("first statement should be base table creation")
	}

	// Check that dimension column comes next
	if !strings.Contains(statements[1], "ALTER TABLE documents ADD COLUMN status") {
		t.Error("second statement should be dimension column addition")
	}

	// Check that schema version table is included
	hasVersionTable := false
	for _, stmt := range statements {
		if strings.Contains(stmt, "CREATE TABLE IF NOT EXISTS schema_version") {
			hasVersionTable = true
			break
		}
	}
	if !hasVersionTable {
		t.Error("schema should include schema_version table")
	}
}

func TestSchemaBuilder_GenerateMigrationSQL(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending", "completed"},
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
			{
				Name:   "priority",
				Type:   types.Enumerated,
				Values: []string{"low", "high"},
			},
		},
	}

	builder := NewSchemaBuilder(config)

	// Simulate existing database with only status dimension
	existingDimensions := []string{"status"}
	statements := builder.GenerateMigrationSQL(existingDimensions)

	// Should add parent_uuid and priority columns, plus all indexes
	if len(statements) < 2 {
		t.Errorf("expected at least 2 migration statements, got %d", len(statements))
	}

	// Check that only new dimensions are added
	hasParent := false
	hasPriority := false
	hasStatus := false

	for _, stmt := range statements {
		if strings.Contains(stmt, "ADD COLUMN parent_uuid") {
			hasParent = true
		}
		if strings.Contains(stmt, "ADD COLUMN priority") {
			hasPriority = true
		}
		if strings.Contains(stmt, "ADD COLUMN status") {
			hasStatus = true
		}
	}

	if !hasParent {
		t.Error("migration should add parent_uuid column")
	}
	if !hasPriority {
		t.Error("migration should add priority column")
	}
	if hasStatus {
		t.Error("migration should not re-add existing status column")
	}
}

func TestSchemaBuilder_ValidateSchemaCompatibility(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending", "completed"},
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	builder := NewSchemaBuilder(config)

	tests := []struct {
		name            string
		existingColumns map[string]string
		shouldError     bool
		errorMsg        string
	}{
		{
			name: "compatible schema",
			existingColumns: map[string]string{
				"status":      "TEXT",
				"parent_uuid": "TEXT",
			},
			shouldError: false,
		},
		{
			name: "incompatible enumerated type",
			existingColumns: map[string]string{
				"status": "INTEGER",
			},
			shouldError: true,
			errorMsg:    "dimension 'status' exists with incompatible type 'INTEGER'",
		},
		{
			name: "incompatible hierarchical type",
			existingColumns: map[string]string{
				"parent_uuid": "INTEGER",
			},
			shouldError: true,
			errorMsg:    "hierarchical dimension 'parent' field 'parent_uuid' exists with incompatible type 'INTEGER'",
		},
		{
			name:            "no existing columns",
			existingColumns: map[string]string{},
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.ValidateSchemaCompatibility(tt.existingColumns)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSchemaBuilder_GetExpectedColumns(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending"},
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	}

	builder := NewSchemaBuilder(config)
	columns := builder.GetExpectedColumns()

	// Should have all core columns plus dimension columns
	expectedColumns := map[string]string{
		"uuid":        "TEXT",
		"title":       "TEXT",
		"body":        "TEXT",
		"created_at":  "INTEGER",
		"updated_at":  "INTEGER",
		"status":      "TEXT",
		"parent_uuid": "TEXT",
	}

	if len(columns) != len(expectedColumns) {
		t.Errorf("expected %d columns, got %d", len(expectedColumns), len(columns))
	}

	for name, expectedType := range expectedColumns {
		if actualType, exists := columns[name]; !exists {
			t.Errorf("missing expected column: %s", name)
		} else if actualType != expectedType {
			t.Errorf("column %s has type %s, expected %s", name, actualType, expectedType)
		}
	}
}
