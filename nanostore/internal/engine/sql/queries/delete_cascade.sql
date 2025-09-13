-- delete_cascade.sql: Delete a document and all its descendants
-- Uses a recursive CTE to find all descendants before deletion
WITH RECURSIVE descendants AS (
    -- Start with the target document
    SELECT uuid FROM documents WHERE uuid = ?
    
    UNION ALL
    
    -- Recursively find all children
    SELECT d.uuid 
    FROM documents d
    INNER JOIN descendants desc ON d.parent_uuid = desc.uuid
)
DELETE FROM documents
WHERE uuid IN (SELECT uuid FROM descendants);