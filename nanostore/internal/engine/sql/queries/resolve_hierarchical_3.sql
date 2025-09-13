-- Resolve a 3-part hierarchical ID (e.g., "1.2.c1") in a single query
-- Parameters: root_status, root_offset, child1_status, child1_offset, child2_status, child2_offset
--
-- This query efficiently resolves deep hierarchical IDs by chaining CTEs that progressively
-- narrow down the search space. Each CTE:
-- 1. Finds documents at a specific level (root, child1, child2)
-- 2. Uses ROW_NUMBER() to match the positional offset within that level
-- 3. Filters by parent relationship from the previous level
--
-- The -1 offset converts from 1-based user IDs to 0-based row numbers
WITH root_doc AS (
    SELECT 
        uuid,
        ROW_NUMBER() OVER (PARTITION BY status ORDER BY created_at) - 1 as row_num
    FROM documents
    WHERE parent_uuid IS NULL AND status = ?1
),
child1_doc AS (
    SELECT 
        d.uuid,
        ROW_NUMBER() OVER (PARTITION BY d.status ORDER BY d.created_at) - 1 as row_num
    FROM documents d
    INNER JOIN root_doc r ON d.parent_uuid = r.uuid
    WHERE r.row_num = ?2 AND d.status = ?3
),
child2_doc AS (
    SELECT 
        d.uuid,
        ROW_NUMBER() OVER (PARTITION BY d.status ORDER BY d.created_at) - 1 as row_num
    FROM documents d
    INNER JOIN child1_doc c1 ON d.parent_uuid = c1.uuid
    WHERE c1.row_num = ?4 AND d.status = ?5
)
SELECT uuid FROM child2_doc WHERE row_num = ?6