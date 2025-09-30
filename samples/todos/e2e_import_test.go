package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore"
)

func TestImportFromDirectory(t *testing.T) {
	// Create a temporary todos app
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "todos.json")

	app, err := NewTodoApp(storePath)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create test directory with import files
	importDir := filepath.Join(tempDir, "import")
	if err := os.Mkdir(importDir, 0755); err != nil {
		t.Fatalf("failed to create import directory: %v", err)
	}

	// Create test files to import
	files := map[string]string{
		"task1.txt": `status: active
priority: high
---

Buy groceries

Need to get milk, bread, and eggs for the week.`,
		"task2.txt": `status: pending
priority: medium
---

Write report

Complete the quarterly project summary.`,
		"task3.txt": `status: done
priority: low
---

Exercise routine

30 minutes of running in the park.`,
	}

	for filename, content := range files {
		path := filepath.Join(importDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create import file %s: %v", filename, err)
		}
	}

	// Import documents
	options := nanostore.DefaultImportOptions()
	result, err := nanostore.ImportFromPath(app.store.Store(), importDir, options)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify import results
	if len(result.Imported) != 3 {
		t.Errorf("imported %d documents, want 3", len(result.Imported))
	}

	if len(result.Failed) != 0 {
		t.Errorf("had %d failed imports, want 0: %v", len(result.Failed), result.Failed)
	}

	// Verify documents in store
	todos, err := app.GetAllTodos()
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 3 {
		t.Errorf("store has %d todos, want 3", len(todos))
	}

	// Verify specific documents and their metadata
	expectedTitles := map[string]string{
		"Buy groceries":    "active",
		"Write report":     "pending",
		"Exercise routine": "done",
	}

	for _, todo := range todos {
		expectedStatus, found := expectedTitles[todo.Title]
		if !found {
			t.Errorf("unexpected todo title: %s", todo.Title)
			continue
		}
		if todo.Status != expectedStatus {
			t.Errorf("todo %s has status %s, want %s", todo.Title, todo.Status, expectedStatus)
		}
	}
}

func TestImportWithExistingData(t *testing.T) {
	// Create a todos app with existing data
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "todos.json")

	app, err := NewTodoApp(storePath)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Add existing todo
	existing, err := app.CreateTodo("Existing task", &TodoItem{
		Status:   "active",
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("failed to create existing todo: %v", err)
	}

	// Create import file
	importDir := filepath.Join(tempDir, "import")
	if err := os.Mkdir(importDir, 0755); err != nil {
		t.Fatalf("failed to create import directory: %v", err)
	}

	importFile := filepath.Join(importDir, "new_task.txt")
	content := `status: pending
priority: medium
---

New imported task

This task was imported from a file.`

	if err := os.WriteFile(importFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	// Import
	options := nanostore.DefaultImportOptions()
	result, err := nanostore.ImportFromPath(app.store.Store(), importDir, options)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify import results
	if len(result.Imported) != 1 {
		t.Errorf("imported %d documents, want 1", len(result.Imported))
	}

	// Verify total todos
	todos, err := app.GetAllTodos()
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("store has %d todos, want 2", len(todos))
	}

	// Verify both existing and new todo exist
	titles := make(map[string]bool)
	for _, todo := range todos {
		titles[todo.Title] = true
	}

	if !titles["Existing task"] {
		t.Error("existing task should still be present")
	}
	if !titles["New imported task"] {
		t.Error("imported task should be present")
	}

	// Verify existing task UUID unchanged
	existingDoc, err := app.GetTodo(existing)
	if err != nil {
		t.Fatalf("failed to get existing document: %v", err)
	}
	if existingDoc.UUID != existing {
		t.Errorf("existing document UUID changed")
	}
}

func TestImportExportRoundTrip(t *testing.T) {
	// Create source app with test data
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.json")

	sourceApp, err := NewTodoApp(sourcePath)
	if err != nil {
		t.Fatalf("failed to create source app: %v", err)
	}
	defer sourceApp.Close()

	// Add test todos
	testTodos := []struct {
		title string
		item  *TodoItem
	}{
		{"Task 1", &TodoItem{Status: "active", Priority: "high", Activity: "active"}},
		{"Task 2", &TodoItem{Status: "pending", Priority: "medium", Activity: "active"}},
		{"Task 3", &TodoItem{Status: "done", Priority: "low", Activity: "active"}},
	}

	for _, todo := range testTodos {
		if _, err := sourceApp.CreateTodo(todo.title, todo.item); err != nil {
			t.Fatalf("failed to create todo: %v", err)
		}
	}

	// Export to zip
	exportPath := filepath.Join(tempDir, "export.zip")
	exportOpts := nanostore.ExportOptions{
		DocumentFormat: formats.PlainText,
	}
	if err := nanostore.ExportToPath(sourceApp.store.Store(), exportOpts, exportPath); err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	// Create destination app
	destPath := filepath.Join(tempDir, "dest.json")
	destApp, err := NewTodoApp(destPath)
	if err != nil {
		t.Fatalf("failed to create dest app: %v", err)
	}
	defer destApp.Close()

	// Import from zip
	importOpts := nanostore.DefaultImportOptions()
	result, err := nanostore.ImportFromPath(destApp.store.Store(), exportPath, importOpts)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify import
	if len(result.Imported) != len(testTodos) {
		t.Errorf("imported %d documents, want %d", len(result.Imported), len(testTodos))
	}

	// Verify all todos preserved
	destTodos, err := destApp.GetAllTodos()
	if err != nil {
		t.Fatalf("failed to get dest todos: %v", err)
	}
	if len(destTodos) != len(testTodos) {
		t.Errorf("destination has %d todos, want %d", len(destTodos), len(testTodos))
	}

	// Verify content preserved
	for _, expectedTodo := range testTodos {
		found := false
		for _, actualTodo := range destTodos {
			if actualTodo.Title == expectedTodo.title {
				found = true
				if actualTodo.Status != expectedTodo.item.Status {
					t.Errorf("todo %s status = %s, want %s",
						actualTodo.Title, actualTodo.Status, expectedTodo.item.Status)
				}
				if actualTodo.Priority != expectedTodo.item.Priority {
					t.Errorf("todo %s priority = %s, want %s",
						actualTodo.Title, actualTodo.Priority, expectedTodo.item.Priority)
				}
				break
			}
		}
		if !found {
			t.Errorf("todo %s not found in destination", expectedTodo.title)
		}
	}
}

func TestImportValidation(t *testing.T) {
	// Create app
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "todos.json")

	app, err := NewTodoApp(storePath)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create import directory with invalid files
	importDir := filepath.Join(tempDir, "import")
	if err := os.Mkdir(importDir, 0755); err != nil {
		t.Fatalf("failed to create import directory: %v", err)
	}

	// File with empty title (should fail)
	invalidFile := filepath.Join(importDir, "invalid.txt")
	invalidContent := `status: active
---



This has no title.`

	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to create invalid file: %v", err)
	}

	// Valid file
	validFile := filepath.Join(importDir, "valid.txt")
	validContent := `status: pending
---

Valid task

This task has a proper title.`

	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatalf("failed to create valid file: %v", err)
	}

	// Import
	options := nanostore.DefaultImportOptions()
	result, err := nanostore.ImportFromPath(app.store.Store(), importDir, options)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Should have 1 success and 1 failure
	if len(result.Imported) != 1 {
		t.Errorf("imported %d documents, want 1", len(result.Imported))
	}
	if len(result.Failed) != 1 {
		t.Errorf("failed %d documents, want 1", len(result.Failed))
	}

	// Verify only the valid task was imported
	todos, err := app.GetAllTodos()
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("store has %d todos, want 1", len(todos))
	}

	if len(todos) > 0 && todos[0].Title != "Valid task" {
		t.Errorf("imported todo title = %s, want 'Valid task'", todos[0].Title)
	}
}
