package nanostore_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	_ "modernc.org/sqlite"
)

func TestDatabaseConsistencyAfterPanic(t *testing.T) {
	// Create file-based database for persistence test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "panic_test.db")

	// Initial setup
	store1, err := nanostore.NewTestStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add some documents
	id1, err := store1.Add("Before Panic 1", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	id2, err := store1.Add("Before Panic 2", nil)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Close normally
	err = store1.Close()
	if err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Simulate a crash by opening the database directly and leaving a transaction open
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Insert a document in the transaction but don't commit
	_, err = tx.Exec(`
		INSERT INTO documents (uuid, title, body, status, parent_uuid, created_at, updated_at)
		VALUES ('panic-doc', 'Should Not Exist', '', 'pending', NULL, 1234567890, 1234567890)
	`)
	if err != nil {
		t.Fatalf("failed to insert in transaction: %v", err)
	}

	// Close database without committing (simulating crash)
	_ = db.Close()

	// Reopen with nanostore
	store2, err := nanostore.NewTestStore(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer func() { _ = store2.Close() }()

	// Verify only committed documents exist
	docs, err := store2.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents after recovery, got %d", len(docs))
	}

	// Verify the specific documents
	foundID1 := false
	foundID2 := false
	for _, doc := range docs {
		if doc.UUID == id1 {
			foundID1 = true
		}
		if doc.UUID == id2 {
			foundID2 = true
		}
		if doc.UUID == "panic-doc" {
			t.Error("found uncommitted document - transaction not rolled back")
		}
	}

	if !foundID1 || !foundID2 {
		t.Error("original documents not found after recovery")
	}
}

func TestRollbackOnConstraintViolation(t *testing.T) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a parent document
	parentID, err := store.Add("Parent", nil)
	if err != nil {
		t.Fatalf("failed to add parent: %v", err)
	}

	// Try to add a child with invalid parent UUID
	// This should fail and not leave partial data
	invalidParent := "invalid-uuid-that-does-not-exist"
	_, err = store.Add("Orphan Child", map[string]interface{}{"parent_uuid": invalidParent})

	if err == nil {
		t.Error("expected foreign key constraint error")
	}

	// Verify database is still consistent
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document after failed insert, got %d", len(docs))
	}

	if docs[0].UUID != parentID {
		t.Error("parent document corrupted after constraint violation")
	}
}
