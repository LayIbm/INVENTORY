-- =============================================================
-- Device Inventory — Billing Module
-- =============================================================
-- Table:
--   invoices — billing records managed by the inventory system
-- =============================================================

-- ── Enums ────────────────────────────────────────────────────

CREATE TYPE invoice_status AS ENUM (
    'PENDIENTE',
    'PAGADA',
    'CANCELADA'
);

-- ── Table: invoices ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS invoices (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Invoice number / folio, e.g. "INV-0001"
    number       TEXT           NOT NULL UNIQUE,
    -- Client or recipient name
    client       TEXT           NOT NULL,
    -- Short description of the billed concept
    concept      TEXT           NOT NULL DEFAULT '',
    -- Amount in numeric form (supports decimals)
    amount       NUMERIC(12, 2) NOT NULL DEFAULT 0,
    status       invoice_status NOT NULL DEFAULT 'PENDIENTE',
    -- Free-text date of issue, e.g. "16/06/2025"
    issue_date   TEXT           NOT NULL DEFAULT '',
    -- Free-text due date
    due_date     TEXT           NOT NULL DEFAULT '',
    -- Additional notes
    notes        TEXT           NOT NULL DEFAULT '',
    -- Username of the user who created the invoice
    created_by   TEXT           NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- ── Trigger: auto-update updated_at ──────────────────────────
CREATE TRIGGER invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
