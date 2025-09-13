-- Update document status
UPDATE documents 
SET status = ?
WHERE uuid = ?