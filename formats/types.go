package formats

import (
	"fmt"
	"strings"
)

// DocumentFormat defines how documents are serialized and deserialized
type DocumentFormat struct {
	// Name is the format identifier (alphanumeric, dashes, underscores, lowercase)
	Name string

	// Extension is the file extension including the dot (e.g., ".txt", ".md")
	Extension string

	// Serialize converts title and content into the formatted document string
	Serialize func(title, content string) string

	// Deserialize extracts title and content from the formatted document string
	// Returns empty title if none found, error if both title and content are empty
	Deserialize func(document string) (title string, content string, err error)
}

// registry holds all available document formats
var registry = make(map[string]*DocumentFormat)

// Register adds a new document format to the registry
func Register(format *DocumentFormat) error {
	// Validate format name (alphanumeric, dashes, underscores, lowercase)
	if !isValidFormatName(format.Name) {
		return fmt.Errorf("invalid format name %q: must be lowercase alphanumeric with dashes and underscores only", format.Name)
	}

	// Normalize extension
	if !strings.HasPrefix(format.Extension, ".") {
		format.Extension = "." + format.Extension
	}

	// Check if format already exists
	if _, exists := registry[format.Name]; exists {
		return fmt.Errorf("format %q already registered", format.Name)
	}

	registry[format.Name] = format
	return nil
}

// Get returns a document format by name
func Get(name string) (*DocumentFormat, error) {
	format, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unknown format %q", name)
	}
	return format, nil
}

// List returns all registered format names
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// isValidFormatName checks if a format name is valid
func isValidFormatName(name string) bool {
	if name == "" {
		return false
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
