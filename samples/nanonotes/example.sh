#!/usr/bin/env bash
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Create a temporary database file
DB_FILE=$(mktemp /tmp/nanonotes-example.XXXXXX.db)
echo -e "${BLUE}${BOLD}Created temporary database: ${NC}$DB_FILE"
echo

# Ensure nano-db is available
NANO_DB="../../bin/nano-db"
if [ ! -f "$NANO_DB" ]; then
    echo -e "${RED}Error: nano-db binary not found at $NANO_DB${NC}"
    echo "Please run './scripts/build' from the project root first."
    exit 1
fi

# Clean up on exit
trap "rm -f $DB_FILE" EXIT

echo -e "${GREEN}${BOLD}=== NanoNotes Example with SQL Query Logging ===${NC}"
echo
echo -e "${YELLOW}Note: This script demonstrates the nano-db CLI with query logging.${NC}"
echo -e "${YELLOW}      Watch for the SQL queries being logged to stdout!${NC}"
echo -e "${YELLOW}      Some commands may show mock data as they're not fully implemented yet.${NC}"
echo

# Helper function to display commands and their output
run_command() {
    local description="$1"
    shift
    echo -e "${YELLOW}${BOLD}$description${NC}"
    echo -e "${BLUE}Command:${NC} $@"
    echo -e "${GREEN}Output:${NC}"
    "$@"
    echo
    echo "---"
    echo
}

# Create several notes
echo -e "${GREEN}${BOLD}1. Creating Notes${NC}"
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
echo -e "${GREEN}${BOLD}2. Listing Notes${NC}"
echo

run_command "List all notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries

run_command "List only work notes" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" list \
    --x-log-queries \
    --category=work

# Query notes with various filters
echo -e "${GREEN}${BOLD}3. Querying Notes${NC}"
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
echo -e "${GREEN}${BOLD}4. Updating Notes${NC}"
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
echo -e "${GREEN}${BOLD}5. Getting Note Details${NC}"
echo

run_command "Get details of note ID 1" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" get 1 \
    --x-log-queries

# Delete a note
echo -e "${GREEN}${BOLD}6. Deleting Notes${NC}"
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
echo -e "${GREEN}${BOLD}7. Complex Queries${NC}"
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
echo -e "${GREEN}${BOLD}8. Database Statistics${NC}"
echo

run_command "Get store statistics" \
    $NANO_DB --x-type Note --x-db="$DB_FILE" stats \
    --x-log-queries

echo -e "${GREEN}${BOLD}=== Example Complete ===${NC}"
echo -e "Database file was: $DB_FILE (will be deleted on exit)"