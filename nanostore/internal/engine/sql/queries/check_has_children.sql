-- check_has_children.sql: Check if a document has any child documents
SELECT COUNT(*) > 0 as has_children
FROM documents
WHERE parent_uuid = ?;