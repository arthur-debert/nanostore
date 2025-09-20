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

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate that all documents conform to the schema",
	Long:  "Check that all documents have consistent fields and valid values according to the current schema.",
	Args:  cobra.NoArgs,
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
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

	// Run validation
	api := migration.NewAPI()
	result := api.ValidateSchema(docs, config, migration.Options{
		DryRun:  dryRun,
		Verbose: verbose,
	})

	handleResult(result)
	return nil
}
