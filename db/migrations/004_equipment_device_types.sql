-- =============================================================
-- Migration 004 — Extend equipment device_type CHECK constraint
-- =============================================================
-- Adds Mouse, Keyboard, Headset, Cable to the allowed device types.
-- Existing rows (Desktop, Monitor, Adapter) are unaffected.
-- =============================================================

ALTER TABLE equipment
    DROP CONSTRAINT IF EXISTS equipment_device_type_check;

ALTER TABLE equipment
    ADD CONSTRAINT equipment_device_type_check
    CHECK (device_type IN ('Desktop', 'Monitor', 'Adapter', 'Mouse', 'Keyboard', 'Headset', 'Cable'));
