-- Update document fields
-- Only updates non-null fields
UPDATE documents 
SET 
    title = COALESCE(?, title),
    body = COALESCE(?, body)
WHERE uuid = ?