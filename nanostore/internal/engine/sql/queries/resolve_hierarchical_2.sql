-- Resolve a 2-part hierarchical ID (e.g., "1.2" or "c1.3") in a single query
-- Parameters: root_status, root_offset, child_status, child_offset
WITH root_doc AS (
    SELECT 
        uuid,
        ROW_NUMBER() OVER (PARTITION BY status ORDER BY created_at) - 1 as row_num
    FROM documents
    WHERE parent_uuid IS NULL AND status = ?1
),
child_doc AS (
    SELECT 
        d.uuid,
        ROW_NUMBER() OVER (PARTITION BY d.status ORDER BY d.created_at) - 1 as row_num
    FROM documents d
    INNER JOIN root_doc r ON d.parent_uuid = r.uuid
    WHERE r.row_num = ?2 AND d.status = ?3
)
SELECT uuid FROM child_doc WHERE row_num = ?4