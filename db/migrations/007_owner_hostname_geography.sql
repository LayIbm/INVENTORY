-- =============================================================
-- Migration 007 — Add owner, hostname, geography to laptops + equipment
-- =============================================================
-- owner:     'IBM' | 'USAA'  — who owns the physical device
-- hostname:  machine hostname (e.g. "MXIBM-LAYSSA")
-- geography: physical location (e.g. "CIC1-B Guadalajara")
-- =============================================================

-- Laptops
ALTER TABLE laptops
    ADD COLUMN IF NOT EXISTS owner      TEXT NOT NULL DEFAULT 'IBM',
    ADD COLUMN IF NOT EXISTS hostname   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS geography  TEXT NOT NULL DEFAULT '';

-- Equipment (Desktops, Monitors, Peripherals)
ALTER TABLE equipment
    ADD COLUMN IF NOT EXISTS owner      TEXT NOT NULL DEFAULT 'IBM',
    ADD COLUMN IF NOT EXISTS hostname   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS geography  TEXT NOT NULL DEFAULT '';
