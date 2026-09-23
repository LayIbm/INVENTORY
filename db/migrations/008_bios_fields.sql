-- =============================================================
-- Migration 008 — Add BIOS tracking fields to laptops
-- =============================================================
-- last_bios_update: free-text date (DD/MM/YYYY) of the last
--   BIOS password change (separate from bios_password itself)
-- bios_details: free-text notes about the BIOS status, e.g.
--   "password on record did not match", "BIOS reset required"
-- bluetooth_disabled_bios: USAA policy requires Bluetooth to be
--   disabled at the BIOS level. This tracks compliance.
--
-- All columns default to safe values so existing rows are
-- unaffected.
-- =============================================================

ALTER TABLE laptops
    ADD COLUMN IF NOT EXISTS last_bios_update         TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bios_details             TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bluetooth_disabled_bios  BOOLEAN NOT NULL DEFAULT false;
