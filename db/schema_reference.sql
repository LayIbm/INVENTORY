-- =============================================================
-- DeviceInventory — Schema de referencia completo
-- =============================================================
-- Este archivo refleja el estado FINAL de la base de datos,
-- incorporando el esquema inicial (db/init/) y todas las
-- migraciones (db/migrations/) hasta la 011.
--
-- NO usar este archivo para inicializar la DB — usar db/init/.
-- Su propósito es documentación y referencia rápida.
--
-- Última actualización: refleja hasta migration 011_laptop_usage
-- =============================================================

-- ── Extensiones ──────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";  -- gen_random_uuid()

-- =============================================================
-- ENUMS
-- =============================================================

-- Roles de usuario
CREATE TYPE user_role AS ENUM (
    'viewer',    -- Solo lectura
    'manager',   -- Lectura + crear/editar/eliminar + importar/exportar
    'admin'      -- Todo lo anterior + gestión de usuarios
);

-- Estado de disponibilidad de dispositivos
CREATE TYPE availability_status AS ENUM (
    'DISPONIBLE',    -- En stock, sin asignar
    'ASIGNADA',      -- Asignada a un empleado
    'NO_DISPONIBLE', -- Fuera de servicio temporal
    'SCRAP'          -- Dado de baja / sin reparación posible
);

-- Estado de preparación de laptops
CREATE TYPE prep_status AS ENUM (
    'LISTA',          -- Lista para asignar
    'NECESITA_PREP',  -- Necesita preparación antes de asignar
    'NO_FUNCIONAL'    -- No funciona
);

-- Estado de comodato (acuerdo de préstamo)
CREATE TYPE comodato_status AS ENUM (
    'FIRMADO',   -- Comodato firmado por el empleado
    'PENDIENTE', -- Pendiente de firma
    'N/A'        -- No aplica
);

-- Estado de factura
CREATE TYPE invoice_status AS ENUM (
    'PENDIENTE',  -- Sin pagar
    'PAGADA',     -- Liquidada
    'CANCELADA'   -- Cancelada
);

-- =============================================================
-- TABLA: users
-- =============================================================
-- Cuentas del sistema. Las contraseñas se hashean con bcrypt
-- en la capa de aplicación (Go) antes de insertarse.
-- Los 3 usuarios semilla los crea el API en cada arranque
-- si la tabla está vacía (ver internal/seed/seed.go).
-- =============================================================

CREATE TABLE IF NOT EXISTS users (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Nombre completo para mostrar, ej: "Layssa González"
    name                 TEXT        NOT NULL,
    -- Nombre de usuario para login, ej: "layssa.g"
    username             TEXT        NOT NULL UNIQUE,
    -- Hash bcrypt — nunca en texto plano
    password_hash        TEXT        NOT NULL,
    role                 user_role   NOT NULL DEFAULT 'viewer',
    -- Si la cuenta está activa
    active               BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Fuerza cambio de contraseña en el primer login
    must_change_password BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Usuarios semilla (creados por el API al arrancar):
--   admin   / role=admin   / SEED_ADMIN_PASSWORD
--   manager / role=manager / SEED_MANAGER_PASSWORD
--   user    / role=viewer  / SEED_USER_PASSWORD
--   Todos con must_change_password = TRUE

-- =============================================================
-- TABLA: laptops
-- =============================================================
-- Dispositivos laptop. Incluye todos los campos agregados
-- por migraciones 002, 005, 007, 008, 009, 010 y 011.
-- =============================================================

CREATE TABLE IF NOT EXISTS laptops (
    id                      UUID                PRIMARY KEY DEFAULT gen_random_uuid(),

    -- ── Identificación ─────────────────────────────────────
    -- Número de serie físico (clave de negocio visible en UI)
    serial                  TEXT                NOT NULL UNIQUE,
    model                   TEXT                NOT NULL,
    -- Sub-modelo / número de parte, ej: "20N3-S5DV14"
    variant                 TEXT                NOT NULL DEFAULT '',
    brand                   TEXT                NOT NULL DEFAULT 'Lenovo',

    -- ── Estado físico ──────────────────────────────────────
    -- Condición: Nuevo | Bueno | Regular | Dañado
    condition               TEXT                NOT NULL DEFAULT 'Bueno',
    availability            availability_status NOT NULL DEFAULT 'DISPONIBLE',
    prep                    prep_status         NOT NULL DEFAULT 'NECESITA_PREP',
    comodato                comodato_status     NOT NULL DEFAULT 'N/A',
    -- Si enciende: "OK" | "NO ENCIENDE"
    powers_on               TEXT                NOT NULL DEFAULT 'OK',
    -- Pantalla: "OK" | "NO ENCIENDE"
    lcd_ok                  TEXT                NOT NULL DEFAULT 'OK',  -- migration 002

    -- ── Software / Sistema operativo ───────────────────────
    -- Cadena de SO instalado, ej: "Windows 11"
    os                      TEXT                NOT NULL DEFAULT '',
    -- Windows 11 instalado y configurado
    win11_ready             BOOLEAN             NOT NULL DEFAULT FALSE,

    -- ── BIOS ───────────────────────────────────────────────
    -- Contraseña BIOS actual: cadena | "SIN CONTRASEÑA" | "DESCONOCIDA"
    bios_password           TEXT                NOT NULL DEFAULT 'SIN CONTRASEÑA',
    -- Fecha del último cambio de contraseña BIOS (libre DD/MM/YYYY, vacío = nunca)
    last_bios_update        TEXT                NOT NULL DEFAULT '',    -- migration 008
    -- Notas sobre el estado BIOS, ej: "la contraseña registrada no coincidió"
    bios_details            TEXT                NOT NULL DEFAULT '',    -- migration 008
    -- USAA policy: Bluetooth deshabilitado a nivel BIOS
    bluetooth_disabled_bios BOOLEAN             NOT NULL DEFAULT FALSE, -- migration 008

    -- ── Conectividad ───────────────────────────────────────
    -- "Habilitado" | "Deshabilitado" | "Desconocido"
    wifi                    TEXT                NOT NULL DEFAULT 'Deshabilitado',
    bluetooth               TEXT                NOT NULL DEFAULT 'Deshabilitado',
    -- IPv4 (campo implícito en el modelo) / IPv6
    ipv6                    TEXT                NOT NULL DEFAULT '',    -- migration 010

    -- ── Hardware extra ─────────────────────────────────────
    charger_included        BOOLEAN             NOT NULL DEFAULT TRUE,

    -- ── Fechas ─────────────────────────────────────────────
    -- Fecha del último formateo de SO (libre DD/MM/YYYY)
    last_format_date        TEXT                NOT NULL DEFAULT '',
    -- Fecha de devolución esperada (libre DD/MM/YYYY, vacío = sin fecha)
    expected_return_date    TEXT                NOT NULL DEFAULT '',    -- migration 002

    -- ── Asignación ─────────────────────────────────────────
    -- Nombre del empleado asignado (NULL = sin asignar)
    employee_name           TEXT,
    -- Email corporativo IBM del empleado
    employee_email          TEXT                NOT NULL DEFAULT '',    -- migration 002
    -- IBM Talent ID del empleado (texto)
    employee_talent_id      TEXT                NOT NULL DEFAULT '',    -- migration 002
    -- Email del manager del empleado
    employee_manager_email  TEXT                NOT NULL DEFAULT '',    -- migration 005

    -- ── Propiedad y ubicación ──────────────────────────────
    -- Propietario físico del dispositivo: 'IBM' | 'USAA'
    owner                   TEXT                NOT NULL DEFAULT 'IBM',    -- migration 007
    -- Hostname de la máquina, ej: "MXIBM-LAYSSA"
    hostname                TEXT                NOT NULL DEFAULT '',       -- migration 007
    -- Ubicación física, ej: "CIC1-B Guadalajara"
    geography               TEXT                NOT NULL DEFAULT '',       -- migration 007

    -- ── EPD ────────────────────────────────────────────────
    -- Estado de despliegue EPD: 'Deployed'|'Roll Off'|'Transition'|'Internal Off'|''
    epd_status              TEXT                NOT NULL DEFAULT '',       -- migration 009

    -- ── Uso / categoría ────────────────────────────────────
    -- 'Exclusive IBM' | 'IBM Client' | 'Exclusive Client' | '' (no establecido)
    usage                   TEXT                NOT NULL DEFAULT '',       -- migration 011

    -- ── Notas y auditoría ──────────────────────────────────
    notes                   TEXT                NOT NULL DEFAULT '',
    -- Array JSONB de HistoryEntry: [{date, type, tone, employee, notes}, ...]
    -- Gestionado exclusivamente por el API (Create: "Registro", Update: "Actualización")
    history                 JSONB               NOT NULL DEFAULT '[]',

    created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- =============================================================
-- TABLA: equipment
-- =============================================================
-- Dispositivos que no son laptops: Desktop, Monitor, Adapter,
-- periféricos (Mouse, Keyboard, Headset, Cable) y equipos de
-- red (Switch, Router, Firewall, AccessPoint, License, OtherNetwork).
-- Incluye campos de migraciones 004, 006, 007, 010.
-- =============================================================

CREATE TABLE IF NOT EXISTS equipment (
    id                       UUID                PRIMARY KEY DEFAULT gen_random_uuid(),

    -- ── Identificación ─────────────────────────────────────
    -- Número de serie físico (clave de negocio)
    serial                   TEXT                NOT NULL UNIQUE,
    -- Tipo de dispositivo (constraint actualizado por migration 004 + 006):
    -- Desktop | Monitor | Adapter | Mouse | Keyboard | Headset | Cable
    -- Switch | Router | Firewall | AccessPoint | License | OtherNetwork
    device_type              TEXT                NOT NULL CHECK (device_type IN (
                                 'Desktop','Monitor','Adapter',
                                 'Mouse','Keyboard','Headset','Cable',
                                 'Switch','Router','Firewall','AccessPoint','License','OtherNetwork'
                             )),
    brand                    TEXT                NOT NULL DEFAULT 'Lenovo',
    model                    TEXT                NOT NULL,
    -- Sub-modelo / número de parte
    variant                  TEXT                NOT NULL DEFAULT '',

    -- ── Estado físico ──────────────────────────────────────
    condition                TEXT                NOT NULL DEFAULT 'Bueno',
    availability             availability_status NOT NULL DEFAULT 'DISPONIBLE',
    -- Si está listo para asignar
    assignability            prep_status         NOT NULL DEFAULT 'LISTA',
    comodato                 comodato_status     NOT NULL DEFAULT 'N/A',

    -- ── Técnico (solo Desktop; vacío para los demás) ────────
    powers_on                TEXT                NOT NULL DEFAULT 'OK',
    os                       TEXT                NOT NULL DEFAULT '',
    bios_password            TEXT                NOT NULL DEFAULT '',
    wifi                     TEXT                NOT NULL DEFAULT '',
    bluetooth                TEXT                NOT NULL DEFAULT '',
    charger_included         BOOLEAN             NOT NULL DEFAULT FALSE,
    last_format_date         TEXT                NOT NULL DEFAULT '',
    ipv6                     TEXT                NOT NULL DEFAULT '',   -- migration 010

    -- ── Asignación ─────────────────────────────────────────
    -- NULL = sin asignar
    employee_name            TEXT,
    employee_email           TEXT                NOT NULL DEFAULT '',
    employee_talent_id       TEXT                NOT NULL DEFAULT '',
    transaction_date         TEXT                NOT NULL DEFAULT '',
    expected_return_date     TEXT                NOT NULL DEFAULT '',

    -- ── Emparejamiento con monitor (solo Desktop) ──────────
    monitor_included         BOOLEAN             NOT NULL DEFAULT FALSE,
    monitor_serial           TEXT                NOT NULL DEFAULT '',

    -- ── Propiedad y ubicación ──────────────────────────────
    owner                    TEXT                NOT NULL DEFAULT 'IBM',    -- migration 007
    hostname                 TEXT                NOT NULL DEFAULT '',       -- migration 007
    geography                TEXT                NOT NULL DEFAULT '',       -- migration 007

    -- ── Red: identificación (migration 006) ────────────────
    -- Número de producto / part number del fabricante
    product_id               TEXT                DEFAULT '',
    -- Tipo de red: '' | 'N/A' | 'Primary' | 'Secondary'
    net_type                 TEXT                DEFAULT '',

    -- ── Red: ciclo de vida (migration 006) ─────────────────
    -- Nota: estas tres columnas usan tipo DATE (no TEXT como el resto)
    end_of_sale              DATE,
    end_of_life              DATE,
    end_contract_support     DATE,

    -- ── Red: costos (migration 006) ────────────────────────
    -- NUMERIC(15,2) para aritmética exacta sin redondeo flotante
    device_cost              NUMERIC(15,2),
    contract_cost            NUMERIC(15,2),

    -- ── Red: ubicación física (migration 006) ──────────────
    -- Sala: '' | 'ODC' | 'ODC2' | 'ODC2-AT&T'
    room                     TEXT                DEFAULT '',

    -- ── Red: estados (migration 006) ───────────────────────
    eol_status               TEXT                DEFAULT '',
    contract_status          TEXT                DEFAULT '',

    -- ── Red: contactos de soporte (migration 006) ──────────
    device_company           TEXT                DEFAULT '',
    contact_name             TEXT                DEFAULT '',
    contact_phone            TEXT                DEFAULT '',
    contact_email            TEXT                DEFAULT '',
    ibm_network_email_support TEXT               DEFAULT '',
    ibm_local_email_support  TEXT                DEFAULT '',

    -- ── Notas y auditoría ──────────────────────────────────
    notes                    TEXT                NOT NULL DEFAULT '',
    -- Array JSONB de HistoryEntry: [{date, type, tone, employee, notes}, ...]
    history                  JSONB               NOT NULL DEFAULT '[]',

    created_at               TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- =============================================================
-- TABLA: invoices
-- =============================================================
-- Registros de facturación gestionados por el sistema.
-- =============================================================

CREATE TABLE IF NOT EXISTS invoices (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Número / folio de factura, ej: "INV-0001"
    number     TEXT           NOT NULL UNIQUE,
    -- Cliente o destinatario
    client     TEXT           NOT NULL,
    -- Descripción corta del concepto facturado
    concept    TEXT           NOT NULL DEFAULT '',
    -- Monto (soporta decimales)
    amount     NUMERIC(12, 2) NOT NULL DEFAULT 0,
    status     invoice_status NOT NULL DEFAULT 'PENDIENTE',
    -- Fecha de emisión (libre DD/MM/YYYY)
    issue_date TEXT           NOT NULL DEFAULT '',
    -- Fecha de vencimiento (libre DD/MM/YYYY)
    due_date   TEXT           NOT NULL DEFAULT '',
    notes      TEXT           NOT NULL DEFAULT '',
    -- Usuario que creó la factura
    created_by TEXT           NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- =============================================================
-- TABLA: enlaces
-- =============================================================
-- Enlaces WAN/ISP gestionados por el sistema.
-- =============================================================

CREATE TABLE IF NOT EXISTS enlaces (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Tipo de enlace, ej: "MPLS", "Internet", "DIA"
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
    -- Identificador único del enlace (usado para el índice único parcial)
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

-- =============================================================
-- ÍNDICES ÚNICOS
-- =============================================================

-- Índice parcial: identifier_link único solo cuando no está vacío
-- (migration zz_enlaces_unique_idx.sql)
CREATE UNIQUE INDEX IF NOT EXISTS enlaces_identifier_link_unique_idx
    ON enlaces (identifier_link)
    WHERE identifier_link <> '';

-- =============================================================
-- FUNCIÓN Y TRIGGERS: auto-actualizar updated_at
-- =============================================================

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

CREATE TRIGGER equipment_updated_at
    BEFORE UPDATE ON equipment
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER enlaces_updated_at
    BEFORE UPDATE ON enlaces
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- =============================================================
-- RESUMEN RÁPIDO
-- =============================================================
--
-- Tablas:
--   users     — Cuentas del sistema (3 roles: viewer/manager/admin)
--   laptops   — Laptops (30+ campos incluyendo BIOS, EPD, usage, IPv6)
--   equipment — Equipos no-laptop (Desktop, Monitor, Periféricos, Red)
--   invoices  — Facturas
--   enlaces   — Enlaces WAN/ISP
--
-- Enums:
--   user_role           : viewer | manager | admin
--   availability_status : DISPONIBLE | ASIGNADA | NO_DISPONIBLE | SCRAP
--   prep_status         : LISTA | NECESITA_PREP | NO_FUNCIONAL
--   comodato_status     : FIRMADO | PENDIENTE | N/A
--   invoice_status      : PENDIENTE | PAGADA | CANCELADA
--
-- Notas importantes:
--   • Todas las fechas son TEXT en formato DD/MM/YYYY
--     EXCEPTO equipment.end_of_sale / end_of_life / end_contract_support
--     que son tipo DATE (migration 006).
--   • device_cost / contract_cost son NUMERIC(15,2), no TEXT.
--   • La columna history es JSONB gestionada SOLO por el API.
--   • Los UUIDs se generan con gen_random_uuid() (pgcrypto).
-- =============================================================
