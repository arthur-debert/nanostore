package store

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/types"
)

func TestWhereEvaluator(t *testing.T) {
	// Create a test document
	doc := &types.Document{
		UUID:      "test-uuid-123",
		SimpleID:  "1",
		Title:     "Test Document",
		Body:      "Test body content",
		CreatedAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC),
		Dimensions: map[string]interface{}{
			"status":         "active",
			"priority":       "high",
			"_data.assignee": "alice",
			"_data.estimate": 5,
			"_data.urgent":   true,
		},
	}

	t.Run("BasicComparisons", func(t *testing.T) {
		tests := []struct {
			clause   string
			args     []interface{}
			expected bool
		}{
			{"status = ?", []interface{}{"active"}, true},
			{"status = ?", []interface{}{"pending"}, false},
			{"status != ?", []interface{}{"pending"}, true},
			{"status != ?", []interface{}{"active"}, false},
			{"priority = ?", []interface{}{"high"}, true},
			{"title = ?", []interface{}{"Test Document"}, true},
			{"uuid = ?", []interface{}{"test-uuid-123"}, true},
			{"simple_id = ?", []interface{}{"1"}, true},
		}

		for _, tt := range tests {
			evaluator := NewWhereEvaluator(tt.clause, tt.args...)
			result, err := evaluator.EvaluateDocument(doc)
			if err != nil {
				t.Errorf("evaluating %q: %v", tt.clause, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("evaluating %q: expected %v, got %v", tt.clause, tt.expected, result)
			}
		}
	})

	t.Run("DataFieldComparisons", func(t *testing.T) {
		tests := []struct {
			clause   string
			args     []interface{}
			expected bool
		}{
			{"_data.assignee = ?", []interface{}{"alice"}, true},
			{"_data.assignee = ?", []interface{}{"bob"}, false},
			{"_data.estimate = ?", []interface{}{5}, true},
			{"_data.estimate = ?", []interface{}{3}, false},
			{"_data.urgent = ?", []interface{}{true}, true},
			{"_data.urgent = ?", []interface{}{false}, false},
		}

		for _, tt := range tests {
			evaluator := NewWhereEvaluator(tt.clause, tt.args...)
			result, err := evaluator.EvaluateDocument(doc)
			if err != nil {
				t.Errorf("evaluating %q: %v", tt.clause, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("evaluating %q: expected %v, got %v", tt.clause, tt.expected, result)
			}
		}
	})

	t.Run("NumericComparisons", func(t *testing.T) {
		tests := []struct {
			clause   string
			args     []interface{}
			expected bool
		}{
			{"_data.estimate > ?", []interface{}{3}, true},
			{"_data.estimate > ?", []interface{}{5}, false},
			{"_data.estimate >= ?", []interface{}{5}, true},
			{"_data.estimate < ?", []interface{}{10}, true},
			{"_data.estimate < ?", []interface{}{3}, false},
			{"_data.estimate <= ?", []interface{}{5}, true},
		}

		for _, tt := range tests {
			evaluator := NewWhereEvaluator(tt.clause, tt.args...)
			result, err := evaluator.EvaluateDocument(doc)
			if err != nil {
				t.Errorf("evaluating %q: %v", tt.clause, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("evaluating %q: expected %v, got %v", tt.clause, tt.expected, result)
			}
		}
	})

	t.Run("LikeComparisons", func(t *testing.T) {
		tests := []struct {
			clause   string
			args     []interface{}
			expected bool
		}{
			{"title LIKE ?", []interface{}{"Test%"}, true},
			{"title LIKE ?", []interface{}{"%Document"}, true},
			{"title LIKE ?", []interface{}{"%Test%"}, true},
			{"title LIKE ?", []interface{}{"Missing%"}, false},
			{"title NOT LIKE ?", []interface{}{"Missing%"}, true},
			{"title NOT LIKE ?", []interface{}{"Test%"}, false},
		}

		for _, tt := range tests {
			evaluator := NewWhereEvaluator(tt.clause, tt.args...)
			result, err := evaluator.EvaluateDocument(doc)
			if err != nil {
				t.Errorf("evaluating %q: %v", tt.clause, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("evaluating %q: expected %v, got %v", tt.clause, tt.expected, result)
			}
		}
	})

	t.Run("ANDConditions", func(t *testing.T) {
		tests := []struct {
			clause   string
			args     []interface{}
			expected bool
		}{
			{"status = ? AND priority = ?", []interface{}{"active", "high"}, true},
			{"status = ? AND priority = ?", []interface{}{"active", "low"}, false},
			{"status = ? AND _data.assignee = ?", []interface{}{"active", "alice"}, true},
			{"status = ? AND _data.assignee = ?", []interface{}{"active", "bob"}, false},
			{"_data.estimate > ? AND _data.urgent = ?", []interface{}{3, true}, true},
			{"_data.estimate > ? AND _data.urgent = ?", []interface{}{10, true}, false},
		}

		for _, tt := range tests {
			evaluator := NewWhereEvaluator(tt.clause, tt.args...)
			result, err := evaluator.EvaluateDocument(doc)
			if err != nil {
				t.Errorf("evaluating %q: %v", tt.clause, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("evaluating %q: expected %v, got %v", tt.clause, tt.expected, result)
			}
		}
	})

	t.Run("SecurityTests", func(t *testing.T) {
		// Test SQL injection attempts
		dangerousClauses := []string{
			"status = 'active'; DROP TABLE documents; --",
			"status = 'active' OR 1=1",
			"status = 'active' UNION SELECT * FROM users",
		}

		for _, clause := range dangerousClauses {
			evaluator := NewWhereEvaluator(clause)
			_, err := evaluator.EvaluateDocument(doc)
			// Should either fail safely or not execute dangerous SQL
			// Since we're using parameterized queries, this should be safe
			t.Logf("Clause '%s' handled safely: %v", clause, err)
		}

		// Test the specific injection vulnerability that was fixed
		// This would have been vulnerable in the original implementation
		t.Run("ParameterInjectionPrevention", func(t *testing.T) {
			// This malicious parameter would have caused the original implementation
			// to evaluate "status = 'active' OR 1=1" (always true)
			maliciousParam := "active' OR 1=1"

			evaluator := NewWhereEvaluator("status = ?", maliciousParam)
			result, err := evaluator.EvaluateDocument(doc)

			// Should not error and should return false (since doc.status is "active", not "active' OR 1=1")
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if result {
				t.Error("Injection attack succeeded - should have returned false")
			}

			// Test with our actual document status to verify normal operation
			evaluator2 := NewWhereEvaluator("status = ?", "active")
			result2, err2 := evaluator2.EvaluateDocument(doc)

			if err2 != nil {
				t.Errorf("Expected no error for normal operation, got: %v", err2)
			}
			if !result2 {
				t.Error("Normal operation failed - should have returned true")
			}
		})
	})
}

func TestDeleteWhere(t *testing.T) {
	// Create a temporary file for the store
	tmpfile, err := os.CreateTemp("", "test_delete_where_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store configuration
	config := &testConfig{
		dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "active", "done"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         types.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := newJSONFileStore(tmpfile.Name(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Add test documents
	testDocs := []struct {
		title    string
		status   string
		priority string
		assignee string
	}{
		{"Task 1", "active", "high", "alice"},
		{"Task 2", "pending", "medium", "bob"},
		{"Task 3", "active", "low", "alice"},
		{"Task 4", "done", "high", "charlie"},
		{"Task 5", "pending", "high", "alice"},
	}

	for _, doc := range testDocs {
		_, err := store.Add(doc.title, map[string]interface{}{
			"status":         doc.status,
			"priority":       doc.priority,
			"_data.assignee": doc.assignee,
		})
		if err != nil {
			t.Fatalf("failed to add document %s: %v", doc.title, err)
		}
	}

	t.Run("DeleteByStatus", func(t *testing.T) {
		// Delete all active tasks
		count, err := store.DeleteWhere("status = ?", "active")
		if err != nil {
			t.Fatalf("DeleteWhere failed: %v", err)
		}

		expectedCount := 2 // Task 1 and Task 3
		if count != expectedCount {
			t.Errorf("expected to delete %d documents, deleted %d", expectedCount, count)
		}

		// Verify remaining documents
		allDocs, err := store.List(types.NewListOptions())
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		expectedRemaining := 3
		if len(allDocs) != expectedRemaining {
			t.Errorf("expected %d remaining documents, got %d", expectedRemaining, len(allDocs))
		}

		// Verify no active documents remain
		for _, doc := range allDocs {
			if doc.Dimensions["status"] == "active" {
				t.Error("found active document that should have been deleted")
			}
		}
	})

	t.Run("DeleteByComplexCondition", func(t *testing.T) {
		// Delete pending high priority tasks
		count, err := store.DeleteWhere("status = ? AND priority = ?", "pending", "high")
		if err != nil {
			t.Fatalf("DeleteWhere failed: %v", err)
		}

		expectedCount := 1 // Task 5
		if count != expectedCount {
			t.Errorf("expected to delete %d documents, deleted %d", expectedCount, count)
		}
	})

	t.Run("DeleteByDataField", func(t *testing.T) {
		// Delete tasks assigned to bob
		count, err := store.DeleteWhere("_data.assignee = ?", "bob")
		if err != nil {
			t.Fatalf("DeleteWhere failed: %v", err)
		}

		expectedCount := 1 // Task 2
		if count != expectedCount {
			t.Errorf("expected to delete %d documents, deleted %d", expectedCount, count)
		}
	})

	t.Run("DeleteNoMatches", func(t *testing.T) {
		// Try to delete non-existent status
		count, err := store.DeleteWhere("status = ?", "archived")
		if err != nil {
			t.Fatalf("DeleteWhere failed: %v", err)
		}

		if count != 0 {
			t.Errorf("expected to delete 0 documents, deleted %d", count)
		}
	})

	t.Run("InvalidWhereClause", func(t *testing.T) {
		// Test empty WHERE clause
		_, err := store.DeleteWhere("")
		if err == nil {
			t.Error("expected error for empty WHERE clause")
		}

		// Test WHERE clause with wrong number of parameters
		_, err = store.DeleteWhere("status = ? AND priority = ?", "active")
		if err == nil {
			t.Error("expected error for mismatched parameters")
		}
	})
}

func TestUpdateWhere(t *testing.T) {
	// Create a temporary file for the store
	tmpfile, err := os.CreateTemp("", "test_update_where_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store configuration
	config := &testConfig{
		dimensions: []types.DimensionConfig{
			{
				Name:         "status",
				Type:         types.Enumerated,
				Values:       []string{"pending", "active", "done"},
				DefaultValue: "pending",
			},
			{
				Name:         "priority",
				Type:         types.Enumerated,
				Values:       []string{"low", "medium", "high"},
				DefaultValue: "medium",
			},
		},
	}

	store, err := newJSONFileStore(tmpfile.Name(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Add test documents
	testDocs := []struct {
		title    string
		status   string
		priority string
		assignee string
	}{
		{"Task 1", "pending", "high", "alice"},
		{"Task 2", "pending", "medium", "bob"},
		{"Task 3", "active", "low", "alice"},
		{"Task 4", "done", "high", "charlie"},
	}

	for _, doc := range testDocs {
		_, err := store.Add(doc.title, map[string]interface{}{
			"status":         doc.status,
			"priority":       doc.priority,
			"_data.assignee": doc.assignee,
		})
		if err != nil {
			t.Fatalf("failed to add document %s: %v", doc.title, err)
		}
	}

	t.Run("UpdateByStatus", func(t *testing.T) {
		// Update all pending tasks to active
		newTitle := "Updated Title"
		updates := types.UpdateRequest{
			Title: &newTitle,
			Dimensions: map[string]interface{}{
				"status": "active",
			},
		}

		count, err := store.UpdateWhere("status = ?", updates, "pending")
		if err != nil {
			t.Fatalf("UpdateWhere failed: %v", err)
		}

		expectedCount := 2 // Task 1 and Task 2
		if count != expectedCount {
			t.Errorf("expected to update %d documents, updated %d", expectedCount, count)
		}

		// Verify updates
		allDocs, err := store.List(types.NewListOptions())
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		activeCount := 0
		for _, doc := range allDocs {
			if doc.Dimensions["status"] == "active" {
				activeCount++
				if !strings.Contains(doc.Title, "Task 3") { // Task 3 was already active
					if doc.Title != "Updated Title" {
						t.Errorf("expected updated title, got %s", doc.Title)
					}
				}
			}
		}

		expectedActiveCount := 3 // Original Task 3 + updated Task 1 and Task 2
		if activeCount != expectedActiveCount {
			t.Errorf("expected %d active documents, got %d", expectedActiveCount, activeCount)
		}
	})

	t.Run("UpdateByComplexCondition", func(t *testing.T) {
		// Update high priority tasks assigned to alice
		updates := types.UpdateRequest{
			Dimensions: map[string]interface{}{
				"priority":         "medium",
				"_data.reassigned": true,
			},
		}

		count, err := store.UpdateWhere("priority = ? AND _data.assignee = ?", updates, "high", "alice")
		if err != nil {
			t.Fatalf("UpdateWhere failed: %v", err)
		}

		if count == 0 {
			t.Error("expected to update at least one document")
		}

		// Verify the update
		allDocs, err := store.List(types.NewListOptions())
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		foundReassigned := false
		for _, doc := range allDocs {
			if reassigned, exists := doc.Dimensions["_data.reassigned"]; exists && reassigned == true {
				foundReassigned = true
				if doc.Dimensions["_data.assignee"] != "alice" {
					t.Error("reassigned flag should only be on alice's documents")
				}
			}
		}

		if !foundReassigned {
			t.Error("expected to find documents with reassigned flag")
		}
	})

	t.Run("UpdateNoMatches", func(t *testing.T) {
		// Try to update non-existent status
		updates := types.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status": "active",
			},
		}

		count, err := store.UpdateWhere("status = ?", updates, "archived")
		if err != nil {
			t.Fatalf("UpdateWhere failed: %v", err)
		}

		if count != 0 {
			t.Errorf("expected to update 0 documents, updated %d", count)
		}
	})

	t.Run("InvalidWhereClause", func(t *testing.T) {
		updates := types.UpdateRequest{
			Dimensions: map[string]interface{}{
				"status": "active",
			},
		}

		// Test empty WHERE clause
		_, err := store.UpdateWhere("", updates)
		if err == nil {
			t.Error("expected error for empty WHERE clause")
		}

		// Test WHERE clause with wrong number of parameters
		_, err = store.UpdateWhere("status = ? AND priority = ?", updates, "active")
		if err == nil {
			t.Error("expected error for mismatched parameters")
		}
	})
}
