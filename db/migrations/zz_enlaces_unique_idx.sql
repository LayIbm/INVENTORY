CREATE UNIQUE INDEX IF NOT EXISTS enlaces_identifier_link_unique_idx
ON enlaces (identifier_link)
WHERE identifier_link <> '';
