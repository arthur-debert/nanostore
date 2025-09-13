-- Check if setting document ?1 as parent of document ?2 would create a circular reference
-- Returns 1 if circular reference would be created, 0 otherwise
WITH RECURSIVE ancestors AS (
    -- Start with the proposed new parent
    SELECT uuid, parent_uuid FROM documents WHERE uuid = ?1
    
    UNION ALL
    
    -- Recursively find all ancestors
    SELECT d.uuid, d.parent_uuid 
    FROM documents d
    INNER JOIN ancestors a ON d.uuid = a.parent_uuid
)
SELECT COUNT(*) > 0 as would_be_circular
FROM ancestors
WHERE uuid = ?2