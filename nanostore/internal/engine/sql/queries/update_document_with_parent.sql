-- Update document fields including parent
-- Only updates fields where a value is explicitly provided
-- For parent_uuid: 
--   - NULL in query means don't change
--   - Empty string means set to NULL (make root)
--   - Non-empty string means set new parent
UPDATE documents 
SET 
    title = COALESCE(?, title),
    body = COALESCE(?, body),
    parent_uuid = CASE
        WHEN ?4 IS NULL THEN parent_uuid  -- No change
        WHEN ?4 = '' THEN NULL            -- Make root
        ELSE ?4                           -- Set new parent
    END,
    updated_at = strftime('%s', 'now')
WHERE uuid = ?5