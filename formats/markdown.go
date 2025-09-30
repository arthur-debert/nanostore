package formats

import (
	"fmt"
	"regexp"
	"strings"
)

// markdownTitleRegex matches markdown h1 headers (must be at very start, no leading space)
var markdownTitleRegex = regexp.MustCompile(`^#\s+(.+?)[\s]*$`)

// Markdown format implementation
// Serialization: # Title followed by blank line, then content
// Deserialization: extract title from # Title pattern at first line
var Markdown = &DocumentFormat{
	Name:      "markdown",
	Extension: ".md",
	Serialize: func(title, content string) string {
		if title == "" {
			return content
		}
		return "# " + title + "\n\n" + content
	},
	Deserialize: func(document string) (string, string, error) {
		// Empty document check
		if strings.TrimSpace(document) == "" {
			return "", "", fmt.Errorf("empty document: both title and content are empty")
		}

		lines := strings.Split(document, "\n")

		// Check if first line is a markdown title
		if len(lines) > 0 {
			matches := markdownTitleRegex.FindStringSubmatch(lines[0])
			if len(matches) > 1 {
				// Extract title
				title := strings.TrimSpace(matches[1])

				// Find where content starts (after title and any blank lines)
				contentStart := 1
				for contentStart < len(lines) && isBlankLine(lines[contentStart]) {
					contentStart++
				}

				if contentStart < len(lines) {
					content := strings.Join(lines[contentStart:], "\n")
					return title, strings.TrimSpace(content), nil
				}

				// Only title, no content
				if title != "" {
					return title, "", nil
				}
				return "", "", fmt.Errorf("empty document: both title and content are empty")
			}
		}

		// No markdown title found, entire document is content
		return "", document, nil
	},
}

func init() {
	if err := Register(Markdown); err != nil {
		panic(fmt.Sprintf("failed to register Markdown format: %v", err))
	}
}
