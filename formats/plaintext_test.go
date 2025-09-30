package formats

import (
	"testing"
)

func TestPlainTextSerialize(t *testing.T) {
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
			want:    "My Document\n\nThis is the content.",
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
			want:    "Just Title\n\n",
		},
		{
			name:    "multiline content",
			title:   "Title",
			content: "Line 1\nLine 2\nLine 3",
			want:    "Title\n\nLine 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializeWithoutMetadata(PlainText, tt.title, tt.content)
			if got != tt.want {
				t.Errorf("Serialize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlainTextDeserialize(t *testing.T) {
	tests := []struct {
		name        string
		document    string
		wantTitle   string
		wantContent string
		wantError   bool
	}{
		{
			name:        "title and content",
			document:    "My Title\n\nContent goes here.",
			wantTitle:   "My Title",
			wantContent: "Content goes here.",
			wantError:   false,
		},
		{
			name:        "title with multiple blank lines",
			document:    "My Title\n\n\n\nContent after blanks.",
			wantTitle:   "My Title",
			wantContent: "Content after blanks.",
			wantError:   false,
		},
		{
			name:        "no blank line after first line",
			document:    "This is all content\nNo blank line here",
			wantTitle:   "",
			wantContent: "This is all content\nNo blank line here",
			wantError:   false,
		},
		{
			name:        "only title",
			document:    "Just Title\n\n",
			wantTitle:   "Just Title",
			wantContent: "",
			wantError:   false,
		},
		{
			name:        "whitespace lines",
			document:    "Title\n   \t   \nContent",
			wantTitle:   "Title",
			wantContent: "Content",
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
			name:        "multiline content",
			document:    "Title\n\nLine 1\nLine 2\nLine 3",
			wantTitle:   "Title",
			wantContent: "Line 1\nLine 2\nLine 3",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, content, err := deserializeWithoutMetadata(PlainText, tt.document)

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

func TestPlainTextRoundTrip(t *testing.T) {
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
			name:    "multiline document",
			title:   "Complex Document",
			content: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
		},
		{
			name:    "empty title",
			title:   "",
			content: "Content without title.",
		},
		{
			name:    "special characters",
			title:   "Title with 特殊字符 and émojis 🎉",
			content: "Content with various characters: @#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serialized := serializeWithoutMetadata(PlainText, tt.title, tt.content)

			// Deserialize
			gotTitle, gotContent, err := deserializeWithoutMetadata(PlainText, serialized)
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
