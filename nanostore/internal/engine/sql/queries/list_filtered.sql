-- Simple filtered list without hierarchical numbering
-- This is used when filters would break the tree structure
--
-- When filtering by status or search terms, we can't maintain proper hierarchical
-- IDs because we might be excluding parent documents. For example, if we filter
-- for completed items only, a completed child might have a pending parent that's
-- filtered out. In such cases, we fall back to simple sequential numbering
-- within the filtered result set.
--
-- The WHERE_CLAUSE placeholder is replaced with filter conditions at runtime
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