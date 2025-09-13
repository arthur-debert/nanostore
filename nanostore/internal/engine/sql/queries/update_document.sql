-- Update document fields
-- Only updates non-null fields
-- Also updates the timestamp directly to avoid trigger issues
UPDATE documents 
SET 
    title = COALESCE(?, title),
    body = COALESCE(?, body),
    updated_at = strftime('%s', 'now')
WHERE uuid = ?