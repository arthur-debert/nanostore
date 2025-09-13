-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_parent ON documents(parent_uuid);
CREATE INDEX IF NOT EXISTS idx_documents_created ON documents(created_at);

-- Note: We handle updated_at timestamps directly in UPDATE queries
-- to avoid potential issues with triggers and foreign key constraints
-- in edge cases (e.g., corrupted data).