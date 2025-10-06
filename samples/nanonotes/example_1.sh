#!/usr/bin/env bash
set -euo pipefail

# Source the base setup
source "$(dirname "$0")/example_base.sh"

echo "=== Example 1: Basic CRUD Operations and Queries ==="
echo

# Create notes
run_query "Creating a personal note" \
    $NANO_DB create \
    "Shopping List" \
    --body="Milk, Eggs, Bread, Coffee" \
    --category=personal

run_query "Creating a work note" \
    $NANO_DB create \
    "Important Meeting" \
    --body="Team sync at 2pm, Discuss Q4 goals" \
    --category=work \
    --content="Meeting agenda and notes"

run_query "Creating an idea note with tags" \
    $NANO_DB create \
    "Project Ideas" \
    --body="1. Build a CLI tool\n2. Write documentation\n3. Create examples" \
    --category=idea \
    --tags="development,cli,documentation"

run_query "Creating a reference note" \
    $NANO_DB create \
    "Git Commands" \
    --body="Common git commands reference" \
    --category=reference \
    --content="git status, git add, git commit, git push" \
    --tags="git,reference,commands"

# List notes
run_query "List all notes" \
    $NANO_DB list

run_query "List only work notes" \
    $NANO_DB list \
    --category=work

# Query notes
run_query "Find idea and reference notes" \
    $NANO_DB list \
    --category=idea --or --category=reference

run_query "Find notes with 'work' tag" \
    $NANO_DB list \
    --tags__contains=work

run_query "Find notes containing 'Meeting' in title" \
    $NANO_DB list \
    --title__contains=Meeting

# Update notes
run_query "Update the shopping list (ID 1)" \
    $NANO_DB update 1 \
    --body="Milk, Eggs, Bread, Coffee, Butter, Cheese" \
    --tags="shopping,urgent" \
    --content="Updated shopping list with more items"

run_query "Change git commands note to personal category (ID 4)" \
    $NANO_DB update 4 \
    --category=personal

# Get note details
run_query "Get details of note ID 1" \
    $NANO_DB get 1

# Delete notes
run_query "Delete the project ideas note (ID 3)" \
    $NANO_DB delete 3

run_query "Try to get the deleted note (should fail)" \
    $NANO_DB get 3 || true

run_query "List remaining notes" \
    $NANO_DB list

# Complex queries
run_query "Find work notes with specific tags" \
    $NANO_DB list \
    --category=work --tags__contains=development

run_query "Find notes that are personal OR have urgent tag" \
    $NANO_DB list \
    --category=personal --or --tags__contains=urgent

# Stats
run_query "Get store statistics" \
    $NANO_DB stats

# Show final database contents
show_db_contents