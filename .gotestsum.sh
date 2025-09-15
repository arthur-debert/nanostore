#!/usr/bin/env bash
# Script to run all tests with gotestsum
# Usage: gotestsum --raw-command -- ./.gotestsum.sh

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Run nanostore and internal tests
go test ./nanostore/... ./internal/... -json

# Run example app tests if they exist
if [ -d "examples/apps/todo" ]; then
    cd examples/apps/todo
    go test ./... -json
    cd "$SCRIPT_DIR"
fi

if [ -d "examples/apps/notes" ]; then
    cd examples/apps/notes
    go test ./... -json
    cd "$SCRIPT_DIR"
fi

# Run C binding tests if test scripts exist
if [ -f "c-bindings/python/test.sh" ]; then
    # Run Python tests but suppress output for gotestsum (it expects JSON)
    if ./c-bindings/python/test.sh > /dev/null 2>&1; then
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"pass","Package":"c-bindings/python","Test":"TestPythonBindings","Elapsed":1}'
    else
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"fail","Package":"c-bindings/python","Test":"TestPythonBindings","Elapsed":1}'
    fi
fi

if [ -f "c-bindings/nodejs/test.sh" ]; then
    # Run Node.js tests but suppress output for gotestsum (it expects JSON)
    if ./c-bindings/nodejs/test.sh > /dev/null 2>&1; then
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"pass","Package":"c-bindings/nodejs","Test":"TestNodeJSBindings","Elapsed":1}'
    else
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"fail","Package":"c-bindings/nodejs","Test":"TestNodeJSBindings","Elapsed":1}'
    fi
fi