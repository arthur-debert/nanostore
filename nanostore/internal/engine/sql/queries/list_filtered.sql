-- Simple filtered list without hierarchical numbering
-- This is used when filters would break the tree structure
SELECT 
    uuid,
    CASE 
        WHEN status = 'completed' THEN 
            'c' || CAST(ROW_NUMBER() OVER (
                PARTITION BY status 
                ORDER BY created_at
            ) AS TEXT)
        ELSE 
            CAST(ROW_NUMBER() OVER (
                PARTITION BY status 
                ORDER BY created_at
            ) AS TEXT)
    END as user_facing_id,
    title,
    body,
    status,
    parent_uuid,
    created_at,
    updated_at
FROM documents
WHERE 1=1 {{WHERE_CLAUSE}}
ORDER BY created_at