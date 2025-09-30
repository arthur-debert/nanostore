#!/bin/bash

# Build script for nanostore CLI tools
# Creates binaries in the .bin directory

set -e

# Create .bin directory if it doesn't exist
mkdir -p .bin

echo "Building nanostore CLI tools..."

# Build main nanostore CLI
echo "  Building nanostore..."
go build -o .bin/nanostore ./nanostore/cmd

# Build todos sample CLI (has separate module)
echo "  Building todos..."
(cd samples/todos && go build -o ../../.bin/todos ./cmd)

echo "✅ Build completed!"
echo "Binaries created in .bin/:"
ls -la .bin/

echo ""
echo "Usage:"
echo "  ./.bin/nanostore --help"
echo "  ./.bin/todos --help"