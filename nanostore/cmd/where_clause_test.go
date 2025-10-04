package main

import (
	"fmt"
	"os"
	"testing"
)

func TestWhereClauseIntegration(t *testing.T) {
	// Setup test database
	testDB := "test_where_integration.db"
	defer os.Remove(testDB)

	// Create registry and executor
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create test tasks with different statuses and priorities
	testTasks := []struct {
		title    string
		status   string
		priority string
	}{
		{"Active High Priority Task", "active", "high"},
		{"Pending Medium Task", "pending", "medium"},
		{"Done Low Task", "done", "low"},
		{"Active Medium Task", "active", "medium"},
	}

	var taskIDs []string
	for _, task := range testTasks {
		data := map[string]interface{}{
			"status":   task.status,
			"priority": task.priority,
		}

		result, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}

		taskID, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string ID, got %T", result)
		}
		taskIDs = append(taskIDs, taskID)
	}

	// Test cases for WHERE clauses
	tests := []struct {
		name          string
		whereClause   string
		whereArgs     []interface{}
		expectedCount int
		description   string
	}{
		{
			name:          "SimpleEquality",
			whereClause:   "status = ?",
			whereArgs:     []interface{}{"active"},
			expectedCount: 2,
			description:   "Find all active tasks",
		},
		{
			name:          "ANDCondition",
			whereClause:   "status = ? AND priority = ?",
			whereArgs:     []interface{}{"active", "high"},
			expectedCount: 1,
			description:   "Find active high priority tasks",
		},
		{
			name:          "NotEqual",
			whereClause:   "status != ?",
			whereArgs:     []interface{}{"done"},
			expectedCount: 3,
			description:   "Find all non-done tasks",
		},
		{
			name:          "MultipleAND",
			whereClause:   "status = ? AND priority = ? AND title LIKE ?",
			whereArgs:     []interface{}{"active", "medium", "%Medium%"},
			expectedCount: 1,
			description:   "Find active medium priority tasks with 'Medium' in title",
		},
		{
			name:          "NoMatches",
			whereClause:   "status = ? AND priority = ?",
			whereArgs:     []interface{}{"done", "high"},
			expectedCount: 0,
			description:   "Find done high priority tasks (should be none)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute query with WHERE clause
			result, err := executor.ExecuteQuery("Task", testDB, tt.whereClause, tt.whereArgs, "", 0, 0)
			if err != nil {
				t.Fatalf("Failed to execute WHERE query: %v", err)
			}

			// Verify result type
			tasks, ok := result.([]TaskDocument)
			if !ok {
				t.Fatalf("Expected []TaskDocument, got %T", result)
			}

			// Verify count
			if len(tasks) != tt.expectedCount {
				t.Errorf("%s: expected %d tasks, got %d", tt.description, tt.expectedCount, len(tasks))
			}

			// Log results for debugging
			t.Logf("%s: WHERE '%s' with args %v returned %d tasks",
				tt.description, tt.whereClause, tt.whereArgs, len(tasks))
		})
	}
}

func TestWhereClauseWithSortingAndPagination(t *testing.T) {
	// Setup test database
	testDB := "test_where_sorting.db"
	defer os.Remove(testDB)

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create multiple tasks with same status but different priorities
	priorities := []string{"low", "medium", "high", "high", "medium"}
	for i, priority := range priorities {
		data := map[string]interface{}{
			"status":   "active",
			"priority": priority,
		}

		title := fmt.Sprintf("Task %d", i+1)
		_, err := executor.ExecuteCreate("Task", testDB, title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", title, err)
		}
	}

	// Test sorting
	t.Run("WithSorting", func(t *testing.T) {
		result, err := executor.ExecuteQuery("Task", testDB, "status = ?", []interface{}{"active"}, "priority", 0, 0)
		if err != nil {
			t.Fatalf("Failed to execute sorted query: %v", err)
		}

		tasks, ok := result.([]TaskDocument)
		if !ok {
			t.Fatalf("Expected []TaskDocument, got %T", result)
		}

		if len(tasks) != 5 {
			t.Errorf("Expected 5 tasks, got %d", len(tasks))
		}

		t.Logf("Sorted tasks by priority: %d results", len(tasks))
	})

	// Test limiting
	t.Run("WithLimit", func(t *testing.T) {
		result, err := executor.ExecuteQuery("Task", testDB, "status = ?", []interface{}{"active"}, "", 3, 0)
		if err != nil {
			t.Fatalf("Failed to execute limited query: %v", err)
		}

		tasks, ok := result.([]TaskDocument)
		if !ok {
			t.Fatalf("Expected []TaskDocument, got %T", result)
		}

		if len(tasks) != 3 {
			t.Errorf("Expected 3 tasks (limited), got %d", len(tasks))
		}

		t.Logf("Limited query returned %d tasks", len(tasks))
	})

	// Test offset
	t.Run("WithOffset", func(t *testing.T) {
		result, err := executor.ExecuteQuery("Task", testDB, "status = ?", []interface{}{"active"}, "", 0, 2)
		if err != nil {
			t.Fatalf("Failed to execute offset query: %v", err)
		}

		tasks, ok := result.([]TaskDocument)
		if !ok {
			t.Fatalf("Expected []TaskDocument, got %T", result)
		}

		if len(tasks) != 3 { // 5 total - 2 offset = 3
			t.Errorf("Expected 3 tasks (with offset), got %d", len(tasks))
		}

		t.Logf("Offset query returned %d tasks", len(tasks))
	})
}

func TestWhereClauseErrorHandling(t *testing.T) {
	testDB := "test_where_errors.db"
	defer os.Remove(testDB)

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test with non-existent database file
	t.Run("NonExistentDatabase", func(t *testing.T) {
		_, err := executor.ExecuteQuery("Task", "/non/existent/path.db", "status = ?", []interface{}{"active"}, "", 0, 0)
		if err == nil {
			t.Error("Expected error for non-existent database, but got none")
		}
		t.Logf("Non-existent database correctly returned error: %v", err)
	})

	// Test with unsupported type
	t.Run("UnsupportedType", func(t *testing.T) {
		_, err := executor.ExecuteQuery("UnsupportedType", testDB, "status = ?", []interface{}{"active"}, "", 0, 0)
		if err == nil {
			t.Error("Expected error for unsupported type, but got none")
		}
		t.Logf("Unsupported type correctly returned error: %v", err)
	})
}