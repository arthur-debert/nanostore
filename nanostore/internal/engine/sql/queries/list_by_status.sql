-- List documents filtered by status with generated IDs
WITH RECURSIVE id_tree AS (
    -- Base case: root documents with status filter
    SELECT 
        d.uuid,
        d.title,
        d.body,
        d.status,
        d.parent_uuid,
        d.created_at,
        d.updated_at,
        0 as depth,
        CASE 
            WHEN d.status = 'completed' THEN 
                'c' || CAST(ROW_NUMBER() OVER (
                    PARTITION BY d.parent_uuid, d.status 
                    ORDER BY d.created_at
                ) AS TEXT)
            ELSE 
                CAST(ROW_NUMBER() OVER (
                    PARTITION BY d.parent_uuid, d.status 
                    ORDER BY d.created_at
                ) AS TEXT)
        END as user_facing_id
    FROM documents d
    WHERE d.parent_uuid IS NULL
    AND d.status = ?
    
    UNION ALL
    
    -- Recursive case: child documents
    SELECT 
        d.uuid,
        d.title,
        d.body,
        d.status,
        d.parent_uuid,
        d.created_at,
        d.updated_at,
        p.depth + 1,
        p.user_facing_id || '.' || 
        CASE 
            WHEN d.status = 'completed' THEN 
                'c' || CAST(ROW_NUMBER() OVER (
                    PARTITION BY d.parent_uuid, d.status 
                    ORDER BY d.created_at
                ) AS TEXT)
            ELSE 
                CAST(ROW_NUMBER() OVER (
                    PARTITION BY d.parent_uuid, d.status 
                    ORDER BY d.created_at
                ) AS TEXT)
        END as user_facing_id
    FROM documents d
    INNER JOIN id_tree p ON d.parent_uuid = p.uuid
    WHERE d.status = ?
)
SELECT 
    uuid,
    user_facing_id,
    title,
    body,
    status,
    parent_uuid,
    created_at,
    updated_at
FROM id_tree
ORDER BY depth, created_at