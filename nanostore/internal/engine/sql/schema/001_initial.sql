-- Initial schema for document store
CREATE TABLE IF NOT EXISTS documents (
    uuid TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    parent_uuid TEXT,
    created_at INTEGER NOT NULL,  -- Unix timestamp for consistent ordering
    updated_at INTEGER NOT NULL,  -- Unix timestamp, updated on modifications
    FOREIGN KEY (parent_uuid) REFERENCES documents(uuid) ON DELETE CASCADE
);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);