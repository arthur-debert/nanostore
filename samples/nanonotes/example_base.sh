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

# Helper function to show database contents
show_db_contents() {
    echo -e "${DIM}=== Database Contents ===${RESET}"
    echo -e "${DIM}$($NANO_DB list --x-format=json | jq '.')${RESET}"
}