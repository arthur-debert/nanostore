-- delete_cascade.sql: Delete a document and all its descendants
-- Uses a recursive CTE to find all descendants before deletion
--
-- The recursive CTE builds a complete list of documents to delete by:
-- 1. Starting with the target document (base case)
-- 2. Recursively adding all documents whose parent is in the current set
-- 3. Continuing until no more children are found
--
-- SQLite's foreign key constraints ensure we can't accidentally orphan documents
-- that weren't found by the CTE (belt and suspenders approach)
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