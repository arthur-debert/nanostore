-- Resolve a user-facing ID component to a UUID
-- Parameters: parent_uuid, status, offset (0-based)
-- For root documents, parent_uuid should be NULL
SELECT uuid
FROM documents
WHERE 
    (parent_uuid = ? OR (? IS NULL AND parent_uuid IS NULL))
    AND status = ?
ORDER BY created_at
LIMIT 1 OFFSET ?