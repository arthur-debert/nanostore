package main

import (
	"fmt"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/migration"
	"github.com/spf13/cobra"
)

var (
	transformer string
)

var transformFieldCmd = &cobra.Command{
	Use:   "transform-field <field-name>",
	Short: "Transform field values across all documents",
	Long: `Transform field values using a built-in transformer.
Available transformers:
  - toString: Convert any value to string
  - toInt: Convert to integer
  - toFloat: Convert to float
  - toLowerCase: Convert string to lowercase
  - toUpperCase: Convert string to uppercase
  - trim: Trim whitespace from strings`,
	Args: cobra.ExactArgs(1),
	RunE: runTransformField,
}

func init() {
	transformFieldCmd.Flags().StringVar(&transformer, "transformer", "", "name of the transformer to apply (required)")
	_ = transformFieldCmd.MarkFlagRequired("transformer")
}

func runTransformField(cmd *cobra.Command, args []string) error {
	fieldName := args[0]

	// Load store
	store, err := loadStore()
	if err != nil {
		return fmt.Errorf("failed to load store: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Get all documents
	docs, err := store.List(nanostore.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	// TODO: Get actual config from store
	config := nanostore.Config{}

	// Run migration
	api := migration.NewAPI()
	result := api.TransformField(docs, config, fieldName, transformer, migration.Options{
		DryRun:  dryRun,
		Verbose: verbose,
	})

	handleResult(result)
	return nil
}
