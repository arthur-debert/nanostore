package export

import (
	"testing"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/types"
)

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name     string
		doc      types.Document
		expected string
	}{
		{
			name: "basic title with order",
			doc: types.Document{
				UUID:     "abc123",
				SimpleID: "1",
				Title:    "My Document",
				Body:     "Some content",
			},
			expected: "abc123-1-my-document.txt",
		},
		{
			name: "hierarchical order",
			doc: types.Document{
				UUID:     "def456",
				SimpleID: "1.2.c3",
				Title:    "Nested Document",
				Body:     "Nested content",
			},
			expected: "def456-1-2-c3-nested-document.txt",
		},
		{
			name: "no title, use body content",
			doc: types.Document{
				UUID:     "ghi789",
				SimpleID: "c2",
				Title:    "",
				Body:     "This is the body content that will be used as title",
			},
			expected: "ghi789-c2-this-is-the-body-content-that-will-be-us.txt",
		},
		{
			name: "title with special characters",
			doc: types.Document{
				UUID:     "jkl012",
				SimpleID: "5",
				Title:    "Title with Special! @#$% Characters & Spaces",
				Body:     "Content",
			},
			expected: "jkl012-5-title-with-special-characters-spaces.txt",
		},
		{
			name: "long title truncation",
			doc: types.Document{
				UUID:     "mno345",
				SimpleID: "1",
				Title:    "This is a very long title that should be truncated to exactly forty characters maximum",
				Body:     "Content",
			},
			expected: "mno345-1-this-is-a-very-long-title-that-should-be.txt",
		},
		{
			name: "no title and no body",
			doc: types.Document{
				UUID:     "pqr678",
				SimpleID: "2",
				Title:    "",
				Body:     "",
			},
			expected: "pqr678-2-untitled.txt",
		},
		{
			name: "no order in SimpleID",
			doc: types.Document{
				UUID:     "stu901",
				SimpleID: "",
				Title:    "Document Without Order",
				Body:     "Content",
			},
			expected: "stu901-document-without-order.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFilename(tt.doc, formats.PlainText)
			if result != tt.expected {
				t.Errorf("generateFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic text",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "special characters",
			input:    "Special! @#$% Characters",
			expected: "special-characters",
		},
		{
			name:     "multiple spaces",
			input:    "Multiple    Spaces   Here",
			expected: "multiple-spaces-here",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Leading and Trailing  ",
			expected: "leading-and-trailing",
		},
		{
			name:     "consecutive dashes",
			input:    "Some--Text---With----Dashes",
			expected: "some-text-with-dashes",
		},
		{
			name:     "underscores preserved",
			input:    "Keep_Underscores_Here",
			expected: "keep_underscores_here",
		},
		{
			name:     "numbers preserved",
			input:    "Document 123 Version 2",
			expected: "document-123-version-2",
		},
		{
			name:     "long text truncation",
			input:    "This is a very long text that exceeds forty characters and should be truncated",
			expected: "this-is-a-very-long-text-that-exceeds-fo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "untitled",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTitle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateFilenameWithFormats(t *testing.T) {
	doc := types.Document{
		UUID:     "test123",
		SimpleID: "1",
		Title:    "Test Document",
		Body:     "Some content",
	}

	tests := []struct {
		name     string
		format   *formats.DocumentFormat
		expected string
	}{
		{
			name:     "plaintext format",
			format:   formats.PlainText,
			expected: "test123-1-test-document.txt",
		},
		{
			name:     "markdown format",
			format:   formats.Markdown,
			expected: "test123-1-test-document.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFilename(doc, tt.format)
			if result != tt.expected {
				t.Errorf("generateFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractOrderFromSimpleID(t *testing.T) {
	tests := []struct {
		name     string
		simpleID string
		expected string
	}{
		{
			name:     "single number",
			simpleID: "1",
			expected: "1",
		},
		{
			name:     "single letter",
			simpleID: "c",
			expected: "c",
		},
		{
			name:     "hierarchical with dots",
			simpleID: "1.2.c3",
			expected: "1-2-c3",
		},
		{
			name:     "complex hierarchy",
			simpleID: "1.a.2.b.3",
			expected: "1-a-2-b-3",
		},
		{
			name:     "empty string",
			simpleID: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOrderFromSimpleID(tt.simpleID)
			if result != tt.expected {
				t.Errorf("extractOrderFromSimpleID() = %v, want %v", result, tt.expected)
			}
		})
	}
}
