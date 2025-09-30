package imports

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore/storage"
)

// ReadImportDataFromDirectory reads import data from a directory of files
func ReadImportDataFromDirectory(dirPath string, format *formats.DocumentFormat) (*ImportData, error) {
	if format == nil {
		format = formats.PlainText
	}

	importData := &ImportData{
		Documents: make([]ImportDocument, 0),
		Metadata: ImportMetadata{
			Version:      "1.0",
			ImportedFrom: "directory",
			ImportedAt:   time.Now(),
		},
	}

	// Check for db.json file
	dbMetadata, err := readDBMetadata(filepath.Join(dirPath, "db.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read db.json: %w", err)
	}

	// Walk through directory and process files
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-matching files
		if info.IsDir() || !strings.HasSuffix(path, format.Extension) {
			return nil
		}

		// Skip db.json
		if filepath.Base(path) == "db.json" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Deserialize using format
		title, body, metadata, err := format.Deserialize(string(content))
		if err != nil {
			return fmt.Errorf("failed to deserialize %s: %w", path, err)
		}

		// Create import document
		doc := ImportDocument{
			Title:      title,
			Body:       body,
			SourceFile: filepath.Base(path),
			Dimensions: make(map[string]interface{}),
		}

		// Apply metadata precedence: file metadata > db.json metadata > defaults
		applyMetadataPrecedence(&doc, metadata, dbMetadata, filepath.Base(path))

		importData.Documents = append(importData.Documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return importData, nil
}

// ReadImportDataFromZip reads import data from a zip file
func ReadImportDataFromZip(zipPath string) (*ImportData, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer func() { _ = r.Close() }()

	importData := &ImportData{
		Documents: make([]ImportDocument, 0),
		Metadata: ImportMetadata{
			Version:      "1.0",
			ImportedFrom: "zip",
			ImportedAt:   time.Now(),
		},
	}

	// First pass: look for db.json and determine format
	var dbMetadata map[string]interface{}
	var detectedFormat *formats.DocumentFormat
	formatCounts := make(map[string]int)

	for _, f := range r.File {
		if f.Name == "db.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open db.json: %w", err)
			}

			content, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read db.json: %w", err)
			}

			var storeData storage.StoreData
			if err := json.Unmarshal(content, &storeData); err != nil {
				return nil, fmt.Errorf("failed to parse db.json: %w", err)
			}

			// Convert to metadata map
			dbMetadata = convertStoreDataToMetadata(&storeData)
			importData.Metadata.Version = storeData.Metadata.Version
			importData.Metadata.ImportedFrom = "export" // This is a nanostore export
		} else {
			// Count file extensions
			ext := filepath.Ext(f.Name)
			if ext != "" {
				formatCounts[ext]++
			}
		}
	}

	// Determine format based on most common extension
	detectedFormat = detectFormatFromExtensions(formatCounts)
	if detectedFormat == nil {
		return nil, fmt.Errorf("could not determine document format from zip contents")
	}

	// Second pass: process document files
	for _, f := range r.File {
		// Skip directories, db.json, and non-matching extensions
		if f.FileInfo().IsDir() || f.Name == "db.json" || !strings.HasSuffix(f.Name, detectedFormat.Extension) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", f.Name, err)
		}

		// Deserialize using detected format
		title, body, metadata, err := detectedFormat.Deserialize(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize %s: %w", f.Name, err)
		}

		// Create import document
		doc := ImportDocument{
			Title:      title,
			Body:       body,
			SourceFile: filepath.Base(f.Name),
			Dimensions: make(map[string]interface{}),
		}

		// Apply metadata precedence
		applyMetadataPrecedence(&doc, metadata, dbMetadata, filepath.Base(f.Name))

		importData.Documents = append(importData.Documents, doc)
	}

	return importData, nil
}

// readDBMetadata reads and parses db.json metadata
func readDBMetadata(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var storeData storage.StoreData
	if err := json.Unmarshal(content, &storeData); err != nil {
		return nil, fmt.Errorf("failed to parse db.json: %w", err)
	}

	return convertStoreDataToMetadata(&storeData), nil
}

// convertStoreDataToMetadata converts StoreData to a metadata map indexed by filename
func convertStoreDataToMetadata(storeData *storage.StoreData) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Create a map of documents by their potential filenames
	for _, doc := range storeData.Documents {
		// Try to match by UUID-based filename or simple ID-based filename
		// This is a simplified approach - in practice, you might need more sophisticated matching
		fileKey := doc.UUID

		docMeta := make(map[string]interface{})
		docMeta["uuid"] = doc.UUID
		docMeta["simple_id"] = doc.SimpleID
		docMeta["created_at"] = doc.CreatedAt
		docMeta["updated_at"] = doc.UpdatedAt

		// Add dimensions
		for k, v := range doc.Dimensions {
			docMeta[k] = v
		}

		metadata[fileKey] = docMeta
	}

	return metadata
}

// applyMetadataPrecedence applies metadata following precedence rules
func applyMetadataPrecedence(doc *ImportDocument, fileMetadata, dbMetadata map[string]interface{}, filename string) {
	// Start with db.json metadata if available
	var baseMeta map[string]interface{}
	// Try to find metadata for this specific file
	// First try by UUID in filename
	for key, meta := range dbMetadata {
		if strings.Contains(filename, key) {
			if m, ok := meta.(map[string]interface{}); ok {
				baseMeta = m
				break
			}
		}
	}

	// Apply db.json metadata
	if baseMeta != nil {
		applyMetadataToDocument(doc, baseMeta)
	}

	// Override with file metadata (higher precedence)
	if fileMetadata != nil {
		applyMetadataToDocument(doc, fileMetadata)
	}
}

// applyMetadataToDocument applies metadata to a document
func applyMetadataToDocument(doc *ImportDocument, metadata map[string]interface{}) {
	for key, value := range metadata {
		switch key {
		case "uuid":
			if s, ok := value.(string); ok && s != "" {
				doc.UUID = &s
			}
		case "simple_id":
			if s, ok := value.(string); ok && s != "" {
				doc.SimpleID = &s
			}
		case "created_at":
			if t, ok := parseTime(value); ok {
				doc.CreatedAt = &t
			}
		case "updated_at":
			if t, ok := parseTime(value); ok {
				doc.UpdatedAt = &t
			}
		default:
			// Everything else goes into dimensions
			doc.Dimensions[key] = value
		}
	}
}

// parseTime attempts to parse various time formats
func parseTime(value interface{}) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		// Try RFC3339
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, true
		}
		// Try other common formats
		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// detectFormatFromExtensions determines the document format based on file extensions
func detectFormatFromExtensions(counts map[string]int) *formats.DocumentFormat {
	maxCount := 0
	var dominantExt string

	for ext, count := range counts {
		if count > maxCount {
			maxCount = count
			dominantExt = ext
		}
	}

	// Map extensions to formats
	switch dominantExt {
	case ".md":
		return formats.Markdown
	case ".txt":
		return formats.PlainText
	default:
		// Default to PlainText if unknown
		return formats.PlainText
	}
}
