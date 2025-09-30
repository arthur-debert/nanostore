package types

import "time"

// Document represents a document in the store with its generated ID
type Document struct {
	UUID       string                 // Stable internal identifier
	SimpleID   string                 // Generated ID like "1", "c2", "1.2.c3"
	Title      string                 // Document title
	Body       string                 // Optional document body
	Dimensions map[string]interface{} // All dimension values and data (data prefixed with "_data.")
	CreatedAt  time.Time              // Creation timestamp
	UpdatedAt  time.Time              // Last update timestamp
}

// ListOptions configures how documents are listed
type ListOptions struct {
	// Filters allows filtering by any configured dimension
	// Key is dimension name, value can be a single value or slice of values
	// Example: {"status": []string{"active", "pending"}, "priority": "high"}
	Filters map[string]interface{}

	// FilterBySearch performs a text search on title and body
	// Empty string returns all documents (no filtering)
	FilterBySearch string

	// OrderBy specifies the order of results
	// Each OrderClause contains a field name and direction
	OrderBy []OrderClause

	// Limit specifies the maximum number of results to return
	// nil or negative values mean no limit
	// 0 returns no results
	Limit *int

	// Offset specifies the number of results to skip
	// nil or negative values mean no offset (start from beginning)
	// Values greater than result count return empty results
	Offset *int
}

// OrderClause represents a single ORDER BY clause
type OrderClause struct {
	Column     string
	Descending bool
}

// NewListOptions creates a new ListOptions with empty filters
func NewListOptions() ListOptions {
	return ListOptions{
		Filters: make(map[string]interface{}),
	}
}

// UpdateRequest specifies fields to update on a document
type UpdateRequest struct {
	Title      *string
	Body       *string
	Dimensions map[string]interface{} // Optional: dimension values to update (e.g., "status": "completed", "parent_uuid": "some-uuid")
}

// Store defines the public interface for the document store
type Store interface {
	// List returns documents based on the provided options
	// The returned documents include generated user-facing IDs
	List(opts ListOptions) ([]Document, error)

	// Add creates a new document with the given title and dimension values
	// The dimensions map allows setting any dimension values, including:
	// - Enumerated dimensions (e.g., "status": "pending")
	// - Hierarchical dimensions (e.g., "parent_uuid": "parent-id")
	// Dimensions not specified will use their default values from the configuration
	// Returns the UUID of the created document
	Add(title string, dimensions map[string]interface{}) (string, error)

	// Update modifies an existing document
	Update(id string, updates UpdateRequest) error

	// ResolveUUID converts a simple ID (e.g., "1.2.c3") to a UUID
	ResolveUUID(simpleID string) (string, error)

	// Delete removes a document and optionally its children
	// If cascade is true, all child documents are also deleted
	// If cascade is false and the document has children, an error is returned
	Delete(id string, cascade bool) error

	// DeleteByDimension removes all documents matching dimension filters
	// For example: DeleteByDimension(map[string]interface{}{"status": "archived"})
	// Multiple filters are combined with AND
	// Returns the number of documents deleted
	DeleteByDimension(filters map[string]interface{}) (int, error)

	// DeleteWhere removes all documents matching a custom WHERE clause
	// The where clause should not include the "WHERE" keyword itself
	// For example: DeleteWhere("status = 'archived' AND priority = 'low'")
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents deleted
	DeleteWhere(whereClause string, args ...interface{}) (int, error)

	// UpdateByDimension updates all documents matching dimension filters
	// For example: UpdateByDimension(map[string]interface{}{"status": "pending"}, UpdateRequest{Title: &newTitle})
	// Multiple filters are combined with AND
	// Returns the number of documents updated
	UpdateByDimension(filters map[string]interface{}, updates UpdateRequest) (int, error)

	// UpdateWhere updates all documents matching a custom WHERE clause
	// The where clause should not include the "WHERE" keyword itself
	// For example: UpdateWhere("created_at < ?", UpdateRequest{...}, time.Now().AddDate(0, -1, 0))
	// Use with caution as it allows arbitrary SQL conditions
	// Returns the number of documents updated
	UpdateWhere(whereClause string, updates UpdateRequest, args ...interface{}) (int, error)

	// UpdateByUUIDs updates multiple documents by their UUIDs in a single operation
	// For example: UpdateByUUIDs([]string{"uuid1", "uuid2"}, UpdateRequest{Title: &newTitle})
	// Returns the number of documents successfully updated
	UpdateByUUIDs(uuids []string, updates UpdateRequest) (int, error)

	// DeleteByUUIDs deletes multiple documents by their UUIDs in a single operation
	// For example: DeleteByUUIDs([]string{"uuid1", "uuid2"})
	// Returns the number of documents successfully deleted
	DeleteByUUIDs(uuids []string) (int, error)

	// Close releases any resources held by the store
	Close() error
}
