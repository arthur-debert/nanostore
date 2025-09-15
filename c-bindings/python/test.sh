#!/bin/bash
set -euo pipefail

# Test Python bindings for nanostore
echo "Testing Python bindings..."

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "∅ Python3 not found, skipping Python tests"
    exit 0
fi

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if the C library exists
if [ ! -f "../libnanostore.so" ] && [ ! -f "../../bin/libnanostore.so" ]; then
    echo "∅ C library not found. Build it first with: scripts/build"
    exit 1
fi

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment and install dependencies
source venv/bin/activate

# Install the package in development mode with test dependencies
echo "Installing nanostore Python package with test dependencies..."
pip install -e ".[test]" > /dev/null 2>&1

# Run tests
echo "Running Python tests..."
if python -m pytest tests/ -v; then
    echo "✓ Python tests passed"
    deactivate
    exit 0
else
    echo "∅ Python tests failed"
    deactivate
    exit 1
fi