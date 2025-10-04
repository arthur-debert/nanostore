package api_test

// IMPORTANT: This test must follow the testing patterns established in:
// nanostore/testutil/model_test.go
//
// Key principles:
// 1. Use testutil.LoadUniverse() for standard test setup
// 2. Leverage fixture data instead of creating test data
// 3. Use assertion helpers for cleaner test code
// 4. Only create fresh stores for specific scenarios (see model_test.go)

import (
	"os"
	"strings"
	"testing"

	_ "github.com/arthur-debert/nanostore/nanostore" // for embedded Document type
	"github.com/arthur-debert/nanostore/nanostore/api"
)

func TestQueryRobustness(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	// Create store
	store, err := api.New[TodoItem](tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test 1: Extreme string inputs in filters
	t.Run("ExtremeStringInputs", func(t *testing.T) {
		extremeStrings := []struct {
			name  string
			value string
		}{
			{"empty string", ""},
			{"single quote", "'"},
			{"double quote", "\""},
			{"sql injection attempt", "'; DROP TABLE documents; --"},
			{"unicode emoji", "🚀💥🎉"},
			{"very long string", strings.Repeat("a", 10000)},
			{"null bytes", "hello\x00world"},
			{"newlines and tabs", "line1\nline2\ttab"},
			{"backslashes", "\\\\escaped\\\\"},
			{"mixed quotes", `"'mixed'"quotes"'`},
		}

		// Create test documents with extreme titles
		for _, test := range extremeStrings {
			_, err := store.Create(test.value, &TodoItem{
				Status: "pending",
			})
			if err != nil {
				// Some strings might be invalid - that's ok
				t.Logf("Failed to create with %s: %v", test.name, err)
			}
		}

		// Try to query with extreme filter values
		for _, test := range extremeStrings {
			results, err := store.Query().
				Status(test.value).
				Find()
			// Should not panic or corrupt data
			if err != nil {
				t.Logf("Query with %s filter failed: %v", test.name, err)
			} else {
				t.Logf("Query with %s filter returned %d results", test.name, len(results))
			}
		}
	})

	// Test 2: Nil and empty values in multi-value filters
	t.Run("NilAndEmptyInMultiValueFilters", func(t *testing.T) {
		// Create some test data
		_, _ = store.Create("Test doc", &TodoItem{Status: "active"})

		testCases := []struct {
			name   string
			values []string
		}{
			{"empty slice", []string{}},
			{"single empty string", []string{""}},
			{"mixed empty and valid", []string{"", "active", ""}},
			{"all empty strings", []string{"", "", ""}},
			{"very many values", make([]string, 1000)}, // 1000 empty strings
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := store.Query().
					StatusIn(tc.values...).
					Find()

				// Should handle gracefully
				if err != nil {
					t.Logf("StatusIn with %s failed: %v", tc.name, err)
				} else {
					t.Logf("StatusIn with %s returned %d results", tc.name, len(results))
				}
			})
		}
	})

	// Test 3: Boundary values for pagination
	t.Run("PaginationBoundaries", func(t *testing.T) {
		// Create 5 test documents
		for i := 0; i < 5; i++ {
			_, _ = store.Create("Doc", &TodoItem{})
		}

		paginationTests := []struct {
			name   string
			limit  *int
			offset *int
			valid  bool
		}{
			{"negative limit", intPtr(-1), nil, false},
			{"negative offset", nil, intPtr(-1), false},
			{"zero limit", intPtr(0), nil, true},
			{"zero offset", nil, intPtr(0), true},
			{"huge limit", intPtr(999999999), nil, true},
			{"huge offset", nil, intPtr(999999999), true},
			{"both negative", intPtr(-5), intPtr(-10), false},
			{"max int values", intPtr(9223372036854775807), intPtr(9223372036854775807), true},
		}

		for _, test := range paginationTests {
			t.Run(test.name, func(t *testing.T) {
				query := store.Query()
				if test.limit != nil {
					query = query.Limit(*test.limit)
				}
				if test.offset != nil {
					query = query.Offset(*test.offset)
				}

				results, err := query.Find()
				if err != nil && test.valid {
					t.Errorf("Expected valid query but got error: %v", err)
				} else if err == nil {
					t.Logf("Query returned %d results", len(results))
				}
			})
		}
	})

	// Test 4: Complex ordering edge cases
	t.Run("ComplexOrderingEdgeCases", func(t *testing.T) {
		// Create documents with edge case values
		edgeCaseDocs := []struct {
			title    string
			priority string
		}{
			{"", "high"},                             // empty title
			{"\x00null", "medium"},                   // null byte in title
			{"Z" + strings.Repeat("z", 1000), "low"}, // very long title
			{"123", "high"},                          // numeric title
			{"!@#$%", "medium"},                      // special characters
		}

		for _, doc := range edgeCaseDocs {
			_, _ = store.Create(doc.title, &TodoItem{Priority: doc.priority})
		}

		// Test multiple orderings
		orderingTests := []struct {
			name    string
			orderBy []string
			desc    []bool
		}{
			{"empty column name", []string{""}, []bool{false}},
			{"non-existent column", []string{"nonexistent"}, []bool{false}},
			{"multiple same column", []string{"title", "title", "title"}, []bool{false, true, false}},
			{"mixed valid/invalid", []string{"title", "", "priority"}, []bool{false, true, true}},
			{"many order clauses", make([]string, 50), make([]bool, 50)}, // 50 empty order clauses
		}

		for _, test := range orderingTests {
			t.Run(test.name, func(t *testing.T) {
				query := store.Query()
				for i, col := range test.orderBy {
					if i < len(test.desc) && test.desc[i] {
						query = query.OrderByDesc(col)
					} else {
						query = query.OrderBy(col)
					}
				}

				results, err := query.Find()
				// Should not panic
				if err != nil {
					t.Logf("Query with %s failed: %v", test.name, err)
				} else {
					t.Logf("Query with %s returned %d results", test.name, len(results))
				}
			})
		}
	})

	// Test 5: Search with malicious patterns
	t.Run("SearchPatternRobustness", func(t *testing.T) {
		// Create some searchable content
		_, _ = store.Create("Normal document", &TodoItem{})
		_, _ = store.Create("Document with special chars !@#$", &TodoItem{})

		searchPatterns := []struct {
			name    string
			pattern string
		}{
			{"empty search", ""},
			{"single wildcard", "*"},
			{"sql wildcard", "%"},
			{"regex pattern", ".*"},
			{"unclosed quote", "search'"},
			{"null byte", "search\x00term"},
			{"very long search", strings.Repeat("search", 1000)},
			{"unicode", "搜索词汇🔍"},
			{"control characters", "\n\r\t\b"},
			{"nested quotes", `"'nested"'quotes'"`},
		}

		for _, test := range searchPatterns {
			t.Run(test.name, func(t *testing.T) {
				results, err := store.Query().
					Search(test.pattern).
					Find()

				// Should handle gracefully without SQL injection
				if err != nil {
					t.Logf("Search with %s failed: %v", test.name, err)
				} else {
					t.Logf("Search with %s returned %d results", test.name, len(results))
					// Verify no data corruption
					allDocs, _ := store.Query().Find()
					if len(allDocs) < 2 {
						t.Error("Data might be corrupted after search")
					}
				}
			})
		}
	})
}

func intPtr(i int) *int {
	return &i
}
