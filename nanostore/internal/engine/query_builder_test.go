package engine

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore/types"
)

func TestQueryBuilder_GenerateListQuery(t *testing.T) {
	tests := []struct {
		name          string
		config        types.Config
		filters       map[string]interface{}
		expectedParts []string
		expectedArgs  int
		shouldContain []string
		shouldNotHave []string
	}{
		{
			name: "simple enumerated dimension",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			filters: map[string]interface{}{},
			shouldContain: []string{
				"WITH RECURSIVE",
				"root_docs AS",
				"child_docs AS",
				"id_tree AS",
				"ROW_NUMBER() OVER",
				"PARTITION BY status",
				"ORDER BY depth, created_at",
			},
		},
		{
			name: "no hierarchical dimension",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"completed": "c"},
					},
				},
			},
			filters: map[string]interface{}{},
			shouldContain: []string{
				"WITH RECURSIVE",
				"root_docs AS",
				"id_tree AS",
				"SELECT * FROM root_docs", // No hierarchy, just use root
				"ORDER BY created_at",     // No depth ordering
			},
			shouldNotHave: []string{
				"child_docs AS",
				"ORDER BY depth",
			},
		},
		{
			name: "with search filter",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			filters: map[string]interface{}{
				"search": "test",
			},
			shouldContain: []string{
				"WHERE",
				"title LIKE ?",
				"body LIKE ?",
			},
			expectedArgs: 2, // Two args for search (title and body)
		},
		{
			name: "with status filter",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			filters: map[string]interface{}{
				"status": "completed",
			},
			shouldContain: []string{
				"WHERE",
				"status = ?",
			},
			expectedArgs: 1,
		},
		{
			name: "with multiple status filter",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:         "status",
						Type:         types.Enumerated,
						Values:       []string{"pending", "completed"},
						Prefixes:     map[string]string{"completed": "c"},
						DefaultValue: "pending",
					},
					{
						Name:     "parent",
						Type:     types.Hierarchical,
						RefField: "parent_uuid",
					},
				},
			},
			filters: map[string]interface{}{
				"status": []string{"pending", "completed"},
			},
			shouldContain: []string{
				"WHERE",
				"status IN (?,?)",
			},
			expectedArgs: 2,
		},
		{
			name: "multiple enumerated dimensions",
			config: types.Config{
				Dimensions: []types.DimensionConfig{
					{
						Name:     "status",
						Type:     types.Enumerated,
						Values:   []string{"pending", "completed"},
						Prefixes: map[string]string{"completed": "c"},
					},
					{
						Name:     "priority",
						Type:     types.Enumerated,
						Values:   []string{"low", "high"},
						Prefixes: map[string]string{"high": "h"},
					},
				},
			},
			filters: map[string]interface{}{},
			shouldContain: []string{
				"PARTITION BY status, priority",
				"WHEN status = 'completed'",
				"WHEN status = 'pending'",
				"'c' || CAST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			qb := NewQueryBuilder(tt.config)
			query, args, err := qb.GenerateListQuery(tt.filters)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check expected parts
			for _, part := range tt.shouldContain {
				if !strings.Contains(query, part) {
					t.Errorf("query should contain '%s'\nGot:\n%s", part, query)
				}
			}

			// Check parts that should NOT be present
			for _, part := range tt.shouldNotHave {
				if strings.Contains(query, part) {
					t.Errorf("query should NOT contain '%s'\nGot:\n%s", part, query)
				}
			}

			// Check argument count
			if tt.expectedArgs > 0 && len(args) != tt.expectedArgs {
				t.Errorf("expected %d args, got %d: %v", tt.expectedArgs, len(args), args)
			}

			// Basic SQL syntax validation
			if !strings.HasPrefix(strings.TrimSpace(query), "WITH RECURSIVE") {
				t.Error("query should start with WITH RECURSIVE")
			}

			if !strings.Contains(query, "SELECT") {
				t.Error("query should contain SELECT")
			}
		})
	}
}

func TestQueryBuilder_IDGeneration(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed", "blocked"},
				Prefixes:     map[string]string{"completed": "c", "blocked": "b"},
				DefaultValue: "pending",
			},
		},
	}

	qb := NewQueryBuilder(config)

	// Test root query generation
	enumDims := GetEnumeratedDimensions(config)
	rootQuery := qb.generateRootQuery(enumDims, nil)

	// Check CASE statement for different statuses
	expectedCases := []string{
		"WHEN status = 'completed' THEN",
		"'c' || CAST(ROW_NUMBER()",
		"WHEN status = 'blocked' THEN",
		"'b' || CAST(ROW_NUMBER()",
		"WHEN status = 'pending' THEN",
		"'' || CAST(ROW_NUMBER()", // No prefix for default
	}

	for _, expected := range expectedCases {
		if !strings.Contains(rootQuery, expected) {
			t.Errorf("root query should contain '%s'\nGot:\n%s", expected, rootQuery)
		}
	}
}

func TestQueryBuilder_HierarchicalQuery(t *testing.T) {
	config := types.Config{
		Dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "completed"},
				Prefixes:     map[string]string{"completed": "c"},
				DefaultValue: "pending",
			},
			{
				Name:     "parent",
				Type:     types.Hierarchical,
				RefField: "parent_uuid",
			},
		},
	} // Has both status and parent dimensions

	qb := NewQueryBuilder(config)
	query, _, err := qb.GenerateListQuery(nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check hierarchical query structure
	hierarchicalParts := []string{
		"root_docs AS",
		"child_docs AS",
		"parent_uuid IS NULL",                   // Root condition
		"parent_uuid IS NOT NULL",               // Child condition
		"UNION ALL",                             // Recursive union
		"p.depth + 1",                           // Depth increment
		"p.user_facing_id || '.' || c.local_id", // ID concatenation
	}

	for _, part := range hierarchicalParts {
		if !strings.Contains(query, part) {
			t.Errorf("hierarchical query should contain '%s'", part)
		}
	}
}
