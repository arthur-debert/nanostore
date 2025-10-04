package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)
//
// This stress test suite focuses on testing system behavior under high load
// and with many operations to ensure robustness and performance.

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestHighVolumeOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Workaround for import usage detection
	_ = os.TempDir()
	var _ *api.Store[TodoItem]

	store, cleanup := createFreshStore(t)
	defer cleanup()

	t.Run("MassDocumentCreation", func(t *testing.T) {
		const numDocs = 1000
		var createdUUIDs []string

		start := time.Now()
		for i := 0; i < numDocs; i++ {
			title := fmt.Sprintf("Mass Document %04d", i)
			status := []string{"pending", "active", "done"}[i%3]
			priority := []string{"low", "medium", "high"}[i%3]
			activity := []string{"active", "archived", "deleted"}[i%3]

			uuid, err := store.Create(title, &TodoItem{
				Status:   status,
				Priority: priority,
				Activity: activity,
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}
			createdUUIDs = append(createdUUIDs, uuid)

			// Log progress periodically
			if (i+1)%100 == 0 {
				t.Logf("Created %d documents", i+1)
			}
		}
		elapsed := time.Since(start)

		t.Logf("Created %d documents in %v (%.2f docs/sec)",
			numDocs, elapsed, float64(numDocs)/elapsed.Seconds())

		// Verify all documents exist
		allDocs, err := store.Query().Find()
		if err != nil {
			t.Fatalf("failed to query all documents: %v", err)
		}

		if len(allDocs) != numDocs {
			t.Errorf("expected %d documents, found %d", numDocs, len(allDocs))
		}

		// Test mass retrieval
		start = time.Now()
		for i, uuid := range createdUUIDs {
			_, err := store.Get(uuid)
			if err != nil {
				t.Errorf("failed to retrieve document %d (UUID: %s): %v", i, uuid, err)
			}

			if (i+1)%100 == 0 {
				t.Logf("Retrieved %d documents", i+1)
			}
		}
		retrievalElapsed := time.Since(start)

		t.Logf("Retrieved %d documents in %v (%.2f retrievals/sec)",
			numDocs, retrievalElapsed, float64(numDocs)/retrievalElapsed.Seconds())
	})
}

func TestComplexQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	store, cleanup := createFreshStore(t)
	defer cleanup()

	// Create a substantial dataset for performance testing
	const numDocs = 500
	for i := 0; i < numDocs; i++ {
		title := fmt.Sprintf("Performance Test Document %04d", i)
		status := []string{"pending", "active", "done"}[i%3]
		priority := []string{"low", "medium", "high"}[rand.Intn(3)]
		activity := []string{"active", "archived", "deleted"}[rand.Intn(3)]

		_, err := store.AddRaw(title, map[string]interface{}{
			"status":           status,
			"priority":         priority,
			"activity":         activity,
			"_data.assignee":   fmt.Sprintf("user_%d", rand.Intn(10)),
			"_data.estimate":   rand.Intn(20) + 1,
			"_data.complexity": rand.Intn(5) + 1,
			"_data.Team":       fmt.Sprintf("dept_%d", rand.Intn(5)),
		})
		if err != nil {
			t.Fatalf("failed to create document %d: %v", i, err)
		}
	}

	t.Run("SimpleQueries", func(t *testing.T) {
		queries := []struct {
			name  string
			query func() ([]TodoItem, error)
		}{
			{
				name: "Status filter",
				query: func() ([]TodoItem, error) {
					return store.Query().Status("active").Find()
				},
			},
			{
				name: "Priority filter",
				query: func() ([]TodoItem, error) {
					return store.Query().Priority("high").Find()
				},
			},
			{
				name: "Activity filter",
				query: func() ([]TodoItem, error) {
					return store.Query().Activity("active").Find()
				},
			},
		}

		for _, q := range queries {
			t.Run(q.name, func(t *testing.T) {
				start := time.Now()
				results, err := q.query()
				elapsed := time.Since(start)

				if err != nil {
					t.Fatalf("query failed: %v", err)
				}

				t.Logf("%s: found %d results in %v", q.name, len(results), elapsed)
			})
		}
	})

	t.Run("ComplexQueries", func(t *testing.T) {
		complexQueries := []struct {
			name  string
			query func() ([]TodoItem, error)
		}{
			{
				name: "Multi-dimension filter",
				query: func() ([]TodoItem, error) {
					return store.Query().
						Status("active").
						Priority("high").
						Activity("active").
						Find()
				},
			},
			{
				name: "OR operations",
				query: func() ([]TodoItem, error) {
					return store.Query().
						StatusIn("active", "pending").
						PriorityIn("high", "medium").
						Find()
				},
			},
			{
				name: "NOT operations",
				query: func() ([]TodoItem, error) {
					return store.Query().
						StatusNot("done").
						PriorityNot("low").
						ActivityNot("deleted").
						Find()
				},
			},
			{
				name: "Data field filters",
				query: func() ([]TodoItem, error) {
					return store.Query().
						Data("assignee", "user_1").
						DataIn("Team", "dept_0", "dept_1").
						Find()
				},
			},
			{
				name: "Mixed complex query",
				query: func() ([]TodoItem, error) {
					return store.Query().
						StatusIn("active", "pending").
						PriorityNot("low").
						Activity("active").
						Data("complexity", 5).
						OrderBy("created_at").
						Limit(50).
						Find()
				},
			},
		}

		for _, q := range complexQueries {
			t.Run(q.name, func(t *testing.T) {
				start := time.Now()
				results, err := q.query()
				elapsed := time.Since(start)

				if err != nil {
					t.Fatalf("complex query failed: %v", err)
				}

				t.Logf("%s: found %d results in %v", q.name, len(results), elapsed)
			})
		}
	})

	t.Run("PaginationPerformance", func(t *testing.T) {
		pageSize := 25
		start := time.Now()

		var totalResults []TodoItem
		for offset := 0; offset < numDocs; offset += pageSize {
			page, err := store.Query().
				OrderBy("title").
				Limit(pageSize).
				Offset(offset).
				Find()
			if err != nil {
				t.Fatalf("pagination failed at offset %d: %v", offset, err)
			}

			totalResults = append(totalResults, page...)

			if len(page) < pageSize {
				break // Last page
			}
		}

		elapsed := time.Since(start)
		t.Logf("Paginated through %d results in %v", len(totalResults), elapsed)

		if len(totalResults) != numDocs {
			t.Errorf("pagination didn't return all results: got %d, expected %d",
				len(totalResults), numDocs)
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	store, cleanup := createFreshStore(t)
	defer cleanup()

	t.Run("ConcurrentReads", func(t *testing.T) {
		// First create some documents
		const numDocs = 100
		var docUUIDs []string

		for i := 0; i < numDocs; i++ {
			uuid, err := store.Create(fmt.Sprintf("Concurrent Test Doc %d", i), &TodoItem{
				Status:   "active",
				Priority: "medium",
				Activity: "active",
			})
			if err != nil {
				t.Fatalf("failed to create document %d: %v", i, err)
			}
			docUUIDs = append(docUUIDs, uuid)
		}

		// Now perform concurrent reads
		const numGoroutines = 10
		const readsPerGoroutine = 50

		var wg sync.WaitGroup
		var mu sync.Mutex
		var totalReads int
		var errors []error

		start := time.Now()

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for r := 0; r < readsPerGoroutine; r++ {
					// Random read operation
					docIndex := rand.Intn(len(docUUIDs))
					_, err := store.Get(docUUIDs[docIndex])

					mu.Lock()
					totalReads++
					if err != nil {
						errors = append(errors, err)
					}
					mu.Unlock()
				}
			}(g)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Performed %d concurrent reads in %v with %d goroutines (%.2f reads/sec)",
			totalReads, elapsed, numGoroutines, float64(totalReads)/elapsed.Seconds())

		if len(errors) > 0 {
			t.Errorf("encountered %d errors during concurrent reads", len(errors))
			for i, err := range errors {
				if i < 5 { // Log first 5 errors
					t.Logf("Error %d: %v", i+1, err)
				}
			}
		}
	})

	t.Run("ConcurrentQueries", func(t *testing.T) {
		const numGoroutines = 8
		const queriesPerGoroutine = 20

		var wg sync.WaitGroup
		var mu sync.Mutex
		var totalQueries int
		var totalResults int
		var errors []error

		queries := []func() ([]TodoItem, error){
			func() ([]TodoItem, error) { return store.Query().Status("active").Find() },
			func() ([]TodoItem, error) { return store.Query().Priority("medium").Find() },
			func() ([]TodoItem, error) { return store.Query().Activity("active").Find() },
			func() ([]TodoItem, error) { return store.Query().StatusIn("active", "pending").Find() },
			func() ([]TodoItem, error) { return store.Query().PriorityNot("low").Find() },
			func() ([]TodoItem, error) { return store.Query().OrderBy("created_at").Limit(10).Find() },
		}

		start := time.Now()

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for q := 0; q < queriesPerGoroutine; q++ {
					// Execute random query
					queryFunc := queries[rand.Intn(len(queries))]
					results, err := queryFunc()

					mu.Lock()
					totalQueries++
					if err != nil {
						errors = append(errors, err)
					} else {
						totalResults += len(results)
					}
					mu.Unlock()
				}
			}(g)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Executed %d concurrent queries in %v with %d goroutines (%.2f queries/sec)",
			totalQueries, elapsed, numGoroutines, float64(totalQueries)/elapsed.Seconds())
		t.Logf("Total results returned: %d", totalResults)

		if len(errors) > 0 {
			t.Errorf("encountered %d errors during concurrent queries", len(errors))
			for i, err := range errors {
				if i < 5 { // Log first 5 errors
					t.Logf("Error %d: %v", i+1, err)
				}
			}
		}
	})
}

func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	store, cleanup := createFreshStore(t)
	defer cleanup()

	t.Run("LargeDocumentCreation", func(t *testing.T) {
		const numDocs = 200
		largeTitle := strings.Repeat("A", 1000) // 1KB title

		start := time.Now()
		for i := 0; i < numDocs; i++ {
			title := fmt.Sprintf("%s_%04d", largeTitle, i)

			// Create documents with large custom data
			_, err := store.AddRaw(title, map[string]interface{}{
				"status":            "active",
				"priority":          "medium",
				"activity":          "active",
				"_data.large_field": strings.Repeat("X", 5000), // 5KB field
				"_data.description": strings.Repeat("Description ", 100),
				"_data.metadata":    fmt.Sprintf("metadata_%d", i),
				"_data.tags":        fmt.Sprintf("tag1,tag2,tag3,tag4,tag5_%d", i),
			})
			if err != nil {
				t.Fatalf("failed to create large document %d: %v", i, err)
			}

			if (i+1)%50 == 0 {
				t.Logf("Created %d large documents", i+1)
			}
		}
		elapsed := time.Since(start)

		t.Logf("Created %d large documents in %v", numDocs, elapsed)

		// Test querying large documents
		queryStart := time.Now()
		results, err := store.Query().Status("active").Find()
		queryElapsed := time.Since(queryStart)

		if err != nil {
			t.Fatalf("failed to query large documents: %v", err)
		}

		t.Logf("Queried %d large documents in %v", len(results), queryElapsed)
	})

	t.Run("RepeatedQueryExecution", func(t *testing.T) {
		const numIterations = 1000

		start := time.Now()
		for i := 0; i < numIterations; i++ {
			_, err := store.Query().
				StatusIn("active", "pending").
				PriorityNot("low").
				Activity("active").
				OrderBy("created_at").
				Limit(20).
				Find()

			if err != nil {
				t.Fatalf("query iteration %d failed: %v", i, err)
			}

			if (i+1)%100 == 0 {
				t.Logf("Completed %d query iterations", i+1)
			}
		}
		elapsed := time.Since(start)

		t.Logf("Executed %d repeated queries in %v (%.2f queries/sec)",
			numIterations, elapsed, float64(numIterations)/elapsed.Seconds())
	})
}
