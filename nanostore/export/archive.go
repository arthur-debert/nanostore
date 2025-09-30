package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateExportArchive takes export data and creates a zip file
// This function creates the actual zip file from the export data structure
func CreateExportArchive(exportData *ExportData, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil && outputPath != "" {
			// Log error but don't override the primary error
			fmt.Fprintf(os.Stderr, "Warning: failed to close archive file: %v\n", err)
		}
	}()

	// Create a new zip writer
	zipWriter := zip.NewWriter(file)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close zip writer: %v\n", err)
		}
	}()

	// Add the database file
	if err := addDatabaseToZip(zipWriter, exportData.Contents.DB); err != nil {
		return fmt.Errorf("failed to add database to zip: %w", err)
	}

	// Add each object file
	for _, objectFile := range exportData.Contents.Objects {
		if err := addObjectToZip(zipWriter, objectFile); err != nil {
			return fmt.Errorf("failed to add object %s to zip: %w", objectFile.Filename, err)
		}
	}

	return nil
}

// CreateExportArchiveToTempDir creates an export archive in a temporary directory
// Returns the path to the created archive
func CreateExportArchiveToTempDir(exportData *ExportData) (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "nanostore-export-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create the archive in the temp directory
	archivePath := filepath.Join(tempDir, exportData.ArchiveFilename)
	if err := CreateExportArchive(exportData, archivePath); err != nil {
		// Clean up on error
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up temp directory: %v\n", err)
		}
		return "", err
	}

	return archivePath, nil
}

// addDatabaseToZip adds the database JSON file to the zip archive
func addDatabaseToZip(zipWriter *zip.Writer, dbFile DatabaseFile) error {
	// Create the file header
	header := &zip.FileHeader{
		Name:     dbFile.Filename,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}

	// Create the file in the zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create database file in zip: %w", err)
	}

	// Convert the database contents to JSON
	jsonData, err := json.MarshalIndent(dbFile.Contents, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database contents: %w", err)
	}

	// Write the JSON data
	_, err = writer.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write database contents: %w", err)
	}

	return nil
}

// addObjectToZip adds an object file to the zip archive with proper timestamps
func addObjectToZip(zipWriter *zip.Writer, objectFile ObjectFile) error {
	// Create the file header with modification time
	header := &zip.FileHeader{
		Name:     objectFile.Filename,
		Method:   zip.Deflate,
		Modified: objectFile.Modified,
	}

	// Create the file in the zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create object file in zip: %w", err)
	}

	// Write the content
	_, err = writer.Write([]byte(objectFile.Content))
	if err != nil {
		return fmt.Errorf("failed to write object content: %w", err)
	}

	return nil
}

// ExtractExportArchive extracts a zip archive and returns the export data
// This is useful for testing and for implementing import functionality later
func ExtractExportArchive(archivePath string) (*ExportData, error) {
	// Open the zip file
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close zip reader: %v\n", err)
		}
	}()

	exportData := &ExportData{
		ArchiveFilename: filepath.Base(archivePath),
		Contents: ExportContent{
			Objects: make([]ObjectFile, 0),
		},
	}

	// Process each file in the archive
	for _, file := range reader.File {
		if file.Name == "db.json" {
			// Handle database file
			if err := extractDatabaseFromZip(file, &exportData.Contents.DB); err != nil {
				return nil, fmt.Errorf("failed to extract database: %w", err)
			}
		} else {
			// Handle object files
			objectFile, err := extractObjectFromZip(file)
			if err != nil {
				return nil, fmt.Errorf("failed to extract object %s: %w", file.Name, err)
			}
			exportData.Contents.Objects = append(exportData.Contents.Objects, *objectFile)
		}
	}

	return exportData, nil
}

// extractDatabaseFromZip extracts the database file from the zip
func extractDatabaseFromZip(file *zip.File, dbFile *DatabaseFile) error {
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open database file: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close zip reader: %v\n", err)
		}
	}()

	// Read the JSON content
	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read database content: %w", err)
	}

	// Parse the JSON
	var dbContent interface{}
	if err := json.Unmarshal(content, &dbContent); err != nil {
		return fmt.Errorf("failed to parse database JSON: %w", err)
	}

	dbFile.Filename = file.Name
	dbFile.Contents = dbContent

	return nil
}

// extractObjectFromZip extracts an object file from the zip
func extractObjectFromZip(file *zip.File) (*ObjectFile, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open object file: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close zip reader: %v\n", err)
		}
	}()

	// Read the content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object content: %w", err)
	}

	objectFile := &ObjectFile{
		Filename: file.Name,
		Modified: file.ModTime(),
		Created:  file.ModTime(), // We can't distinguish created vs modified from zip
		Content:  string(content),
	}

	return objectFile, nil
}
