-- Recursively deletes document and all its descendants
-- Parameters:
--   $1 - UUID of the root document to delete
--   %s - Hierarchical reference field name (e.g., parent_uuid)
WITH RECURSIVE descendants AS (
    -- Start with the document to delete
    SELECT uuid FROM documents WHERE uuid = $1
    UNION ALL
    -- Recursively find all children and descendants
    SELECT d.uuid 
    FROM documents d
    INNER JOIN descendants desc ON d.%s = desc.uuid
)
DELETE FROM documents WHERE uuid IN (SELECT uuid FROM descendants)