-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_parent ON documents(parent_uuid);
CREATE INDEX IF NOT EXISTS idx_documents_created ON documents(created_at);

-- Trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_timestamp 
AFTER UPDATE ON documents
FOR EACH ROW
WHEN OLD.updated_at = NEW.updated_at
BEGIN
    UPDATE documents SET updated_at = strftime('%s', 'now') WHERE uuid = NEW.uuid;
END;