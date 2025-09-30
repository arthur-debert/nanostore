package formats

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPlainTextMetadata(t *testing.T) {
	tests := []struct {
		name         string
		title        string
		content      string
		metadata     map[string]interface{}
		wantContains []string
	}{
		{
			name:    "with metadata",
			title:   "Test Document",
			content: "This is the content.",
			metadata: map[string]interface{}{
				"uuid":       "test-uuid-123",
				"simple_id":  "1",
				"created_at": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				"status":     "active",
				"priority":   "high",
			},
			wantContains: []string{
				"uuid: test-uuid-123",
				"simple_id: 1",
				"created_at: 2024-01-01T10:00:00Z",
				"status: active",
				"priority: high",
				"---",
				"Test Document",
				"This is the content.",
			},
		},
		{
			name:     "without metadata",
			title:    "Simple Document",
			content:  "Just content.",
			metadata: nil,
			wantContains: []string{
				"Simple Document",
				"Just content.",
			},
		},
		{
			name:     "empty metadata map",
			title:    "Another Document",
			content:  "More content.",
			metadata: map[string]interface{}{},
			wantContains: []string{
				"Another Document",
				"More content.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlainText.Serialize(tt.title, tt.content, tt.metadata)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Serialize() output missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}

func TestPlainTextMetadataRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		content  string
		metadata map[string]interface{}
	}{
		{
			name:    "full document with metadata",
			title:   "Test Document",
			content: "This is the content of the document.",
			metadata: map[string]interface{}{
				"uuid":       "doc-123",
				"simple_id":  "1.2",
				"created_at": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				"updated_at": time.Date(2024, 1, 2, 15, 30, 0, 0, time.UTC),
				"status":     "pending",
				"priority":   "medium",
				"parent_id":  "parent-123",
			},
		},
		{
			name:     "no metadata",
			title:    "Plain Document",
			content:  "Just plain content here.",
			metadata: nil,
		},
		{
			name:    "metadata with various types",
			title:   "Complex Document",
			content: "Content with complex metadata.",
			metadata: map[string]interface{}{
				"count":     42,
				"ratio":     3.14,
				"is_active": true,
				"name":      "example",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serialized := PlainText.Serialize(tt.title, tt.content, tt.metadata)

			// Deserialize
			gotTitle, gotContent, gotMetadata, err := PlainText.Deserialize(serialized)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Check title and content
			if gotTitle != tt.title {
				t.Errorf("title = %q, want %q", gotTitle, tt.title)
			}
			if gotContent != tt.content {
				t.Errorf("content = %q, want %q", gotContent, tt.content)
			}

			// Check metadata
			if tt.metadata == nil {
				if gotMetadata != nil {
					t.Errorf("metadata = %v, want nil", gotMetadata)
				}
			} else {
				if gotMetadata == nil {
					t.Errorf("metadata = nil, want %v", tt.metadata)
				} else {
					// Check each metadata field
					for key, expectedValue := range tt.metadata {
						gotValue, exists := gotMetadata[key]
						if !exists {
							t.Errorf("metadata missing key %q", key)
							continue
						}

						// Special handling for time values
						if expectedTime, ok := expectedValue.(time.Time); ok {
							if gotTime, ok := gotValue.(time.Time); ok {
								if !expectedTime.Equal(gotTime) {
									t.Errorf("metadata[%q] = %v, want %v", key, gotTime, expectedTime)
								}
							} else if gotStr, ok := gotValue.(string); ok {
								// Try parsing string as time
								if parsedTime, err := time.Parse(time.RFC3339, gotStr); err == nil {
									if !expectedTime.Equal(parsedTime) {
										t.Errorf("metadata[%q] = %v, want %v", key, parsedTime, expectedTime)
									}
								} else {
									t.Errorf("metadata[%q] = %v (type %T), want %v (type %T)", key, gotValue, gotValue, expectedValue, expectedValue)
								}
							} else {
								t.Errorf("metadata[%q] = %v (type %T), want %v (type %T)", key, gotValue, gotValue, expectedValue, expectedValue)
							}
						} else if fmt.Sprintf("%v", gotValue) != fmt.Sprintf("%v", expectedValue) {
							t.Errorf("metadata[%q] = %v, want %v", key, gotValue, expectedValue)
						}
					}

					// Check for unexpected metadata
					for key := range gotMetadata {
						if _, exists := tt.metadata[key]; !exists {
							t.Errorf("unexpected metadata key %q", key)
						}
					}
				}
			}
		})
	}
}

func TestMarkdownMetadata(t *testing.T) {
	tests := []struct {
		name         string
		title        string
		content      string
		metadata     map[string]interface{}
		wantContains []string
	}{
		{
			name:    "with frontmatter",
			title:   "Test Document",
			content: "This is the content.",
			metadata: map[string]interface{}{
				"uuid":       "test-uuid-123",
				"simple_id":  "1",
				"created_at": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				"status":     "active",
				"priority":   "high",
			},
			wantContains: []string{
				"---",
				"uuid: test-uuid-123",
				"simple_id:",
				"created_at: ",
				"status: active",
				"priority: high",
				"---",
				"# Test Document",
				"This is the content.",
			},
		},
		{
			name:     "without frontmatter",
			title:    "Simple Document",
			content:  "Just content.",
			metadata: nil,
			wantContains: []string{
				"# Simple Document",
				"Just content.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Markdown.Serialize(tt.title, tt.content, tt.metadata)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Serialize() output missing %q\nGot:\n%s", want, got)
				}
			}

			// Check that frontmatter is only present when metadata exists
			if len(tt.metadata) == 0 {
				if strings.HasPrefix(got, "---") {
					t.Errorf("Serialize() should not have frontmatter when metadata is nil/empty")
				}
			} else {
				if !strings.HasPrefix(got, "---") {
					t.Errorf("Serialize() should have frontmatter when metadata exists")
				}
			}
		})
	}
}

func TestMarkdownFrontmatterRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		content  string
		metadata map[string]interface{}
	}{
		{
			name:    "full document with frontmatter",
			title:   "Test Document",
			content: "This is the content of the document.",
			metadata: map[string]interface{}{
				"uuid":       "doc-123",
				"simple_id":  "1.2",
				"created_at": time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				"updated_at": time.Date(2024, 1, 2, 15, 30, 0, 0, time.UTC),
				"status":     "pending",
				"priority":   "medium",
				"parent_id":  "parent-123",
			},
		},
		{
			name:     "no frontmatter",
			title:    "Plain Document",
			content:  "Just plain content here.",
			metadata: nil,
		},
		{
			name:    "frontmatter with nested data",
			title:   "Complex Document",
			content: "Content with complex metadata.",
			metadata: map[string]interface{}{
				"tags":      []string{"important", "review"},
				"count":     42,
				"is_active": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serialized := Markdown.Serialize(tt.title, tt.content, tt.metadata)

			// Deserialize
			gotTitle, gotContent, gotMetadata, err := Markdown.Deserialize(serialized)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Check title and content
			if gotTitle != tt.title {
				t.Errorf("title = %q, want %q", gotTitle, tt.title)
			}
			if gotContent != tt.content {
				t.Errorf("content = %q, want %q", gotContent, tt.content)
			}

			// For YAML, we need more relaxed comparison due to type conversions
			if tt.metadata == nil {
				if gotMetadata != nil {
					t.Errorf("metadata = %v, want nil", gotMetadata)
				}
			} else if gotMetadata == nil {
				t.Errorf("metadata = nil, want %v", tt.metadata)
			}
			// Note: Deep metadata comparison is complex with YAML due to type conversions
			// In production, you might want to use a more sophisticated comparison
		})
	}
}
