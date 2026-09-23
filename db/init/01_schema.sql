-- =============================================================
-- Device Inventory — Database Schema
-- =============================================================
-- Tables:
--   users   — system accounts (viewer / manager / admin)
--   laptops — physical devices tracked in the inventory
--
-- Notes:
--   • Passwords are bcrypt-hashed by the Go application before insert.
--   • The 'history' column on laptops is a JSONB array of loan/event
--     entries appended by the API on create/update operations.
--   • Seeding (initial 3 users) is done by the Go app at startup,
--     NOT here, so bcrypt is applied correctly.
-- =============================================================

-- ── Extensions ───────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";  -- for gen_random_uuid()

-- ── Enums ────────────────────────────────────────────────────

-- User roles
CREATE TYPE user_role AS ENUM ('viewer', 'manager', 'admin');

-- Laptop availability
CREATE TYPE availability_status AS ENUM (
    'DISPONIBLE',
    'ASIGNADA',
    'NO_DISPONIBLE',
    'SCRAP'
);

-- Laptop preparation state
CREATE TYPE prep_status AS ENUM (
    'LISTA',
    'NECESITA_PREP',
    'NO_FUNCIONAL'
);

-- Comodato (loan agreement) state
CREATE TYPE comodato_status AS ENUM (
    'FIRMADO',
    'PENDIENTE',
    'N/A'
);

-- ── Table: users ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Full display name, e.g. "Layssa González"
    name                TEXT        NOT NULL,
    -- Login username, e.g. "layssa.g"
    username            TEXT        NOT NULL UNIQUE,
    -- bcrypt hash — never stored in plaintext
    password_hash       TEXT        NOT NULL,
    role                user_role   NOT NULL DEFAULT 'viewer',
    -- Whether the account is enabled
    active              BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Forces a password-change screen on first login
    must_change_password BOOLEAN    NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Table: laptops ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS laptops (
    id               UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Physical serial number (business key shown in UI)
    serial           TEXT               NOT NULL UNIQUE,
    model            TEXT               NOT NULL,
    -- Sub-model / part number, e.g. "20N3-S5DV14"
    variant          TEXT               NOT NULL DEFAULT '',
    brand            TEXT               NOT NULL DEFAULT 'Lenovo',
    -- Physical condition: Nuevo / Bueno / Regular / Dañado
    condition        TEXT               NOT NULL DEFAULT 'Bueno',
    availability     availability_status NOT NULL DEFAULT 'DISPONIBLE',
    prep             prep_status         NOT NULL DEFAULT 'NECESITA_PREP',
    comodato         comodato_status     NOT NULL DEFAULT 'N/A',
    -- Whether the device powers on: "OK" or "NO ENCIENDE"
    powers_on        TEXT               NOT NULL DEFAULT 'OK',
    -- Installed OS string, e.g. "Windows 11"
    os               TEXT               NOT NULL DEFAULT '',
    -- True when Windows 11 is installed and configured
    win11_ready      BOOLEAN            NOT NULL DEFAULT FALSE,
    -- BIOS password string (or "SIN CONTRASEÑA" / "DESCONOCIDA")
    bios_password    TEXT               NOT NULL DEFAULT 'SIN CONTRASEÑA',
    -- "Habilitado" / "Deshabilitado" / "Desconocido"
    wifi             TEXT               NOT NULL DEFAULT 'Deshabilitado',
    bluetooth        TEXT               NOT NULL DEFAULT 'Deshabilitado',
    charger_included BOOLEAN            NOT NULL DEFAULT TRUE,
    -- Free-text date of last OS format, e.g. "16/06/2025"
    last_format_date TEXT               NOT NULL DEFAULT '',
    -- Name of the employee the device is assigned to (null = unassigned)
    employee_name    TEXT,
    -- IBM corporate email of the assigned employee
    employee_email   TEXT               NOT NULL DEFAULT '',
    -- IBM Talent ID of the assigned employee (stored as text)
    employee_talent_id TEXT             NOT NULL DEFAULT '',
    -- IBM corporate email of the assigned employee's manager
    employee_manager_email TEXT         NOT NULL DEFAULT '',
    -- Built-in screen status: "OK" or "NO ENCIENDE"
    lcd_ok           TEXT               NOT NULL DEFAULT 'OK',
    -- Expected device return date (free-text DD/MM/YYYY, empty = no date set)
    expected_return_date TEXT           NOT NULL DEFAULT '',
    -- Physical owner of the device: 'IBM' or 'USAA'
    owner            TEXT               NOT NULL DEFAULT 'IBM',
    -- Machine hostname (e.g. "MXIBM-LAYSSA")
    hostname         TEXT               NOT NULL DEFAULT '',
    -- Physical location / geography (e.g. "CIC1-B Guadalajara")
    geography        TEXT               NOT NULL DEFAULT '',
    -- Date of last BIOS password change (free-text DD/MM/YYYY, empty = never recorded)
    last_bios_update TEXT               NOT NULL DEFAULT '',
    -- Notes about BIOS status (e.g. "password on record did not match")
    bios_details     TEXT               NOT NULL DEFAULT '',
    -- USAA policy: Bluetooth must be disabled at BIOS level
    bluetooth_disabled_bios BOOLEAN     NOT NULL DEFAULT false,
    -- EPD deployment status: 'Deployed' | 'Roll Off' | 'Transition' | 'Internal Off' | ''
    epd_status       TEXT               NOT NULL DEFAULT '',
    -- Free-text notes about the device
    notes            TEXT               NOT NULL DEFAULT '',
    -- JSONB array of HistoryEntry objects:
    -- [{ date, type, tone, employee, notes }, ...]
    history          JSONB              NOT NULL DEFAULT '[]',
    created_at       TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

-- ── Trigger: auto-update updated_at ──────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER laptops_updated_at
    BEFORE UPDATE ON laptops
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
