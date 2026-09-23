import { useState } from 'react'
import { Equipment, CreateEquipmentPayload, UpdateEquipmentPayload } from '../hooks/useEquipment'
import { generateCommodatumPDF } from '../utils/generateCommodatum'

type Mode = 'view' | 'edit' | 'create'

interface Props {
  mode: Mode
  equipment: Equipment | null
  role: string
  /** 'network' renders the network-specific form; defaults to the standard form. */
  context?: 'network' | 'standard'
  /** Restrict the Device type dropdown to these types only (create + edit). */
  allowedTypes?: string[]
  onClose: () => void
  onCreate: (payload: CreateEquipmentPayload) => Promise<void>
  onUpdate: (id: string, payload: UpdateEquipmentPayload) => Promise<void>
  onDelete: (e: Equipment) => void
}

const toneColor: Record<string, string> = {
  ok: 'var(--cds-support-success)',
  warn: 'var(--cds-support-warning)',
  danger: 'var(--cds-support-error)',
  neutral: 'var(--cds-text-secondary)',
}

function todayStr() {
  const d = new Date()
  return `${String(d.getDate()).padStart(2,'0')}/${String(d.getMonth()+1).padStart(2,'0')}/${d.getFullYear()}`
}

const emptyForm: CreateEquipmentPayload = {
  serial: '', device_type: 'Desktop', brand: 'Lenovo', model: '', variant: '',
  condition: 'Good', availability: 'DISPONIBLE', assignability: 'LISTA', comodato: 'N/A',
  powers_on: 'OK', os: '', bios_password: '', wifi: 'Enabled', bluetooth: 'Disabled',
  charger_included: false, last_format_date: todayStr(),
  employee_name: null, employee_email: '', employee_talent_id: '',
  transaction_date: '', expected_return_date: '',
  monitor_included: false, monitor_serial: '',
  owner: 'IBM', hostname: '', geography: '',
  notes: '',
  // network defaults
  product_id: '', net_type: 'N/A',
  end_of_sale: null, end_of_life: null, end_contract_support: null,
  device_cost: null, contract_cost: null,
  room: '', eol_status: '', contract_status: '',
  device_company: '', contact_name: '', contact_phone: '', contact_email: '',
  ibm_network_email_support: '', ibm_local_email_support: '',
}

const emptyNetworkForm: CreateEquipmentPayload = {
  ...emptyForm,
  device_type: 'Switch',
  brand: '',
}

function equipmentToForm(e: Equipment): CreateEquipmentPayload {
  return {
    serial: e.serial, device_type: e.device_type, brand: e.brand,
    model: e.model, variant: e.variant, condition: e.condition,
    availability: e.availability, assignability: e.assignability, comodato: e.comodato,
    powers_on: e.powers_on, os: e.os, bios_password: e.bios_password,
    wifi: e.wifi, bluetooth: e.bluetooth, charger_included: e.charger_included,
    last_format_date: e.last_format_date, employee_name: e.employee_name,
    employee_email: e.employee_email, employee_talent_id: e.employee_talent_id,
    transaction_date: e.transaction_date, expected_return_date: e.expected_return_date,
    monitor_included: e.monitor_included, monitor_serial: e.monitor_serial,
    owner: e.owner, hostname: e.hostname, geography: e.geography,
    notes: e.notes,
    // network
    product_id: e.product_id, net_type: e.net_type,
    end_of_sale: e.end_of_sale, end_of_life: e.end_of_life,
    end_contract_support: e.end_contract_support,
    device_cost: e.device_cost, contract_cost: e.contract_cost,
    room: e.room, eol_status: e.eol_status, contract_status: e.contract_status,
    device_company: e.device_company, contact_name: e.contact_name,
    contact_phone: e.contact_phone, contact_email: e.contact_email,
    ibm_network_email_support: e.ibm_network_email_support,
    ibm_local_email_support: e.ibm_local_email_support,
  }
}

const deviceTypeLabel: Record<string, string> = {
  Desktop: 'Desktop / TinyPC', Monitor: 'Monitor', Adapter: 'Adapter',
  Mouse: 'Mouse', Keyboard: 'Keyboard', Headset: 'Headset', Cable: 'Cable',
  Switch: 'Switch', Router: 'Router', Firewall: 'Firewall',
  AccessPoint: 'Access Point', License: 'License', OtherNetwork: 'Other network device',
}

// ── Network-specific type of device options ───────────────────────────────────
const NETWORK_DEVICE_TYPES = ['Switch', 'Router', 'Firewall', 'AccessPoint', 'License', 'OtherNetwork'] as const
const NETWORK_DEVICE_LABELS = ['Switch', 'Router', 'Firewall', 'Access Point', 'License', 'Other network device']

// ── Standard (non-network) device type options ────────────────────────────────
const STANDARD_DEVICE_TYPES = ['Desktop', 'Monitor', 'Adapter', 'Mouse', 'Keyboard', 'Headset', 'Cable', 'Switch', 'Router', 'Firewall', 'AccessPoint', 'License', 'OtherNetwork']
const STANDARD_DEVICE_LABELS = ['Desktop / TinyPC', 'Monitor', 'Adapter', 'Mouse', 'Keyboard', 'Headset', 'Cable', 'Switch', 'Router', 'Firewall', 'Access Point', 'License', 'Other network device']

// Simple email validation regex — allows corporate addresses
function isValidEmail(v: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)
}

export default function EquipmentModal({ mode, equipment: eq, role, context = 'standard', allowedTypes, onClose, onCreate, onUpdate, onDelete }: Props) {
  const canWrite = role === 'manager' || role === 'admin'
  const isNetwork = context === 'network'

  // Build filtered device-type options (respects allowedTypes when provided)
  const filteredTypeIndices = allowedTypes
    ? STANDARD_DEVICE_TYPES.map((t, i) => ({ t, i })).filter(({ t }) => allowedTypes.includes(t))
    : STANDARD_DEVICE_TYPES.map((t, i) => ({ t, i }))
  const filteredTypes  = filteredTypeIndices.map(({ t }) => t)
  const filteredLabels = filteredTypeIndices.map(({ i }) => STANDARD_DEVICE_LABELS[i])
  const [activeTab, setActiveTab] = useState<'tech' | 'loans' | 'log' | 'bios'>('tech')
  const [localMode, setLocalMode] = useState<Mode>(mode)
  const [form, setForm] = useState<CreateEquipmentPayload>(() => {
    if (mode !== 'create') return equipmentToForm(eq!)
    const base = isNetwork ? emptyNetworkForm : emptyForm
    // If allowedTypes is provided and the default type is not in the list,
    // pick the first allowed type so the dropdown is never out of sync.
    if (allowedTypes && allowedTypes.length > 0 && !allowedTypes.includes(base.device_type)) {
      return { ...base, device_type: allowedTypes[0] as CreateEquipmentPayload['device_type'] }
    }
    return base
  })
  const [emailErrors, setEmailErrors] = useState<Record<string, string>>({})

  function set<K extends keyof CreateEquipmentPayload>(k: K, v: CreateEquipmentPayload[K]) {
    setForm(prev => ({ ...prev, [k]: v }))
  }

  const isCreate = localMode === 'create'
  const isView   = localMode === 'view'
  const isEdit   = localMode === 'edit'

  // For desktop-only fields we check the currently-selected device type in form
  const effectiveDeviceType = (isCreate || isEdit) ? form.device_type : (eq?.device_type ?? '')
  const isDesktop = effectiveDeviceType === 'Desktop'

  function validateEmails(): boolean {
    const errs: Record<string, string> = {}
    if (form.contact_email && !isValidEmail(form.contact_email)) {
      errs['contact_email'] = 'Invalid email'
    }
    if (form.ibm_network_email_support && !isValidEmail(form.ibm_network_email_support)) {
      errs['ibm_network_email_support'] = 'Invalid email'
    }
    if (form.ibm_local_email_support && !isValidEmail(form.ibm_local_email_support)) {
      errs['ibm_local_email_support'] = 'Invalid email'
    }
    setEmailErrors(errs)
    return Object.keys(errs).length === 0
  }

  async function handleSave() {
    if (isNetwork && !validateEmails()) return
    if (isCreate) {
      await onCreate({ ...form, serial: form.serial.toUpperCase().trim() })
    } else if (eq) {
      const { serial: _s, ...rest } = form
      void _s
      await onUpdate(eq.id, rest as UpdateEquipmentPayload)
    }
    onClose()
  }

  const availBadge = (e: Equipment) => {
    if (e.availability === 'DISPONIBLE') return <span className="badge ok">Available</span>
    if (e.availability === 'ASIGNADA')   return <span className="badge neutral">Assigned</span>
    return <span className="badge danger">Unavailable</span>
  }
  const assignBadge = (e: Equipment) => {
    if (e.assignability === 'LISTA')         return <span className="badge ok">Ready</span>
    if (e.assignability === 'NECESITA_PREP') return <span className="badge warn">Needs prep.</span>
    return <span className="badge danger">Non-functional</span>
  }
  const comodatoBadge = (e: Equipment) => {
    if (e.comodato === 'FIRMADO')   return <span className="badge ok">Signed</span>
    if (e.comodato === 'PENDIENTE') return <span className="badge warn">Pending</span>
    return <span className="badge neutral">N/A</span>
  }

  const titleBadge = eq
    ? <span className={`badge ${eq.availability === 'DISPONIBLE' ? 'ok' : eq.availability === 'ASIGNADA' ? 'neutral' : 'danger'}`}>
        {eq.availability === 'DISPONIBLE' ? 'Available' : eq.availability === 'ASIGNADA' ? 'Assigned' : 'Unavailable'}
      </span>
    : null

  return (
    <div className="overlay" style={{ display: 'flex' }}>
      <div className="modal">
        {/* Header */}
        <div className="modal-head">
          <div className="modal-head-top">
            <div>
              <div className="modal-title-row">
                <h3>
                  {isCreate
                    ? (isNetwork ? 'Add network device' : 'Add device')
                    : `${eq?.brand && eq.brand !== 'Unknown' ? eq.brand + ' ' : ''}${eq?.model}`}
                </h3>
                {!isCreate && titleBadge}
              </div>
              <div className="modal-sub">
                {isCreate
                  ? (isNetwork ? 'Fill in the network device details.' : 'Fill in the new device details.')
                  : eq && <><span className="tag" style={{ marginRight: 8 }}>{eq.serial}</span>{deviceTypeLabel[eq.device_type] ?? eq.device_type}</>
                }
              </div>
            </div>
            <button className="modal-close" onClick={onClose}>&times;</button>
          </div>
          <div className="tabs">
            <button className={`tab${activeTab === 'tech' ? ' active' : ''}`} onClick={() => setActiveTab('tech')}>
              Technical details
            </button>
            {!isCreate && (
              <button className={`tab${activeTab === 'loans' ? ' active' : ''}`} onClick={() => setActiveTab('loans')}>
                Loans
              </button>
            )}
            {!isCreate && (
              <button className={`tab${activeTab === 'log' ? ' active' : ''}`} onClick={() => setActiveTab('log')}>
                Log
              </button>
            )}
            {!isCreate && (eq?.device_type === 'Desktop') && (
              <button className={`tab${activeTab === 'bios' ? ' active' : ''}`} onClick={() => setActiveTab('bios')}>
                BIOS Password
              </button>
            )}
          </div>
        </div>

        {/* Body */}
        <div className="modal-body">
          {activeTab === 'tech' && (
            <>
              {isNetwork
                ? <NetworkForm
                    form={form} eq={eq ?? null} isCreate={isCreate} isView={isView} isEdit={isEdit}
                    canWrite={canWrite} set={set} emailErrors={emailErrors}
                    availBadge={availBadge} assignBadge={assignBadge} comodatoBadge={comodatoBadge}
                  />
                : <StandardForm
                    form={form} eq={eq ?? null} isCreate={isCreate} isView={isView} isEdit={isEdit}
                    isDesktop={isDesktop} set={set}
                    availBadge={availBadge} assignBadge={assignBadge} comodatoBadge={comodatoBadge}
                    filteredTypes={filteredTypes} filteredLabels={filteredLabels}
                  />
              }
            </>
          )}

          {/* BIOS history tab */}
          {activeTab === 'bios' && eq && (() => {
            const biosHistory = [...eq.history]
              .reverse()
              .filter(h => h.notes && h.notes.toLowerCase().includes('bios'))

            return biosHistory.length === 0 ? (
              <div style={{ padding: '32px 0', textAlign: 'center', color: 'var(--cds-text-placeholder)', fontStyle: 'italic', fontSize: 13 }}>
                No BIOS password changes recorded.
              </div>
            ) : (
              <ul className="timeline">
                {biosHistory.map((h, i) => {
                  const match = h.notes.match(/BIOS:\s*([^\s|→]+)\s*→\s*([^\s|]+)/)
                  const from = match ? match[1] : '—'
                  const to   = match ? match[2] : '—'
                  return (
                    <li key={i}>
                      <div className="t-dot" style={{ background: 'var(--cds-blue-60)' }} />
                      <div className="t-body">
                        <div className="t-head">
                          <span className="badge neutral">BIOS change</span>
                          <span className="t-emp">{h.employee}</span>
                          <span className="t-date">{h.date}</span>
                        </div>
                        {match ? (
                          <div className="t-note" style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
                            <span style={{ fontFamily: 'var(--cds-code-01-font-family, monospace)', fontSize: 12.5, background: 'var(--cds-layer-02)', padding: '2px 7px', borderRadius: 4 }}>{from}</span>
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M5 12h14M13 6l6 6-6 6"/></svg>
                            <span style={{ fontFamily: 'var(--cds-code-01-font-family, monospace)', fontSize: 12.5, background: 'var(--cds-layer-02)', padding: '2px 7px', borderRadius: 4, fontWeight: 600 }}>{to}</span>
                          </div>
                        ) : (
                          <div className="t-note">{h.notes}</div>
                        )}
                      </div>
                    </li>
                  )
                })}
              </ul>
            )
          })()}

          {/* Loans tab */}
          {activeTab === 'loans' && eq && (() => {
            const loanHistory = [...eq.history]
              .reverse()
              .filter(h =>
                h.notes && (
                  h.notes.toLowerCase().includes('empleado:') ||
                  h.type === 'Entrega' || h.type === 'Recibo' ||
                  h.type === 'Asignada' || h.type === 'LOAN_OUT' ||
                  h.type === 'RETURN'  || h.type === 'ASSIGNED'
                )
              )
            return loanHistory.length === 0 ? (
              <div style={{ padding: '32px 0', textAlign: 'center', color: 'var(--cds-text-placeholder)', fontStyle: 'italic', fontSize: 13 }}>
                No loans or assignments recorded.
              </div>
            ) : (
              <ul className="timeline">
                {loanHistory.map((h, i) => (
                  <li key={i}>
                    <div className="t-dot" style={{ background: toneColor[h.tone] ?? toneColor.neutral }} />
                    <div className="t-body">
                      <div className="t-head">
                        <span className={`badge ${h.tone === 'ok' ? 'ok' : h.tone === 'warn' ? 'warn' : h.tone === 'danger' ? 'danger' : 'neutral'}`}>{h.type}</span>
                        <span className="t-emp">{h.employee}</span>
                        <span className="t-date">{h.date}</span>
                      </div>
                      <div className="t-note">{h.notes}</div>
                    </div>
                  </li>
                ))}
              </ul>
            )
          })()}

          {/* Log tab — all changes */}
          {activeTab === 'log' && eq && (
            <ul className="timeline">
              {[...eq.history].reverse().map((h, i) => (
                <li key={i}>
                  <div className="t-dot" style={{ background: toneColor[h.tone] ?? toneColor.neutral }} />
                  <div className="t-body">
                    <div className="t-head">
                      <span className={`badge ${h.tone === 'ok' ? 'ok' : h.tone === 'warn' ? 'warn' : h.tone === 'danger' ? 'danger' : 'neutral'}`}>{h.type}</span>
                      <span className="t-emp">{h.employee}</span>
                      <span className="t-date">{h.date}</span>
                    </div>
                    <div className="t-note">{h.notes}</div>
                  </div>
                </li>
              ))}
              {eq.history.length === 0 && (
                <li style={{ color: 'var(--cds-text-placeholder)', fontStyle: 'italic', fontSize: '0.875rem' }}>
                  No history recorded.
                </li>
              )}
            </ul>
          )}
        </div>

        {/* Footer */}
        <div className="modal-foot">
          <div className="left">
            {isView && canWrite && (
              <>
                <button className="btn btn-ghost" onClick={() => setLocalMode('edit')}>Edit</button>
                <button className="btn btn-danger-ghost" onClick={() => eq && onDelete(eq)}>Delete</button>
              </>
            )}
            {isView && eq && eq.employee_name && (
              <button
                className="btn btn-ghost"
                onClick={() => {
                  const isDesktopType = eq.device_type === 'Desktop'
                  const description = isDesktopType ? 'TINY PC' : eq.device_type.toUpperCase()
                  generateCommodatumPDF({
                    description,
                    deviceType: '-',
                    model: eq.model,
                    serial: eq.serial,
                    employeeName: eq.employee_name!,
                    employeeTalentId: eq.employee_talent_id,
                    date: new Date().toISOString().split('T')[0],
                    os: isDesktopType ? (eq.os || undefined) : undefined,
                    chargerIncluded: isDesktopType ? eq.charger_included : false,
                    monitorIncluded: isDesktopType ? eq.monitor_included : false,
                    monitorSerial: isDesktopType ? eq.monitor_serial : undefined,
                  })
                }}
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" style={{ marginRight: 4 }}>
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                Download loan agreement
              </button>
            )}
            {(isEdit || isCreate) && (
              <button className="btn btn-ghost" onClick={() => isCreate ? onClose() : setLocalMode('view')}>
                Cancel
              </button>
            )}
          </div>
          <div className="right">
            {isView && <button className="btn btn-ghost" onClick={onClose}>Close</button>}
            {(isEdit || isCreate) && (
              <button className="btn btn-primary" onClick={handleSave}>
                {isCreate ? 'Create device' : 'Save changes'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ── Standard (non-network) form ────────────────────────────────────────────────

interface StandardFormProps {
  form: CreateEquipmentPayload
  eq: Equipment | null
  isCreate: boolean
  isView: boolean
  isEdit: boolean
  isDesktop: boolean
  set: <K extends keyof CreateEquipmentPayload>(k: K, v: CreateEquipmentPayload[K]) => void
  availBadge: (e: Equipment) => React.ReactNode
  assignBadge: (e: Equipment) => React.ReactNode
  comodatoBadge: (e: Equipment) => React.ReactNode
  filteredTypes: string[]
  filteredLabels: string[]
}

function StandardForm({ form, eq, isCreate, isView, isEdit, isDesktop, set, availBadge, assignBadge, comodatoBadge, filteredTypes, filteredLabels }: StandardFormProps) {
  const deviceTypeLabel: Record<string, string> = {
    Desktop: 'Desktop / TinyPC', Monitor: 'Monitor', Adapter: 'Adapter',
    Mouse: 'Mouse', Keyboard: 'Keyboard', Headset: 'Headset', Cable: 'Cable',
    Switch: 'Switch', Router: 'Router', Firewall: 'Firewall',
    AccessPoint: 'Access Point', License: 'License', OtherNetwork: 'Other network device',
  }
  return (
    <>
      <div className="spec-grid">
        {/* ── IDENTITY (always shown in create) ── */}
        {isCreate && (
          <>
            <SpecField label="Serial number *" value={form.serial} onChange={v => set('serial', v)} placeholder="e.g. MJ4W30XR" />
            <SpecSelect label="Device type *" value={form.device_type}
              options={filteredTypes}
              labels={filteredLabels}
              onChange={v => set('device_type', v as CreateEquipmentPayload['device_type'])} />
            <SpecField label="Brand" value={form.brand} onChange={v => set('brand', v)} placeholder="e.g. Lenovo" />
            <SpecField label="Model *" value={form.model} onChange={v => set('model', v)} placeholder="e.g. M920Q" />
            <SpecField label="Variant / sub-model" value={form.variant} onChange={v => set('variant', v)} placeholder="e.g. 10MQS..." />
          </>
        )}

        {/* ── VIEW MODE ── */}
        {isView && eq && (
          <>
            <SpecView label="Type" value={deviceTypeLabel[eq.device_type] ?? eq.device_type} />
            <SpecView label="Condition" value={eq.condition} />
            <SpecView label="Availability" value={availBadge(eq)} />
            <SpecView label="Readiness" value={assignBadge(eq)} />
            <SpecView label="Loan agreement" value={comodatoBadge(eq)} />
            <SpecView label="Assigned employee" value={eq.employee_name ?? 'Unassigned'} />
            <SpecView label="Employee email" value={eq.employee_email || '—'} />
            <SpecView label="Talent ID" value={eq.employee_talent_id || '—'} mono />
            <SpecView label="Transaction date" value={eq.transaction_date || '—'} />
            <SpecView label="Expected return date" value={eq.expected_return_date || '—'} />
            {isDesktop && (
              <>
                <SpecView label="Powers on" value={eq.powers_on} />
                <SpecView label="Operating system" value={eq.os || '—'} />
                <SpecView label="BIOS password" value={eq.bios_password || '—'} mono />
                <SpecView label="WiFi" value={eq.wifi || '—'} />
                <SpecView label="Bluetooth" value={eq.bluetooth || '—'} />
                <SpecView label="Charger included" value={eq.charger_included ? 'Yes' : 'No'} />
                <SpecView label="Last format" value={eq.last_format_date || '—'} />
                <SpecView label="Monitor included" value={eq.monitor_included ? 'Yes' : 'No'} />
                {eq.monitor_included && <SpecView label="Monitor serial" value={eq.monitor_serial || '—'} mono />}
              </>
            )}
          </>
        )}

        {/* ── EDIT / CREATE FORM ── */}
        {(isEdit || isCreate) && (
          <>
            {isEdit && (
              <SpecSelect label="Device type" value={form.device_type}
                options={filteredTypes}
                labels={filteredLabels}
                onChange={v => set('device_type', v as CreateEquipmentPayload['device_type'])} />
            )}
            <SpecSelect label="Condition" value={form.condition}
              options={['New','Good','Fair','Damaged']} onChange={v => set('condition', v)} />
            <SpecSelect label="Availability"
              value={form.availability}
              options={['DISPONIBLE','ASIGNADA','NO_DISPONIBLE','SCRAP']}
              labels={['Available','Assigned','Unavailable','Scrap']}
              onChange={v => set('availability', v as CreateEquipmentPayload['availability'])} />
            <SpecSelect label="Readiness"
              value={form.assignability}
              options={['LISTA','NECESITA_PREP','NO_FUNCIONAL']}
              labels={['Ready','Needs prep','Non-functional']}
              onChange={v => set('assignability', v as CreateEquipmentPayload['assignability'])} />
            <SpecSelect label="Loan agreement"
              value={form.comodato}
              options={['FIRMADO','PENDIENTE','N/A']}
              labels={['Signed','Pending','N/A']}
              onChange={v => set('comodato', v as CreateEquipmentPayload['comodato'])} />
            <SpecField label="Assigned employee" value={form.employee_name ?? ''}
              onChange={v => set('employee_name', v || null)} placeholder="Leave empty if not assigned" />
            <SpecField label="Employee email" value={form.employee_email}
              onChange={v => set('employee_email', v)} placeholder="name@ibm.com" />
            <SpecField label="Talent ID" value={form.employee_talent_id}
              onChange={v => set('employee_talent_id', v)} placeholder="e.g. 104697" />
            <SpecField label="Transaction date" value={form.transaction_date}
              onChange={v => set('transaction_date', v)} placeholder="DD/MM/YYYY" />
            <SpecField label="Expected return date" value={form.expected_return_date}
              onChange={v => set('expected_return_date', v)} placeholder="DD/MM/YYYY (leave empty if N/A)" />

            {/* Desktop-only fields */}
            {isDesktop && (
              <>
                <SpecSelect label="Powers on" value={form.powers_on}
                  options={['OK','NO ENCIENDE']} labels={['OK','Does not power on']} onChange={v => set('powers_on', v)} />
                <SpecField label="Operating system" value={form.os}
                  onChange={v => set('os', v)} placeholder="e.g. Windows 11" />
                <SpecField label="BIOS password" value={form.bios_password}
                  onChange={v => set('bios_password', v)} placeholder="e.g. USAA88IBM" />
                <SpecSelect label="WiFi" value={form.wifi}
                  options={['Enabled','Disabled','Unknown']} onChange={v => set('wifi', v)} />
                <SpecSelect label="Bluetooth" value={form.bluetooth}
                  options={['Disabled','Enabled','Unknown']} onChange={v => set('bluetooth', v)} />
                <SpecSelect label="Charger included" value={form.charger_included ? 'Yes' : 'No'}
                  options={['Yes','No']} onChange={v => set('charger_included', v === 'Yes')} />
                <SpecField label="Last format" value={form.last_format_date}
                  onChange={v => set('last_format_date', v)} placeholder="DD/MM/YYYY" />
                <SpecSelect label="Monitor included" value={form.monitor_included ? 'Yes' : 'No'}
                  options={['Yes','No']} onChange={v => set('monitor_included', v === 'Yes')} />
                {form.monitor_included && (
                  <SpecField label="Monitor serial" value={form.monitor_serial}
                    onChange={v => set('monitor_serial', v)} placeholder="e.g. 033NTUY..." />
                )}
              </>
            )}
          </>
        )}
      </div>

      {/* Notes — always shown */}
      <div className="spec-note">
        <div className="l">Notes</div>
        {isView
          ? <p>{eq?.notes || '—'}</p>
          : <textarea value={form.notes} onChange={e => set('notes', e.target.value)}
              placeholder="Notes about this device…" />
        }
      </div>
    </>
  )
}

// ── Network form ───────────────────────────────────────────────────────────────

interface NetworkFormProps {
  form: CreateEquipmentPayload
  eq: Equipment | null
  isCreate: boolean
  isView: boolean
  isEdit: boolean
  canWrite: boolean
  set: <K extends keyof CreateEquipmentPayload>(k: K, v: CreateEquipmentPayload[K]) => void
  emailErrors: Record<string, string>
  availBadge: (e: Equipment) => React.ReactNode
  assignBadge: (e: Equipment) => React.ReactNode
  comodatoBadge: (e: Equipment) => React.ReactNode
}

/** Format a YYYY-MM-DD date string from a DATE column into a readable display value. */
function fmtDate(v: string | null | undefined): string {
  if (!v) return '—'
  // Postgres DATE returned as YYYY-MM-DD (first 10 chars when cast ::text)
  return v.slice(0, 10)
}

/** Convert a date string from an <input type="date"> (YYYY-MM-DD) or null/empty into a nullable string. */
function dateInputToPtr(v: string): string | null {
  return v.trim() === '' ? null : v.trim()
}

function NetworkForm({ form, eq, isCreate, isView, isEdit, set, emailErrors }: NetworkFormProps) {
  const deviceTypeLabel: Record<string, string> = {
    Switch: 'Switch', Router: 'Router', Firewall: 'Firewall',
    AccessPoint: 'Access Point', License: 'License', OtherNetwork: 'Other network device',
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>

      {/* ── Identification ──────────────────────────────────────────────── */}
      <SectionHeader label="Identification" />
      <div className="spec-grid">
        {isCreate && (
          <SpecField label="Serial number *" value={form.serial} onChange={v => set('serial', v)} placeholder="e.g. FOC2318Y4PV" />
        )}
        {isView && eq && (
          <>
            <SpecView label="Device type" value={deviceTypeLabel[eq.device_type] ?? eq.device_type} />
            <SpecView label="Product ID" value={eq.product_id || '—'} mono />
            <SpecView label="Model / Description" value={`${eq.brand && eq.brand !== 'Unknown' ? eq.brand + ' ' : ''}${eq.model}`} />
            <SpecView label="Network type" value={eq.net_type || '—'} />
          </>
        )}
        {(isEdit || isCreate) && (
          <>
            <SpecSelect label="Device type"
              value={form.device_type}
              options={[...NETWORK_DEVICE_TYPES]}
              labels={NETWORK_DEVICE_LABELS}
              onChange={v => set('device_type', v as CreateEquipmentPayload['device_type'])} />
            <SpecField label="Product ID" value={form.product_id} onChange={v => set('product_id', v)} placeholder="e.g. WS-C3650-48PD-S" />
            <SpecField label="Brand" value={form.brand} onChange={v => set('brand', v)} placeholder="e.g. Cisco" />
            <SpecField label="Model / Description *" value={form.model} onChange={v => set('model', v)} placeholder="e.g. Catalyst 3650" />
            <SpecField label="Variant / sub-model" value={form.variant} onChange={v => set('variant', v)} placeholder="e.g. 48-port" />
            <SpecSelect label="Network type"
              value={form.net_type}
              options={['N/A','Primary','Secondary']}
              onChange={v => set('net_type', v)} />
          </>
        )}
      </div>

      {/* ── Lifecycle & contract ────────────────────────────────────────── */}
      <SectionHeader label="Lifecycle & Contract" />
      <div className="spec-grid">
        {isView && eq && (
          <>
            <SpecView label="End of Sale" value={fmtDate(eq.end_of_sale)} />
            <SpecView label="End of Life" value={fmtDate(eq.end_of_life)} />
            <SpecView label="End of Contract Support" value={fmtDate(eq.end_contract_support)} />
            <SpecView label="EOL Status" value={eq.eol_status || '—'} />
            <SpecView label="Contract Status" value={eq.contract_status || '—'} />
          </>
        )}
        {(isEdit || isCreate) && (
          <>
            <SpecDate label="End of Sale" value={form.end_of_sale ?? ''} onChange={v => set('end_of_sale', dateInputToPtr(v))} />
            <SpecDate label="End of Life" value={form.end_of_life ?? ''} onChange={v => set('end_of_life', dateInputToPtr(v))} />
            <SpecDate label="End of Contract Support" value={form.end_contract_support ?? ''} onChange={v => set('end_contract_support', dateInputToPtr(v))} />
            <SpecField label="EOL Status" value={form.eol_status} onChange={v => set('eol_status', v)} placeholder="e.g. End of Life" />
            <SpecField label="Contract Status" value={form.contract_status} onChange={v => set('contract_status', v)} placeholder="e.g. Active" />
          </>
        )}
      </div>

      {/* ── Costs ───────────────────────────────────────────────────────── */}
      <SectionHeader label="Costs" />
      <div className="spec-grid">
        {isView && eq && (
          <>
            <SpecView label="Device Cost" value={eq.device_cost ?? '—'} />
            <SpecView label="Contract Cost" value={eq.contract_cost ?? '—'} />
          </>
        )}
        {(isEdit || isCreate) && (
          <>
            <SpecCost label="Device Cost" value={form.device_cost ?? ''} onChange={v => set('device_cost', v === '' ? null : v)} />
            <SpecCost label="Contract Cost" value={form.contract_cost ?? ''} onChange={v => set('contract_cost', v === '' ? null : v)} />
          </>
        )}
      </div>

      {/* ── Location ────────────────────────────────────────────────────── */}
      <SectionHeader label="Location" />
      <div className="spec-grid">
        {isView && eq && (
          <>
            <SpecView label="Location" value={eq.variant || '—'} />
            <SpecView label="Room" value={eq.room || '—'} />
          </>
        )}
        {(isEdit || isCreate) && (
          <>
            <SpecField label="Location" value={form.variant} onChange={v => set('variant', v)} placeholder="e.g. CDMX Office" />
            <SpecField label="Room" value={form.room} onChange={v => set('room', v)} placeholder="e.g. ODC, ODC2, ODC2-AT&T" />
          </>
        )}
      </div>

      {/* ── Vendor & support ────────────────────────────────────────────── */}
      <SectionHeader label="Vendor & Support" />
      <div className="spec-grid">
        {isView && eq && (
          <>
            <SpecView label="Device Company" value={eq.device_company || '—'} />
            <SpecView label="Provider contact name" value={eq.contact_name || '—'} />
            <SpecView label="Provider contact phone" value={eq.contact_phone || '—'} />
            <SpecView label="Provider contact email" value={eq.contact_email || '—'} />
            <SpecView label="IBM network support email" value={eq.ibm_network_email_support || '—'} />
            <SpecView label="IBM local support email" value={eq.ibm_local_email_support || '—'} />
          </>
        )}
        {(isEdit || isCreate) && (
          <>
            <SpecField label="Device Company" value={form.device_company} onChange={v => set('device_company', v)} placeholder="e.g. Cisco Systems" />
            <SpecField label="Provider contact name" value={form.contact_name} onChange={v => set('contact_name', v)} placeholder="e.g. John Smith" />
            <SpecField label="Provider contact phone" value={form.contact_phone} onChange={v => set('contact_phone', v)} placeholder="e.g. +52 55 1234 5678" />
            <SpecField label="Provider contact email" value={form.contact_email} onChange={v => set('contact_email', v)} placeholder="vendor@company.com" error={emailErrors['contact_email']} />
            <SpecField label="IBM network support email" value={form.ibm_network_email_support} onChange={v => set('ibm_network_email_support', v)} placeholder="network@ibm.com" error={emailErrors['ibm_network_email_support']} />
            <SpecField label="IBM local support email" value={form.ibm_local_email_support} onChange={v => set('ibm_local_email_support', v)} placeholder="local@ibm.com" error={emailErrors['ibm_local_email_support']} />
          </>
        )}
      </div>

      {/* ── Notes ───────────────────────────────────────────────────────── */}
      <div className="spec-note">
        <div className="l">Notes</div>
        {isView
          ? <p style={{ whiteSpace: 'pre-wrap' }}>{eq?.notes || '—'}</p>
          : <textarea
              value={form.notes}
              onChange={e => set('notes', e.target.value)}
              placeholder="Comments about this device…"
              rows={4}
            />
        }
      </div>
    </div>
  )
}

// ── Small reusable form components ────────────────────────────────────────────

function SectionHeader({ label }: { label: string }) {
  return (
    <div style={{
      fontSize: '0.75rem', fontWeight: 600, textTransform: 'uppercase',
      letterSpacing: '0.08em', color: 'var(--cds-text-secondary)',
      borderBottom: '1px solid var(--cds-border-subtle-01)',
      paddingBottom: 4, marginBottom: -8,
    }}>
      {label}
    </div>
  )
}

function SpecView({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <div className={`v${mono ? ' mono' : ''}`}>{value}</div>
    </div>
  )
}

function SpecField({ label, value, onChange, placeholder, error }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; error?: string
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        style={error ? { borderColor: 'var(--cds-support-error)' } : undefined}
      />
      {error && <div style={{ fontSize: '0.75rem', color: 'var(--cds-support-error)', marginTop: 2 }}>{error}</div>}
    </div>
  )
}

function SpecDate({ label, value, onChange }: {
  label: string; value: string; onChange: (v: string) => void
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <input
        type="date"
        value={value}
        onChange={e => onChange(e.target.value)}
      />
    </div>
  )
}

/** Numeric cost field — accepts decimals, rejects negatives and non-numeric input. */
function SpecCost({ label, value, onChange }: {
  label: string; value: string; onChange: (v: string) => void
}) {
  function handleChange(raw: string) {
    // Allow empty, digits, and a single decimal point; disallow negatives
    if (raw === '' || /^\d+(\.\d{0,2})?$/.test(raw)) {
      onChange(raw)
    }
  }
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <input
        type="text"
        inputMode="decimal"
        value={value}
        onChange={e => handleChange(e.target.value)}
        placeholder="0.00"
      />
    </div>
  )
}

function SpecSelect({ label, value, options, labels, onChange }: {
  label: string; value: string; options: string[]; labels?: string[]; onChange: (v: string) => void
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <select value={value} onChange={e => onChange(e.target.value)}>
        {options.map((o, i) => <option key={o} value={o}>{labels?.[i] ?? o}</option>)}
      </select>
    </div>
  )
}
