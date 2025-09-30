package export

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/types"
)

// generateFilename creates a filename for an object using the format:
// <uuid>-<order>-<title>.<ext> or <uuid>-<title>.<ext>
// where title is sanitized according to the rules:
// 1. Only alphanumeric, dash, underscore
// 2. Spaces replaced with dash
// 3. Order (in canonical) is prefixed if object has order
// 4. Truncated to 40 chars
// 5. If no title, use first 40 chars of content with same rules
func generateFilename(doc types.Document, format *formats.DocumentFormat) string {
	uuid := doc.UUID
	title := doc.Title
	content := doc.Body

	// If no title, use first 40 chars of content
	if title == "" && content != "" {
		if len(content) > 40 {
			title = content[:40]
		} else {
			title = content
		}
	}

	// If still no title, use a default
	if title == "" {
		title = "untitled"
	}

	// Sanitize the title
	sanitized := sanitizeTitle(title)

	// Check if document has canonical order (SimpleID contains numbers/letters indicating order)
	orderPrefix := extractOrderFromSimpleID(doc.SimpleID)

	// Build filename: <uuid>-[<order>-]<sanitized_title>.<ext>
	// Use format's extension or default to .txt
	ext := format.Extension
	if ext == "" {
		ext = ".txt"
	}

	var filename string
	if orderPrefix != "" {
		filename = uuid + "-" + orderPrefix + "-" + sanitized + ext
	} else {
		filename = uuid + "-" + sanitized + ext
	}

	return filename
}

// sanitizeTitle cleans a title according to export rules
func sanitizeTitle(title string) string {
	// Convert to lowercase and replace spaces with dashes
	result := strings.ToLower(title)
	result = strings.ReplaceAll(result, " ", "-")

	// Keep only alphanumeric, dash, underscore
	var builder strings.Builder
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}

	// Remove consecutive dashes
	result = builder.String()
	re := regexp.MustCompile("-+")
	result = re.ReplaceAllString(result, "-")

	// Trim leading/trailing dashes
	result = strings.Trim(result, "-")

	// Truncate to 40 chars
	if len(result) > 40 {
		result = result[:40]
	}

	// If empty after sanitization, use default
	if result == "" {
		result = "untitled"
	}

	return result
}

// extractOrderFromSimpleID extracts the order prefix from a SimpleID
// For example: "1.2.c3" -> "1-2-c3", "c2" -> "c2", "1" -> "1"
func extractOrderFromSimpleID(simpleID string) string {
	if simpleID == "" {
		return ""
	}

	// Replace dots with dashes for order prefix
	return strings.ReplaceAll(simpleID, ".", "-")
}
