package formats

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// markdownTitleRegex matches markdown h1 headers (must be at very start, no leading space)
var markdownTitleRegex = regexp.MustCompile(`^#\s+(.+?)[\s]*$`)

// Markdown format implementation
// Serialization:
//   - If metadata exists: YAML frontmatter, then # Title, blank line, content
//   - If no metadata: # Title followed by blank line, then content
//
// Deserialization: extract frontmatter if present, then title from # Title pattern
var Markdown = &DocumentFormat{
	Name:      "markdown",
	Extension: ".md",
	Serialize: func(title, content string, metadata map[string]interface{}) string {
		var result strings.Builder

		// Add frontmatter if metadata exists
		if len(metadata) > 0 {
			result.WriteString("---\n")
			yamlBytes, err := yaml.Marshal(metadata)
			if err == nil {
				result.Write(yamlBytes)
			}
			result.WriteString("---\n\n")
		}

		// Add title if present
		if title != "" {
			result.WriteString("# ")
			result.WriteString(title)
			result.WriteString("\n\n")
		}

		// Add content
		result.WriteString(content)

		return result.String()
	},
	Deserialize: func(document string) (string, string, map[string]interface{}, error) {
		// Empty document check
		if strings.TrimSpace(document) == "" {
			return "", "", nil, fmt.Errorf("empty document: both title and content are empty")
		}

		var metadata map[string]interface{}
		var contentStartIndex int

		// Check for frontmatter
		if strings.HasPrefix(document, "---\n") {
			var err error
			metadata, contentStartIndex, err = parseFrontmatter(document)
			if err != nil {
				return "", "", nil, err
			}
			document = document[contentStartIndex:]
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
					return title, strings.TrimSpace(content), metadata, nil
				}

				// Only title, no content
				if title != "" {
					return title, "", metadata, nil
				}
				return "", "", nil, fmt.Errorf("empty document: both title and content are empty")
			}
		}

		// No markdown title found, entire document is content
		return "", document, metadata, nil
	},
}

func init() {
	if err := Register(Markdown); err != nil {
		panic(fmt.Sprintf("failed to register Markdown format: %v", err))
	}
}

// parseFrontmatter parses YAML frontmatter from the document
func parseFrontmatter(document string) (map[string]interface{}, int, error) {
	// Look for closing ---
	endIndex := strings.Index(document[4:], "\n---\n")
	if endIndex == -1 {
		// Try with different line endings
		endIndex = strings.Index(document[4:], "\n---\r\n")
		if endIndex == -1 {
			return nil, 0, fmt.Errorf("frontmatter opening found but no closing delimiter")
		}
	}

	// Extract YAML content
	yamlContent := document[4 : endIndex+4]

	// Parse YAML
	metadata := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return nil, 0, fmt.Errorf("failed to parse frontmatter YAML: %w", err)
	}

	// Calculate where content starts (after closing --- and newline)
	contentStart := endIndex + 4 + 4 // 4 for "---\n" at start, endIndex+4 for content, 4 for "\n---\n"

	// Skip any blank lines after frontmatter
	remaining := document[contentStart:]
	for strings.HasPrefix(remaining, "\n") || strings.HasPrefix(remaining, "\r\n") {
		if strings.HasPrefix(remaining, "\r\n") {
			contentStart += 2
			remaining = remaining[2:]
		} else {
			contentStart++
			remaining = remaining[1:]
		}
	}

	return metadata, contentStart, nil
}
