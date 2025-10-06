package main

import (
	"os"
	"testing"
	"time"
)

func TestReflectionExecutorIntegration(t *testing.T) {
	// Setup test database
	testDB := "test_reflection.db"
	defer func() { _ = os.Remove(testDB) }()

	// Create registry and executor
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test Task creation
	taskData := map[string]interface{}{
		"status":      "active",
		"priority":    "high",
		"description": "Test task description",
		"assignee":    "test-user",
	}

	// Test Create
	createResult, err := executor.ExecuteCreate("Task", testDB, "Test Task", taskData)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Extract the simple ID
	simpleID, ok := createResult.(string)
	if !ok {
		t.Fatalf("Expected string (simple ID), got %T", createResult)
	}

	if simpleID == "" {
		t.Error("Expected non-empty simple ID")
	}

	// Test Get using the simple ID
	getResult, err := executor.ExecuteGet("Task", testDB, simpleID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	retrievedDoc, ok := getResult.(*TaskDocument)
	if !ok {
		t.Fatalf("Expected *TaskDocument, got %T", getResult)
	}

	if retrievedDoc.Title != "Test Task" {
		t.Errorf("Expected title 'Test Task', got '%s'", retrievedDoc.Title)
	}

	// Test Update
	updateData := map[string]interface{}{
		"status":   "done",
		"priority": "low",
	}

	updateResult, err := executor.ExecuteUpdate("Task", testDB, simpleID, updateData)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	updateCount, ok := updateResult.(int)
	if !ok {
		t.Fatalf("Expected int (update count), got %T", updateResult)
	}

	if updateCount != 1 {
		t.Errorf("Expected update count 1, got %d", updateCount)
	}

	// Verify the update by getting the document again
	getResult2, err := executor.ExecuteGet("Task", testDB, simpleID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	updatedDoc, ok := getResult2.(*TaskDocument)
	if !ok {
		t.Fatalf("Expected *TaskDocument, got %T", getResult2)
	}

	if updatedDoc.Status != "done" {
		t.Errorf("Expected status 'done', got '%s'", updatedDoc.Status)
	}

	// Test List
	query := &Query{
		Groups: []FilterGroup{
			{
				Conditions: []FilterCondition{
					{Field: "status", Operator: "eq", Value: "done"},
				},
			},
		},
	}
	listResult, err := executor.ExecuteList("Task", testDB, query, "created_at", 10, 0)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	// The result should be a slice of documents
	docs, ok := listResult.([]TaskDocument)
	if !ok {
		t.Fatalf("Expected []TaskDocument, got %T", listResult)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document in list result")
	}

	// Test Delete
	err = executor.ExecuteDelete("Task", testDB, simpleID, false)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify deletion - get should fail now
	_, err = executor.ExecuteGet("Task", testDB, simpleID)
	if err == nil {
		t.Error("Expected error when getting deleted document, but got none")
	}
}

func TestReflectionExecutorWithNote(t *testing.T) {
	// Setup test database
	testDB := "test_note_reflection.db"
	defer func() { _ = os.Remove(testDB) }()

	// Create registry and executor
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test Note creation
	noteData := map[string]interface{}{
		"category": "work",
		"tags":     "meeting,q4",
		"content":  "This is a test note content",
	}

	// Test Create
	createResult, err := executor.ExecuteCreate("Note", testDB, "Test Note", noteData)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Extract the simple ID
	simpleID, ok := createResult.(string)
	if !ok {
		t.Fatalf("Expected string (simple ID), got %T", createResult)
	}

	if simpleID == "" {
		t.Error("Expected non-empty simple ID")
	}

	// Get the created document to verify values
	getResult, err := executor.ExecuteGet("Note", testDB, simpleID)
	if err != nil {
		t.Fatalf("Failed to get note: %v", err)
	}

	createdDoc, ok := getResult.(*NoteDocument)
	if !ok {
		t.Fatalf("Expected *NoteDocument, got %T", getResult)
	}

	if createdDoc.Title != "Test Note" {
		t.Errorf("Expected title 'Test Note', got '%s'", createdDoc.Title)
	}

	if createdDoc.Category != "work" {
		t.Errorf("Expected category 'work', got '%s'", createdDoc.Category)
	}

	if createdDoc.Tags != "meeting,q4" {
		t.Errorf("Expected tags 'meeting,q4', got '%s'", createdDoc.Tags)
	}

	if createdDoc.Content != "This is a test note content" {
		t.Errorf("Expected content 'This is a test note content', got '%s'", createdDoc.Content)
	}
}

func TestTypeConversions(t *testing.T) {
	registry := NewEnhancedTypeRegistry()
	if err := registry.LoadBuiltinTypes(); err != nil {
		t.Fatalf("Failed to load builtin types: %v", err)
	}

	executor := NewReflectionExecutor(registry)

	// Test date conversion
	now := time.Now()
	task := &TaskDocument{}
	data := map[string]interface{}{
		"due_date": now.Format(time.RFC3339),
	}

	executor.populateDocumentFromMap(task, data)

	if task.DueDate == nil {
		t.Error("Expected DueDate to be set, but it was nil")
	} else {
		// Allow for small differences due to precision
		diff := task.DueDate.Sub(now)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Expected DueDate to be close to %v, got %v", now, task.DueDate)
		}
	}
}
