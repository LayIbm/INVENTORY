-- =============================================================
-- Migration 005 — Add employee_manager_email to laptops
-- =============================================================
ALTER TABLE laptops
    ADD COLUMN IF NOT EXISTS employee_manager_email TEXT NOT NULL DEFAULT '';
