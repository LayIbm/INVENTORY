-- =============================================================
-- Migration 009 — Add EPD status field to laptops
-- =============================================================
-- epd_status: tracks the employee's EPD deployment state.
--   Values: 'Deployed', 'Roll Off', 'Transition', 'Internal Off', or ''
--   Sourced from the EPD list Excel (Sheet2, Status column).
-- =============================================================

ALTER TABLE laptops
    ADD COLUMN IF NOT EXISTS epd_status TEXT NOT NULL DEFAULT '';
