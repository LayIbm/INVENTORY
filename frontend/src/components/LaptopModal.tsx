import { useState } from 'react'
import { Laptop, CreateLaptopPayload, UpdateLaptopPayload } from '../hooks/useLaptops'
import { generateCommodatumPDF } from '../utils/generateCommodatum'

function availBadge(l: Laptop) {
  if (l.availability === 'DISPONIBLE') return <span className="badge ok">Available</span>
  if (l.availability === 'ASIGNADA')   return <span className="badge neutral">Assigned</span>
  return <span className="badge danger">Unavailable</span>
}

function prepBadge(l: Laptop) {
  if (l.prep === 'LISTA')          return <span className="badge ok">Ready</span>
  if (l.prep === 'NECESITA_PREP')  return <span className="badge warn">Needs prep.</span>
  return <span className="badge danger">Non-functional</span>
}

function comodatoBadge(l: Laptop) {
  if (l.comodato === 'FIRMADO')   return <span className="badge ok">Signed</span>
  if (l.comodato === 'PENDIENTE') return <span className="badge warn">Pending</span>
  return <span className="badge neutral">N/A</span>
}

type Mode = 'view' | 'edit' | 'create'

interface Props {
  mode: Mode
  laptop: Laptop | null
  role: string
  /** Pre-select a specific tab when the modal opens. Defaults to 'tech'. */
  initialTab?: 'tech' | 'loans' | 'log' | 'bios'
  onClose: () => void
  onCreate: (payload: CreateLaptopPayload) => Promise<void>
  onUpdate: (id: string, payload: UpdateLaptopPayload) => Promise<void>
  onDelete: (l: Laptop) => void
}

const toneColor: Record<string, string> = {
  ok: 'var(--ok)', warn: 'var(--warn)', danger: 'var(--danger)', neutral: 'var(--ink-faint)',
}

function todayStr() {
  const d = new Date()
  return `${String(d.getDate()).padStart(2,'0')}/${String(d.getMonth()+1).padStart(2,'0')}/${d.getFullYear()}`
}

const emptyForm: CreateLaptopPayload = {
  serial: '', model: '', variant: '', brand: 'Lenovo', condition: 'Good',
  availability: 'DISPONIBLE', prep: 'NECESITA_PREP', comodato: 'N/A',
  powers_on: 'OK', os: 'Windows 11', win11_ready: false, bios_password: 'NO PASSWORD',
  wifi: 'Disabled', bluetooth: 'Disabled', charger_included: true,
  last_format_date: todayStr(), employee_name: null,
  employee_email: '', employee_talent_id: '', employee_manager_email: '', lcd_ok: 'OK', expected_return_date: '',
  owner: 'IBM', hostname: '', geography: '',
  last_bios_update: '', bios_details: '', bluetooth_disabled_bios: false,
  epd_status: '', ipv6: 'Disabled', usage: '',
  notes: '',
}

export default function LaptopModal({ mode, laptop, role, initialTab = 'tech', onClose, onCreate, onUpdate, onDelete }: Props) {
  const canWrite = role === 'manager' || role === 'admin'
  const [activeTab, setActiveTab] = useState<'tech' | 'loans' | 'log' | 'bios'>(initialTab)
  const [localMode, setLocalMode] = useState<Mode>(mode)

  const [form, setForm] = useState<CreateLaptopPayload>(
    mode === 'create' ? emptyForm : laptopToForm(laptop!)
  )

  function laptopToForm(l: Laptop): CreateLaptopPayload {
    return {
      serial: l.serial, model: l.model, variant: l.variant, brand: l.brand,
      condition: l.condition, availability: l.availability, prep: l.prep, comodato: l.comodato,
      powers_on: l.powers_on, os: l.os, win11_ready: l.win11_ready, bios_password: l.bios_password,
      wifi: l.wifi, bluetooth: l.bluetooth, charger_included: l.charger_included,
      last_format_date: l.last_format_date, employee_name: l.employee_name,
      employee_email: l.employee_email, employee_talent_id: l.employee_talent_id,
      employee_manager_email: l.employee_manager_email,
      lcd_ok: l.lcd_ok, expected_return_date: l.expected_return_date,
      owner: l.owner, hostname: l.hostname, geography: l.geography,
      last_bios_update: l.last_bios_update, bios_details: l.bios_details,
      bluetooth_disabled_bios: l.bluetooth_disabled_bios,
      epd_status: l.epd_status, ipv6: l.ipv6, usage: l.usage,
      notes: l.notes,
    }
  }

  function set<K extends keyof CreateLaptopPayload>(k: K, v: CreateLaptopPayload[K]) {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  async function handleSave() {
    if (localMode === 'create') {
      await onCreate({ ...form, serial: form.serial.toUpperCase().trim() })
    } else if (laptop) {
      const { serial: _s, ...rest } = form
      void _s
      await onUpdate(laptop.id, rest as UpdateLaptopPayload)
    }
    onClose()
  }

  const isCreate = localMode === 'create'
  const isView = localMode === 'view'
  const isEdit = localMode === 'edit'

  return (
    <div className="overlay" style={{ display: 'flex' }}>
      <div className="modal">
        <div className="modal-head">
          <div className="modal-head-top">
            <div>
              <div className="modal-title-row">
                <h3>{isCreate ? 'Add laptop' : `${laptop?.brand} ${laptop?.model}`}</h3>
                {!isCreate && laptop && (
                  <span className={`badge ${laptop.availability === 'DISPONIBLE' ? 'ok' : laptop.availability === 'ASIGNADA' ? 'neutral' : 'danger'}`}>
                    {laptop.availability === 'DISPONIBLE' ? 'Available' : laptop.availability === 'ASIGNADA' ? 'Assigned' : 'Unavailable'}
                  </span>
                )}
              </div>
              <div className="modal-sub">
                {isCreate
                  ? 'Fill in the new device details.'
                  : laptop && <><span className="tag" style={{ marginRight: 8 }}>{laptop.serial}</span>{laptop.variant}</>
                }
              </div>
            </div>
            <button className="modal-close" onClick={onClose}>&times;</button>
          </div>
          <div className="tabs">
            <button className={`tab${activeTab === 'tech'  ? ' active' : ''}`} onClick={() => setActiveTab('tech')}>Technical details</button>
            {!isCreate && <button className={`tab${activeTab === 'loans' ? ' active' : ''}`} onClick={() => setActiveTab('loans')}>Loans</button>}
            {!isCreate && <button className={`tab${activeTab === 'log'   ? ' active' : ''}`} onClick={() => setActiveTab('log')}>Log</button>}
            {!isCreate && <button className={`tab${activeTab === 'bios'  ? ' active' : ''}`} onClick={() => setActiveTab('bios')}>BIOS Password</button>}
          </div>
        </div>

        <div className="modal-body">
          {activeTab === 'tech' && (
            <>
              <div className="spec-grid">
                {isCreate && (
                  <>
                    <SpecField label="Serial number" value={form.serial} onChange={(v) => set('serial', v)} />
                    <SpecField label="Brand" value={form.brand} onChange={(v) => set('brand', v)} />
                    <SpecField label="Model" value={form.model} onChange={(v) => set('model', v)} />
                    <SpecField label="Variant / sub-model" value={form.variant} onChange={(v) => set('variant', v)} />
                  </>
                )}
                {isView && laptop ? (
                  <>
                    <SpecView label="Condition" value={laptop.condition} />
                    <SpecView label="Powers on" value={laptop.powers_on} />
                    <SpecView label="Operating system" value={laptop.os} />
                    <SpecView label="Windows 11 ready" value={laptop.win11_ready ? 'Yes' : 'No'} />
                    <SpecView label="BIOS password" value={laptop.bios_password} mono />
                    <SpecView label="WiFi" value={laptop.wifi} />
                    <SpecView label="Bluetooth" value={laptop.bluetooth} />
                    <SpecView label="Charger included" value={laptop.charger_included ? 'Yes' : 'No'} />
                    <SpecView label="Last format" value={laptop.last_format_date} />
                    <SpecView label="Availability" value={availBadge(laptop)} />
                    <SpecView label="Readiness" value={prepBadge(laptop)} />
                    <SpecView label="Loan agreement" value={comodatoBadge(laptop)} />
                    <SpecView label="Assigned employee" value={laptop.employee_name ?? 'Unassigned'} />
                    <SpecView label="Employee email" value={laptop.employee_email || '—'} />
                    <SpecView label="Manager email" value={laptop.employee_manager_email || '—'} />
                    <SpecView label="Talent ID" value={laptop.employee_talent_id || '—'} mono />
                    <SpecView label="Screen (LCD)" value={laptop.lcd_ok} />
                    <SpecView label="Expected return date" value={laptop.expected_return_date || '—'} />
                    <SpecView label="Device owner" value={laptop.owner || 'IBM'} />
                    <SpecView label="Hostname" value={laptop.hostname || '—'} mono />
                    <SpecView label="Geography" value={laptop.geography || '—'} />
                    <SpecView label="IPv6" value={laptop.ipv6 || '—'} mono />
                    <SpecView label="Usage" value={laptop.usage || '—'} />
                  </>
                ) : (
                  <>
                    <SpecSelect label="Condition" value={form.condition} options={['New','Good','Fair','Damaged']} onChange={(v) => set('condition', v)} />
                    <SpecSelect label="Powers on" value={form.powers_on} options={['OK','NO ENCIENDE']} labels={['OK','Does not power on']} onChange={(v) => set('powers_on', v)} />
                    <SpecField label="Operating system" value={form.os} onChange={(v) => set('os', v)} />
                    <SpecSelect label="Windows 11 ready" value={form.win11_ready ? 'Yes' : 'No'} options={['Yes','No']} onChange={(v) => set('win11_ready', v === 'Yes')} />
                    <SpecField label="BIOS password" value={form.bios_password} onChange={(v) => set('bios_password', v)} />
                    <SpecSelect label="WiFi" value={form.wifi} options={['Enabled','Disabled','Unknown']} onChange={(v) => set('wifi', v)} />
                    <SpecSelect label="Bluetooth" value={form.bluetooth} options={['Disabled','Enabled','Unknown']} onChange={(v) => set('bluetooth', v)} />
                    <SpecSelect label="Charger included" value={form.charger_included ? 'Yes' : 'No'} options={['Yes','No']} onChange={(v) => set('charger_included', v === 'Yes')} />
                    <SpecField label="Last format" value={form.last_format_date} onChange={(v) => set('last_format_date', v)} />
                    <SpecSelect label="Availability" value={form.availability} options={['DISPONIBLE','ASIGNADA','NO_DISPONIBLE','SCRAP']} labels={['Available','Assigned','Unavailable','Scrap']} onChange={(v) => set('availability', v as CreateLaptopPayload['availability'])} />
                    <SpecSelect label="Readiness" value={form.prep} options={['LISTA','NECESITA_PREP','NO_FUNCIONAL']} labels={['Ready','Needs prep','Non-functional']} onChange={(v) => set('prep', v as CreateLaptopPayload['prep'])} />
                    <SpecSelect label="Loan agreement" value={form.comodato} options={['FIRMADO','PENDIENTE','N/A']} labels={['Signed','Pending','N/A']} onChange={(v) => set('comodato', v as CreateLaptopPayload['comodato'])} />
                    <SpecField label="Assigned employee" value={form.employee_name ?? ''} onChange={(v) => set('employee_name', v || null)} placeholder="Leave empty if not assigned" />
                    <SpecField label="Employee email" value={form.employee_email} onChange={(v) => set('employee_email', v)} placeholder="name@ibm.com" />
                    <SpecField label="Manager email" value={form.employee_manager_email} onChange={(v) => set('employee_manager_email', v)} placeholder="manager@ibm.com" />
                    <SpecField label="Talent ID" value={form.employee_talent_id} onChange={(v) => set('employee_talent_id', v)} placeholder="e.g. 104697" />
                    <SpecSelect label="Screen (LCD)" value={form.lcd_ok} options={['OK','NO ENCIENDE']} labels={['OK','Does not power on']} onChange={(v) => set('lcd_ok', v)} />
                    <SpecField label="Expected return date" value={form.expected_return_date} onChange={(v) => set('expected_return_date', v)} placeholder="DD/MM/YYYY (leave empty if N/A)" />
                    <SpecSelect label="Device owner" value={form.owner} options={['IBM','USAA']} onChange={(v) => set('owner', v as 'IBM' | 'USAA')} />
                    <SpecField label="Hostname" value={form.hostname} onChange={(v) => set('hostname', v)} placeholder="e.g. MXIBM-LAYSSA" />
                    <SpecField label="Geography" value={form.geography} onChange={(v) => set('geography', v)} placeholder="e.g. CIC1-B Guadalajara" />
                    <SpecSelect label="IPv6" value={form.ipv6} options={['Disabled','Enabled']} onChange={(v) => set('ipv6', v)} />
                    <SpecSelect label="Usage" value={form.usage} options={['','Exclusive IBM','IBM Client','Exclusive Client']} labels={['— Not set —','Exclusive IBM','IBM Client','Exclusive Client']} onChange={(v) => set('usage', v)} />
                  </>
                )}
              </div>

              <div className="spec-note">
                <div className="l">Notes</div>
                {isView
                  ? <p>{laptop?.notes || '—'}</p>
                  : <textarea value={form.notes} onChange={(e) => set('notes', e.target.value)} placeholder="Notes about this device…" />
                }
              </div>
            </>
          )}

          {/* Loans tab — solo cambios de asignación de empleado */}
          {activeTab === 'loans' && laptop && (() => {
            // Tipos explícitos de préstamo (ingresados manualmente o por flujos legacy)
            const LOAN_TYPES = new Set(['Entrega', 'Recibo', 'Asignada', 'LOAN_OUT', 'RETURN', 'ASSIGNED'])

            // Extrae solo el fragmento "Empleado: X → Y" de una nota compuesta
            function extractEmpleado(notes: string): string | null {
              // Nota del backend: "Empleado: Sin asignar → Juan | Correo empleado: ..."
              const match = notes.match(/Empleado:\s*([^|]+)/i)
              if (match) return match[1].trim()
              return null
            }

            const loanHistory = [...laptop.history]
              .reverse()
              .filter(h => {
                if (!h.notes) return false
                // Tipo explícito de asignación
                if (LOAN_TYPES.has(h.type)) return true
                // Actualización que incluye cambio de empleado
                if (h.notes.match(/Empleado:/i)) return true
                return false
              })

            return loanHistory.length === 0 ? (
              <div style={{ padding: '32px 0', textAlign: 'center', color: 'var(--cds-text-placeholder)', fontStyle: 'italic', fontSize: 13 }}>
                No employee assignments recorded.
              </div>
            ) : (
              <ul className="timeline">
                {loanHistory.map((h, i) => {
                  // Para actualizaciones generales, mostrar solo el fragmento de empleado
                  const noteToShow = LOAN_TYPES.has(h.type)
                    ? h.notes
                    : (extractEmpleado(h.notes) ?? h.notes)
                  return (
                    <li key={i}>
                      <div className="t-dot" style={{ background: toneColor[h.tone] ?? toneColor.neutral }} />
                      <div className="t-body">
                        <div className="t-head">
                          <span className={`badge ${h.tone === 'ok' ? 'ok' : h.tone === 'warn' ? 'warn' : h.tone === 'danger' ? 'danger' : 'neutral'}`}>
                            {LOAN_TYPES.has(h.type) ? h.type : 'Assignment change'}
                          </span>
                          <span className="t-emp">{h.employee}</span>
                          <span className="t-date">{h.date}</span>
                        </div>
                        <div className="t-note">{noteToShow}</div>
                      </div>
                    </li>
                  )
                })}
              </ul>
            )
          })()}

          {/* Log tab — all changes */}
          {activeTab === 'log' && laptop && (
            <ul className="timeline">
              {[...laptop.history].reverse().map((h, i) => (
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
          )}

          {activeTab === 'bios' && laptop && (() => {
            const biosHistory = [...laptop.history]
              .reverse()
              .filter(h => h.notes && h.notes.toLowerCase().includes('bios'))

            return (
              <>
                {/* ── Estado actual de BIOS ── */}
                <div className="spec-grid" style={{ marginBottom: 20 }}>
                  {isView ? (
                    <>
                      <div className="spec-item">
                        <div className="l">Current BIOS password</div>
                              <div className="v mono">{laptop.bios_password || 'NO PASSWORD'}</div>
                      </div>
                      <div className="spec-item">
                        <div className="l">Last password change</div>
                        <div className="v">{laptop.last_bios_update || '—'}</div>
                      </div>
                      <div className="spec-item">
                        <div className="l">Bluetooth disabled in BIOS</div>
                        <div className="v">
                          <span className={`badge ${laptop.bluetooth_disabled_bios ? 'ok' : 'danger'}`}>
                            {laptop.bluetooth_disabled_bios ? 'Yes — Disabled' : 'No — Enabled'}
                          </span>
                        </div>
                      </div>
                      <div className="spec-item">
                        <div className="l">IPv6</div>
                        <div className="v" style={{ fontFamily: 'monospace', fontSize: 12.5 }}>{laptop.ipv6 || '—'}</div>
                      </div>
                      <div className="spec-item" style={{ gridColumn: '1 / -1' }}>
                        <div className="l">BIOS details / observations</div>
                        <div className="v">{laptop.bios_details || '—'}</div>
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="spec-item">
                        <div className="l">Current BIOS password</div>
                        <input type="text" value={form.bios_password}
                          onChange={e => set('bios_password', e.target.value)}
                          placeholder="e.g. USAA88IBM" />
                      </div>
                      <div className="spec-item">
                        <div className="l">Last password change</div>
                        <input type="text" value={form.last_bios_update}
                          onChange={e => set('last_bios_update', e.target.value)}
                          placeholder="DD/MM/YYYY" />
                      </div>
                      <div className="spec-item">
                        <div className="l">Bluetooth disabled in BIOS</div>
                        <select value={form.bluetooth_disabled_bios ? 'Yes' : 'No'}
                          onChange={e => set('bluetooth_disabled_bios', e.target.value === 'Yes')}>
                          <option value="Yes">Yes — Disabled (meets USAA policy)</option>
                          <option value="No">No — Enabled (does NOT meet USAA policy)</option>
                        </select>
                      </div>
                      <div className="spec-item">
                        <div className="l">IPv6</div>
                        <select value={form.ipv6} onChange={e => set('ipv6', e.target.value)}>
                          <option value="Disabled">Disabled</option>
                          <option value="Enabled">Enabled</option>
                        </select>
                      </div>
                      <div className="spec-item" style={{ gridColumn: '1 / -1' }}>
                        <div className="l">BIOS details / observations</div>
                        <textarea value={form.bios_details}
                          onChange={e => set('bios_details', e.target.value)}
                          placeholder="e.g. Recorded password did not match, performed reset…"
                          rows={3} style={{ width: '100%', resize: 'vertical' }} />
                      </div>
                    </>
                  )}
                </div>

                {/* ── Historial de cambios BIOS ── */}
                <div style={{ fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '.06em',
                  color: 'var(--cds-text-secondary)', borderBottom: '1px solid var(--cds-border-subtle-01)',
                  paddingBottom: 4, marginBottom: 14 }}>
                  Change history
                </div>
                {biosHistory.length === 0 ? (
                  <div style={{ padding: '16px 0', textAlign: 'center', color: 'var(--cds-text-placeholder)', fontStyle: 'italic', fontSize: 13 }}>
                    No BIOS password changes recorded.
                  </div>
                ) : (
                  <ul className="timeline">
                    {biosHistory.map((h, i) => {
                  // Extract BIOS change from notes: "BIOS: OLD → NEW"
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
                )}
              </>
            )
          })()}

        </div>

        <div className="modal-foot">
          <div className="left">
            {isView && canWrite && (
              <>
                <button className="btn btn-ghost" onClick={() => setLocalMode('edit')}>Edit</button>
                <button className="btn btn-danger-ghost" onClick={() => laptop && onDelete(laptop)}>Delete</button>
              </>
            )}
            {isView && laptop && laptop.employee_name && (
              <button
                className="btn btn-ghost"
                onClick={() => generateCommodatumPDF({
                  description: 'LAPTOP',
                  deviceType: laptop.variant || '-',
                  model: laptop.model,
                  serial: laptop.serial,
                  employeeName: laptop.employee_name!,
                  employeeTalentId: laptop.employee_talent_id,
                  date: new Date().toISOString().split('T')[0],
                  os: laptop.os || undefined,
                  chargerIncluded: laptop.charger_included,
                })}
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
              <button className="btn btn-ghost" onClick={() => isCreate ? onClose() : setLocalMode('view')}>Cancel</button>
            )}
          </div>
          <div className="right">
            {isView && <button className="btn btn-ghost" onClick={onClose}>Close</button>}
            {(isEdit || isCreate) && (
              <button className="btn btn-primary" onClick={handleSave}>
                {isCreate ? 'Create laptop' : 'Save changes'}
              </button>
            )}
          </div>
        </div>
      </div>
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

function SpecField({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
    </div>
  )
}

function SpecSelect({ label, value, options, labels, onChange }: { label: string; value: string; options: string[]; labels?: string[]; onChange: (v: string) => void }) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((o, i) => <option key={o} value={o}>{labels?.[i] ?? o}</option>)}
      </select>
    </div>
  )
}
