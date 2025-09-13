package nanostore_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
	_ "github.com/mattn/go-sqlite3"
)

func TestTransactionRollback(t *testing.T) {
	t.Skip("Transaction support not yet exposed in public API")

	// This test documents expected transaction behavior
	// Currently, each operation is auto-committed
	// Future enhancement: Add transaction support to Store interface

	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// In a transaction-aware API, this might look like:
	// tx, err := store.BeginTx()
	// defer tx.Rollback()
	//
	// id1, err := tx.Add("Document 1", nil)
	// id2, err := tx.Add("Document 2", nil)
	//
	// // Simulate error condition
	// if someError {
	//     tx.Rollback()
	//     // Both documents should be rolled back
	// } else {
	//     tx.Commit()
	// }
}

func TestDatabaseConsistencyAfterPanic(t *testing.T) {
	// Create file-based database for persistence test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "panic_test.db")

	// Initial setup
	store1, err := nanostore.New(dbPath)
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
	db, err := sql.Open("sqlite3", dbPath)
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
	store2, err := nanostore.New(dbPath)
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

func TestConcurrentTransactionIsolation(t *testing.T) {
	t.Skip("SQLite has limited concurrent write support")

	// This test documents expected behavior for concurrent transactions
	// SQLite uses database-level locking, so concurrent writes will serialize
	// This is acceptable for the nanostore use case

	store, err := nanostore.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// In a real concurrent scenario:
	// - Multiple goroutines attempt writes
	// - SQLite serializes them automatically
	// - No dirty reads or phantom reads occur
	// - Write operations may block waiting for lock
}

func TestRollbackOnConstraintViolation(t *testing.T) {
	store, err := nanostore.New(":memory:")
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
	_, err = store.Add("Orphan Child", &invalidParent)

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

func TestAtomicBatchOperations(t *testing.T) {
	t.Skip("Batch operations not yet implemented")

	// Future enhancement: Add batch operations that execute in a single transaction
	// Example API:
	// batch := store.NewBatch()
	// batch.Add("Doc 1", nil)
	// batch.Add("Doc 2", nil)
	// batch.SetStatus(id, nanostore.StatusCompleted)
	// err := batch.Execute() // All or nothing
}
