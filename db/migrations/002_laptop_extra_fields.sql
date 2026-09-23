-- =============================================================
-- Migration 002 — Add employee_email, employee_talent_id,
--                 lcd_ok, and expected_return_date to laptops
-- =============================================================
-- Safe to run multiple times (ADD COLUMN IF NOT EXISTS).
-- Run against a live DB:
--   docker compose exec db psql -U $DB_USER -d $DB_NAME -f /migrations/002_laptop_extra_fields.sql

ALTER TABLE laptops
    ADD COLUMN IF NOT EXISTS employee_email        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS employee_talent_id    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lcd_ok                TEXT NOT NULL DEFAULT 'OK',
    ADD COLUMN IF NOT EXISTS expected_return_date  TEXT NOT NULL DEFAULT '';
