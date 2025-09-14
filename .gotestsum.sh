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