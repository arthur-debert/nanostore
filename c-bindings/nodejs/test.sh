#!/bin/bash
set -euo pipefail

# Test Node.js bindings for nanostore
echo "Testing Node.js bindings..."

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    echo "∅ Node.js not found, skipping Node.js tests"
    exit 0
fi


# Check if npm is available
if ! command -v npm &> /dev/null; then
    echo "∅ npm not found, skipping Node.js tests"
    exit 0
fi

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if the C library exists
LIB_FOUND=false
for lib in "../../bin/libnanostore.so" "../../bin/libnanostore.dylib" "../libnanostore.so" "../libnanostore.dylib"; do
    if [ -f "$lib" ]; then
        LIB_FOUND=true
        break
    fi
done

if [ "$LIB_FOUND" = false ]; then
    echo "∅ C library not found. Build it first with: scripts/build"
    exit 1
fi

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install > /dev/null 2>&1
fi

# Run tests
echo "Running Node.js tests..."
if npm test; then
    echo "✓ Node.js tests passed"
    exit 0
else
    echo "∅ Node.js tests failed"
    exit 1
fi