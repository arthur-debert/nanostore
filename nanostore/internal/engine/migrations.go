package engine

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// migrate runs all pending migrations
func (s *store) migrate() error {
	// Get current version
	currentVersion, err := s.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Get all migration files
	entries, err := sqlFiles.ReadDir("sql/schema")
	if err != nil {
		return fmt.Errorf("failed to read schema directory: %w", err)
	}

	// Sort migration files by version number
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	// Apply migrations
	for _, migration := range migrations {
		// Extract version number from filename (e.g., "001_initial.sql" -> 1)
		parts := strings.Split(migration, "_")
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid migration filename %s: %w", migration, err)
		}

		if version <= currentVersion {
			continue // Skip already applied migrations
		}

		// Read and execute migration
		content, err := sqlFiles.ReadFile(filepath.Join("sql/schema", migration))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration, err)
		}

		// Update version
		if err := s.updateVersion(version); err != nil {
			return fmt.Errorf("failed to update version after %s: %w", migration, err)
		}
	}

	return nil
}

// getCurrentVersion returns the current schema version
func (s *store) getCurrentVersion() (int, error) {
	var version int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		// Table doesn't exist yet, this is fine
		return 0, nil
	}
	return version, nil
}

// updateVersion records a new schema version
func (s *store) updateVersion(version int) error {
	_, err := s.db.Exec(
		"INSERT INTO schema_version (version, applied_at) VALUES (?, strftime('%s', 'now'))",
		version,
	)
	return err
}
