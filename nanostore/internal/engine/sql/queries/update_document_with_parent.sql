-- Update document fields including parent
-- Only updates fields where a value is explicitly provided
-- For parent_uuid: 
--   - NULL in query means don't change
--   - Empty string means set to NULL (make root)
--   - Non-empty string means set new parent
UPDATE documents 
SET 
    title = COALESCE(?1, title),
    body = COALESCE(?2, body),
    parent_uuid = CASE
        WHEN ?3 IS NULL THEN parent_uuid  -- No change
        WHEN ?3 = '' THEN NULL            -- Make root
        ELSE ?3                           -- Set new parent
    END,
    updated_at = strftime('%s', 'now')
WHERE uuid = ?4