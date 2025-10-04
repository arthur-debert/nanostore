package main

import (
	"fmt"
	"os"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/store"
	"github.com/arthur-debert/nanostore/types"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nanostore",
	Short: "Nanostore CLI - Document and ID store management",
	Long: `Nanostore is a document and ID store library that uses JSON file storage
to manage document storage and dynamically generate user-facing, hierarchical IDs.

This CLI provides basic operations for managing nanostore databases.`,
}

var (
	storePath  string
	configFile string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&storePath, "store", "s", "store.json", "Path to the nanostore database file")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (optional)")

	// Add the export command
	// For this basic CLI, we'll use a simple default configuration
	// In a real application, this would be loaded from a config file
	defaultConfig := types.Config{
		Dimensions: []nanostore.DimensionConfig{
			{
				Name:   "status",
				Type:   types.Enumerated,
				Values: []string{"pending", "active", "completed"},
				Prefixes: map[string]string{
					"pending":   "p",
					"active":    "a",
					"completed": "c",
				},
				DefaultValue: "pending",
			},
		},
	}

	// Create export command with dynamic store path
	exportCmd := &cobra.Command{
		Use:   "export [id1 id2 ...]",
		Short: "Export documents from the nanostore",
		Long: `Export documents from the nanostore to a zip archive.

The export command creates a zip file containing:
- db.json: Complete database representation  
- Individual files for each document's content

Examples:
  # Export all documents
  nanostore export

  # Export specific documents by ID
  nanostore export 1 c2 1.3

  # Export to a specific path
  nanostore export --output /path/to/export.zip

  # Use different store file
  nanostore --store mydata.json export`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the output flag
			outputPath, _ := cmd.Flags().GetString("output")

			// Open the store
			store, err := store.New(storePath, &defaultConfig)
			if err != nil {
				return fmt.Errorf("failed to open store at %s: %w", storePath, err)
			}
			defer func() {
				if err := store.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "Error closing store: %v\n", err)
				}
			}()

			// Prepare export options
			options := nanostore.ExportOptions{}
			if len(args) > 0 {
				options.IDs = args
				fmt.Printf("Exporting %d specific documents...\n", len(args))
			} else {
				fmt.Println("Exporting all documents...")
			}

			// Get metadata
			metadata, err := nanostore.GetExportMetadata(store, options)
			if err != nil {
				return fmt.Errorf("failed to get export metadata: %w", err)
			}

			if metadata.DocumentCount == 0 {
				fmt.Println("No documents found to export.")
				return nil
			}

			fmt.Printf("Found %d documents to export\n", metadata.DocumentCount)

			// Perform export
			var archivePath string
			if outputPath != "" {
				err = nanostore.ExportToPath(store, options, outputPath)
				if err != nil {
					return fmt.Errorf("failed to export: %w", err)
				}
				archivePath = outputPath
			} else {
				archivePath, err = nanostore.Export(store, options)
				if err != nil {
					return fmt.Errorf("failed to export: %w", err)
				}
			}

			fmt.Printf("✅ Export completed: %s\n", archivePath)
			return nil
		},
	}

	exportCmd.Flags().StringP("output", "o", "", "Output path for the export archive")
	rootCmd.AddCommand(exportCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
