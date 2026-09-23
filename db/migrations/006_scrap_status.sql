-- =============================================================
-- Migration 006 — Add SCRAP to availability_status enum
-- =============================================================
-- SCRAP marks devices that are beyond repair / written off.
-- =============================================================

ALTER TYPE availability_status ADD VALUE IF NOT EXISTS 'SCRAP';
