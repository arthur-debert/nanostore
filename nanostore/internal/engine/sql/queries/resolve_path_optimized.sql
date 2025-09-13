-- Resolve a complete hierarchical user-facing ID to a UUID in a single query
-- This query is called multiple times but with different depths
-- Parameters: 
--   For root: status, offset
--   For level 1: parent_status, parent_offset, child_status, child_offset
--   For level 2: p_status, p_offset, c1_status, c1_offset, c2_status, c2_offset
--   etc.

-- This is the optimized query for depth 1 (single ID part like "1" or "c2")
WITH numbered_roots AS (
    SELECT 
        uuid,
        ROW_NUMBER() OVER (PARTITION BY status ORDER BY created_at) - 1 as row_num
    FROM documents
    WHERE parent_uuid IS NULL AND status = ?
)
SELECT uuid FROM numbered_roots WHERE row_num = ?