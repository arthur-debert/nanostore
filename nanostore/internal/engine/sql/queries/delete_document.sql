-- delete_document.sql: Delete a single document by UUID
DELETE FROM documents
WHERE uuid = ?;