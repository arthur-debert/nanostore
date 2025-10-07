#!/usr/bin/env bash
set -euo pipefail

# Colors for better output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
DIM='\033[2m'
RESET='\033[0m'

# Setup
DB_FILE=$(mktemp /tmp/nano-db-showcase.XXXXXX.db)
PROJECT_ROOT=$(dirname "$(dirname "$(realpath "$0")")")
NANO_DB="$PROJECT_ROOT/bin/nano-db"

# Set environment variables
export NANOSTORE_DB="$DB_FILE"
export NANOSTORE_TYPE="Note"
export NANOSTORE_LOG_QUERIES="false"  # Set to true to see SQL queries
export NANOSTORE_LOG_RESULTS="false"  # Set to true to see operation results

if [ ! -f "$NANO_DB" ]; then
    echo -e "${RED}Error: nano-db binary not found at $NANO_DB${RESET}"
    echo "Please run './scripts/build' from the project root first."
    exit 1
fi

# Clean up on exit
trap "rm -f $DB_FILE" EXIT

# Helper function to run commands with nice formatting
run_command() {
    local description="$1"
    local command="$2"
    
    echo -e "\n${BLUE}=== $description ===${RESET}"
    echo -e "${DIM}$command${RESET}"
    
    # Execute command and capture output
    local output=$($command 2>&1)
    echo -e "${output}"
}

# Helper function to show database contents
show_database() {
    echo -e "\n${CYAN}📊 Current Database Contents:${RESET}"
    echo -e "${DIM}$NANO_DB list --x-format=table${RESET}"
    $NANO_DB list --x-format=table
}

echo -e "${GREEN}🚀 nano-db CLI Showcase: Blog Management System${RESET}"
echo -e "${DIM}This demo showcases the nano-db CLI with a blog post management system.${RESET}"
echo -e "${DIM}Database: $DB_FILE${RESET}"

# =============================================================================
# SESSION 1: Basic CRUD Operations
# =============================================================================

echo -e "\n${YELLOW}📝 SESSION 1: Basic CRUD Operations${RESET}"

# Create some blog posts
run_command "Creating a technical blog post" \
    "$NANO_DB create Getting_Started_with_Go --category=work --tags=golang,programming --content='Learn the basics of Go programming language' --author=alice --status=published"

run_command "Creating a lifestyle blog post" \
    "$NANO_DB create Healthy_Morning_Routine --category=personal --tags=health,wellness --content='Start your day with these simple habits' --author=bob --status=draft"

run_command "Creating a business blog post" \
    "$NANO_DB create Remote_Work_Best_Practices --category=work --tags=remote,productivity --content='Tips for effective remote work' --author=alice --status=published"

run_command "Creating a tech tutorial" \
    "$NANO_DB create Building_REST_APIs_in_Go --category=work --tags=golang,api,backend --content='Step-by-step guide to building REST APIs' --author=charlie --status=published"

show_database

# Read operations
run_command "Getting a specific blog post" \
    "$NANO_DB get 1"

run_command "Getting post metadata" \
    "$NANO_DB get-metadata 2"

# Update operations
run_command "Updating a blog post" \
    "$NANO_DB update 2 --status=published --tags=health,wellness,motivation"

run_command "Updating post content" \
    "$NANO_DB update 4 --content='Comprehensive guide to building scalable REST APIs in Go with best practices and examples'"

show_database

# =============================================================================
# SESSION 2: Advanced Querying and Filtering
# =============================================================================

echo -e "\n${YELLOW}🔍 SESSION 2: Advanced Querying and Filtering${RESET}"

run_command "Filter by category" \
    "$NANO_DB list --category=work"

run_command "Filter by author" \
    "$NANO_DB list --author=alice"

run_command "Filter by status" \
    "$NANO_DB list --status=published"

run_command "Search by content (contains)" \
    "$NANO_DB list --content__contains=REST"

run_command "Search by tags (contains)" \
    "$NANO_DB list --tags__contains=golang"

run_command "Complex filter: published work posts" \
    "$NANO_DB list --category=work --status=published"

run_command "Filter by author AND category" \
    "$NANO_DB list --author=alice --category=work"

show_database

# =============================================================================
# SESSION 3: Logical Operators (AND, OR)
# =============================================================================

echo -e "\n${YELLOW}🔗 SESSION 3: Logical Operators (AND, OR)${RESET}"

run_command "OR operator: work OR personal posts" \
    "$NANO_DB list --category=work --or --category=personal"

run_command "OR operator: posts by alice OR charlie" \
    "$NANO_DB list --author=alice --or --author=charlie"

run_command "OR operator: published OR draft posts" \
    "$NANO_DB list --status=published --or --status=draft"

run_command "Complex OR: posts containing 'Go' OR 'remote'" \
    "$NANO_DB list --content__contains=Go --or --content__contains=remote"

run_command "Combined AND/OR: (work AND published) OR (personal AND published)" \
    "$NANO_DB list --category=work --status=published --or --category=personal --status=published"

# Advanced logical combinations
run_command "Complex query: posts by alice that are (work OR personal) and published" \
    "$NANO_DB list --author=alice --category=work --or --category=personal --status=published"

show_database

# =============================================================================
# SESSION 4: Bulk Operations
# =============================================================================

echo -e "\n${YELLOW}⚡ SESSION 4: Bulk Operations${RESET}"

# Add a few more posts for bulk operations
run_command "Adding more posts for bulk operations" \
    "$NANO_DB create Advanced_Go_Patterns --category=work --tags=golang,advanced --author=charlie --status=draft"

run_command "Adding another draft post" \
    "$NANO_DB create Go_Testing_Best_Practices --category=work --tags=golang,testing --author=alice --status=draft"

show_database

echo -e "\n${PURPLE}🔧 Bulk Update Operations:${RESET}"

run_command "Bulk update: Publish all draft work posts" \
    "$NANO_DB update-by-dimension --sql --category=work --status=draft --data --status=published"

run_command "Bulk update: Add 'tutorial' tag to all work posts" \
    "$NANO_DB update-by-dimension --sql --category=work --data --tags__contains=tutorial"

run_command "Bulk update: Change author for specific posts" \
    "$NANO_DB update-where --sql --author=charlie --status=published --data --author=charlie_senior"

show_database

echo -e "\n${PURPLE}🗑️  Bulk Delete Operations:${RESET}"

run_command "Bulk delete: Remove all draft posts" \
    "$NANO_DB delete-by-dimension --status=draft"

run_command "Bulk delete: Remove posts by specific author" \
    "$NANO_DB delete-where --author=bob"

show_database

# =============================================================================
# SESSION 5: Sorting, Limiting, and Formatting
# =============================================================================

echo -e "\n${YELLOW}📊 SESSION 5: Sorting, Limiting, and Formatting${RESET}"

# Add a few more posts with different timestamps
run_command "Adding more posts for sorting demo" \
    "$NANO_DB create Go_Concurrency_Patterns --category=work --tags=golang,concurrency --author=alice --status=published"

run_command "Adding another post" \
    "$NANO_DB create Microservices_with_Go --category=work --tags=golang,microservices --author=charlie_senior --status=published"

show_database

run_command "Sort by title (ascending)" \
    "$NANO_DB list --x-format=table --sort=title"

run_command "Sort by author (descending)" \
    "$NANO_DB list --x-format=table --sort=-author"

run_command "Limit results to 3 posts" \
    "$NANO_DB list --x-format=table --limit=3"

run_command "Get first 2 work posts sorted by title" \
    "$NANO_DB list --category=work --x-format=table --sort=title --limit=2"

run_command "Output in JSON format" \
    "$NANO_DB list --category=work --x-format=json"

run_command "Output in YAML format" \
    "$NANO_DB list --category=work --x-format=yaml"

# =============================================================================
# SESSION 6: Store Statistics and Metadata
# =============================================================================

echo -e "\n${YELLOW}📈 SESSION 6: Store Statistics and Metadata${RESET}"

run_command "Get store statistics" \
    "$NANO_DB stats"

run_command "Get field usage statistics" \
    "$NANO_DB field-stats"

run_command "Count total posts" \
    "$NANO_DB count"

run_command "Count work posts" \
    "$NANO_DB count --category=work"

run_command "Count posts by author" \
    "$NANO_DB count --author=alice"

# =============================================================================
# SESSION 7: Advanced Search and Complex Queries
# =============================================================================

echo -e "\n${YELLOW}🔎 SESSION 7: Advanced Search and Complex Queries${RESET}"

run_command "Search for posts containing 'Go' in content" \
    "$NANO_DB list --content__contains=Go --x-format=table"

run_command "Find posts with 'golang' tag" \
    "$NANO_DB list --tags__contains=golang --x-format=table"

run_command "Complex search: published work posts containing 'API' or 'REST'" \
    "$NANO_DB list --category=work --status=published --content__contains=API --or --content__contains=REST"

run_command "Find posts by multiple authors" \
    "$NANO_DB list --author=alice --or --author=charlie_senior"

run_command "Posts with specific tag combinations" \
    "$NANO_DB list --tags__contains=golang --tags__contains=advanced"

# =============================================================================
# SESSION 8: Error Handling and Edge Cases
# =============================================================================

echo -e "\n${YELLOW}⚠️  SESSION 8: Error Handling and Edge Cases${RESET}"

run_command "Try to get non-existent post" \
    "$NANO_DB get 999 || echo 'Expected: Post not found'"

run_command "Try to update non-existent post" \
    "$NANO_DB update 999 --status=published || echo 'Expected: Update failed'"

run_command "Try to delete non-existent post" \
    "$NANO_DB delete 999 || echo 'Expected: Delete failed'"

run_command "Query with no results" \
    "$NANO_DB list --category=nonexistent"

run_command "Bulk update with no matching records" \
    "$NANO_DB update-by-dimension --sql --category=nonexistent --data --status=published"

# =============================================================================
# FINAL SHOWCASE
# =============================================================================

echo -e "\n${GREEN}🎉 Final Database State${RESET}"
show_database

echo -e "\n${GREEN}📊 Final Statistics${RESET}"
run_command "Final store statistics" \
    "$NANO_DB stats"

echo -e "\n${GREEN}✨ nano-db CLI Showcase Complete!${RESET}"
echo -e "${DIM}This demo showed:${RESET}"
echo -e "${DIM}• Basic CRUD operations (create, read, update, delete)${RESET}"
echo -e "${DIM}• Advanced filtering and querying${RESET}"
echo -e "${DIM}• Logical operators (AND, OR)${RESET}"
echo -e "${DIM}• Bulk operations (update-by-dimension, update-where, delete-by-dimension, delete-where)${RESET}"
echo -e "${DIM}• Sorting, limiting, and different output formats${RESET}"
echo -e "${DIM}• Store statistics and metadata${RESET}"
echo -e "${DIM}• Complex search queries${RESET}"
echo -e "${DIM}• Error handling${RESET}"
echo -e "\n${DIM}Database file: $DB_FILE${RESET}"
echo -e "${DIM}Clean up: rm -f $DB_FILE${RESET}"
