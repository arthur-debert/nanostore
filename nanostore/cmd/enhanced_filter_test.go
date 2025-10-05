package main

import (
	"os"
	"testing"
	"time"
)

func TestEnhancedFilteringFlags(t *testing.T) {
	// Setup test database
	testDB := "test_enhanced_filtering.db"
	defer func() { _ = os.Remove(testDB) }()

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create test tasks with various properties for filtering
	testTasks := []struct {
		title       string
		status      string
		priority    string
		assignee    string
		description string
	}{
		{"High Priority Task", "active", "high", "alice", "Important task for project Alpha"},
		{"Medium Task", "pending", "medium", "bob", "Regular maintenance work"},
		{"Low Priority Item", "done", "low", "alice", "Cleanup and documentation"},
		{"Critical Bug Fix", "active", "high", "charlie", "Fix login authentication"},
		{"Research Task", "pending", "medium", "", "Investigate new technology"},
	}

	for _, task := range testTasks {
		data := map[string]interface{}{
			"status":   task.status,
			"priority": task.priority,
		}
		if task.assignee != "" {
			data["assignee"] = task.assignee
		}
		if task.description != "" {
			data["description"] = task.description
		}

		_, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}
	}

	// Test enhanced filtering functionality
	tests := []struct {
		name          string
		filterEq      []string
		filterNe      []string
		filterGt      []string
		filterLt      []string
		filterGte     []string
		filterLte     []string
		filterLike    []string
		status        string
		priority      string
		expectedCount int
		description   string
	}{
		{
			name:          "EqualityFilter",
			filterEq:      []string{"status=active"},
			expectedCount: 2, // Should find active tasks
			description:   "Filter by status=active using filter-eq",
		},
		{
			name:          "NotEqualFilter",
			filterNe:      []string{"priority=low"},
			expectedCount: 4, // Should find all non-low priority tasks
			description:   "Filter by priority!=low using filter-ne",
		},
		{
			name:          "LikePatternFilter",
			filterLike:    []string{"title=%Task%"},
			expectedCount: 3, // Should find tasks with "Task" in title
			description:   "Filter by title pattern using filter-like",
		},
		// Note: IN filters are not supported due to WhereEvaluator OR logic limitations
		{
			name:          "ConvenienceStatusFilter",
			status:        "pending",
			expectedCount: 2, // Should find pending tasks
			description:   "Filter by status using convenience flag",
		},
		{
			name:          "ConveniencePriorityFilter",
			priority:      "high",
			expectedCount: 2, // Should find high priority tasks
			description:   "Filter by priority using convenience flag",
		},
		// Note: statusIn and priorityIn filters are not supported due to WhereEvaluator OR logic limitations
		{
			name:          "CombinedFilters",
			filterEq:      []string{"status=active"},
			filterNe:      []string{"assignee=bob"},
			expectedCount: 2, // Should find active tasks not assigned to bob
			description:   "Combine multiple filter types",
		},
		{
			name:          "MultipleEqualityFilters",
			filterEq:      []string{"status=active", "priority=high"},
			expectedCount: 2, // Should find active high priority tasks
			description:   "Multiple equality filters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build enhanced filter WHERE clause
			whereClause, whereArgs, err := executor.buildFilterWhere(
				"", "", "", "", // No date filters
				nil, nil, // No NULL filters
				"", "", "", false, // No text search
				tt.filterEq, tt.filterNe, tt.filterGt, tt.filterLt,
				tt.filterGte, tt.filterLte, tt.filterLike,
				tt.status, tt.priority)
			if err != nil {
				t.Fatalf("Failed to build enhanced filter WHERE clause: %v", err)
			}

			// Check if we should expect a WHERE clause
			shouldHaveWhere := len(tt.filterEq) > 0 || len(tt.filterNe) > 0 ||
				len(tt.filterGt) > 0 || len(tt.filterLt) > 0 || len(tt.filterGte) > 0 ||
				len(tt.filterLte) > 0 || len(tt.filterLike) > 0 || tt.status != "" || tt.priority != ""

			if whereClause == "" && shouldHaveWhere {
				t.Fatalf("Expected non-empty WHERE clause for test %s", tt.name)
			}

			// Skip empty query tests
			if whereClause == "" {
				t.Logf("Skipping test %s - no filter criteria", tt.name)
				return
			}

			// Execute query
			result, err := executor.ExecuteQuery("Task", testDB, whereClause, whereArgs, "", 0, 0)
			if err != nil {
				t.Fatalf("Failed to execute enhanced filter query: %v", err)
			}

			// Verify result type
			tasks, ok := result.([]TaskDocument)
			if !ok {
				t.Fatalf("Expected []TaskDocument, got %T", result)
			}

			// Verify count
			if len(tasks) != tt.expectedCount {
				t.Errorf("%s: expected %d tasks, got %d", tt.description, tt.expectedCount, len(tasks))
				for i, task := range tasks {
					t.Logf("  Task %d: %q (status: %s, priority: %s)", i, task.Title, task.Status, task.Priority)
				}
			}

			t.Logf("%s: WHERE '%s' with args %v returned %d tasks",
				tt.description, whereClause, whereArgs, len(tasks))
		})
	}
}

func TestComplexFilterCombinations(t *testing.T) {
	// Setup test database
	testDB := "test_complex_filtering.db"
	defer func() { _ = os.Remove(testDB) }()

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create test tasks
	testTasks := []struct {
		title    string
		status   string
		priority string
		assignee string
	}{
		{"Urgent Bug Fix", "active", "high", "alice"},
		{"Feature Request", "pending", "medium", "bob"},
		{"Documentation Update", "done", "low", "alice"},
		{"Code Review", "active", "medium", "charlie"},
		{"Performance Optimization", "pending", "high", ""},
	}

	for _, task := range testTasks {
		data := map[string]interface{}{
			"status":   task.status,
			"priority": task.priority,
		}
		if task.assignee != "" {
			data["assignee"] = task.assignee
		}

		_, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}
	}

	// Test complex filter combinations
	t.Run("DateRangeWithStatusFilter", func(t *testing.T) {
		// Get current time for date filtering
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)

		// Build combined filter: created after yesterday AND status = active
		whereClause, whereArgs, err := executor.buildFilterWhere(
			yesterday.Format(time.RFC3339), "", "", "", // Date filters
			nil, nil, // No NULL filters
			"", "", "", false, // No text search
			[]string{"status=active"}, nil, nil, nil, nil, nil, nil, // Enhanced filters
			"", "") // No convenience filters
		if err != nil {
			t.Fatalf("Failed to build combined filter: %v", err)
		}

		result, err := executor.ExecuteQuery("Task", testDB, whereClause, whereArgs, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to execute combined query: %v", err)
		}

		tasks, ok := result.([]TaskDocument)
		if !ok {
			t.Fatalf("Expected []TaskDocument, got %T", result)
		}

		// Should find active tasks created after yesterday (which should be all active tasks)
		expectedCount := 2 // "Urgent Bug Fix" and "Code Review" are active
		if len(tasks) != expectedCount {
			t.Errorf("Expected %d tasks, got %d", expectedCount, len(tasks))
		}

		t.Logf("Date + status filter: WHERE '%s' returned %d tasks", whereClause, len(tasks))
	})

	t.Run("TextSearchWithPriorityFilter", func(t *testing.T) {
		// Combine text search with priority filter
		whereClause, whereArgs, err := executor.buildFilterWhere(
			"", "", "", "", // No date filters
			nil, nil, // No NULL filters
			"bug", "", "", false, // Text search for "bug"
			nil, nil, nil, nil, nil, nil, nil, // No enhanced filters
			"", "high") // Convenience priority filter
		if err != nil {
			t.Fatalf("Failed to build text + priority filter: %v", err)
		}

		result, err := executor.ExecuteQuery("Task", testDB, whereClause, whereArgs, "", 0, 0)
		if err != nil {
			t.Fatalf("Failed to execute text + priority query: %v", err)
		}

		tasks, ok := result.([]TaskDocument)
		if !ok {
			t.Fatalf("Expected []TaskDocument, got %T", result)
		}

		// Should find high priority tasks containing "bug" (should be 1: "Urgent Bug Fix")
		expectedCount := 1
		if len(tasks) != expectedCount {
			t.Errorf("Expected %d tasks, got %d", expectedCount, len(tasks))
		}

		t.Logf("Text search + priority filter: WHERE '%s' returned %d tasks", whereClause, len(tasks))
	})
}
