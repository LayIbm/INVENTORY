-- Add usage field to laptops
-- Values: 'Exclusive IBM' | 'IBM Client' | 'Exclusive Client' | '' (empty = not set)
ALTER TABLE laptops ADD COLUMN IF NOT EXISTS usage TEXT NOT NULL DEFAULT '';
