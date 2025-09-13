package engine

// Engine defines the internal storage engine interface with concrete types.
// This interface uses strongly-typed parameters and return values to ensure
// compile-time type safety and eliminate the need for runtime type assertions.
type Engine interface {
	// List returns documents matching the provided options.
	// The returned documents are strongly typed, eliminating the need
	// for type assertions in the adapter layer.
	List(opts ListOptions) ([]Document, error)

	// Add creates a new document with the given title and optional parent.
	// Returns the UUID of the created document.
	Add(title string, parentID *string) (string, error)

	// Update modifies an existing document using a strongly-typed request.
	// This eliminates the previous map[string]*string approach that required
	// runtime type checking.
	Update(id string, updates UpdateRequest) error

	// SetStatus changes the status of a document.
	// Status is passed as a string internally but validated at the API boundary.
	SetStatus(id string, status string) error

	// ResolveUUID converts a user-facing ID to a UUID.
	// Examples: "1" -> uuid, "1.2" -> uuid, "1.c1" -> uuid
	ResolveUUID(userFacingID string) (string, error)

	// Close releases any resources held by the engine.
	Close() error
}
