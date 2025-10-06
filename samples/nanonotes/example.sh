#!/usr/bin/env bash
set -euo pipefail

# Colors
DIM='\033[2m'
RESET='\033[0m'

# Setup
DB_FILE=$(mktemp /tmp/nanonotes-example.XXXXXX.db)
NANO_DB="${PROJECT_ROOT}/bin/nano-db"
# we can use the env var to avoid passing the db path
export NANO_DB

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
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Shopping List" \
    --body="Milk, Eggs, Bread, Coffee" \
    --category=personal

run_query "Creating a work note" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Important Meeting" \
    --body="Team sync at 2pm, Discuss Q4 goals" \
    --category=work \
    --content="Meeting agenda and notes"

run_query "Creating an idea note with tags" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Project Ideas" \
    --body="1. Build a CLI tool\n2. Write documentation\n3. Create examples" \
    --category=idea \
    --tags="development,cli,documentation"

run_query "Creating a reference note" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Git Commands" \
    --body="Common git commands reference" \
    --category=reference \
    --content="git status, git add, git commit, git push" \
    --tags="git,reference,commands"

# List notes
run_query "List all notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries

run_query "List only work notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=work

# Query notes
run_query "Find idea and reference notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=idea --or --category=reference

run_query "Find notes with 'work' tag" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --tags__contains=work

run_query "Find notes containing 'Meeting' in title" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --title__contains=Meeting

# Update notes
run_query "Update the shopping list (ID 1)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" update 1 \
    --x-log-queries \
    --body="Milk, Eggs, Bread, Coffee, Butter, Cheese" \
    --tags="shopping,urgent" \
    --content="Updated shopping list with more items"

run_query "Change git commands note to personal category (ID 4)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" update 4 \
    --x-log-queries \
    --category=personal

# Get note details
run_query "Get details of note ID 1" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" get 1 \
    --x-log-queries

# Delete notes
run_query "Delete the project ideas note (ID 3)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" delete 3 \
    --x-log-queries

run_query "Try to get the deleted note (should fail)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" get 3 \
    --x-log-queries || true

run_query "List remaining notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries

# Complex queries
run_query "Find work notes with specific tags" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=work --tags__contains=development

run_query "Find notes that are personal OR have urgent tag" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=personal --or --tags__contains=urgent

# Stats
run_query "Get store statistics" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" stats \
    --x-log-queries

# Database dump
echo -e "${DIM}=== Database Contents ===${RESET}"
echo -e "${DIM}$($NANO_DB --x-type Note --x-db="$DB_FILE" list --x-format=json | jq '.')${RESET}"

