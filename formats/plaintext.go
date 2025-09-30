package formats

import (
	"fmt"
	"strings"
)

// PlainText format implementation
// Serialization: title on first line, blank line, then content
// Deserialization: if first line followed by blank line, use as title
var PlainText = &DocumentFormat{
	Name:      "plaintext",
	Extension: ".txt",
	Serialize: func(title, content string) string {
		if title == "" {
			return content
		}
		return title + "\n\n" + content
	},
	Deserialize: func(document string) (string, string, error) {
		// Empty document check
		if strings.TrimSpace(document) == "" {
			return "", "", fmt.Errorf("empty document: both title and content are empty")
		}

		lines := strings.Split(document, "\n")

		// If we have at least 2 lines and the second is blank
		if len(lines) >= 2 && isBlankLine(lines[1]) {
			// First line is title, rest is content
			title := strings.TrimSpace(lines[0])

			// Find where content starts (after blank lines)
			contentStart := 2
			for contentStart < len(lines) && isBlankLine(lines[contentStart]) {
				contentStart++
			}

			// Check if we have content
			var content string
			if contentStart < len(lines) {
				content = strings.TrimSpace(strings.Join(lines[contentStart:], "\n"))
			}

			// Return title and content (content may be empty)
			if title != "" || content != "" {
				return title, content, nil
			}
			return "", "", fmt.Errorf("empty document: both title and content are empty")
		}

		// No title pattern found, entire document is content
		return "", document, nil
	},
}

func init() {
	if err := Register(PlainText); err != nil {
		panic(fmt.Sprintf("failed to register PlainText format: %v", err))
	}
}
