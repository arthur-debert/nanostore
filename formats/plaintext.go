package formats

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PlainText format implementation
// Serialization:
//   - If metadata exists: metadata section, separator (---), blank line, title, blank line, content
//   - If no metadata: title on first line, blank line, then content
//
// Deserialization: parses metadata section if present, then title/content
var PlainText = &DocumentFormat{
	Name:      "plaintext",
	Extension: ".txt",
	Serialize: func(title, content string, metadata map[string]interface{}) string {
		var result strings.Builder

		// Add metadata section if present
		if len(metadata) > 0 {
			for key, value := range metadata {
				result.WriteString(key)
				result.WriteString(": ")
				result.WriteString(formatValue(value))
				result.WriteString("\n")
			}
			result.WriteString("---\n\n")
		}

		// Add title if present
		if title != "" {
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

		lines := strings.Split(document, "\n")
		var metadata map[string]interface{}
		var contentStartIndex int

		// Check for metadata section
		if hasMetadataSection(lines) {
			var err error
			metadata, contentStartIndex, err = parseMetadataSection(lines)
			if err != nil {
				return "", "", nil, err
			}
			lines = lines[contentStartIndex:]
		}

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
				return title, content, metadata, nil
			}
			return "", "", nil, fmt.Errorf("empty document: both title and content are empty")
		}

		// No title pattern found, entire document is content
		return "", strings.Join(lines, "\n"), metadata, nil
	},
}

func init() {
	if err := Register(PlainText); err != nil {
		panic(fmt.Sprintf("failed to register PlainText format: %v", err))
	}
}

// hasMetadataSection checks if the document starts with a metadata section
func hasMetadataSection(lines []string) bool {
	// Look for key: value pattern in first line and separator within reasonable distance
	if len(lines) < 2 {
		return false
	}

	// Check first line has key: value pattern
	if !strings.Contains(lines[0], ": ") {
		return false
	}

	// Look for separator line within first 20 lines
	for i := 1; i < len(lines) && i < 20; i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return true
		}
	}

	return false
}

// parseMetadataSection parses the metadata section and returns the metadata and index where content starts
func parseMetadataSection(lines []string) (map[string]interface{}, int, error) {
	metadata := make(map[string]interface{})
	separatorIndex := -1

	// Find the separator
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			separatorIndex = i
			break
		}
	}

	if separatorIndex == -1 {
		return nil, 0, fmt.Errorf("metadata section found but no separator")
	}

	// Parse metadata lines
	for i := 0; i < separatorIndex; i++ {
		line := lines[i]
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := parseValue(strings.TrimSpace(parts[1]))
			metadata[key] = value
		}
	}

	// Skip separator and any blank lines after it
	contentStart := separatorIndex + 1
	for contentStart < len(lines) && isBlankLine(lines[contentStart]) {
		contentStart++
	}

	return metadata, contentStart, nil
}

// formatValue converts a value to string representation
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// parseValue attempts to parse a string value into appropriate type
func parseValue(s string) interface{} {
	// Try to parse as time
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	// Try to parse as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}

	// Try to parse as int
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	// Try to parse as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Default to string
	return s
}
