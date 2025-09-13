-- Check if setting document ?1 as parent of document ?2 would create a circular reference
-- Returns 1 if circular reference would be created, 0 otherwise
--
-- This prevents hierarchy corruption by checking if the proposed parent (?1) is actually
-- a descendant of the document being updated (?2). We do this by:
-- 1. Starting from the proposed parent
-- 2. Walking up the ancestor chain recursively
-- 3. Checking if we ever encounter the document being updated
--
-- If we find the document in the ancestor chain, it means setting the parent would
-- create a cycle: A -> B -> C -> A
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