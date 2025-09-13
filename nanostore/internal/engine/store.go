package engine

import (
	"database/sql"
	"embed"
	"fmt"
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
	id := uuid.New().String()
	now := time.Now().Unix()

	query, err := loadQuery("queries/insert_document.sql")
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(query, id, title, "", "pending", parentID, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	return id, nil
}

// Update modifies an existing document
func (s *store) Update(id string, updates types.UpdateRequest) error {
	query, err := loadQuery("queries/update_document.sql")
	if err != nil {
		return err
	}

	// Direct access to strongly-typed fields from the public type
	result, err := s.db.Exec(query, updates.Title, updates.Body, id)
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

	return nil
}

// SetStatus changes the status of a document
func (s *store) SetStatus(id string, status types.Status) error {
	query, err := loadQuery("queries/set_status.sql")
	if err != nil {
		return err
	}

	// Convert Status to string for SQL query
	result, err := s.db.Exec(query, string(status), id)
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

	return nil
}

// List returns documents based on options
func (s *store) List(opts types.ListOptions) ([]types.Document, error) {
	// TODO: In the future, use opts to filter results
	// For now, we'll implement the basic list functionality

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
