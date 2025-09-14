package nanostore_test

import (
	"fmt"
	"testing"

	"github.com/arthur-debert/nanostore/nanostore"
)

func BenchmarkAdd(b *testing.B) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Add(fmt.Sprintf("Document %d", i), nil, nil)
		if err != nil {
			b.Fatalf("failed to add document: %v", err)
		}
	}
}

func BenchmarkList10(b *testing.B) {
	benchmarkList(b, 10)
}

func BenchmarkList100(b *testing.B) {
	benchmarkList(b, 100)
}

func BenchmarkList1000(b *testing.B) {
	benchmarkList(b, 1000)
}

func BenchmarkList10000(b *testing.B) {
	benchmarkList(b, 10000)
}

func benchmarkList(b *testing.B, count int) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add documents
	for i := 0; i < count; i++ {
		_, err := store.Add(fmt.Sprintf("Document %d", i), nil, nil)
		if err != nil {
			b.Fatalf("failed to add document: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			b.Fatalf("failed to list: %v", err)
		}
		if len(docs) != count {
			b.Fatalf("expected %d documents, got %d", count, len(docs))
		}
	}
}

func BenchmarkResolveUUID(b *testing.B) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchy
	root, _ := store.Add("Root", nil, nil)
	parent := root
	for i := 0; i < 5; i++ {
		child, _ := store.Add(fmt.Sprintf("Level %d", i), &parent, nil)
		parent = child
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Resolve a deep ID
		_, err := store.ResolveUUID("1.1.1.1.1")
		if err != nil {
			b.Fatalf("failed to resolve: %v", err)
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		id, err := store.Add(fmt.Sprintf("Document %d", i), nil, nil)
		if err != nil {
			b.Fatalf("failed to add document: %v", err)
		}
		ids[i] = id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		title := fmt.Sprintf("Updated %d", i)
		err := store.Update(ids[i%100], nanostore.UpdateRequest{
			Title: &title,
		})
		if err != nil {
			b.Fatalf("failed to update: %v", err)
		}
	}
}

func BenchmarkHierarchicalList(b *testing.B) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create hierarchical structure
	// 10 roots, each with 10 children, each child with 10 grandchildren = 1110 docs
	for i := 0; i < 10; i++ {
		root, _ := store.Add(fmt.Sprintf("Root %d", i), nil, nil)
		for j := 0; j < 10; j++ {
			child, _ := store.Add(fmt.Sprintf("Child %d.%d", i, j), &root, nil)
			for k := 0; k < 10; k++ {
				_, _ = store.Add(fmt.Sprintf("Grandchild %d.%d.%d", i, j, k), &child, nil)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			b.Fatalf("failed to list: %v", err)
		}
		if len(docs) != 1110 {
			b.Fatalf("expected 1110 documents, got %d", len(docs))
		}
	}
}

func BenchmarkMixedStatusList(b *testing.B) {
	store, err := nanostore.NewTestStore(":memory:")
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create documents with mixed status
	for i := 0; i < 1000; i++ {
		id, err := store.Add(fmt.Sprintf("Document %d", i), nil, nil)
		if err != nil {
			b.Fatalf("failed to add document: %v", err)
		}
		// Mark every third as completed
		if i%3 == 0 {
			_ = nanostore.SetStatus(store, id, "completed")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		docs, err := store.List(nanostore.ListOptions{})
		if err != nil {
			b.Fatalf("failed to list: %v", err)
		}
		if len(docs) != 1000 {
			b.Fatalf("expected 1000 documents, got %d", len(docs))
		}
	}
}
