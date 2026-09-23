-- =============================================================
-- Device Inventory — Enlaces Module
-- =============================================================
-- Table:
--   enlaces — WAN / ISP links managed by the inventory system
-- =============================================================

CREATE TABLE IF NOT EXISTS enlaces (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type               TEXT        NOT NULL,
    company            TEXT        NOT NULL DEFAULT '',
    public_ip          TEXT        NOT NULL DEFAULT '',
    ip                 TEXT        NOT NULL DEFAULT '',
    velocity           TEXT        NOT NULL DEFAULT '',
    end_date_contract  TEXT        NOT NULL DEFAULT '',
    months_of_contract TEXT        NOT NULL DEFAULT '',
    additional_service TEXT        NOT NULL DEFAULT '',
    contract_number    TEXT        NOT NULL DEFAULT '',
    client_number      TEXT        NOT NULL DEFAULT '',
    identifier_link    TEXT        NOT NULL DEFAULT '',
    name_contact       TEXT        NOT NULL DEFAULT '',
    phone_contact      TEXT        NOT NULL DEFAULT '',
    email_contact      TEXT        NOT NULL DEFAULT '',
    support_phone      TEXT        NOT NULL DEFAULT '',
    support_clave      TEXT        NOT NULL DEFAULT '',
    current_po         TEXT        NOT NULL DEFAULT '',
    comments           TEXT        NOT NULL DEFAULT '',
    created_by         TEXT        NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER enlaces_updated_at
    BEFORE UPDATE ON enlaces
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
