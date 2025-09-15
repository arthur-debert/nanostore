#!/usr/bin/env bash
# Script to run all tests with gotestsum
# Usage: gotestsum --raw-command -- ./.gotestsum.sh

set -e

# Run nanostore tests
go test ./... -json

# Run todo app tests
cd examples/apps/todo
go test ./... -json

# Run notes app tests
cd ../notes
go test ./... -json

# Run Python binding tests if available
cd ../../c-bindings/python
if command -v python3 &> /dev/null; then
    # Run Python tests but suppress output for gotestsum (it expects JSON)
    python3 tests/test_nanostore.py > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"pass","Package":"examples/c-bindings/python","Test":"TestPythonBindings","Elapsed":1}'
    else
        echo '{"Time":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","Action":"fail","Package":"examples/c-bindings/python","Test":"TestPythonBindings","Elapsed":1}'
    fi
fi