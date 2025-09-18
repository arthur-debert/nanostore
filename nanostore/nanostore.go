// Package nanostore provides a document and ID store library that uses SQLite
// to manage document storage and dynamically generate user-facing, contiguous IDs.
//
// This package replaces pkg/idm and parts of pkg/too/store with a cleaner,
// more focused approach to document management with configurable ID schemes.
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
