package nanostore

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestResolveUUIDPerformance demonstrates the performance improvement of the optimized ResolveUUID
func TestResolveUUIDPerformance(t *testing.T) {
	config := Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "done"},
				Prefixes:     map[string]string{"done": "d"},
				DefaultValue: "pending",
			},
		},
	}

	store, err := New(":memory:", config)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a substantial number of documents for performance testing
	numDocs := 1000

	start := time.Now()
	for i := 0; i < numDocs; i++ {
		title := fmt.Sprintf("Document %d", i+1)
		dimensions := map[string]string{"status": "pending"}
		if i%10 == 0 { // Every 10th document is "done"
			dimensions["status"] = "done"
		}

		_, err := store.Add(title, nil, dimensions)
		if err != nil {
			t.Fatalf("Failed to add document %d: %v", i+1, err)
		}
	}
	addDuration := time.Since(start)
	t.Logf("Added %d documents in %v (%.2f docs/sec)", numDocs, addDuration, float64(numDocs)/addDuration.Seconds())

	// Test ResolveUUID performance with various IDs
	testCases := []struct {
		id          string
		description string
	}{
		{"1", "first pending document"},
		{"500", "middle pending document"},
		{"900", "late pending document"},
		{"d1", "first done document"},
		{"d50", "middle done document"},
		{"d100", "last done document"},
	}

	// Warm up the query engine
	for _, tc := range testCases {
		_, err := store.ResolveUUID(tc.id)
		if err != nil && tc.id != "d100" { // d100 might not exist if we have fewer than 100 done documents
			t.Logf("Warmup resolution of %s (%s) failed: %v", tc.id, tc.description, err)
		}
	}

	// Performance test: resolve multiple IDs
	iterations := 100
	start = time.Now()
	successfulResolves := 0

	for i := 0; i < iterations; i++ {
		for _, tc := range testCases {
			_, err := store.ResolveUUID(tc.id)
			if err == nil {
				successfulResolves++
			}
		}
	}

	duration := time.Since(start)
	avgPerResolve := duration / time.Duration(iterations*len(testCases))

	t.Logf("Resolved %d IDs in %v", iterations*len(testCases), duration)
	t.Logf("Average time per ResolveUUID: %v", avgPerResolve)
	t.Logf("Successful resolves: %d/%d", successfulResolves, iterations*len(testCases))

	// Performance should be well under 1ms per resolve for the optimized version
	if avgPerResolve > time.Millisecond {
		t.Logf("WARNING: ResolveUUID averaging %v per call, may indicate performance issue", avgPerResolve)
	}

	// Test that we can resolve actual document IDs
	docs, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(docs) < 10 {
		t.Skip("Need at least 10 documents for meaningful test")
	}

	// Test resolving first 10 actual document IDs
	start = time.Now()
	for i := 0; i < 10; i++ {
		expectedUUID := docs[i].UUID
		actualUUID, err := store.ResolveUUID(docs[i].UserFacingID)
		if err != nil {
			t.Fatalf("Failed to resolve actual document ID %s: %v", docs[i].UserFacingID, err)
		}
		if actualUUID != expectedUUID {
			t.Fatalf("UUID mismatch for ID %s: expected %s, got %s", docs[i].UserFacingID, expectedUUID, actualUUID)
		}
	}
	actualResolveDuration := time.Since(start)
	t.Logf("Resolved 10 actual document IDs in %v (%.2f μs per resolve)",
		actualResolveDuration, float64(actualResolveDuration.Nanoseconds())/float64(10*1000))
}

// BenchmarkResolveUUID measures ResolveUUID performance
func BenchmarkResolveUUID(b *testing.B) {
	config := Config{
		Dimensions: []DimensionConfig{
			{
				Name:         "status",
				Type:         Enumerated,
				Values:       []string{"pending", "done", "blocked"},
				Prefixes:     map[string]string{"done": "d", "blocked": "b"},
				DefaultValue: "pending",
			},
		},
	}

	// Use temporary file for more realistic performance
	tmpFile := "/tmp/nanostore_bench.db"
	defer func() { _ = os.Remove(tmpFile) }()

	store, err := New(tmpFile, config)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test documents
	numDocs := 100
	for i := 0; i < numDocs; i++ {
		status := "pending"
		if i%5 == 0 {
			status = "done"
		} else if i%7 == 0 {
			status = "blocked"
		}

		_, err := store.Add(fmt.Sprintf("Doc %d", i+1), nil, map[string]string{"status": status})
		if err != nil {
			b.Fatalf("Failed to add document: %v", err)
		}
	}

	// Test cases for benchmarking
	testIDs := []string{"1", "10", "50", "d1", "d5", "b1", "b3"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, id := range testIDs {
			_, err := store.ResolveUUID(id)
			if err != nil && err.Error() != "document not found" {
				b.Fatalf("Unexpected error resolving %s: %v", id, err)
			}
		}
	}
}
