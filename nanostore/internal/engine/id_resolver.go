package engine

import (
	"database/sql"
	"fmt"
	"strings"
)

// idPart represents a parsed component of a hierarchical ID
type idPart struct {
	status string
	offset int
}

// ResolveUUID converts a user-facing ID to a UUID using optimized queries
// Handles hierarchical IDs like "1", "c2", "1.2", "1.c3" with a single query
func (s *store) ResolveUUID(userFacingID string) (string, error) {
	// Validate input doesn't contain SQL injection attempts
	if strings.ContainsAny(userFacingID, "'\"`;\\") {
		return "", fmt.Errorf("invalid ID format: contains illegal characters")
	}

	// Split the ID by dots to handle hierarchical IDs
	parts := strings.Split(userFacingID, ".")

	// Parse all parts first to validate format
	parsedParts := make([]idPart, len(parts))

	for i, part := range parts {
		// Extract status and number from the part
		var status string
		var number int

		if strings.HasPrefix(part, "c") {
			status = "completed"
			consumed, err := fmt.Sscanf(part[1:], "%d", &number)
			if err != nil || consumed != 1 || len(part[1:]) != len(fmt.Sprintf("%d", number)) {
				return "", fmt.Errorf("invalid ID format: %s", part)
			}
		} else {
			status = "pending"
			consumed, err := fmt.Sscanf(part, "%d", &number)
			if err != nil || consumed != 1 || len(part) != len(fmt.Sprintf("%d", number)) {
				return "", fmt.Errorf("invalid ID format: %s", part)
			}
		}

		// Validate number is positive
		if number < 1 {
			return "", fmt.Errorf("invalid ID format: number must be positive")
		}

		parsedParts[i] = idPart{
			status: status,
			offset: number - 1, // Convert to 0-based offset
		}
	}

	// Choose the appropriate query based on depth
	var query string
	var args []interface{}
	var err error

	switch len(parts) {
	case 1:
		// Simple root document
		query, err = loadQuery("queries/resolve_path_optimized.sql")
		if err != nil {
			return "", err
		}
		args = []interface{}{parsedParts[0].status, parsedParts[0].offset}

	case 2:
		// Two-level hierarchy
		query, err = loadQuery("queries/resolve_hierarchical_2.sql")
		if err != nil {
			return "", err
		}
		args = []interface{}{
			parsedParts[0].status, parsedParts[0].offset,
			parsedParts[1].status, parsedParts[1].offset,
		}

	case 3:
		// Three-level hierarchy
		query, err = loadQuery("queries/resolve_hierarchical_3.sql")
		if err != nil {
			return "", err
		}
		args = []interface{}{
			parsedParts[0].status, parsedParts[0].offset,
			parsedParts[1].status, parsedParts[1].offset,
			parsedParts[2].status, parsedParts[2].offset,
		}

	default:
		// For deeper nesting, fall back to the original iterative approach
		// This could be extended with more queries for 4, 5, etc. levels
		return s.resolveUUIDIterative(userFacingID, parsedParts)
	}

	// Execute the single query
	var uuid string
	err = s.db.QueryRow(query, args...).Scan(&uuid)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("document not found for ID: %s", userFacingID)
	} else if err != nil {
		return "", fmt.Errorf("failed to resolve ID: %w", err)
	}

	return uuid, nil
}

// resolveUUIDIterative is the fallback for very deep hierarchies (4+ levels)
func (s *store) resolveUUIDIterative(userFacingID string, parts []idPart) (string, error) {
	// Start with root documents
	currentParentUUID := sql.NullString{Valid: false}
	var finalUUID string

	for i, part := range parts {
		// Query to find the document at this level
		query, err := loadQuery("queries/resolve_id.sql")
		if err != nil {
			return "", err
		}

		var uuid string
		var scanErr error

		if currentParentUUID.Valid {
			// Looking for a child document
			scanErr = s.db.QueryRow(query, currentParentUUID.String, currentParentUUID.String, part.status, part.offset).Scan(&uuid)
		} else {
			// Looking for a root document
			scanErr = s.db.QueryRow(query, nil, nil, part.status, part.offset).Scan(&uuid)
		}

		if scanErr == sql.ErrNoRows {
			return "", fmt.Errorf("document not found for ID: %s", userFacingID)
		} else if scanErr != nil {
			return "", fmt.Errorf("failed to resolve ID: %w", scanErr)
		}

		// Update parent for next iteration
		currentParentUUID = sql.NullString{String: uuid, Valid: true}

		// If this is the last part, this is our final UUID
		if i == len(parts)-1 {
			finalUUID = uuid
		}
	}

	return finalUUID, nil
}
