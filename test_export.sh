#!/bin/bash

# Test script for nanostore export functionality
# Builds todos CLI, creates test data, exports, and verifies

set -e  # Exit on error

echo "=== Nanostore Export Test Script ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test directory
TEST_DIR="test_export_tmp"
TODOS_DB="$TEST_DIR/todos.json"
EXPORT_DIR="$TEST_DIR/exports"

# Clean up function
cleanup() {
    echo "Cleaning up test files..."
    rm -rf "$TEST_DIR"
    rm -f todos.json  # Clean up default location too
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Create test directories
echo "1. Setting up test environment..."
mkdir -p "$TEST_DIR"
mkdir -p "$EXPORT_DIR"

# Build the todos CLI
echo "2. Building todos CLI..."
(cd samples/todos && go build -o ../../.bin/todos ./cmd) || {
    echo -e "${RED}Failed to build todos CLI${NC}"
    exit 1
}
echo -e "${GREEN}✓ Build successful${NC}"

# Create test data
echo
echo "3. Creating test todos..."
TODOS_CMD="./.bin/todos --store $TODOS_DB"

# Create root todos
$TODOS_CMD add "Project Alpha"
$TODOS_CMD add "Shopping List" 
$TODOS_CMD add --priority high "Urgent: Fix Production Bug"
$TODOS_CMD add "Learn Go"

# Add subtasks
$TODOS_CMD add --parent 1 "Design database schema"
$TODOS_CMD add --parent 1 "Implement API endpoints"
$TODOS_CMD add --parent 1 "Write tests"
$TODOS_CMD add --parent 2 "Buy milk"
$TODOS_CMD add --parent 2 "Get bread"
$TODOS_CMD add --parent 4 "Complete Go tour"
$TODOS_CMD add --parent 4 "Read Effective Go"

# Complete some tasks
$TODOS_CMD complete 8  # Complete "Buy milk"
$TODOS_CMD complete 7  # Complete "Write tests"

echo -e "${GREEN}✓ Created test data${NC}"

# Show current state
echo
echo "4. Current todo list:"
$TODOS_CMD list --all

# Test 1: Export all todos
echo
echo "5. Testing full export..."
EXPORT_ALL="$EXPORT_DIR/export_all.zip"
$TODOS_CMD export --output "$EXPORT_ALL"

# Verify export exists and has content
if [ ! -f "$EXPORT_ALL" ]; then
    echo -e "${RED}✗ Export file not created${NC}"
    exit 1
fi

SIZE=$(stat -f%z "$EXPORT_ALL" 2>/dev/null || stat -c%s "$EXPORT_ALL" 2>/dev/null)
if [ "$SIZE" -lt 1000 ]; then
    echo -e "${RED}✗ Export file suspiciously small: $SIZE bytes${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Full export created: $SIZE bytes${NC}"

# Extract and verify
EXTRACT_DIR="$EXPORT_DIR/extracted_all"
mkdir -p "$EXTRACT_DIR"
unzip -q "$EXPORT_ALL" -d "$EXTRACT_DIR"

# Check for db.json
if [ ! -f "$EXTRACT_DIR/db.json" ]; then
    echo -e "${RED}✗ db.json not found in export${NC}"
    exit 1
fi

# Count document files (should be 11 todos)
DOC_COUNT=$(find "$EXTRACT_DIR" -name "*.txt" | wc -l)
echo "  Found $DOC_COUNT document files"

# Test 2: Export specific todos
echo
echo "6. Testing selective export..."
EXPORT_SELECTED="$EXPORT_DIR/export_selected.zip"
$TODOS_CMD export 1 h1 4 --output "$EXPORT_SELECTED" --verbose

# Verify selective export
EXTRACT_SELECTED="$EXPORT_DIR/extracted_selected"
mkdir -p "$EXTRACT_SELECTED"
unzip -q "$EXPORT_SELECTED" -d "$EXTRACT_SELECTED"

SELECTED_COUNT=$(find "$EXTRACT_SELECTED" -name "*.txt" | wc -l)
if [ "$SELECTED_COUNT" -ne 3 ]; then
    echo -e "${RED}✗ Expected 3 documents, found $SELECTED_COUNT${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Selective export successful: $SELECTED_COUNT documents${NC}"

# Test 3: Export to temp directory
echo
echo "7. Testing export to temp directory..."
TEMP_EXPORT=$($TODOS_CMD export | grep "Archive created:" | sed 's/.*Archive created: //')
if [ -z "$TEMP_EXPORT" ]; then
    echo -e "${RED}✗ Failed to extract temp export path${NC}"
    exit 1
fi

if [ ! -f "$TEMP_EXPORT" ]; then
    echo -e "${RED}✗ Temp export file not found: $TEMP_EXPORT${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Temp export created at: $TEMP_EXPORT${NC}"

# Test 4: Verify file naming
echo
echo "8. Verifying file naming conventions..."
EXTRACT_NAMES="$EXPORT_DIR/extracted_names"
mkdir -p "$EXTRACT_NAMES"
unzip -q "$EXPORT_ALL" -d "$EXTRACT_NAMES"

# Check for UUID pattern in filenames
INVALID_NAMES=$(find "$EXTRACT_NAMES" -name "*.txt" ! -regex ".*/[a-f0-9-]\{36\}-.*\.txt" | wc -l)
if [ "$INVALID_NAMES" -gt 0 ]; then
    echo -e "${RED}✗ Found $INVALID_NAMES files with invalid naming${NC}"
    exit 1
fi

# Check for order prefixes
if ! ls "$EXTRACT_NAMES"/*-1-*.txt >/dev/null 2>&1; then
    echo -e "${RED}✗ No files with order prefix '1' found${NC}"
    exit 1
fi

if ! ls "$EXTRACT_NAMES"/*-h1-*.txt >/dev/null 2>&1; then
    echo -e "${RED}✗ No files with high priority prefix 'h1' found${NC}"
    exit 1
fi

if ! ls "$EXTRACT_NAMES"/*-d*.txt >/dev/null 2>&1; then
    echo -e "${RED}✗ No completed files with 'd' prefix found${NC}"
    exit 1
fi

echo -e "${GREEN}✓ File naming conventions verified${NC}"

# Test 5: Verify export contents
echo
echo "9. Verifying export contents..."

# Check db.json structure
if ! jq -e '.documents | length > 0' "$EXTRACT_DIR/db.json" >/dev/null 2>&1; then
    echo -e "${RED}✗ db.json doesn't contain documents array${NC}"
    exit 1
fi

# Check for metadata
if ! jq -e '.metadata.version' "$EXTRACT_DIR/db.json" >/dev/null 2>&1; then
    echo -e "${RED}✗ db.json missing metadata${NC}"
    exit 1
fi

# Verify a specific document
PROJECT_FILE=$(ls "$EXTRACT_DIR"/*-1-project-alpha.txt 2>/dev/null | head -1)
if [ -z "$PROJECT_FILE" ]; then
    echo -e "${RED}✗ Could not find 'Project Alpha' document file${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Export contents verified${NC}"

# Test 6: Empty export handling
echo
echo "10. Testing empty export..."
EMPTY_DB="$TEST_DIR/empty.json"
EMPTY_EXPORT="$EXPORT_DIR/empty_export.zip"
TODOS_EMPTY="./.bin/todos --store $EMPTY_DB"

# Try to export from empty store (should handle gracefully)
OUTPUT=$($TODOS_EMPTY export --output "$EMPTY_EXPORT" 2>&1)
if [[ "$OUTPUT" != *"No todos found to export"* ]]; then
    echo -e "${RED}✗ Empty export didn't handle gracefully${NC}"
    echo "Output: $OUTPUT"
    exit 1
fi
echo -e "${GREEN}✓ Empty export handled correctly${NC}"

# Summary
echo
echo "=== Export Test Summary ==="
echo -e "${GREEN}✓ All tests passed!${NC}"
echo
echo "Verified:"
echo "  - Full database export"
echo "  - Selective export by IDs"
echo "  - Temporary directory export"
echo "  - File naming conventions"
echo "  - Archive structure and contents"
echo "  - Empty database handling"
echo
echo "Export functionality is working correctly."