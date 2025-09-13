package engine

import (
	"database/sql"
	"fmt"
	"strings"
)

// ResolveUUID converts a user-facing ID to a UUID
// Handles hierarchical IDs like "1", "c2", "1.2", "1.c3"
func (s *store) ResolveUUID(userFacingID string) (string, error) {
	// Validate input doesn't contain SQL injection attempts
	if strings.ContainsAny(userFacingID, "'\"`;\\") {
		return "", fmt.Errorf("invalid ID format: contains illegal characters")
	}

	// Split the ID by dots to handle hierarchical IDs
	parts := strings.Split(userFacingID, ".")

	// Start with root documents
	currentParentUUID := sql.NullString{Valid: false}
	var finalUUID string

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

		// Query to find the document at this level
		query, err := loadQuery("queries/resolve_id.sql")
		if err != nil {
			return "", err
		}

		var uuid string
		var scanErr error

		if currentParentUUID.Valid {
			// Looking for a child document
			scanErr = s.db.QueryRow(query, currentParentUUID.String, currentParentUUID.String, status, number-1).Scan(&uuid)
		} else {
			// Looking for a root document
			scanErr = s.db.QueryRow(query, nil, nil, status, number-1).Scan(&uuid)
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
