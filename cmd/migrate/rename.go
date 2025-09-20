// Part of the nanostore CLI - this file implements the 'nanostore migrate <command>' subcommand.
// Build the CLI with: scripts/build
// This creates bin/nanostore which includes all migration commands.
package main

import (
	"fmt"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/migration"
	"github.com/spf13/cobra"
)

var (
	renameDimensionOnly bool
	renameDataOnly      bool
	renameBoth          bool
)

var renameFieldCmd = &cobra.Command{
	Use:   "rename-field <old-name> <new-name>",
	Short: "Rename a field across all documents",
	Long:  "Rename a field from old-name to new-name in all documents. The field can be either a dimension or a data field.",
	Args:  cobra.ExactArgs(2),
	RunE:  runRenameField,
}

func init() {
	renameFieldCmd.Flags().BoolVar(&renameDimensionOnly, "dimension-only", false, "only rename dimension fields")
	renameFieldCmd.Flags().BoolVar(&renameDataOnly, "data-only", false, "only rename data fields")
	renameFieldCmd.Flags().BoolVar(&renameBoth, "both", false, "explicitly rename both dimension and data fields")
}

func runRenameField(cmd *cobra.Command, args []string) error {
	oldName, newName := args[0], args[1]

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

	// Determine field type based on flags
	fieldType := migration.FieldTypeAuto
	if renameDimensionOnly {
		fieldType = migration.FieldTypeDimension
	} else if renameDataOnly {
		fieldType = migration.FieldTypeData
	} else if renameBoth {
		fieldType = migration.FieldTypeBoth
	}

	// Run migration
	api := migration.NewAPI()
	_, result := api.RenameField(docs, config, oldName, newName, migration.Options{
		DryRun:    dryRun,
		Verbose:   verbose,
		FieldType: fieldType,
	})

	handleResult(result)
	return nil
}
