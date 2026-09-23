-- =============================================================
-- Device Inventory — Equipment Table
-- =============================================================
-- Tracks Desktop/TinyPC, Monitor, and Adapter devices.
-- Reuses enums from 01_schema.sql.
-- =============================================================

CREATE TABLE IF NOT EXISTS equipment (
    id                   UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Physical serial number (business key)
    serial               TEXT               NOT NULL UNIQUE,
    -- Device type: Desktop, Monitor, Adapter
    device_type          TEXT               NOT NULL CHECK (device_type IN ('Desktop','Monitor','Adapter','Mouse','Keyboard','Headset','Cable','Switch','Router','Firewall','AccessPoint','License','OtherNetwork')),
    brand                TEXT               NOT NULL DEFAULT 'Lenovo',
    model                TEXT               NOT NULL,
    -- Sub-model / part number
    variant              TEXT               NOT NULL DEFAULT '',
    -- Physical condition
    condition            TEXT               NOT NULL DEFAULT 'Bueno',
    availability         availability_status NOT NULL DEFAULT 'DISPONIBLE',
    -- Whether the device is ready to assign
    assignability        prep_status         NOT NULL DEFAULT 'LISTA',
    comodato             comodato_status     NOT NULL DEFAULT 'N/A',

    -- Technical fields (Desktop only; empty string for Monitor/Adapter)
    powers_on            TEXT               NOT NULL DEFAULT 'OK',
    os                   TEXT               NOT NULL DEFAULT '',
    bios_password        TEXT               NOT NULL DEFAULT '',
    wifi                 TEXT               NOT NULL DEFAULT '',
    bluetooth            TEXT               NOT NULL DEFAULT '',
    charger_included     BOOLEAN            NOT NULL DEFAULT FALSE,
    last_format_date     TEXT               NOT NULL DEFAULT '',

    -- Assignment
    employee_name        TEXT,
    employee_email       TEXT               NOT NULL DEFAULT '',
    employee_talent_id   TEXT               NOT NULL DEFAULT '',
    transaction_date     TEXT               NOT NULL DEFAULT '',
    expected_return_date TEXT               NOT NULL DEFAULT '',

    -- Monitor pairing (Desktop only)
    monitor_included     BOOLEAN            NOT NULL DEFAULT FALSE,
    monitor_serial       TEXT               NOT NULL DEFAULT '',

    -- Ownership and location
    owner                TEXT               NOT NULL DEFAULT 'IBM',
    hostname             TEXT               NOT NULL DEFAULT '',
    geography            TEXT               NOT NULL DEFAULT '',

    -- Notes and audit
    notes                TEXT               NOT NULL DEFAULT '',
    history              JSONB              NOT NULL DEFAULT '[]',

    created_at           TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

CREATE TRIGGER equipment_updated_at
    BEFORE UPDATE ON equipment
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
