package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arthur-debert/nanostore/formats"
	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/spf13/cobra"
)

var (
	exportOutput string
	exportFormat string
)

var exportCmd = &cobra.Command{
	Use:   "export [id1 id2 ...]",
	Short: "Export todos to a zip archive",
	Long: `Export todos to a zip archive containing the database and individual files.

The export creates a zip file with:
- db.json: Complete database representation
- Individual text files for each todo's content

You can export specific todos by providing their IDs, or export all todos
if no IDs are specified.

Examples:
  todos export                            # Export all todos as plaintext
  todos export --format markdown          # Export all todos as markdown
  todos export 1 h2.1                    # Export specific todos by ID
  todos export --output backup.zip        # Export all to specific file
  todos export --format md --output my.zip # Export as markdown to specific file

Available formats:
  plaintext: Simple text format with title on first line (.txt)
  markdown: Markdown format with # Title header (.md)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := NewTodoApp(storePath)
		if err != nil {
			return fmt.Errorf("failed to open todos: %w", err)
		}
		defer app.Close()

		// Get the document format
		format, err := formats.Get(exportFormat)
		if err != nil {
			return fmt.Errorf("invalid format %q: %w", exportFormat, err)
		}

		// Create export options
		options := nanostore.ExportOptions{
			DocumentFormat: format,
		}
		if len(args) > 0 {
			options.IDs = args
			fmt.Printf("Exporting %d specific todos: %v\n", len(args), args)
		} else {
			fmt.Println("Exporting all todos...")
		}
		fmt.Printf("Format: %s\n", format.Name)

		// Get metadata to show what will be exported
		metadata, err := nanostore.GetExportMetadata(app.store.Store(), options)
		if err != nil {
			return fmt.Errorf("failed to get export metadata: %w", err)
		}

		if metadata.DocumentCount == 0 {
			fmt.Println("No todos found to export.")
			return nil
		}

		fmt.Printf("Found %d todos to export\n", metadata.DocumentCount)

		if verbose {
			fmt.Println("Todos to be exported:")
			for _, doc := range metadata.Documents {
				fmt.Printf("  %s. %s → %s\n", doc.SimpleID, doc.Title, doc.Filename)
			}
			fmt.Printf("Estimated archive size: %d bytes\n", metadata.EstimatedSizeBytes)
			fmt.Println()
		}

		// Perform the export
		var archivePath string
		if exportOutput != "" {
			// Export to specified path
			// Ensure the directory exists
			dir := filepath.Dir(exportOutput)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			err = nanostore.ExportToPath(app.store.Store(), options, exportOutput)
			if err != nil {
				return fmt.Errorf("failed to export to %s: %w", exportOutput, err)
			}
			archivePath = exportOutput
		} else {
			// Export to temporary directory
			archivePath, err = nanostore.Export(app.store.Store(), options)
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

		if verbose {
			fmt.Printf("💡 You can extract and examine the archive contents:\n")
			fmt.Printf("   unzip -l \"%s\"\n", absPath)
			fmt.Printf("   unzip \"%s\" -d extracted/\n", absPath)
		}

		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output path for the export archive")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "plaintext", "Document format (plaintext or markdown)")
	rootCmd.AddCommand(exportCmd)
}
