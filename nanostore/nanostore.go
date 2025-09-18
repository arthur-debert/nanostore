// Package nanostore provides a document and ID store library that uses JSON file storage
// to manage document storage and dynamically generate user-facing, hierarchical IDs.
//
// This package provides a clean approach to document management with configurable
// ID schemes, automatic hierarchical organization, and human-friendly ID generation.
package nanostore

// New creates a new Store instance with the specified dimension configuration
// The store uses a JSON file backend with file locking for concurrent access
func New(filePath string, config Config) (Store, error) {
	// First validate the configuration
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	return newJSONFileStore(filePath, config)
}
