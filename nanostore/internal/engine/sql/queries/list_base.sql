-- Base list query template with placeholders for filtering
-- Uses a multi-step approach to handle SQLite's limitations with window functions in recursive CTEs
-- 
-- This query works around SQLite's restriction that prevents window functions from being used
-- directly in recursive CTEs. We solve this by:
-- 1. Pre-calculating IDs for root documents (with window functions)
-- 2. Pre-calculating local IDs for all child documents (with window functions) 
-- 3. Using a recursive CTE to build the hierarchy without window functions
--
-- Placeholders ROOT_WHERE_CLAUSE and CHILD_WHERE_CLAUSE are replaced at runtime
-- with appropriate filter conditions
WITH RECURSIVE 
-- Step 1: Number root documents by status
root_docs AS (
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
    WHERE parent_uuid IS NULL {{ROOT_WHERE_CLAUSE}}
),
-- Step 2: Number all children by parent and status
child_docs AS (
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
                    PARTITION BY parent_uuid, status 
                    ORDER BY created_at
                ) AS TEXT)
            ELSE 
                CAST(ROW_NUMBER() OVER (
                    PARTITION BY parent_uuid, status 
                    ORDER BY created_at
                ) AS TEXT)
        END as local_id
    FROM documents
    WHERE parent_uuid IS NOT NULL {{CHILD_WHERE_CLAUSE}}
),
-- Step 3: Build the tree with full paths
id_tree AS (
    -- Base case: root documents
    SELECT 
        uuid,
        title,
        body,
        status,
        parent_uuid,
        created_at,
        updated_at,
        0 as depth,
        user_facing_id
    FROM root_docs
    
    UNION ALL
    
    -- Recursive case: children with concatenated IDs
    SELECT 
        c.uuid,
        c.title,
        c.body,
        c.status,
        c.parent_uuid,
        c.created_at,
        c.updated_at,
        p.depth + 1,
        p.user_facing_id || '.' || c.local_id as user_facing_id
    FROM child_docs c
    INNER JOIN id_tree p ON c.parent_uuid = p.uuid
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