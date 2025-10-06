#!/usr/bin/env bash
set -euo pipefail

# Simple test to verify CLI works
source .envrc

echo "Testing nanostore CLI..."

# Test 1: Create a note
echo "Creating note..."
nanostore-cli --type "Note" --db "/tmp/test-notes.db" --format "json" create "Test note" \
    --category "personal" \
    --content "This is a test note"

echo -e "\n✓ CLI test completed"