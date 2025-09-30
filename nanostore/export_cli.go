package nanostore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CreateExportCommand creates a cobra command for exporting nanostore data
// This function allows applications using nanostore to easily add export functionality
// to their CLI by calling this function and adding the returned command to their root command.
//
// Usage example:
//
//	rootCmd := &cobra.Command{Use: "myapp"}
//	exportCmd := nanostore.CreateExportCommand("/path/to/store.json", config)
//	rootCmd.AddCommand(exportCmd)
//
// The command accepts a list of document IDs to export. If no IDs are provided,
// all documents will be exported.
func CreateExportCommand(storePath string, config Config) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "export [id1 id2 ...]",
		Short: "Export documents from the nanostore",
		Long: `Export documents from the nanostore to a zip archive.

The export command creates a zip file containing:
- db.json: Complete database representation
- Individual files for each document's content

Examples:
  # Export all documents
  myapp export

  # Export specific documents by ID
  myapp export 1 c2 1.3

  # Export to a specific path
  myapp export --output /path/to/export.zip

The exported archive contains the database and individual text files for each document,
with filenames in the format: <uuid>-<order>-<title>.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportCommand(storePath, config, args, outputPath)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for the export archive (default: creates in temp directory)")

	return cmd
}

// runExportCommand executes the export operation
func runExportCommand(storePath string, config Config, ids []string, outputPath string) error {
	// Open the store
	store, err := New(storePath, config)
	if err != nil {
		return fmt.Errorf("failed to open store at %s: %w", storePath, err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close store: %v\n", err)
		}
	}()

	// Prepare export options
	options := ExportOptions{}
	if len(ids) > 0 {
		options.IDs = ids
		fmt.Printf("Exporting %d specific documents: %s\n", len(ids), strings.Join(ids, ", "))
	} else {
		fmt.Println("Exporting all documents...")
	}

	// Get metadata to show what will be exported
	metadata, err := GetExportMetadata(store, options)
	if err != nil {
		return fmt.Errorf("failed to get export metadata: %w", err)
	}

	if metadata.DocumentCount == 0 {
		fmt.Println("No documents found to export.")
		return nil
	}

	fmt.Printf("Found %d documents to export (estimated size: %d bytes)\n",
		metadata.DocumentCount, metadata.EstimatedSizeBytes)

	// Perform the export
	var archivePath string
	if outputPath != "" {
		// Export to specified path
		err = ExportToPath(store, options, outputPath)
		if err != nil {
			return fmt.Errorf("failed to export to %s: %w", outputPath, err)
		}
		archivePath = outputPath
	} else {
		// Export to temporary directory
		archivePath, err = Export(store, options)
		if err != nil {
			return fmt.Errorf("failed to export: %w", err)
		}
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		absPath = archivePath
	}

	fmt.Printf("✅ Export completed successfully!\n")
	fmt.Printf("📦 Archive created: %s\n", absPath)

	// Show some info about the archive
	info, err := os.Stat(archivePath)
	if err == nil {
		fmt.Printf("📊 Archive size: %d bytes\n", info.Size())
	}

	return nil
}

// ExportCommandConfig provides configuration for creating export commands
// This can be used by applications that want to customize the export command behavior
type ExportCommandConfig struct {
	StorePath string
	Config    Config
	// Future: additional flags, custom validators, etc.
}

// CreateExportCommandWithConfig creates an export command with additional configuration options
// This is an extended version that allows for more customization in the future
func CreateExportCommandWithConfig(config ExportCommandConfig) *cobra.Command {
	return CreateExportCommand(config.StorePath, config.Config)
}
