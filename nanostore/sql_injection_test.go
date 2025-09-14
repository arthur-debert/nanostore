package nanostore_test

import (
	"strings"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func TestSQLInjectionInAdd(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Various SQL injection attempts
	injectionAttempts := []string{
		"'; DROP TABLE documents; --",
		"\" OR 1=1; --",
		"'); DELETE FROM documents; --",
		"1'; UPDATE documents SET status='completed'; --",
		"\\'; DROP TABLE documents; --",
		"''; DROP TABLE documents; --''",
		"'; INSERT INTO documents (uuid) VALUES ('evil'); --",
		"'; ATTACH DATABASE ':memory:' AS evil; --",
		"'; PRAGMA foreign_keys=OFF; --",
		"Robert'); DROP TABLE documents;--",
		"1' UNION SELECT * FROM documents--",
		"' OR '1'='1",
	}

	// All of these should be safely handled as data, not SQL
	for i, attempt := range injectionAttempts {
		id, err := store.Add(attempt, nil, nil)
		if err != nil {
			t.Errorf("failed to add document with injection attempt %d: %v", i, err)
			continue
		}

		// Verify the document was created with the exact title
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list after injection attempt %d: %v", i, err)
		}

		found := false
		for _, doc := range docs {
			if doc.UUID == id && doc.Title == attempt {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("injection attempt %d: document not found or title mismatch", i)
		}
	}

	// Verify all documents still exist (nothing was dropped/deleted)
	finalDocs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to final list: %v", err)
	}

	if len(finalDocs) != len(injectionAttempts) {
		t.Errorf("expected %d documents, got %d - possible injection succeeded",
			len(injectionAttempts), len(finalDocs))
	}
}

func TestSQLInjectionInUpdate(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	id1, err := store.Add("Document 1", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document 1: %v", err)
	}

	id2, err := store.Add("Document 2", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document 2: %v", err)
	}

	// SQL injection attempts in update
	injectionAttempts := []string{
		"Updated'; DROP TABLE documents; --",
		"Title' WHERE 1=1; --",
		"'; UPDATE documents SET title='Hacked' WHERE 1=1; --",
		"' || (SELECT uuid FROM documents LIMIT 1) || '",
		"', status='completed' WHERE uuid != '",
	}

	for i, attempt := range injectionAttempts {
		err := store.Update(id1, nanostore.UpdateRequest{
			Title: &attempt,
		})
		if err != nil {
			t.Errorf("failed to update with injection attempt %d: %v", i, err)
		}
	}

	// Verify only the target document was updated
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after updates: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d - injection may have affected data", len(docs))
	}

	// Verify document 2 was not affected
	for _, doc := range docs {
		if doc.UUID == id2 && doc.Title != "Document 2" {
			t.Errorf("document 2 was modified: %s", doc.Title)
		}
	}
}

func TestSQLInjectionInResolveUUID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a document
	id, err := store.Add("Test", nil, nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// SQL injection attempts in ID resolution
	// These contain invalid characters or formats that should be rejected
	injectionAttempts := []struct {
		input      string
		shouldFail bool
	}{
		{"1' OR '1'='1", true},                          // Contains quotes
		{"1'; DROP TABLE documents; --", true},          // Contains quotes and semicolon
		{"1' UNION SELECT uuid FROM documents--", true}, // Contains quotes
		{"1' AND 1=1--", true},                          // Contains quotes
		{"1' OR uuid IS NOT NULL--", true},              // Contains quotes
		{"' OR ''='", true},                             // Starts with quote
		{"1\\' OR 1=1--", true},                         // Contains backslash and quote
		{"DROP TABLE", true},                            // Not a number
		{"-1", true},                                    // Negative numbers should be rejected
		{"999999", false},                               // Large number won't find anything
	}

	for i, attempt := range injectionAttempts {
		result, err := store.ResolveUUID(attempt.input)

		if attempt.shouldFail {
			// Should fail with format error, not succeed
			if err == nil {
				t.Errorf("injection attempt %d (%s) succeeded with result %s when it should have failed",
					i, attempt.input, result)
			} else if !strings.Contains(err.Error(), "invalid ID format") &&
				!strings.Contains(err.Error(), "document not found") {
				// Make sure it's failing for the right reason
				t.Logf("injection attempt %d (%s) failed with: %v", i, attempt.input, err)
			}
		} else {
			// Should fail with "not found" error
			if err == nil {
				t.Errorf("attempt %d (%s) found a document when none should exist", i, attempt.input)
			} else if !strings.Contains(err.Error(), "not found") {
				t.Errorf("attempt %d (%s) failed with unexpected error: %v", i, attempt.input, err)
			}
		}
	}

	// Verify the store is still functional
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after injection attempts: %v", err)
	}

	if len(docs) != 1 || docs[0].UUID != id {
		t.Error("database state corrupted after injection attempts")
	}
}

func TestSQLInjectionInHierarchicalID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create parent and child
	parent, err := store.Add("Parent", nil, nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	_, err = store.Add("Child", &parent, nil)
	if err != nil {
		t.Fatalf("failed to add child: %v", err)
	}

	// SQL injection attempts in hierarchical IDs
	injectionAttempts := []struct {
		input      string
		shouldFail bool
	}{
		{"1.1' OR '1'='1", true},                                                      // Contains quotes
		{"1.1'; DROP TABLE documents; --", true},                                      // Contains quotes and semicolon
		{"1' UNION SELECT uuid FROM documents WHERE parent_uuid IS NOT NULL--", true}, // Contains quotes
		{"1.1' OR parent_uuid IS NOT NULL--", true},                                   // Contains quotes
		{"1.' OR ''='", true},                                                         // Contains quotes in second part
		{"1.1\\' OR 1=1--", true},                                                     // Contains backslash and quote
		{"1.DROP TABLE", true},                                                        // Invalid format in second part
		{"1.999", false},                                                              // Valid format but won't find anything
		{"999.1", false},                                                              // Parent doesn't exist
	}

	for i, attempt := range injectionAttempts {
		result, err := store.ResolveUUID(attempt.input)

		if attempt.shouldFail {
			// Should fail with format error, not succeed
			if err == nil {
				t.Errorf("hierarchical injection attempt %d (%s) succeeded with result %s when it should have failed",
					i, attempt.input, result)
			} else if !strings.Contains(err.Error(), "invalid ID format") &&
				!strings.Contains(err.Error(), "document not found") {
				// Make sure it's failing for the right reason
				t.Logf("hierarchical injection attempt %d (%s) failed with: %v", i, attempt.input, err)
			}
		} else {
			// Should fail with "not found" error
			if err == nil {
				t.Errorf("attempt %d (%s) found a document when none should exist", i, attempt.input)
			} else if !strings.Contains(err.Error(), "not found") {
				t.Errorf("attempt %d (%s) failed with unexpected error: %v", i, attempt.input, err)
			}
		}
	}

	// Verify data integrity
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after injection attempts: %v", err)
	}

	if len(docs) != 2 {
		t.Error("database corrupted after hierarchical injection attempts")
	}
}

func TestSQLInjectionWithNullBytes(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Null byte injection attempts
	injectionAttempts := []string{
		"Title\x00'; DROP TABLE documents; --",
		"Title\x00' OR 1=1--",
		"\x00'; DELETE FROM documents; --",
		"Before\x00After",
	}

	for i, attempt := range injectionAttempts {
		id, err := store.Add(attempt, nil, nil)
		if err != nil {
			t.Errorf("failed to add with null byte attempt %d: %v", i, err)
			continue
		}

		// Verify the document exists
		_, err = store.ResolveUUID("1")
		if err != nil {
			t.Errorf("failed to resolve ID after null byte attempt %d: %v", i, err)
		}

		// The title might be truncated at null byte, but injection should not succeed
		_ = id
	}

	// Verify database integrity
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after null byte attempts: %v", err)
	}

	if len(docs) == 0 {
		t.Error("database corrupted after null byte injection attempts")
	}
}

func TestSQLInjectionInParentID(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a valid parent
	parent, err := store.Add("Parent", nil, nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	// SQL injection attempts in parent ID parameter
	injectionAttempts := []string{
		"'; DROP TABLE documents; --",
		"' OR '1'='1",
		"' UNION SELECT uuid FROM documents--",
		parent + "' OR uuid IS NOT NULL--",
		parent + "'; DELETE FROM documents; --",
	}

	for i, attempt := range injectionAttempts {
		// These should fail with foreign key constraint or invalid UUID
		_, err := store.Add("Child", &attempt, nil)
		if err == nil {
			t.Errorf("parent ID injection attempt %d succeeded when it should have failed", i)
		}
	}

	// Verify database integrity
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list after parent ID injection: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d - injection may have succeeded", len(docs))
	}
}
