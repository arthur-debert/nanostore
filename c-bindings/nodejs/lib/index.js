const koffi = require('koffi');
const path = require('path');
const fs = require('fs');

// Buffer sizes (matching Python bindings)
const DEFAULT_BUFFER_SIZE = 4096;
const LIST_BUFFER_SIZE = 65536;

// Try to find the shared library
function findLibrary() {
    const possiblePaths = [
        path.join(__dirname, '../../../bin/libnanostore.so'),
        path.join(__dirname, '../../../bin/libnanostore.dylib'),
        path.join(__dirname, '../../../bin/libnanostore.dll'),
        path.join(__dirname, '../../libnanostore.so'),
        path.join(__dirname, '../../libnanostore.dylib'),
        path.join(__dirname, '../../libnanostore.dll'),
    ];
    
    for (const libPath of possiblePaths) {
        if (fs.existsSync(libPath)) {
            return libPath;
        }
    }
    
    throw new Error('Could not find nanostore library. Please build it first with: scripts/build');
}

// Load the library
const lib = koffi.load(findLibrary());

// Define C function signatures matching the actual C API
const nanostore_new = lib.func('nanostore_new', 'int', ['str', 'str', 'char *', 'int']);
const nanostore_add = lib.func('nanostore_add', 'int', ['str', 'str', 'str', 'char *', 'int']);
const nanostore_list = lib.func('nanostore_list', 'int', ['str', 'str', 'char *', 'int']);
const nanostore_update = lib.func('nanostore_update', 'int', ['str', 'str', 'str', 'char *', 'int']);
const nanostore_delete = lib.func('nanostore_delete', 'int', ['str', 'str', 'int', 'char *', 'int']);
const nanostore_resolve_uuid = lib.func('nanostore_resolve_uuid', 'int', ['str', 'str', 'char *', 'int']);
const nanostore_close = lib.func('nanostore_close', 'int', ['str', 'char *', 'int']);

class NanoStore {
    constructor(dbPath = ':memory:', config = null) {
        // Default config must have at least one dimension
        const defaultConfig = {
            dimensions: [{
                name: "type",
                type: 0, // ENUMERATED
                values: ["default"]
            }]
        };
        
        const configJSON = JSON.stringify(config || defaultConfig);
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_new(dbPath, configJSON, buffer, bufferSize),
            DEFAULT_BUFFER_SIZE
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        this.handle = response.handle;
    }

    _callWithRetry(callFn, initialBufferSize = DEFAULT_BUFFER_SIZE) {
        let bufferSize = initialBufferSize;
        let buffer = Buffer.alloc(bufferSize);
        
        let result = callFn(buffer, bufferSize);
        
        if (result < 0) {
            // Buffer too small, retry with larger buffer
            bufferSize = bufferSize * 4;
            buffer = Buffer.alloc(bufferSize);
            result = callFn(buffer, bufferSize);
            
            if (result < 0) {
                throw new Error('Response too large for buffer');
            }
        }
        
        return JSON.parse(buffer.toString('utf8', 0, result));
    }

    close() {
        if (this.handle) {
            const buffer = Buffer.alloc(DEFAULT_BUFFER_SIZE);
            nanostore_close(this.handle, buffer, DEFAULT_BUFFER_SIZE);
            this.handle = null;
        }
    }

    add(title, dimensions = {}) {
        if (!this.handle) {
            throw new Error('Store is closed');
        }
        
        const dimensionsJSON = JSON.stringify(dimensions);
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_add(this.handle, title, dimensionsJSON, buffer, bufferSize)
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        return response.uuid;
    }

    list(filters = {}) {
        if (!this.handle) {
            throw new Error('Store is closed');
        }
        
        const filtersJSON = JSON.stringify(filters);
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_list(this.handle, filtersJSON, buffer, bufferSize),
            LIST_BUFFER_SIZE
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        // The response might be an array directly, not wrapped in an object
        if (Array.isArray(response)) {
            return response;
        }
        
        return response.documents || [];
    }

    update(id, updates) {
        if (!this.handle) {
            throw new Error('Store is closed');
        }
        
        const updatesJSON = JSON.stringify(updates);
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_update(this.handle, id, updatesJSON, buffer, bufferSize)
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        return true;
    }

    delete(id, cascade = false) {
        if (!this.handle) {
            throw new Error('Store is closed');
        }
        
        const cascadeInt = cascade ? 1 : 0;
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_delete(this.handle, id, cascadeInt, buffer, bufferSize)
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        return true;
    }

    resolveUUID(userFacingId) {
        if (!this.handle) {
            throw new Error('Store is closed');
        }
        
        const response = this._callWithRetry(
            (buffer, bufferSize) => nanostore_resolve_uuid(this.handle, userFacingId, buffer, bufferSize)
        );
        
        if (response.error) {
            throw new Error(response.error);
        }
        
        return response.uuid;
    }
}

// Helper class for documents (matching Python API)
class Document {
    constructor(data) {
        Object.assign(this, data);
    }
}

// Convenience functions for common configurations (matching Python API)
function todoConfig() {
    return {
        dimensions: [
            {
                name: "status",
                type: 0, // ENUMERATED
                values: ["pending", "completed"],
                prefixes: { "completed": "c" },
                default_value: "pending"
            },
            {
                name: "parent",
                type: 1, // HIERARCHICAL
                ref_field: "parent_uuid"
            }
        ]
    };
}

function exampleConfig() {
    return {
        dimensions: [
            {
                name: "category",
                type: 0, // ENUMERATED
                values: ["default", "archived"],
                prefixes: { "archived": "a" },
                default_value: "default"
            },
            {
                name: "parent",
                type: 1, // HIERARCHICAL
                ref_field: "parent_uuid"
            }
        ]
    };
}

module.exports = { NanoStore, Document, todoConfig, exampleConfig };