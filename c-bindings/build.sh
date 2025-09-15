#!/bin/bash
set -euo pipefail

# Build C library from Go code
# This script can be run standalone or called from the main build script

# Allow output directory to be specified, default to current directory
OUTPUT_DIR="${1:-.}"

echo "Building C library..."

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Build shared library
if go build -buildmode=c-shared -o "$OUTPUT_DIR/libnanostore.so" main.go; then
    echo "✓ Built shared library: $OUTPUT_DIR/libnanostore.so"
else
    echo "∅ Failed to build shared library"
    exit 1
fi

# Build static library
if go build -buildmode=c-archive -o "$OUTPUT_DIR/libnanostore.a" main.go; then
    echo "✓ Built static library: $OUTPUT_DIR/libnanostore.a"
else
    echo "∅ Failed to build static library"
    exit 1
fi

# Check header was generated
if [ -f "$OUTPUT_DIR/libnanostore.h" ]; then
    echo "✓ Generated header: $OUTPUT_DIR/libnanostore.h"
else
    echo "∅ Header file not generated"
    exit 1
fi

echo "C library build complete!"