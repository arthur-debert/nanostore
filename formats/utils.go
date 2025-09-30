package formats

import "strings"

// isBlankLine checks if a line contains only whitespace
func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}
