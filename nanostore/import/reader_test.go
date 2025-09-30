package imports

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore/storage"
	"github.com/arthur-debert/nanostore/types"
)

func TestReadImportDataFromDirectory(t *testing.T) {
	// Create test directory structure
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"doc1.txt": "Document 1\n\nThis is the content of document 1.",
		"doc2.txt": `status: active
priority: high
created_at: 2024-01-01T10:00:00Z
---

Document 2

This is the content of document 2.`,
		"doc3.md": `---
status: done
tags: [important, review]
---

# Document 3

This is the content of document 3.`,
	}

	for filename, content := range files {
		path := filepath.Join(tempDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Test reading PlainText files
	t.Run("read plaintext files", func(t *testing.T) {
		importData, err := ReadImportDataFromDirectory(tempDir, formats.PlainText)
		if err != nil {
			t.Fatalf("ReadImportDataFromDirectory() error = %v", err)
		}

		// Should read 2 .txt files
		if len(importData.Documents) != 2 {
			t.Errorf("got %d documents, want 2", len(importData.Documents))
		}

		// Check metadata was parsed
		for _, doc := range importData.Documents {
			if doc.SourceFile == "doc2.txt" {
				if doc.Dimensions["status"] != "active" {
					t.Errorf("doc2 should have status=active")
				}
				if doc.Dimensions["priority"] != "high" {
					t.Errorf("doc2 should have priority=high")
				}
			}
		}
	})

	// Test reading Markdown files
	t.Run("read markdown files", func(t *testing.T) {
		importData, err := ReadImportDataFromDirectory(tempDir, formats.Markdown)
		if err != nil {
			t.Fatalf("ReadImportDataFromDirectory() error = %v", err)
		}

		// Should read 1 .md file
		if len(importData.Documents) != 1 {
			t.Errorf("got %d documents, want 1", len(importData.Documents))
		}

		doc := importData.Documents[0]
		if doc.Title != "Document 3" {
			t.Errorf("got title %q, want %q", doc.Title, "Document 3")
		}
		if doc.Dimensions["status"] != "done" {
			t.Errorf("doc3 should have status=done")
		}
	})

	// Test with db.json
	t.Run("with db.json metadata", func(t *testing.T) {
		// Create a db.json file
		storeData := storage.StoreData{
			Documents: []types.Document{
				{
					UUID:      "doc1-uuid",
					SimpleID:  "1",
					Title:     "Document 1",
					Body:      "Original content",
					CreatedAt: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2023, 12, 2, 10, 0, 0, 0, time.UTC),
					Dimensions: map[string]interface{}{
						"status": "archived",
					},
				},
			},
			Metadata: storage.Metadata{
				Version: "1.0",
			},
		}

		dbJSON, _ := json.Marshal(storeData)
		dbPath := filepath.Join(tempDir, "db.json")
		if err := os.WriteFile(dbPath, dbJSON, 0644); err != nil {
			t.Fatalf("failed to create db.json: %v", err)
		}

		importData, err := ReadImportDataFromDirectory(tempDir, formats.PlainText)
		if err != nil {
			t.Fatalf("ReadImportDataFromDirectory() error = %v", err)
		}

		// File content should override db.json for title/body
		for _, doc := range importData.Documents {
			if doc.SourceFile == "doc1.txt" {
				if doc.Title != "Document 1" {
					t.Errorf("title should come from file, got %q", doc.Title)
				}
				if doc.Body != "This is the content of document 1." {
					t.Errorf("body should come from file")
				}
			}
		}
	})
}

func TestReadImportDataFromZip(t *testing.T) {
	// Create test zip file
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	w := zip.NewWriter(zipFile)

	// Add files to zip
	files := map[string]string{
		"doc1.txt": "Document 1\n\nContent 1",
		"doc2.txt": `uuid: test-uuid-123
status: active
---

Document 2

Content 2`,
		"db.json": `{
			"documents": [{
				"uuid": "doc1-uuid",
				"simple_id": "1",
				"title": "Original Title",
				"body": "Original Body",
				"dimensions": {"priority": "low"},
				"created_at": "2024-01-01T10:00:00Z",
				"updated_at": "2024-01-02T10:00:00Z"
			}],
			"metadata": {"version": "1.0"}
		}`,
	}

	for filename, content := range files {
		f, err := w.Create(filename)
		if err != nil {
			t.Fatalf("failed to create file in zip: %v", err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write to zip: %v", err)
		}
	}

	_ = w.Close()
	_ = zipFile.Close()

	// Test reading zip
	importData, err := ReadImportDataFromZip(zipPath)
	if err != nil {
		t.Fatalf("ReadImportDataFromZip() error = %v", err)
	}

	// Check results
	if len(importData.Documents) != 2 {
		t.Errorf("got %d documents, want 2", len(importData.Documents))
	}

	// Check it detected as export
	if importData.Metadata.ImportedFrom != "export" {
		t.Errorf("should detect as nanostore export, got %s", importData.Metadata.ImportedFrom)
	}

	// Check metadata precedence
	for _, doc := range importData.Documents {
		if doc.SourceFile == "doc2.txt" {
			// File metadata should override db.json
			if doc.UUID == nil || *doc.UUID != "test-uuid-123" {
				t.Errorf("UUID should be from file metadata")
			}
			if doc.Dimensions["status"] != "active" {
				t.Errorf("status should be from file metadata")
			}
		}
	}
}

func TestDetectFormatFromExtensions(t *testing.T) {
	tests := []struct {
		name       string
		counts     map[string]int
		wantFormat *formats.DocumentFormat
	}{
		{
			name:       "majority markdown",
			counts:     map[string]int{".md": 5, ".txt": 2},
			wantFormat: formats.Markdown,
		},
		{
			name:       "majority plaintext",
			counts:     map[string]int{".txt": 10, ".md": 1},
			wantFormat: formats.PlainText,
		},
		{
			name:       "unknown format defaults to plaintext",
			counts:     map[string]int{".doc": 5},
			wantFormat: formats.PlainText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormatFromExtensions(tt.counts)
			if got != tt.wantFormat {
				t.Errorf("detectFormatFromExtensions() = %v, want %v", got.Name, tt.wantFormat.Name)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  time.Time
		ok    bool
	}{
		{
			name:  "time.Time value",
			value: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			want:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			ok:    true,
		},
		{
			name:  "RFC3339 string",
			value: "2024-01-01T10:00:00Z",
			want:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			ok:    true,
		},
		{
			name:  "date only string",
			value: "2024-01-01",
			want:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ok:    true,
		},
		{
			name:  "invalid string",
			value: "not a date",
			want:  time.Time{},
			ok:    false,
		},
		{
			name:  "non-string value",
			value: 42,
			want:  time.Time{},
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTime(tt.value)
			if ok != tt.ok {
				t.Errorf("parseTime() ok = %v, want %v", ok, tt.ok)
			}
			if ok && !got.Equal(tt.want) {
				t.Errorf("parseTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
