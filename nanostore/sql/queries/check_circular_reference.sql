-- Detects circular parent-child references using recursive CTE
-- Parameters: 
--   $1 - UUID of the proposed parent
--   $2 - UUID of the document being updated
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