-- Detects circular parent-child references using recursive CTE
-- Prevents cycles like: A -> B -> C -> A which would break hierarchical ID resolution
--
-- Algorithm:
-- 1. Start with proposed parent document
-- 2. Recursively follow parent_uuid links upward to root
-- 3. Check if the document we're trying to update appears anywhere in that chain
-- 4. Return count > 0 if circular reference would be created
--
-- Parameters: 
--   $1 - UUID of the proposed parent
--   $2 - UUID of the document being updated
--
-- Example scenario that would be detected:
-- - Document A has parent B
-- - Document B has parent C  
-- - Trying to set C's parent to A would create: A -> B -> C -> A (circular)
-- - Query would find A in C's parent chain and return count = 1
--
-- Performance: O(hierarchy_depth) which is typically 2-4 levels in practice

WITH RECURSIVE parent_chain AS (
    -- Start with the proposed parent
    SELECT uuid, parent_uuid FROM documents WHERE uuid = $1
    UNION ALL
    -- Recursively follow parent links upward
    SELECT d.uuid, d.parent_uuid 
    FROM documents d
    JOIN parent_chain pc ON d.uuid = pc.parent_uuid
)
-- Check if the document we're updating appears in the parent chain
-- If count > 0, this would create a circular reference
SELECT COUNT(*) FROM parent_chain WHERE uuid = $2