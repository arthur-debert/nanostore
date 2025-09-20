package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BodyStorageType represents how a document's body is stored
type BodyStorageType string

const (
	// BodyStorageEmbedded means the body is stored directly in the JSON file
	BodyStorageEmbedded BodyStorageType = "embedded"

	// BodyStorageFile means the body is stored in a separate file
	BodyStorageFile BodyStorageType = "file"
)

// BodyFormat represents the format of a document body
type BodyFormat string

const (
	// BodyFormatText is plain text format
	BodyFormatText BodyFormat = "txt"

	// BodyFormatMarkdown is Markdown format
	BodyFormatMarkdown BodyFormat = "md"

	// BodyFormatHTML is HTML format
	BodyFormatHTML BodyFormat = "html"
)

// ParseBodyFormat parses a file extension into a BodyFormat
func ParseBodyFormat(ext string) (BodyFormat, error) {
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "txt":
		return BodyFormatText, nil
	case "md", "markdown":
		return BodyFormatMarkdown, nil
	case "html", "htm":
		return BodyFormatHTML, nil
	default:
		return "", fmt.Errorf("unsupported body format: %s", ext)
	}
}

// BodyMetadata contains information about how a document's body is stored
type BodyMetadata struct {
	// Type indicates whether body is embedded or in a file
	Type BodyStorageType `json:"type"`

	// Format of the body content (txt, md, html)
	Format BodyFormat `json:"format"`

	// Filename is only set when Type is BodyStorageFile
	// It's relative to the bodies directory
	Filename string `json:"filename,omitempty"`

	// Size in bytes (for monitoring/limits)
	Size int64 `json:"size"`
}

// BodyStorage handles reading and writing document bodies
type BodyStorage interface {
	// ReadBody reads a document's body content
	ReadBody(meta BodyMetadata, embeddedContent string) (string, error)

	// WriteBody writes a document's body and returns updated metadata
	// The forceEmbed parameter forces embedded storage regardless of size
	WriteBody(uuid string, content string, format BodyFormat, forceEmbed bool) (BodyMetadata, string, error)

	// DeleteBody removes body content if it's stored in a file
	DeleteBody(meta BodyMetadata) error

	// ValidateBody checks if a body's storage is valid (files exist, etc.)
	ValidateBody(meta BodyMetadata) error

	// ListOrphanedFiles returns body files that aren't referenced by any document
	ListOrphanedFiles(documentMetas []BodyMetadata) ([]string, error)

	// MigrateBody changes storage type for a body
	// The uuid parameter is needed when migrating from embedded to file storage
	MigrateBody(meta BodyMetadata, embeddedContent string, toType BodyStorageType, uuid string) (BodyMetadata, string, error)
}

// HybridBodyStorage implements BodyStorage with support for both embedded and file storage
type HybridBodyStorage struct {
	fs             FileSystemExt
	basePath       string // Base directory for the store
	bodiesDir      string // Subdirectory for body files (e.g., "bodies")
	embedSizeLimit int64  // Maximum size for embedded bodies (default 1KB)
}

// NewHybridBodyStorage creates a new hybrid body storage handler
func NewHybridBodyStorage(fs FileSystemExt, basePath string, embedSizeLimit int64) *HybridBodyStorage {
	if embedSizeLimit <= 0 {
		embedSizeLimit = 1024 // 1KB default
	}

	return &HybridBodyStorage{
		fs:             fs,
		basePath:       basePath,
		bodiesDir:      "bodies",
		embedSizeLimit: embedSizeLimit,
	}
}

// bodiesPath returns the full path to the bodies directory
func (h *HybridBodyStorage) bodiesPath() string {
	return filepath.Join(h.basePath, h.bodiesDir)
}

// bodyFilePath returns the full path for a body file
func (h *HybridBodyStorage) bodyFilePath(filename string) string {
	return filepath.Join(h.bodiesPath(), filename)
}

// ensureBodiesDir creates the bodies directory if it doesn't exist
func (h *HybridBodyStorage) ensureBodiesDir() error {
	bodiesPath := h.bodiesPath()

	// Check if directory exists
	if _, err := h.fs.Stat(bodiesPath); err == nil {
		return nil // Directory already exists
	}

	// Create directory
	return h.fs.MkdirAll(bodiesPath, 0755)
}

// ReadBody implements BodyStorage.ReadBody
func (h *HybridBodyStorage) ReadBody(meta BodyMetadata, embeddedContent string) (string, error) {
	switch meta.Type {
	case BodyStorageEmbedded:
		return embeddedContent, nil

	case BodyStorageFile:
		if meta.Filename == "" {
			return "", fmt.Errorf("body file name not specified")
		}

		content, err := h.fs.ReadFile(h.bodyFilePath(meta.Filename))
		if err != nil {
			return "", fmt.Errorf("failed to read body file %s: %w", meta.Filename, err)
		}

		return string(content), nil

	default:
		return "", fmt.Errorf("unknown body storage type: %s", meta.Type)
	}
}

// WriteBody implements BodyStorage.WriteBody
func (h *HybridBodyStorage) WriteBody(uuid string, content string, format BodyFormat, forceEmbed bool) (BodyMetadata, string, error) {
	size := int64(len(content))

	// Determine storage type based on size and forceEmbed flag
	if forceEmbed {
		// Force embedded storage regardless of size
		return BodyMetadata{
			Type:   BodyStorageEmbedded,
			Format: format,
			Size:   size,
		}, content, nil
	}

	// If not forcing embed, decide based on size
	if size <= h.embedSizeLimit {
		// Small enough to embed
		return BodyMetadata{
			Type:   BodyStorageEmbedded,
			Format: format,
			Size:   size,
		}, content, nil
	}

	// Store in file
	if err := h.ensureBodiesDir(); err != nil {
		return BodyMetadata{}, "", fmt.Errorf("failed to create bodies directory: %w", err)
	}

	// Generate filename based on UUID and format
	filename := fmt.Sprintf("%s.%s", uuid, format)
	fullPath := h.bodyFilePath(filename)

	// Write file
	if err := h.fs.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return BodyMetadata{}, "", fmt.Errorf("failed to write body file: %w", err)
	}

	return BodyMetadata{
		Type:     BodyStorageFile,
		Format:   format,
		Filename: filename,
		Size:     size,
	}, "", nil // Empty embedded content when stored in file
}

// DeleteBody implements BodyStorage.DeleteBody
func (h *HybridBodyStorage) DeleteBody(meta BodyMetadata) error {
	if meta.Type != BodyStorageFile || meta.Filename == "" {
		return nil // Nothing to delete for embedded bodies
	}

	err := h.fs.Remove(h.bodyFilePath(meta.Filename))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete body file %s: %w", meta.Filename, err)
	}

	return nil
}

// ValidateBody implements BodyStorage.ValidateBody
func (h *HybridBodyStorage) ValidateBody(meta BodyMetadata) error {
	if meta.Type != BodyStorageFile {
		return nil // Embedded bodies are always valid
	}

	if meta.Filename == "" {
		return fmt.Errorf("body file name not specified")
	}

	_, err := h.fs.Stat(h.bodyFilePath(meta.Filename))
	if err != nil {
		return fmt.Errorf("body file %s not found: %w", meta.Filename, err)
	}

	return nil
}

// ListOrphanedFiles implements BodyStorage.ListOrphanedFiles
func (h *HybridBodyStorage) ListOrphanedFiles(documentMetas []BodyMetadata) ([]string, error) {
	// Build set of referenced files
	referenced := make(map[string]bool)
	for _, meta := range documentMetas {
		if meta.Type == BodyStorageFile && meta.Filename != "" {
			referenced[meta.Filename] = true
		}
	}

	// List all files in bodies directory
	bodiesPath := h.bodiesPath()

	// Try to read directory directly - if it doesn't exist, ReadDir will return an appropriate error
	entries, err := h.fs.ReadDir(bodiesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No bodies directory, so no orphaned files
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read bodies directory: %w", err)
	}

	orphaned := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		filename := entry.Name()
		// Skip hidden files and our marker files
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// Check if this file is referenced
		if !referenced[filename] {
			orphaned = append(orphaned, filename)
		}
	}

	return orphaned, nil
}

// MigrateBody implements BodyStorage.MigrateBody
func (h *HybridBodyStorage) MigrateBody(meta BodyMetadata, embeddedContent string, toType BodyStorageType, uuid string) (BodyMetadata, string, error) {
	// First read the current content
	content, err := h.ReadBody(meta, embeddedContent)
	if err != nil {
		return meta, embeddedContent, err
	}

	// If migrating to the same type, nothing to do
	if meta.Type == toType {
		return meta, embeddedContent, nil
	}

	// Use UUID from filename if migrating from file and no UUID provided
	if uuid == "" && meta.Type == BodyStorageFile && meta.Filename != "" {
		uuid = strings.TrimSuffix(meta.Filename, filepath.Ext(meta.Filename))
	}

	// Delete old storage if it was a file
	if meta.Type == BodyStorageFile {
		if err := h.DeleteBody(meta); err != nil {
			return meta, embeddedContent, err
		}
	}

	// Write with new storage type
	var newMeta BodyMetadata
	var newEmbedded string

	if toType == BodyStorageEmbedded {
		// Force embedded storage
		newMeta, newEmbedded, err = h.WriteBody(uuid, content, meta.Format, true)
	} else {
		// Force file storage - write directly to file regardless of size
		newMeta, newEmbedded, err = h.writeBodyToFile(uuid, content, meta.Format)
	}

	if err != nil {
		return meta, embeddedContent, err
	}

	return newMeta, newEmbedded, nil
}

// writeBodyToFile forces content to be written to a file regardless of size
func (h *HybridBodyStorage) writeBodyToFile(uuid string, content string, format BodyFormat) (BodyMetadata, string, error) {
	size := int64(len(content))

	// Ensure bodies directory exists
	if err := h.ensureBodiesDir(); err != nil {
		return BodyMetadata{}, "", fmt.Errorf("failed to create bodies directory: %w", err)
	}

	// Generate filename based on UUID and format
	filename := fmt.Sprintf("%s.%s", uuid, format)
	fullPath := h.bodyFilePath(filename)

	// Write file
	if err := h.fs.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return BodyMetadata{}, "", fmt.Errorf("failed to write body file: %w", err)
	}

	return BodyMetadata{
		Type:     BodyStorageFile,
		Format:   format,
		Filename: filename,
		Size:     size,
	}, "", nil // Empty embedded content when stored in file
}
