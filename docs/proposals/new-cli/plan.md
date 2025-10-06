# Implementation Plan: Ergonomic CLI Query Syntax

This document outlines the phased implementation plan for the new CLI query syntax, as detailed in `new-cli-spec.md`.

### Prerequisite: Define the `Query` Struct

Before implementation begins, a clear Go struct will be defined to represent a parsed query. This struct will serve as the contract between the pre-parser and the command execution logic. It will contain fields for filter groups, logical operators (`AND`/`OR`), and the relationships between them.

### Phase 1: The Pre-Parsing POC

- **Goal:** Validate the core mechanism of intercepting and rewriting arguments before passing them to Cobra. This is a low-risk step to prove the architectural approach.
- **Actions:**
    1.  Implement a `preParse` function that runs before `rootCmd.Execute()`.
    2.  Create a temporary `next` subcommand to isolate the new logic from existing commands.
    3.  Inside `preParse`, if the command is `next`, rewrite any known global flags (e.g., `--db`, `--type`) with an `--x-` prefix.
    4.  Pass the rewritten argument slice to Cobra for execution.
    5.  The `next` command itself will simply print the parsed `x-` flags to verify the mechanism works.

### Phase 2: The Option Parser - Simple AND Queries

- **Goal:** Parse all non-`--x-` flags into a `Query` object representing a single group of `AND` conditions.
- **Actions:**
    1.  Finalize the `Query` struct definition in Go.
    2.  Enhance the `preParse` function. For the `next` command, it will now isolate all filter flags (e.g., `--status=active`, `--priority__gte=5`).
    3.  Implement the logic to parse these flags into the `Query` object.
    4.  Write extensive unit tests to assert that various argument combinations produce the correct `Query` object.
    5.  Attach the `Query` object to the `cobra.Command`'s context for later use.

### Phase 3: The Option Parser - AND/OR Logic and Grouping

- **Goal:** Enhance the parser to handle `--and` and `--or` infix flags, respecting the strict left-to-right precedence.
- **Actions:**
    1.  Update the parser to recognize `--and` and `--or` as tokens that separate filter groups.
    2.  Implement the grouping logic based on the left-to-right evaluation rule.
    3.  Update the `Query` object structure if necessary to support multiple groups and the logic connecting them.
    4.  Add comprehensive unit tests for various combinations of `AND` and `OR` to ensure the query structure is built correctly.

### Phase 4: Switch-Over and Finalization

- **Goal:** Make the new parsing logic the default for all relevant commands and deprecate the old filter flags.
- **Actions:**
    1.  Modify the definitions of all relevant Cobra commands (`list`, `get`, etc.) to rename their flags with the `--x-` prefix (e.g., `cmd.Flags().String("x-db", ...)`).
    2.  Remove the check for the `next` subcommand from the `preParse` function, applying the logic to all commands.
    3.  Remove the old `--filter-xx` flags from the `list` command.
    4.  Integrate the parsed `Query` object into the command's execution logic, replacing the old filter mechanism.
