-- Update document status
-- Also updates the timestamp directly to avoid trigger issues
UPDATE documents 
SET 
    status = ?,
    updated_at = strftime('%s', 'now')
WHERE uuid = ?