package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore"
)

func TestMetadataExport(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test-todos.json")

	// Create a new todos app
	app, err := NewTodoApp(storePath)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Add some todos with different metadata
	todo1ID, err := app.store.Create("Buy groceries", &TodoItem{
		Priority: "medium",
		Status:   "pending",
		Activity: "active",
	})
	if err != nil {
		t.Fatalf("failed to create todo 1: %v", err)
	}

	todo2ID, err := app.store.Create("Write report", &TodoItem{
		Priority:    "high",
		Status:      "active",
		Activity:    "active",
		Description: "Quarterly sales report",
	})
	if err != nil {
		t.Fatalf("failed to create todo 2: %v", err)
	}

	// Add a subtask
	_, err = app.store.Create("Include charts", &TodoItem{
		Priority: "medium",
		Status:   "pending",
		Activity: "active",
		ParentID: todo2ID,
	})
	if err != nil {
		t.Fatalf("failed to create subtask: %v", err)
	}

	// Mark first todo as done
	err = app.store.Update(todo1ID, &TodoItem{
		Status: "done",
	})
	if err != nil {
		t.Fatalf("failed to update todo: %v", err)
	}

	// Test plaintext export
	t.Run("plaintext export", func(t *testing.T) {
		exportPath := filepath.Join(tempDir, "plaintext-export.zip")
		options := nanostore.ExportOptions{
			DocumentFormat: formats.PlainText,
		}

		err = nanostore.ExportToPath(app.store.Store(), options, exportPath)
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Extract and check content
		content, err := extractFirstFile(exportPath, ".txt")
		if err != nil {
			t.Fatalf("failed to extract file: %v", err)
		}

		// Check for metadata in plaintext
		expectedStrings := []string{
			"uuid:",
			"simple_id:",
			"created_at:",
			"updated_at:",
			"status:",
			"priority:",
			"activity:",
			"---",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(content, expected) {
				t.Errorf("plaintext export missing %q\nContent:\n%s", expected, content)
			}
		}
	})

	// Test markdown export
	t.Run("markdown export", func(t *testing.T) {
		exportPath := filepath.Join(tempDir, "markdown-export.zip")
		options := nanostore.ExportOptions{
			DocumentFormat: formats.Markdown,
		}

		err = nanostore.ExportToPath(app.store.Store(), options, exportPath)
		if err != nil {
			t.Fatalf("failed to export: %v", err)
		}

		// Extract and check content
		content, err := extractFirstFile(exportPath, ".md")
		if err != nil {
			t.Fatalf("failed to extract file: %v", err)
		}

		// Check for frontmatter
		if !strings.HasPrefix(content, "---\n") {
			t.Error("markdown export should start with frontmatter")
		}

		// Check for metadata in frontmatter
		expectedStrings := []string{
			"uuid:",
			"simple_id:",
			"created_at:",
			"status:",
			"priority:",
			"---",
			"# ", // Markdown title
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(content, expected) {
				t.Errorf("markdown export missing %q\nContent:\n%s", expected, content)
			}
		}
	})
}

func extractFirstFile(archivePath, extension string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, extension) {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			return string(content), nil
		}
	}

	return "", os.ErrNotExist
}
