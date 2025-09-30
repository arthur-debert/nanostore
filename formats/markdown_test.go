package formats

import (
	"testing"
)

func TestMarkdownSerialize(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		want    string
	}{
		{
			name:    "with title and content",
			title:   "My Document",
			content: "This is the content.",
			want:    "# My Document\n\nThis is the content.",
		},
		{
			name:    "empty title",
			title:   "",
			content: "Just content here.",
			want:    "Just content here.",
		},
		{
			name:    "empty content",
			title:   "Just Title",
			content: "",
			want:    "# Just Title\n\n",
		},
		{
			name:    "title with special chars",
			title:   "Title: With Colons & Symbols!",
			content: "Content here.",
			want:    "# Title: With Colons & Symbols!\n\nContent here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializeWithoutMetadata(Markdown, tt.title, tt.content)
			if got != tt.want {
				t.Errorf("Serialize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkdownDeserialize(t *testing.T) {
	tests := []struct {
		name        string
		document    string
		wantTitle   string
		wantContent string
		wantError   bool
	}{
		{
			name:        "standard markdown title",
			document:    "# My Title\n\nContent goes here.",
			wantTitle:   "My Title",
			wantContent: "Content goes here.",
			wantError:   false,
		},
		{
			name:        "title with multiple words",
			document:    "# This is a Long Title\n\nContent.",
			wantTitle:   "This is a Long Title",
			wantContent: "Content.",
			wantError:   false,
		},
		{
			name:        "no space after hash",
			document:    "#Title\n\nThis should not be recognized as title.",
			wantTitle:   "",
			wantContent: "#Title\n\nThis should not be recognized as title.",
			wantError:   false,
		},
		{
			name:        "multiple hashes",
			document:    "## Not H1\n\nContent",
			wantTitle:   "",
			wantContent: "## Not H1\n\nContent",
			wantError:   false,
		},
		{
			name:        "hash not at start",
			document:    " # Not a title\n\nContent",
			wantTitle:   "",
			wantContent: " # Not a title\n\nContent",
			wantError:   false,
		},
		{
			name:        "only title",
			document:    "# Just Title\n\n",
			wantTitle:   "Just Title",
			wantContent: "",
			wantError:   false,
		},
		{
			name:        "title with multiple blank lines",
			document:    "# Title\n\n\n\nContent after blanks.",
			wantTitle:   "Title",
			wantContent: "Content after blanks.",
			wantError:   false,
		},
		{
			name:        "markdown content without title",
			document:    "This is content.\n\n## Section\n\nMore content.",
			wantTitle:   "",
			wantContent: "This is content.\n\n## Section\n\nMore content.",
			wantError:   false,
		},
		{
			name:        "empty document",
			document:    "",
			wantTitle:   "",
			wantContent: "",
			wantError:   true,
		},
		{
			name:        "only whitespace",
			document:    "   \n\t\n   ",
			wantTitle:   "",
			wantContent: "",
			wantError:   true,
		},
		{
			name:        "title with trailing spaces",
			document:    "# Title with spaces   \n\nContent",
			wantTitle:   "Title with spaces",
			wantContent: "Content",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, content, err := deserializeWithoutMetadata(Markdown, tt.document)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if title != tt.wantTitle {
					t.Errorf("title = %q, want %q", title, tt.wantTitle)
				}
				if content != tt.wantContent {
					t.Errorf("content = %q, want %q", content, tt.wantContent)
				}
			}
		})
	}
}

func TestMarkdownRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
	}{
		{
			name:    "simple document",
			title:   "Test Title",
			content: "Test content.",
		},
		{
			name:    "complex markdown",
			title:   "User Guide",
			content: "## Introduction\n\nThis is a guide.\n\n### Section 1\n\nContent here.\n\n- Item 1\n- Item 2",
		},
		{
			name:    "empty title",
			title:   "",
			content: "This is content without a title.\n\nIt doesn't start with # so no ambiguity.",
		},
		{
			name:    "special characters in title",
			title:   "Title: With *Special* [Characters] & More!",
			content: "Content with **bold** and _italic_.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serialized := serializeWithoutMetadata(Markdown, tt.title, tt.content)

			// Deserialize
			gotTitle, gotContent, err := deserializeWithoutMetadata(Markdown, serialized)
			if err != nil {
				t.Fatalf("round-trip deserialization failed: %v", err)
			}

			// Compare
			if gotTitle != tt.title {
				t.Errorf("round-trip title mismatch: got %q, want %q", gotTitle, tt.title)
			}
			if gotContent != tt.content {
				t.Errorf("round-trip content mismatch: got %q, want %q", gotContent, tt.content)
			}
		})
	}
}

func TestMarkdownRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasMatch bool
		title    string
	}{
		{
			name:     "valid h1",
			input:    "# Title",
			hasMatch: true,
			title:    "Title",
		},
		{
			name:     "h1 with multiple words",
			input:    "# Multiple Word Title",
			hasMatch: true,
			title:    "Multiple Word Title",
		},
		{
			name:     "no space after hash",
			input:    "#Title",
			hasMatch: false,
		},
		{
			name:     "h2",
			input:    "## Not H1",
			hasMatch: false,
		},
		{
			name:     "space before hash",
			input:    " # Not H1",
			hasMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := markdownTitleRegex.FindStringSubmatch(tt.input)

			if tt.hasMatch {
				if len(matches) < 2 {
					t.Error("expected regex match but got none")
				} else if matches[1] != tt.title {
					t.Errorf("extracted title = %q, want %q", matches[1], tt.title)
				}
			} else {
				if len(matches) > 0 {
					t.Errorf("expected no match but got %v", matches)
				}
			}
		})
	}
}
