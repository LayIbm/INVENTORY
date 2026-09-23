-- Migration 010: add ipv6 field to laptops and equipment (Desktop only in practice)
ALTER TABLE laptops   ADD COLUMN IF NOT EXISTS ipv6 TEXT NOT NULL DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS ipv6 TEXT NOT NULL DEFAULT '';
