package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDateRangeQueries(t *testing.T) {
	// Setup test database
	testDB := "test_date_range.db"
	defer func() { _ = os.Remove(testDB) }()

	// Create registry and executor
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create test tasks with different creation dates
	testTasks := []struct {
		title      string
		status     string
		priority   string
		daysOffset int // Days to add to baseTime
	}{
		{"Old Task", "done", "low", -30},        // 2023-12-02
		{"Recent Task", "active", "high", -5},   // 2023-12-27
		{"Current Task", "active", "medium", 0}, // 2024-01-01
		{"Future Task", "pending", "low", 15},   // 2024-01-16
	}

	for _, task := range testTasks {
		data := map[string]interface{}{
			"status":   task.status,
			"priority": task.priority,
		}

		_, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}
	}

	// First, let's get the actual creation time from one of the documents to understand the timing
	allDocs, err := executor.ExecuteQuery("Task", testDB, "", nil, "", 0, 0)
	if err != nil {
		t.Fatalf("Failed to get all documents: %v", err)
	}

	docs, ok := allDocs.([]TaskDocument)
	if !ok || len(docs) == 0 {
		t.Fatalf("Expected at least one document, got %T with length %d", allDocs, len(docs))
	}

	// Get creation time from first document
	firstDocTime := docs[0].CreatedAt
	t.Logf("First document created at: %v", firstDocTime)

	// Generate test dates relative to the actual document creation time
	beforeCreation := firstDocTime.Add(-1 * time.Minute)
	afterCreation := firstDocTime.Add(1 * time.Minute)

	// Test date range filtering
	tests := []struct {
		name          string
		createdAfter  string
		createdBefore string
		updatedAfter  string
		updatedBefore string
		expectedCount int
		description   string
	}{
		{
			name:          "CreatedAfter",
			createdAfter:  beforeCreation.Format(time.RFC3339),
			expectedCount: 4, // All tasks created after the before-creation time
			description:   "Find tasks created after before-creation time",
		},
		{
			name:          "CreatedBefore",
			createdBefore: afterCreation.Format(time.RFC3339),
			expectedCount: 4, // All tasks created before the after-creation time
			description:   "Find tasks created before after-creation time",
		},
		{
			name:          "NoMatches",
			createdAfter:  afterCreation.Format(time.RFC3339),
			expectedCount: 0, // No tasks created after the after-creation time
			description:   "Find tasks created after after-creation time (should be none)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build date WHERE clause
			whereClause, whereArgs, err := executor.buildFilterWhere(
				tt.createdAfter, tt.createdBefore, tt.updatedAfter, tt.updatedBefore,
				nil, nil, "", "", "", false,
				nil, nil, nil, nil, nil, nil, nil, // No enhanced filters
				"", "") // No status/priority filters
			if err != nil {
				t.Fatalf("Failed to build date WHERE clause: %v", err)
			}

			if whereClause == "" {
				t.Fatalf("Expected non-empty WHERE clause for test %s", tt.name)
			}

			// Execute query
			result, err := executor.ExecuteQuery("Task", testDB, whereClause, whereArgs, "", 0, 0)
			if err != nil {
				t.Fatalf("Failed to execute date query: %v", err)
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

			t.Logf("%s: WHERE '%s' with args %v returned %d tasks",
				tt.description, whereClause, whereArgs, len(tasks))
		})
	}
}

func TestNullHandling(t *testing.T) {
	// Setup test database
	testDB := "test_null_handling.db"
	defer func() { _ = os.Remove(testDB) }()

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create test tasks with different NULL field patterns
	testTasks := []struct {
		title       string
		status      string
		priority    string
		assignee    string
		description string
		dueDate     *time.Time
	}{
		{
			title:       "Complete Task",
			status:      "active",
			priority:    "high",
			assignee:    "alice",
			description: "Full task description",
			dueDate:     &time.Time{}, // Will be set to actual date
		},
		{
			title:    "Minimal Task",
			status:   "pending",
			priority: "medium",
			// assignee and description left empty (NULL-ish)
			dueDate: nil,
		},
		{
			title:       "Partial Task",
			status:      "active",
			priority:    "low",
			assignee:    "bob",
			description: "", // Empty string
			dueDate:     nil,
		},
	}

	// Set due date for the first task
	dueDate := time.Now().Add(24 * time.Hour)
	testTasks[0].dueDate = &dueDate

	for _, task := range testTasks {
		data := map[string]interface{}{
			"status":   task.status,
			"priority": task.priority,
		}

		// Add optional fields only if they have values
		if task.assignee != "" {
			data["assignee"] = task.assignee
		}
		if task.description != "" {
			data["description"] = task.description
		}
		if task.dueDate != nil && !task.dueDate.IsZero() {
			data["due_date"] = task.dueDate.Format(time.RFC3339)
		}

		_, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}
	}

	// Test NULL field queries
	tests := []struct {
		name          string
		nullFields    []string
		notNullFields []string
		expectedCount int
		description   string
	}{
		{
			name:          "NullAssignee",
			nullFields:    []string{"assignee"},
			expectedCount: 1, // Minimal Task has no assignee
			description:   "Find tasks with no assignee",
		},
		{
			name:          "NotNullAssignee",
			notNullFields: []string{"assignee"},
			expectedCount: 2, // Complete Task and Partial Task have assignees
			description:   "Find tasks with assignee set",
		},
		{
			name:          "NullDueDate",
			nullFields:    []string{"due_date"},
			expectedCount: 2, // Minimal Task and Partial Task have no due date
			description:   "Find tasks with no due date",
		},
		{
			name:          "NotNullDueDate",
			notNullFields: []string{"due_date"},
			expectedCount: 1, // Only Complete Task has due date
			description:   "Find tasks with due date set",
		},
		{
			name:          "MultipleNullFields",
			nullFields:    []string{"assignee", "due_date"},
			expectedCount: 1, // Only Minimal Task has both NULL
			description:   "Find tasks with multiple NULL fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build NULL WHERE clause
			whereClause, whereArgs, err := executor.buildFilterWhere(
				"", "", "", "", tt.nullFields, tt.notNullFields, "", "", "", false,
				nil, nil, nil, nil, nil, nil, nil, // No enhanced filters
				"", "") // No status/priority filters
			if err != nil {
				t.Fatalf("Failed to build NULL WHERE clause: %v", err)
			}

			if whereClause == "" {
				t.Fatalf("Expected non-empty WHERE clause for test %s", tt.name)
			}

			// Execute query
			result, err := executor.ExecuteQuery("Task", testDB, whereClause, whereArgs, "", 0, 0)
			if err != nil {
				t.Fatalf("Failed to execute NULL query: %v", err)
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

			t.Logf("%s: WHERE '%s' returned %d tasks",
				tt.description, whereClause, len(tasks))
		})
	}
}

func TestCombinedDateAndNullQueries(t *testing.T) {
	// Setup test database
	testDB := "test_combined_queries.db"
	defer func() { _ = os.Remove(testDB) }()

	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Create a variety of test tasks
	testTasks := []struct {
		title    string
		status   string
		assignee string
	}{
		{"Recent Active Task", "active", "alice"},
		{"Recent Unassigned Task", "active", ""},
		{"Old Done Task", "done", "bob"},
		{"Old Unassigned Task", "pending", ""},
	}

	for _, task := range testTasks {
		data := map[string]interface{}{
			"status": task.status,
		}
		if task.assignee != "" {
			data["assignee"] = task.assignee
		}

		_, err := executor.ExecuteCreate("Task", testDB, task.title, data)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.title, err)
		}
	}

	// Test combining WHERE clause with date/NULL filters
	t.Run("CombineWhereAndDate", func(t *testing.T) {
		explicitWhere := "status = ?"
		explicitArgs := []interface{}{"active"}

		dateWhere := "created_at > ?"
		yesterday := time.Now().Add(-24 * time.Hour)
		dateArgs := []interface{}{yesterday}

		// Combine clauses
		combinedWhere, combinedArgs := executor.combineWhereClauses(
			explicitWhere, explicitArgs, dateWhere, dateArgs)

		expectedWhere := "(status = ?) AND (created_at > ?)"
		if combinedWhere != expectedWhere {
			t.Errorf("Expected combined WHERE '%s', got '%s'", expectedWhere, combinedWhere)
		}

		if len(combinedArgs) != 2 {
			t.Errorf("Expected 2 combined args, got %d", len(combinedArgs))
		}

		t.Logf("Combined WHERE: %s with %d args", combinedWhere, len(combinedArgs))
	})

	t.Run("CombineWhereAndNull", func(t *testing.T) {
		explicitWhere := "status = ?"
		explicitArgs := []interface{}{"active"}

		nullWhere := "_data.assignee IS NULL"
		var nullArgs []interface{}

		// Combine clauses
		combinedWhere, combinedArgs := executor.combineWhereClauses(
			explicitWhere, explicitArgs, nullWhere, nullArgs)

		expectedWhere := "(status = ?) AND (_data.assignee IS NULL)"
		if combinedWhere != expectedWhere {
			t.Errorf("Expected combined WHERE '%s', got '%s'", expectedWhere, combinedWhere)
		}

		if len(combinedArgs) != 1 {
			t.Errorf("Expected 1 combined arg, got %d", len(combinedArgs))
		}

		t.Logf("Combined WHERE: %s with %d args", combinedWhere, len(combinedArgs))
	})
}

func TestDateParsingValidation(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test invalid date formats
	invalidDates := []string{
		"2024-01-01",           // Missing time
		"invalid-date",         // Completely invalid
		"2024-13-01T00:00:00Z", // Invalid month
		"2024-01-32T00:00:00Z", // Invalid day
	}

	for _, invalidDate := range invalidDates {
		t.Run(fmt.Sprintf("InvalidDate_%s", invalidDate), func(t *testing.T) {
			_, _, err := executor.buildFilterWhere(invalidDate, "", "", "", nil, nil, "", "", "", false,
				nil, nil, nil, nil, nil, nil, nil, // No enhanced filters
				"", "") // No status/priority filters
			if err == nil {
				t.Errorf("Expected error for invalid date '%s', but got none", invalidDate)
			}
			t.Logf("Invalid date '%s' correctly returned error: %v", invalidDate, err)
		})
	}

	// Test valid date formats
	validDates := []string{
		"2024-01-01T00:00:00Z",
		"2024-12-31T23:59:59Z",
		"2024-06-15T12:30:45Z",
	}

	for _, validDate := range validDates {
		t.Run(fmt.Sprintf("ValidDate_%s", validDate), func(t *testing.T) {
			whereClause, args, err := executor.buildFilterWhere(validDate, "", "", "", nil, nil, "", "", "", false,
				nil, nil, nil, nil, nil, nil, nil, // No enhanced filters
				"", "") // No status/priority filters
			if err != nil {
				t.Errorf("Expected no error for valid date '%s', but got: %v", validDate, err)
			}
			if whereClause == "" {
				t.Errorf("Expected non-empty WHERE clause for valid date '%s'", validDate)
			}
			if len(args) != 1 {
				t.Errorf("Expected 1 arg for valid date '%s', got %d", validDate, len(args))
			}
			t.Logf("Valid date '%s' generated WHERE: %s with %d args", validDate, whereClause, len(args))
		})
	}
}
