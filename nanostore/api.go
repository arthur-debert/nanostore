package nanostore

// Store defines the public interface for the document store
type Store interface {
	// List returns documents based on the provided options
	// The returned documents include generated user-facing IDs
	List(opts ListOptions) ([]Document, error)

	// Add creates a new document with the given title and optional parent
	// Returns the UUID of the created document
	Add(title string, parentID *string) (string, error)

	// Update modifies an existing document
	Update(id string, updates UpdateRequest) error

	// SetStatus changes the status of a document
	SetStatus(id string, status Status) error

	// ResolveUUID converts a user-facing ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(userFacingID string) (string, error)

	// Delete removes a document and optionally its children
	// If cascade is true, all child documents are also deleted
	// If cascade is false and the document has children, an error is returned
	Delete(id string, cascade bool) error

	// Close releases any resources held by the store
	Close() error
}

// New creates a new Store instance connected to the given database path
// Use ":memory:" for an in-memory database (useful for testing)
func New(dbPath string) (Store, error) {
	return newStore(dbPath)
}

// NewWithConfig creates a new Store instance with custom dimension configuration
// This allows for configurable ID generation beyond the default status/parent dimensions
func NewWithConfig(dbPath string, config Config) (Store, error) {
	return newStoreWithConfig(dbPath, config)
}
