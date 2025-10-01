#!/bin/bash

# Build todos if not already built
if [ ! -f todos ]; then
    go build -o todos
fi

# Remove any existing test database
rm -f test-todos.json

# Add some todos with metadata
echo "Adding todos..."
./todos -s test-todos.json add "Buy groceries"
./todos -s test-todos.json add --priority high "Complete project report"
./todos -s test-todos.json add "Call dentist" --body "Schedule annual checkup"
./todos -s test-todos.json complete 1

echo -e "\nListing todos..."
./todos -s test-todos.json list

# Export with plaintext format
echo -e "\nExporting with plaintext format..."
./todos -s test-todos.json export --format plaintext --output plaintext-export.zip

# Export with markdown format
echo -e "\nExporting with markdown format..."
./todos -s test-todos.json export --format markdown --output markdown-export.zip

# Extract and show sample files
echo -e "\nExtracting exports..."
rm -rf plaintext-export markdown-export
unzip -q plaintext-export.zip -d plaintext-export
unzip -q markdown-export.zip -d markdown-export

echo -e "\nSample plaintext export (first file):"
find plaintext-export -name "*.txt" | head -1 | xargs cat

echo -e "\n\nSample markdown export (first file):"
find markdown-export -name "*.md" | head -1 | xargs cat

# Cleanup
rm -rf plaintext-export markdown-export plaintext-export.zip markdown-export.zip test-todos.json