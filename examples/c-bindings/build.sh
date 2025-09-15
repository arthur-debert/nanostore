#!/bin/bash

# Build C library from Go code
echo "Building C library..."

# Build as C shared library
go build -buildmode=c-shared -o libnanostore.so main.go

# Also build static library for distribution
go build -buildmode=c-archive -o libnanostore.a main.go

echo "Built:"
echo "  - libnanostore.so (shared library)"
echo "  - libnanostore.a (static library)"
echo "  - libnanostore.h (C header)"