# Nanostore Language Bindings

This directory contains FFI (Foreign Function Interface) bindings for nanostore, allowing it to be used from various programming languages.

## Architecture

The bindings use a C shared library interface (`main.go`) that wraps the Go nanostore API. Each language then uses its native FFI capabilities to call into this shared library.

Key design decisions:
- **Caller-managed memory**: To avoid cross-language memory management issues, the calling language allocates buffers and the C API writes into them
- **JSON data exchange**: All complex data structures are exchanged as JSON strings
- **Handle-based API**: Stores are referenced by string handles to avoid pointer management across languages

## Supported Languages

- [Python](python/) - Full support with ctypes FFI
- Node.js - Coming soon
- Ruby - Planned

## Building

The shared library is automatically built and distributed via GitHub releases. Language packages download the appropriate binary for their platform during installation.

To build manually:
```bash
# Linux/macOS
go build -buildmode=c-shared -o libnanostore.so main.go

# Windows
go build -buildmode=c-shared -o nanostore.dll main.go
```

## C API Reference

All functions use caller-managed buffers and return the number of bytes written (or -1 if buffer too small):

```c
// Create a new store
int nanostore_new(const char* dbPath, const char* configJSON, 
                  char* outBuffer, int bufferSize);

// Add a document
int nanostore_add(const char* handle, const char* title, 
                  const char* dimensionsJSON, char* outBuffer, int bufferSize);

// List documents
int nanostore_list(const char* handle, const char* filtersJSON, 
                   char* outBuffer, int bufferSize);

// Update a document
int nanostore_update(const char* handle, const char* id, 
                     const char* updatesJSON, char* outBuffer, int bufferSize);

// Delete a document
int nanostore_delete(const char* handle, const char* id, int cascade, 
                     char* outBuffer, int bufferSize);

// Resolve user-facing ID to UUID
int nanostore_resolve_uuid(const char* handle, const char* userFacingID, 
                           char* outBuffer, int bufferSize);

// Close a store
int nanostore_close(const char* handle, char* outBuffer, int bufferSize);
```

## Contributing

To add a new language binding:
1. Create a new directory with the language name
2. Implement the FFI wrapper using the C API
3. Add tests and examples
4. Update the build process to include your language
5. Submit a pull request

## License

Same as the main nanostore project.