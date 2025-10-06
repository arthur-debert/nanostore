#!/usr/bin/env bash
set -euo pipefail

# Colors
DIM='\033[2m'
RESET='\033[0m'

# Setup
DB_FILE=$(mktemp /tmp/nanonotes-example.XXXXXX.db)
NANO_DB="../../bin/nano-db"

# Set environment variables for database, type, and query logging
export NANOSTORE_DB="$DB_FILE"
export NANOSTORE_TYPE="Note"
export NANOSTORE_LOG_QUERIES="true"

if [ ! -f "$NANO_DB" ]; then
    echo "Error: nano-db binary not found at $NANO_DB"
    echo "Please run './scripts/build' from the project root first."
    exit 1
fi

# Clean up on exit
trap "rm -f $DB_FILE" EXIT

# Helper function to run queries
run_query() {
    local description="$1"
    shift

    # Echo the description (normal text)
    echo "$description"

    # Build command display, replacing paths with shorter versions
    local cmd_display=$(echo "$@" | sed "s|$NANO_DB|nano-db|g" | sed "s|$DB_FILE|notes.db|g")
    echo "$cmd_display"

    # Capture and display output in dim color
    local output=$("$@" 2>&1)
    echo -e "${DIM}${output}${RESET}"
    echo
}

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

# Database dump
echo -e "${DIM}=== Database Contents ===${RESET}"
echo -e "${DIM}$($NANO_DB list --x-format=json | jq '.')${RESET}"