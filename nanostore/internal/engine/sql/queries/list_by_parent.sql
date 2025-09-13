-- List direct children of a specific parent with correct numbering
WITH numbered_children AS (
    SELECT 
        uuid,
        title,
        body,
        status,
        parent_uuid,
        created_at,
        updated_at,
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
        END as user_facing_id
    FROM documents
    WHERE parent_uuid = ? {{ADDITIONAL_WHERE}}
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
FROM numbered_children
ORDER BY created_at