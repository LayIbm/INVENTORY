-- =============================================================
-- Migration 006 — Add network-equipment specific fields
-- =============================================================
-- Extends the equipment table with fields required for Network
-- Infrastructure devices (Switch, Router, Firewall, AccessPoint,
-- License, OtherNetwork).
--
-- All columns are optional (NULL allowed) so that existing
-- Desktop/Monitor/Peripheral rows are unaffected.
--
-- Date fields use DATE type (no time component, no timezone
-- offset issues). Costs use NUMERIC(15,2) for exact decimal
-- arithmetic without floating-point rounding.
--
-- No existing columns are renamed or dropped.
-- =============================================================

-- 1. Extend device_type CHECK to include 'License'
ALTER TABLE equipment
    DROP CONSTRAINT IF EXISTS equipment_device_type_check;

ALTER TABLE equipment
    ADD CONSTRAINT equipment_device_type_check
    CHECK (device_type IN (
        'Desktop', 'Monitor', 'Adapter',
        'Mouse', 'Keyboard', 'Headset', 'Cable',
        'Switch', 'Router', 'Firewall', 'AccessPoint', 'License', 'OtherNetwork'
    ));

-- 2. Network identification
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS product_id        TEXT         DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS net_type          TEXT         DEFAULT '';   -- N/A | Primary | Secondary

-- 3. Lifecycle dates
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS end_of_sale       DATE;
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS end_of_life       DATE;
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS end_contract_support DATE;

-- 4. Costs  (NUMERIC for exact decimal storage — no assumed currency)
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS device_cost       NUMERIC(15,2);
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS contract_cost     NUMERIC(15,2);

-- 5. Location
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS room              TEXT         DEFAULT '';   -- ODC | ODC2 | ODC2-AT&T

-- 6. Status
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS eol_status        TEXT         DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS contract_status   TEXT         DEFAULT '';

-- 7. Provider / support contacts
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS device_company            TEXT DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS contact_name              TEXT DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS contact_phone             TEXT DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS contact_email             TEXT DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS ibm_network_email_support TEXT DEFAULT '';
ALTER TABLE equipment ADD COLUMN IF NOT EXISTS ibm_local_email_support   TEXT DEFAULT '';
