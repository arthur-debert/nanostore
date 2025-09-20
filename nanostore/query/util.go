package query

import (
	"fmt"
	"time"
)

// valueToString converts any value to a string for comparison
// Special handling for time.Time values to use RFC3339Nano format
func valueToString(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		// Use RFC3339Nano for consistent datetime comparison with nanosecond precision
		return v.Format(time.RFC3339Nano)
	case string:
		// Check if it's a datetime string and normalize it
		// Try various datetime formats
		for _, format := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(format, v); err == nil {
				return t.Format(time.RFC3339Nano)
			}
		}
		// Not a datetime, return as-is
		return v
	default:
		return fmt.Sprintf("%v", value)
	}
}
