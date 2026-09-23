-- =============================================================
-- Migration 003 — Create equipment table
-- =============================================================
-- Safe to run multiple times (CREATE TABLE IF NOT EXISTS).
-- Requires enums from 01_schema.sql to already exist.

CREATE TABLE IF NOT EXISTS equipment (
    id                   UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    serial               TEXT               NOT NULL UNIQUE,
    device_type          TEXT               NOT NULL CHECK (device_type IN ('Desktop','Monitor','Adapter')),
    brand                TEXT               NOT NULL DEFAULT 'Lenovo',
    model                TEXT               NOT NULL,
    variant              TEXT               NOT NULL DEFAULT '',
    condition            TEXT               NOT NULL DEFAULT 'Bueno',
    availability         availability_status NOT NULL DEFAULT 'DISPONIBLE',
    assignability        prep_status         NOT NULL DEFAULT 'LISTA',
    comodato             comodato_status     NOT NULL DEFAULT 'N/A',
    powers_on            TEXT               NOT NULL DEFAULT 'OK',
    os                   TEXT               NOT NULL DEFAULT '',
    bios_password        TEXT               NOT NULL DEFAULT '',
    wifi                 TEXT               NOT NULL DEFAULT '',
    bluetooth            TEXT               NOT NULL DEFAULT '',
    charger_included     BOOLEAN            NOT NULL DEFAULT FALSE,
    last_format_date     TEXT               NOT NULL DEFAULT '',
    employee_name        TEXT,
    employee_email       TEXT               NOT NULL DEFAULT '',
    employee_talent_id   TEXT               NOT NULL DEFAULT '',
    transaction_date     TEXT               NOT NULL DEFAULT '',
    expected_return_date TEXT               NOT NULL DEFAULT '',
    monitor_included     BOOLEAN            NOT NULL DEFAULT FALSE,
    monitor_serial       TEXT               NOT NULL DEFAULT '',
    notes                TEXT               NOT NULL DEFAULT '',
    history              JSONB              NOT NULL DEFAULT '[]',
    created_at           TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'equipment_updated_at'
  ) THEN
    CREATE TRIGGER equipment_updated_at
      BEFORE UPDATE ON equipment
      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;
