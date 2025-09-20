// Part of the nanostore CLI - this file implements the 'nanostore migrate <command>' subcommand.
// Build the CLI with: scripts/build
// This creates bin/nanostore which includes all migration commands.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/arthur-debert/nanostore/nanostore"
	"github.com/arthur-debert/nanostore/nanostore/migration"
	"github.com/spf13/cobra"
)

var (
	defaultValue string
	isDataField  bool
)

var addFieldCmd = &cobra.Command{
	Use:   "add-field <field-name>",
	Short: "Add a field with a default value to all documents",
	Long:  "Add a new field to all documents with the specified default value.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddField,
}

func init() {
	addFieldCmd.Flags().StringVar(&defaultValue, "default", "", "default value for the field (required)")
	addFieldCmd.Flags().BoolVar(&isDataField, "data-field", false, "add as data field (with _data. prefix)")
	_ = addFieldCmd.MarkFlagRequired("default")
}

func runAddField(cmd *cobra.Command, args []string) error {
	fieldName := args[0]

	// Parse default value
	var parsedValue interface{}

	// Try to parse as JSON first
	if err := json.Unmarshal([]byte(defaultValue), &parsedValue); err != nil {
		// If not valid JSON, use as string
		parsedValue = defaultValue
	}

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
	result := api.AddField(docs, config, fieldName, parsedValue, migration.Options{
		DryRun:      dryRun,
		Verbose:     verbose,
		IsDataField: isDataField,
	})

	handleResult(result)
	return nil
}
