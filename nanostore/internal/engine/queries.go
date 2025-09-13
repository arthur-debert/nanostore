package engine

import (
	"fmt"
	"path/filepath"
)

// loadQuery loads a SQL query from the embedded filesystem
func loadQuery(filename string) (string, error) {
	content, err := sqlFiles.ReadFile(filepath.Join("sql", filename))
	if err != nil {
		return "", fmt.Errorf("failed to load query %s: %w", filename, err)
	}
	return string(content), nil
}
