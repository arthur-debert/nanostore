-- Resolve a user-facing ID component to a UUID
-- Parameters: parent_uuid, status, offset (0-based)
-- For root documents, parent_uuid should be NULL
--
-- This query handles both root and child lookups with a single condition:
-- - When ?1 is NULL and we're looking for roots: (NULL = NULL OR (NULL IS NULL AND parent_uuid IS NULL))
--   The first part is false in SQL, but the second part matches root documents
-- - When ?1 has a value and we're looking for children: (parent_uuid = ?1 OR ...)
--   The first part matches children of that parent
--
-- The LIMIT 1 OFFSET pattern efficiently finds the Nth document (0-based) 
-- in the ordered result set
SELECT uuid
FROM documents
WHERE 
    (parent_uuid = ? OR (? IS NULL AND parent_uuid IS NULL))
    AND status = ?
ORDER BY created_at
LIMIT 1 OFFSET ?