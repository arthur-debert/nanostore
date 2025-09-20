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

var removeFieldCmd = &cobra.Command{
	Use:   "remove-field <field-name>",
	Short: "Remove a field from all documents",
	Long:  "Remove a field from all documents. The field can be either a dimension or a data field.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoveField,
}

func runRemoveField(cmd *cobra.Command, args []string) error {
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
	result := api.RemoveField(docs, config, fieldName, migration.Options{
		DryRun:  dryRun,
		Verbose: verbose,
	})

	handleResult(result)
	return nil
}
