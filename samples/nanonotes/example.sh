#!/usr/bin/env bash
set -euo pipefail

# Colors for output
PRIMARY='\033[0m'     # Normal/primary text
MUTED='\033[2m'       # Dim/muted text
NC='\033[0m'          # No Color/Reset

# Create a temporary database file
DB_FILE=$(mktemp /tmp/nanonotes-example.XXXXXX.db)
echo -e "${PRIMARY}Created temporary database: $DB_FILE${NC}"
echo

# Ensure nano-db is available
NANO_DB="../../bin/nano-db"
if [ ! -f "$NANO_DB" ]; then
    echo "Error: nano-db binary not found at $NANO_DB"
    echo "Please run './scripts/build' from the project root first."
    exit 1
fi

# Clean up on exit
trap "rm -f $DB_FILE" EXIT

echo -e "${PRIMARY}=== NanoNotes Example with SQL Query Logging ===${NC}"
echo
echo -e "${MUTED}Note: This script demonstrates the nano-db CLI with query logging."
echo "      Watch for the SQL queries being logged to stdout!"
echo "      Some commands may show mock data as they're not fully implemented yet.${NC}"
echo

# Helper function to display commands and their output
run_command() {
    local description="$1"
    shift
    # Build command string, replacing the full path with just "nano-db"
    local cmd_display=$(echo "$@" | sed "s|$NANO_DB|nano-db|g" | sed "s|$DB_FILE|notes.db|g")
    
    echo -e "${PRIMARY}$description${NC}"
    echo -e "${PRIMARY}$cmd_display${NC}"
    echo -e "${MUTED}"
    "$@"
    echo -e "${NC}"
    echo
}

# Create several notes
echo -e "${PRIMARY}1. Creating Notes${NC}"
echo

run_command "Creating a personal note" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Shopping List" \
    --body="Milk, Eggs, Bread, Coffee" \
    --category=personal

run_command "Creating a work note" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Important Meeting" \
    --body="Team sync at 2pm, Discuss Q4 goals" \
    --category=work \
    --content="Meeting agenda and notes"

run_command "Creating an idea note with tags" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Project Ideas" \
    --body="1. Build a CLI tool\n2. Write documentation\n3. Create examples" \
    --category=idea \
    --tags="development,cli,documentation"

run_command "Creating a reference note" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" create \
    --x-log-queries \
    "Git Commands" \
    --body="Common git commands reference" \
    --category=reference \
    --content="git status, git add, git commit, git push" \
    --tags="git,reference,commands"

# List all notes
echo -e "${PRIMARY}2. Listing Notes${NC}"
echo

run_command "List all notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries

run_command "List only work notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=work

# Query notes with various filters
echo -e "${PRIMARY}3. Querying Notes${NC}"
echo

run_command "Find idea and reference notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=idea --or --category=reference

run_command "Find notes with 'work' tag" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --tags__contains=work

run_command "Find notes containing 'Meeting' in title" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --title__contains=Meeting

# Update a note
echo -e "${PRIMARY}4. Updating Notes${NC}"
echo

run_command "Update the shopping list (ID 1)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" update 1 \
    --x-log-queries \
    --body="Milk, Eggs, Bread, Coffee, Butter, Cheese" \
    --tags="shopping,urgent" \
    --content="Updated shopping list with more items"

run_command "Change git commands note to personal category (ID 4)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" update 4 \
    --x-log-queries \
    --category=personal

# Get specific note details
echo -e "${PRIMARY}5. Getting Note Details${NC}"
echo

run_command "Get details of note ID 1" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" get 1 \
    --x-log-queries

# Delete a note
echo -e "${PRIMARY}6. Deleting Notes${NC}"
echo

run_command "Delete the project ideas note (ID 3)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" delete 3 \
    --x-log-queries

run_command "Try to get the deleted note (should fail)" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" get 3 \
    --x-log-queries || echo "Note not found (as expected)"

run_command "List remaining notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries

# Complex queries
echo -e "${PRIMARY}7. Complex Queries${NC}"
echo

run_command "Find work notes with specific tags" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=work --tags__contains=development

run_command "Find notes that are personal OR have urgent tag" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=personal --or --tags__contains=urgent

# Stats
echo -e "${PRIMARY}8. Database Statistics${NC}"
echo

run_command "Get store statistics" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" stats \
    --x-log-queries

echo -e "${MUTED}=== Database Contents ===${NC}"
echo
echo -e "${MUTED}"
$NANO_DB --x-type Note --x-db="$DB_FILE" list --x-format=json | jq '.'
echo -e "${NC}"