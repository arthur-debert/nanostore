package imports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/types"
)

// ImportFromPath imports documents from a file path (directory or zip)
// This is a convenience function that detects the source type and calls the appropriate reader
func ImportFromPath(store types.Store, path string, options ImportOptions) (*ImportResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var importData *ImportData

	if info.IsDir() {
		// Import from directory
		// Try to detect format from files in directory
		format := detectFormatFromDirectory(path)
		importData, err = ReadImportDataFromDirectory(path, format)
		if err != nil {
			return nil, fmt.Errorf("failed to read from directory: %w", err)
		}
	} else if strings.HasSuffix(path, ".zip") {
		// Import from zip
		importData, err = ReadImportDataFromZip(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read from zip: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", path)
	}

	// Process the import
	return ProcessImportData(store, *importData, options)
}

// detectFormatFromDirectory tries to detect the document format from files in a directory
func detectFormatFromDirectory(dirPath string) *formats.DocumentFormat {
	counts := make(map[string]int)

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != "" && ext != ".json" {
			counts[ext]++
		}

		return nil
	})

	// Use the same detection logic as in reader.go
	return detectFormatFromExtensions(counts)
}

// DefaultImportOptions returns a set of sensible default import options
func DefaultImportOptions() ImportOptions {
	return ImportOptions{
		SkipValidation:       false,
		IgnoreDuplicateUUIDs: false,
		DefaultDimensions:    make(map[string]interface{}),
		DryRun:               false,
	}
}
