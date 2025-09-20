package storage

import "github.com/arthur-debert/nanostore/nanostore/storage/internal"

// NewJSONStorage creates a new JSON file-based storage implementation
func NewJSONStorage(filePath string) Storage {
	return internal.NewJSONStorage(filePath)
}
