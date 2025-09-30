package imports

import (
	"testing"
	"time"
)

func TestValidateDocument(t *testing.T) {
	tests := []struct {
		name    string
		doc     ImportDocument
		wantErr bool
	}{
		{
			name: "valid document",
			doc: ImportDocument{
				Title: "Valid Title",
				Body:  "Content",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			doc: ImportDocument{
				Title: "",
				Body:  "Content",
			},
			wantErr: true,
		},
		{
			name: "invalid UUID format",
			doc: ImportDocument{
				Title: "Title",
				UUID:  stringPtr("not-a-valid-uuid"),
			},
			wantErr: true,
		},
		{
			name: "valid UUID",
			doc: ImportDocument{
				Title: "Title",
				UUID:  stringPtr("550e8400-e29b-41d4-a716-446655440000"),
			},
			wantErr: false,
		},
		{
			name: "invalid created date",
			doc: ImportDocument{
				Title:     "Title",
				CreatedAt: timePtr(time.Time{}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDocument(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDocument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataPrecedence(t *testing.T) {
	doc := ImportDocument{
		Title:      "Test Doc",
		Body:       "Content",
		SourceFile: "test.txt",
		Dimensions: make(map[string]interface{}),
	}

	// DB metadata (lowest precedence)
	dbMeta := map[string]interface{}{
		"uuid":       "db-uuid",
		"created_at": "2024-01-01T10:00:00Z",
		"status":     "pending",
		"priority":   "low",
	}

	// File metadata (highest precedence)
	fileMeta := map[string]interface{}{
		"uuid":     "file-uuid",
		"status":   "active",
		"category": "important",
	}

	// Apply metadata with precedence
	applyMetadataToDocument(&doc, dbMeta)
	applyMetadataToDocument(&doc, fileMeta)

	// Check results
	if doc.UUID == nil || *doc.UUID != "file-uuid" {
		t.Errorf("UUID should be from file metadata, got %v", doc.UUID)
	}

	if doc.Dimensions["status"] != "active" {
		t.Errorf("status should be from file metadata, got %v", doc.Dimensions["status"])
	}

	if doc.Dimensions["priority"] != "low" {
		t.Errorf("priority should be from db metadata, got %v", doc.Dimensions["priority"])
	}

	if doc.Dimensions["category"] != "important" {
		t.Errorf("category should be from file metadata, got %v", doc.Dimensions["category"])
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
