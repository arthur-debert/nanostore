-- Recursively deletes document and all its descendants
-- This is a template that requires column name substitution
-- Parameters:
--   $1 - UUID of the root document to delete
--   Column name must be substituted via fmt.Sprintf for the hierarchical field
WITH RECURSIVE descendants AS (
    -- Start with the document to delete
    SELECT uuid FROM documents WHERE uuid = $1
    UNION ALL
    -- Recursively find all children and descendants
    SELECT d.uuid 
    FROM documents d
    INNER JOIN descendants desc ON d.%s = desc.uuid
)
DELETE FROM documents WHERE uuid IN (SELECT uuid FROM descendants)