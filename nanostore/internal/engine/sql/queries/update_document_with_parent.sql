-- Update document fields including parent
-- Only updates fields where a value is explicitly provided
--
-- This query uses different semantics for NULL handling:
-- - For title/body: COALESCE means NULL values don't change the field
-- - For parent_uuid: We need a CASE statement because:
--   * NULL parameter (?3 IS NULL) means "don't change parent"
--   * Empty string parameter means "remove parent" (make root)
--   * Any other value means "set this as the new parent"
--
-- This allows us to distinguish between "no change" and "make root" operations
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