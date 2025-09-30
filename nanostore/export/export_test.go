package export

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/types"
)

// MockStore implements the Store interface for testing
type MockStore struct {
	documents []types.Document
}

func (m *MockStore) List(opts types.ListOptions) ([]types.Document, error) {
	// Simple implementation that respects basic filters
	var result []types.Document

	for _, doc := range m.documents {
		include := true

		// Apply dimension filters
		for key, value := range opts.Filters {
			if docValue, exists := doc.Dimensions[key]; !exists || docValue != value {
				include = false
				break
			}
		}

		if include {
			result = append(result, doc)
		}
	}

	return result, nil
}

func (m *MockStore) Add(title string, dimensions map[string]interface{}) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *MockStore) Update(id string, updates types.UpdateRequest) error {
	return fmt.Errorf("not implemented")
}

func (m *MockStore) ResolveUUID(simpleID string) (string, error) {
	for _, doc := range m.documents {
		if doc.SimpleID == simpleID {
			return doc.UUID, nil
		}
	}
	return "", fmt.Errorf("document not found: %s", simpleID)
}

func (m *MockStore) Delete(id string, cascade bool) error {
	return fmt.Errorf("not implemented")
}

func (m *MockStore) DeleteByDimension(filters map[string]interface{}) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) DeleteWhere(whereClause string, args ...interface{}) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) UpdateByDimension(filters map[string]interface{}, updates types.UpdateRequest) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) UpdateWhere(whereClause string, updates types.UpdateRequest, args ...interface{}) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) UpdateByUUIDs(uuids []string, updates types.UpdateRequest) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) DeleteByUUIDs(uuids []string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockStore) Close() error {
	return nil
}

func createTestDocuments() []types.Document {
	now := time.Now()
	return []types.Document{
		{
			UUID:     "doc1-uuid",
			SimpleID: "1",
			Title:    "First Document",
			Body:     "This is the content of the first document",
			Dimensions: map[string]interface{}{
				"status":   "active",
				"priority": "high",
			},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			UUID:     "doc2-uuid",
			SimpleID: "1.1",
			Title:    "Child Document",
			Body:     "This is a child document",
			Dimensions: map[string]interface{}{
				"status":      "pending",
				"priority":    "medium",
				"parent_uuid": "doc1-uuid",
			},
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-30 * time.Minute),
		},
		{
			UUID:     "doc3-uuid",
			SimpleID: "c2",
			Title:    "Another Document",
			Body:     "Different content here",
			Dimensions: map[string]interface{}{
				"status":   "completed",
				"priority": "low",
			},
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
	}
}

func TestGenerateExportData(t *testing.T) {
	documents := createTestDocuments()
	store := &MockStore{documents: documents}

	tests := []struct {
		name         string
		options      ExportOptions
		expectedDocs int
		expectError  bool
	}{
		{
			name:         "export all documents",
			options:      ExportOptions{},
			expectedDocs: 3,
			expectError:  false,
		},
		{
			name: "export by specific IDs",
			options: ExportOptions{
				IDs: []string{"1", "c2"},
			},
			expectedDocs: 2,
			expectError:  false,
		},
		{
			name: "export by dimension filter",
			options: ExportOptions{
				DimensionFilters: map[string]interface{}{
					"status": "active",
				},
			},
			expectedDocs: 1,
			expectError:  false,
		},
		{
			name: "export with custom filter query",
			options: ExportOptions{
				FilterQuery: "status = 'pending'",
			},
			expectedDocs: 0,
			expectError:  true, // Not implemented yet
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exportData, err := GenerateExportData(store, tt.options)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exportData == nil {
				t.Fatal("exportData is nil")
			}

			// Check archive filename format
			if exportData.ArchiveFilename == "" {
				t.Error("archive filename is empty")
			}

			// Check database file
			if exportData.Contents.DB.Filename != "db.json" {
				t.Errorf("expected database filename to be 'db.json', got %s", exportData.Contents.DB.Filename)
			}

			// Check that database contents are present
			if exportData.Contents.DB.Contents == nil {
				t.Error("database contents are nil")
			}

			// Check number of object files
			if len(exportData.Contents.Objects) != tt.expectedDocs {
				t.Errorf("expected %d object files, got %d", tt.expectedDocs, len(exportData.Contents.Objects))
			}

			// Verify object files have required fields
			for i, obj := range exportData.Contents.Objects {
				if obj.Filename == "" {
					t.Errorf("object %d has empty filename", i)
				}
				if obj.Content == "" {
					t.Errorf("object %d has empty content", i)
				}
				if obj.Created.IsZero() || obj.Modified.IsZero() {
					t.Errorf("object %d has zero timestamps", i)
				}
			}
		})
	}
}

func TestExportDataJSONSerialization(t *testing.T) {
	documents := createTestDocuments()
	store := &MockStore{documents: documents}

	exportData, err := GenerateExportData(store, ExportOptions{})
	if err != nil {
		t.Fatalf("failed to generate export data: %v", err)
	}

	// Test JSON serialization
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal export data: %v", err)
	}

	// Test JSON deserialization
	var deserializedData ExportData
	err = json.Unmarshal(jsonData, &deserializedData)
	if err != nil {
		t.Fatalf("failed to unmarshal export data: %v", err)
	}

	// Verify key fields are preserved
	if deserializedData.ArchiveFilename != exportData.ArchiveFilename {
		t.Error("archive filename mismatch after serialization")
	}

	if deserializedData.Contents.DB.Filename != exportData.Contents.DB.Filename {
		t.Error("database filename mismatch after serialization")
	}

	if len(deserializedData.Contents.Objects) != len(exportData.Contents.Objects) {
		t.Error("object count mismatch after serialization")
	}
}

func TestGetDocumentsByIDs(t *testing.T) {
	documents := createTestDocuments()
	store := &MockStore{documents: documents}

	tests := []struct {
		name         string
		ids          []string
		expectedDocs int
	}{
		{
			name:         "get by simple IDs",
			ids:          []string{"1", "c2"},
			expectedDocs: 2,
		},
		{
			name:         "get by UUIDs",
			ids:          []string{"doc1-uuid", "doc2-uuid"},
			expectedDocs: 2,
		},
		{
			name:         "mixed IDs and UUIDs",
			ids:          []string{"1", "doc3-uuid"},
			expectedDocs: 2,
		},
		{
			name:         "non-existent ID",
			ids:          []string{"non-existent"},
			expectedDocs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs, err := getDocumentsByIDs(store, tt.ids)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(docs) != tt.expectedDocs {
				t.Errorf("expected %d documents, got %d", tt.expectedDocs, len(docs))
			}
		})
	}
}

func TestGetStoreData(t *testing.T) {
	documents := createTestDocuments()
	store := &MockStore{documents: documents}

	storeData, err := getStoreData(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if storeData == nil {
		t.Fatal("store data is nil")
	}

	if len(storeData.Documents) != len(documents) {
		t.Errorf("expected %d documents in store data, got %d", len(documents), len(storeData.Documents))
	}

	if storeData.Metadata.Version != "1.0" {
		t.Errorf("expected version '1.0', got %s", storeData.Metadata.Version)
	}
}

func TestExportWithDocumentFormats(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	documents := []types.Document{
		{
			UUID:      "test-uuid",
			SimpleID:  "1",
			Title:     "Test Document",
			Body:      "This is the document content.",
			CreatedAt: testTime,
			UpdatedAt: testTime,
			Dimensions: map[string]interface{}{
				"status":   "active",
				"priority": "high",
			},
		},
	}
	store := &MockStore{documents: documents}

	tests := []struct {
		name             string
		format           *formats.DocumentFormat
		expectedExt      string
		expectedContains []string
	}{
		{
			name:        "plaintext format",
			format:      formats.PlainText,
			expectedExt: ".txt",
			expectedContains: []string{
				"uuid: test-uuid",
				"simple_id: 1",
				"created_at: 2024-01-15T10:30:00Z",
				"status: active",
				"priority: high",
				"---",
				"Test Document",
				"This is the document content.",
			},
		},
		{
			name:        "markdown format",
			format:      formats.Markdown,
			expectedExt: ".md",
			expectedContains: []string{
				"---",
				"uuid: test-uuid",
				"# Test Document",
				"This is the document content.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := ExportOptions{
				DocumentFormat: tt.format,
			}

			exportData, err := GenerateExportData(store, options)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(exportData.Contents.Objects) != 1 {
				t.Fatalf("expected 1 object, got %d", len(exportData.Contents.Objects))
			}

			obj := exportData.Contents.Objects[0]

			// Check filename extension
			if !strings.HasSuffix(obj.Filename, tt.expectedExt) {
				t.Errorf("expected filename to end with %s, got %s", tt.expectedExt, obj.Filename)
			}

			// Check serialized content contains expected strings
			for _, expected := range tt.expectedContains {
				if !strings.Contains(obj.Content, expected) {
					t.Errorf("expected content to contain %q\nGot:\n%s", expected, obj.Content)
				}
			}
		})
	}
}
