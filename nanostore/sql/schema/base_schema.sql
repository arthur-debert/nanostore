-- Base documents table with core columns
-- This table is created for all nanostore configurations before dimension-specific columns are added
--
-- Design decisions:
-- - uuid: Stable internal identifier, never shown to users
-- - title/body: User-visible content fields  
-- - created_at/updated_at: Unix timestamps for ordering and ROW_NUMBER() consistency
-- - Additional dimension columns (status, priority, parent_uuid, etc.) are added dynamically
--
-- Index strategy:
-- - created_at index: Critical for ROW_NUMBER() OVER (ORDER BY created_at) performance
-- - updated_at index: Used for last-modified queries and change tracking
-- - Dimension-specific indexes are added by schema_builder.go based on configuration

CREATE TABLE IF NOT EXISTS documents (
    uuid TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Standard indexes for ROW_NUMBER() performance
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_documents_updated_at ON documents(updated_at);