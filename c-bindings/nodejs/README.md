# NanoStore Node.js Bindings

Node.js bindings for NanoStore, providing a native JavaScript interface to the high-performance document and ID store.

## Installation

First, ensure the C library is built:

```bash
cd ../..
./scripts/build
```

Then install the Node.js package dependencies:

```bash
npm install
```

## Usage

```javascript
const NanoStore = require('nanostore');

// Create a new store (in-memory or file-based)
const store = new NanoStore(':memory:');
// const store = new NanoStore('/path/to/database.db');

// Add a document
const id = store.add('users', 'user', JSON.stringify({
    name: 'John Doe',
    email: 'john@example.com'
}));

// Retrieve a document
const doc = store.get(id);
console.log(JSON.parse(doc.content));

// Update a document
store.update(id, JSON.stringify({
    name: 'John Doe',
    email: 'john.doe@example.com',
    updated: true
}));

// Delete a document
store.delete(id);

// List documents
const users = store.list('users');

// Work with hierarchical documents
const projectId = store.add('projects', 'project', '{"name": "My Project"}');
const taskId = store.add('tasks', 'task', '{"title": "Task 1"}', projectId);

// Close the store when done
store.close();
```

## API Reference

### `new NanoStore(dbPath)`
Creates a new NanoStore instance.
- `dbPath`: Path to the database file or `:memory:` for in-memory storage

### `store.add(collection, docType, content, parent?)`
Adds a new document to the store.
- `collection`: Collection name
- `docType`: Document type
- `content`: JSON string of document content
- `parent`: Optional parent document ID for hierarchical documents
- Returns: Generated document ID

### `store.get(id)`
Retrieves a document by ID.
- `id`: Document ID
- Returns: Document object with `id`, `uuid`, `collection`, `doc_type`, `content`, etc.

### `store.update(id, content)`
Updates an existing document.
- `id`: Document ID
- `content`: New JSON content
- Returns: `true` on success

### `store.delete(id)`
Deletes a document.
- `id`: Document ID
- Returns: `true` on success

### `store.list(collection?, docType?, limit?, offset?)`
Lists documents with optional filtering.
- `collection`: Filter by collection (optional)
- `docType`: Filter by document type (optional)
- `limit`: Maximum number of results (default: 100)
- `offset`: Skip this many results (default: 0)
- Returns: Array of document objects

### `store.resolve(uuid)`
Resolves a UUID to its user-friendly ID.
- `uuid`: Document UUID
- Returns: User-friendly ID

### `store.close()`
Closes the database connection. Always call this when done.

## Running Tests

```bash
npm test
```

## Running Example

```bash
npm run example
```

## Requirements

- Node.js >= 16.0.0
- NanoStore C library built and available

The bindings use `koffi`, a modern FFI library that works with all current Node.js versions including v24+.

## License

MIT