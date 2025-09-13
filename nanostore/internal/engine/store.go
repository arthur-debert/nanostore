package engine

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/types"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed all:sql
var sqlFiles embed.FS

// store implements the Store interface
type store struct {
	db *sql.DB
}

// New creates a new store instance.
// Returns the concrete store type since we can't import the Engine interface
// from the parent package (would cause circular dependency).
func New(dbPath string) (*store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	s := &store{db: db}

	// Run migrations
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

// Close releases database resources
func (s *store) Close() error {
	return s.db.Close()
}

// Add creates a new document
func (s *store) Add(title string, parentID *string) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback is a no-op if tx has been committed

	id := uuid.New().String()
	now := time.Now().Unix()

	query, err := loadQuery("queries/insert_document.sql")
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(query, id, title, "", "pending", parentID, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}

// Update modifies an existing document
func (s *store) Update(id string, updates types.UpdateRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback is a no-op if tx has been committed

	// If parent is being updated, check for circular references
	if updates.ParentID != nil {
		// Empty string means "make root", which is always safe
		if *updates.ParentID != "" {
			// First check that we're not setting a document as its own parent
			if *updates.ParentID == id {
				return fmt.Errorf("cannot set document as its own parent")
			}

			// Check if this would create a circular reference
			checkQuery, err := loadQuery("queries/check_circular_reference.sql")
			if err != nil {
				return err
			}

			var wouldBeCircular bool
			err = tx.QueryRow(checkQuery, *updates.ParentID, id).Scan(&wouldBeCircular)
			if err != nil {
				return fmt.Errorf("failed to check for circular reference: %w", err)
			}

			if wouldBeCircular {
				return fmt.Errorf("cannot set parent: would create circular reference")
			}
		}
	}

	// Choose the appropriate update query
	var query string
	var args []interface{}

	if updates.ParentID != nil {
		// Use the query that handles parent updates
		query, err = loadQuery("queries/update_document_with_parent.sql")
		if err != nil {
			return err
		}
		// Pass parent value as-is (nil check already done above)
		args = []interface{}{updates.Title, updates.Body, *updates.ParentID, *updates.ParentID, id}
	} else {
		// Use the simpler query when parent is not being updated
		query, err = loadQuery("queries/update_document.sql")
		if err != nil {
			return err
		}
		args = []interface{}{updates.Title, updates.Body, id}
	}

	// Execute the update
	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SetStatus changes the status of a document
func (s *store) SetStatus(id string, status types.Status) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback is a no-op if tx has been committed

	query, err := loadQuery("queries/set_status.sql")
	if err != nil {
		return err
	}

	// Convert Status to string for SQL query
	result, err := tx.Exec(query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// List returns documents based on options
func (s *store) List(opts types.ListOptions) ([]types.Document, error) {
	// If we have filters, we need to use the templated query
	if hasFilters(opts) {
		return s.listWithFilters(opts)
	}

	// No filters - use the simple query
	query, err := loadQuery("queries/list.sql")
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Using the shared Document type
	var results []types.Document
	for rows.Next() {
		var (
			uuid         string
			userFacingID string
			title        string
			body         string
			status       string
			parentUUID   sql.NullString
			createdAt    int64
			updatedAt    int64
		)

		err := rows.Scan(&uuid, &userFacingID, &title, &body, &status, &parentUUID, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build Document using the shared type
		doc := types.Document{
			UUID:         uuid,
			UserFacingID: userFacingID,
			Title:        title,
			Body:         body,
			Status:       types.Status(status), // Convert string to Status type
			CreatedAt:    time.Unix(createdAt, 0),
			UpdatedAt:    time.Unix(updatedAt, 0),
		}

		// Handle optional parent UUID
		if parentUUID.Valid {
			doc.ParentUUID = &parentUUID.String
		}

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// hasFilters checks if any filters are specified
func hasFilters(opts types.ListOptions) bool {
	return len(opts.FilterByStatus) > 0 ||
		opts.FilterByParent != nil ||
		opts.FilterBySearch != ""
}

// listWithFilters handles filtered queries by building the WHERE clauses into the CTEs
func (s *store) listWithFilters(opts types.ListOptions) ([]types.Document, error) {
	// Special case: filtering by a specific parent (not root)
	if opts.FilterByParent != nil && *opts.FilterByParent != "" {
		return s.listBySpecificParent(opts)
	}

	// If we have status or search filters, use the simple filtered query
	// because these filters can break the hierarchical tree structure
	if len(opts.FilterByStatus) > 0 || opts.FilterBySearch != "" {
		return s.listSimpleFiltered(opts)
	}

	// Load the base query template for hierarchical queries
	baseQuery, err := loadQuery("queries/list_base.sql")
	if err != nil {
		return nil, err
	}

	// Build WHERE conditions and args
	var conditions []string
	var args []interface{}

	// Filter by status
	if len(opts.FilterByStatus) > 0 {
		placeholders := make([]string, len(opts.FilterByStatus))
		for i, status := range opts.FilterByStatus {
			placeholders[i] = "?"
			args = append(args, string(status))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by search (searches in title and body)
	if opts.FilterBySearch != "" {
		searchPattern := "%" + opts.FilterBySearch + "%"
		conditions = append(conditions, "(title LIKE ? OR body LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	// Build the WHERE clause string
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	// Handle parent filter for root documents only
	rootWhereClause := whereClause
	childWhereClause := whereClause

	if opts.FilterByParent != nil && *opts.FilterByParent == "" {
		// Filter for root documents only - child_docs should return nothing
		childWhereClause = " AND 1=0" // This ensures no children are selected
	}

	// Replace placeholders in the query
	finalQuery := strings.Replace(baseQuery, "{{ROOT_WHERE_CLAUSE}}", rootWhereClause, 1)
	finalQuery = strings.Replace(finalQuery, "{{CHILD_WHERE_CLAUSE}}", childWhereClause, 1)

	// We need to duplicate args for both root and child clauses
	// Since both CTEs use the same WHERE conditions, we need the args twice
	allArgs := make([]interface{}, 0, len(args)*2)
	allArgs = append(allArgs, args...) // For root_docs
	allArgs = append(allArgs, args...) // For child_docs

	rows, err := s.db.Query(finalQuery, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents with filters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Parse results using the same logic as the non-filtered version
	var results []types.Document
	for rows.Next() {
		var doc types.Document
		var parentUUID sql.NullString
		var createdUnix, updatedUnix int64

		err := rows.Scan(
			&doc.UUID,
			&doc.UserFacingID,
			&doc.Title,
			&doc.Body,
			&doc.Status,
			&parentUUID,
			&createdUnix,
			&updatedUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert nullable parent UUID
		if parentUUID.Valid {
			doc.ParentUUID = &parentUUID.String
		}

		// Convert Unix timestamps to time.Time
		doc.CreatedAt = time.Unix(createdUnix, 0)
		doc.UpdatedAt = time.Unix(updatedUnix, 0)

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// listBySpecificParent handles listing direct children of a specific parent
func (s *store) listBySpecificParent(opts types.ListOptions) ([]types.Document, error) {
	// Load the parent-specific query
	baseQuery, err := loadQuery("queries/list_by_parent.sql")
	if err != nil {
		return nil, err
	}

	// Start with parent UUID as first arg
	args := []interface{}{*opts.FilterByParent}

	// Build additional WHERE conditions
	var conditions []string

	// Filter by status
	if len(opts.FilterByStatus) > 0 {
		placeholders := make([]string, len(opts.FilterByStatus))
		for i, status := range opts.FilterByStatus {
			placeholders[i] = "?"
			args = append(args, string(status))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by search
	if opts.FilterBySearch != "" {
		searchPattern := "%" + opts.FilterBySearch + "%"
		conditions = append(conditions, "(title LIKE ? OR body LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	// Build additional WHERE clause
	additionalWhere := ""
	if len(conditions) > 0 {
		additionalWhere = " AND " + strings.Join(conditions, " AND ")
	}

	// Replace placeholder in query
	finalQuery := strings.Replace(baseQuery, "{{ADDITIONAL_WHERE}}", additionalWhere, 1)

	rows, err := s.db.Query(finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list children of parent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Parse results
	var results []types.Document
	for rows.Next() {
		var doc types.Document
		var parentUUID sql.NullString
		var createdUnix, updatedUnix int64

		err := rows.Scan(
			&doc.UUID,
			&doc.UserFacingID,
			&doc.Title,
			&doc.Body,
			&doc.Status,
			&parentUUID,
			&createdUnix,
			&updatedUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert nullable parent UUID
		if parentUUID.Valid {
			doc.ParentUUID = &parentUUID.String
		}

		// Convert Unix timestamps to time.Time
		doc.CreatedAt = time.Unix(createdUnix, 0)
		doc.UpdatedAt = time.Unix(updatedUnix, 0)

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// listSimpleFiltered handles queries with status/search filters using a simple non-hierarchical approach
func (s *store) listSimpleFiltered(opts types.ListOptions) ([]types.Document, error) {
	// Load the simple filtered query
	baseQuery, err := loadQuery("queries/list_filtered.sql")
	if err != nil {
		return nil, err
	}

	// Build WHERE conditions and args
	var conditions []string
	var args []interface{}

	// Filter by status
	if len(opts.FilterByStatus) > 0 {
		placeholders := make([]string, len(opts.FilterByStatus))
		for i, status := range opts.FilterByStatus {
			placeholders[i] = "?"
			args = append(args, string(status))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by search
	if opts.FilterBySearch != "" {
		searchPattern := "%" + opts.FilterBySearch + "%"
		conditions = append(conditions, "(title LIKE ? OR body LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	// Filter by parent (only root documents)
	if opts.FilterByParent != nil && *opts.FilterByParent == "" {
		conditions = append(conditions, "parent_uuid IS NULL")
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	// Replace placeholder in query
	finalQuery := strings.Replace(baseQuery, "{{WHERE_CLAUSE}}", whereClause, 1)

	rows, err := s.db.Query(finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list with simple filter: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Parse results
	var results []types.Document
	for rows.Next() {
		var doc types.Document
		var parentUUID sql.NullString
		var createdUnix, updatedUnix int64

		err := rows.Scan(
			&doc.UUID,
			&doc.UserFacingID,
			&doc.Title,
			&doc.Body,
			&doc.Status,
			&parentUUID,
			&createdUnix,
			&updatedUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert nullable parent UUID
		if parentUUID.Valid {
			doc.ParentUUID = &parentUUID.String
		}

		// Convert Unix timestamps to time.Time
		doc.CreatedAt = time.Unix(createdUnix, 0)
		doc.UpdatedAt = time.Unix(updatedUnix, 0)

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// Delete removes a document and optionally its children
func (s *store) Delete(id string, cascade bool) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// If not cascading, check if the document has children
	if !cascade {
		checkQuery, err := loadQuery("queries/check_has_children.sql")
		if err != nil {
			return err
		}

		var hasChildren bool
		err = tx.QueryRow(checkQuery, id).Scan(&hasChildren)
		if err != nil {
			return fmt.Errorf("failed to check for children: %w", err)
		}

		if hasChildren {
			return fmt.Errorf("cannot delete document with children unless cascade is true")
		}
	}

	// Load the appropriate delete query
	var deleteQuery string
	if cascade {
		deleteQuery, err = loadQuery("queries/delete_cascade.sql")
		if err != nil {
			return err
		}
	} else {
		deleteQuery, err = loadQuery("queries/delete_document.sql")
		if err != nil {
			return err
		}
	}

	// Execute the delete
	result, err := tx.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
