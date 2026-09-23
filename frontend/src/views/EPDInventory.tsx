import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import { Laptop, fetchLaptops, createLaptop, updateLaptop, deleteLaptop, CreateLaptopPayload, UpdateLaptopPayload } from '../hooks/useLaptops'
import LaptopModal from '../components/LaptopModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import InventoryImportModal from '../components/InventoryImportModal'

type ModalMode = 'view' | 'edit' | 'create'
type CountryFilter = 'ALL' | 'MEXICO' | 'PHILIPPINES' | 'INDIA'
type OwnerFilter = 'ALL' | 'IBM' | 'USAA'
type UsageFilter = 'ALL' | 'Exclusive IBM' | 'IBM Client' | 'Exclusive Client' | 'Unassigned'

// ── CSV helpers ───────────────────────────────────────────────────────────────

function csvEscape(v: string | null | undefined): string {
  const s = v == null ? '' : String(v)
  return `"${s.replace(/"/g, '""')}"`
}

const availLabelEPD: Record<string, string> = {
  DISPONIBLE: 'Available', ASIGNADA: 'Assigned',
  NO_DISPONIBLE: 'Unavailable', SCRAP: 'Scrap',
}

const prepLabelEPD: Record<string, string> = {
  LISTA: 'Ready', NECESITA_PREP: 'Needs prep', NO_FUNCIONAL: 'Non-functional',
}
const comodatoLabelEPD: Record<string, string> = {
  FIRMADO: 'Signed', PENDIENTE: 'Pending', 'N/A': 'N/A',
}

function buildEpdCsv(rows: Laptop[]): string {
  const headers = [
    'Serial', 'Brand', 'Model', 'Variant', 'Condition',
    'Availability', 'Preparation', 'Comodato',
    'Owner', 'Geography', 'Hostname', 'EPD Status', 'IPv6',
    'Employee', 'Employee email', 'Employee talent ID', 'Manager email',
    'Expected return date',
    'OS', 'Powers on', 'Win11 ready', 'BIOS password',
    'WiFi', 'Bluetooth', 'Bluetooth disabled (BIOS)',
    'Charger included', 'LCD OK',
    'Last format date', 'Last BIOS update', 'BIOS details',
    'Notes',
  ]
  const csvRows = rows.map(l => [
    csvEscape(l.serial),
    csvEscape(l.brand),
    csvEscape(l.model),
    csvEscape(l.variant),
    csvEscape(l.condition),
    csvEscape(availLabelEPD[l.availability] ?? l.availability),
    csvEscape(prepLabelEPD[l.prep] ?? l.prep),
    csvEscape(comodatoLabelEPD[l.comodato] ?? l.comodato),
    csvEscape(l.owner),
    csvEscape(l.geography),
    csvEscape(l.hostname),
    csvEscape(l.epd_status),
    csvEscape(l.ipv6),
    csvEscape(l.employee_name),
    csvEscape(l.employee_email),
    csvEscape(l.employee_talent_id),
    csvEscape(l.employee_manager_email),
    csvEscape(l.expected_return_date),
    csvEscape(l.os),
    csvEscape(l.powers_on),
    csvEscape(l.win11_ready ? 'Yes' : 'No'),
    csvEscape(l.bios_password),
    csvEscape(l.wifi),
    csvEscape(l.bluetooth),
    csvEscape(l.bluetooth_disabled_bios ? 'Yes' : 'No'),
    csvEscape(l.charger_included ? 'Yes' : 'No'),
    csvEscape(l.lcd_ok),
    csvEscape(l.last_format_date),
    csvEscape(l.last_bios_update),
    csvEscape(l.bios_details),
    csvEscape(l.notes),
  ].join(','))
  return '\uFEFF' + [headers.map(csvEscape).join(','), ...csvRows].join('\r\n')
}

function downloadCsvEpd(content: string, filename: string) {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.style.display = 'none'
  document.body.appendChild(a); a.click()
  document.body.removeChild(a); URL.revokeObjectURL(url)
}

function todayFilenameEpd(suffix: string): string {
  const d = new Date()
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `epd-${suffix}-${yyyy}-${mm}-${dd}.csv`
}

function ownerBadge(owner: string) {
  const isUSAA = owner?.toUpperCase() === 'USAA'
  return (
    <span className={`badge ${isUSAA ? 'warn' : 'neutral'}`}>
      {owner || 'IBM'}
    </span>
  )
}

export default function EPDInventory() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()

  const [laptops, setLaptops]             = useState<Laptop[]>([])
  const [loading, setLoading]             = useState(true)
  const [error, setError]                 = useState<string | null>(null)
  const [search, setSearch]               = useState('')
  const [countryFilter, setCountryFilter] = useState<CountryFilter>('ALL')
  const [ownerFilter, setOwnerFilter]     = useState<OwnerFilter>('ALL')
  const [usageFilter, setUsageFilter]     = useState<UsageFilter>('ALL')
  const [modalMode, setModalMode]         = useState<ModalMode>('view')
  const [selected, setSelected]           = useState<Laptop | null>(null)
  const [modalOpen, setModalOpen]         = useState(false)
  const [confirmTarget, setConfirmTarget] = useState<Laptop | null>(null)
  const [importOpen, setImportOpen]       = useState(false)
  const [createOpen, setCreateOpen]       = useState(false)
  const [exportMenuOpen, setExportMenuOpen] = useState(false)

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setLaptops(await fetchLaptops())
    } catch {
      setError('Error loading inventory.')
      addToast('Error loading data', 'danger')
    } finally {
      setLoading(false)
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const asignadas = laptops.filter(l => !!l.employee_name)

  const countMexico      = asignadas.filter(l => l.geography?.toLowerCase().includes('mexico')).length
  const countPhilippines = asignadas.filter(l => l.geography?.toLowerCase().includes('philip')).length
  const countIndia       = asignadas.filter(l => l.geography?.toLowerCase().includes('india')).length

  const byCountry = countryFilter === 'ALL' ? asignadas : asignadas.filter(l => {
    const g = l.geography?.toLowerCase() ?? ''
    if (countryFilter === 'MEXICO')      return g.includes('mexico')
    if (countryFilter === 'PHILIPPINES') return g.includes('philip')
    if (countryFilter === 'INDIA')       return g.includes('india')
    return true
  })

  const byOwner = ownerFilter === 'ALL' ? byCountry : byCountry.filter(l => {
    const isUSAA = l.owner?.toUpperCase() === 'USAA'
    if (ownerFilter === 'USAA') return isUSAA
    if (ownerFilter === 'IBM')  return !isUSAA
    return true
  })

  const byUsage = usageFilter === 'ALL' ? byOwner : byOwner.filter(l => {
    if (usageFilter === 'Unassigned') return !l.usage || l.usage === ''
    return l.usage === usageFilter
  })

  const filtered = byUsage.filter(l => {
    const q = search.toLowerCase()
    return !q || [
      l.employee_name ?? '', l.employee_email, l.employee_talent_id,
      l.serial, l.model, l.brand, l.variant,
      l.hostname, l.geography, l.owner,
    ].join(' ').toLowerCase().includes(q)
  })

  const total = asignadas.length

  async function handleCreate(payload: CreateLaptopPayload) {
    try {
      await createLaptop(payload)
      addToast(`Employee device ${payload.serial} added`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating device'
      addToast(msg, 'danger')
      throw err
    }
  }

  async function handleUpdate(id: string, payload: UpdateLaptopPayload) {
    try {
      await updateLaptop(id, payload)
      addToast('Record updated')
      await load()
    } catch {
      addToast('Error updating', 'danger')
    }
  }

  async function handleDeleteConfirmed() {
    if (!confirmTarget) return
    try {
      await deleteLaptop(confirmTarget.id)
      addToast(`Laptop ${confirmTarget.serial} deleted`, 'danger')
      await load()
      if (selected?.id === confirmTarget.id) setModalOpen(false)
    } catch {
      addToast('Error deleting', 'danger')
    } finally {
      setConfirmTarget(null)
    }
  }

  function openView(l: Laptop) {
    setSelected(l); setModalMode('view'); setModalOpen(true)
  }
  function openEdit(l: Laptop) {
    setSelected(l); setModalMode('edit'); setModalOpen(true)
  }

  return (
    <>
      {/* Top bar */}
      <div className="topbar">
        <div>
          <h2>EPD — Employee Device Inventory</h2>
          <p>
            {canWrite
              ? 'View and manage devices assigned to each project employee.'
              : 'View devices assigned to each project employee.'
            }
          </p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>
            </svg>
            <input
              type="text"
              placeholder="Search by name, email, serial, hostname…"
              value={search}
              onChange={e => setSearch(e.target.value)}
            />
          </div>

          {canWrite && (
            <div style={{ position: 'relative' }}>
              <button className="btn btn-ghost" onClick={() => setExportMenuOpen(p => !p)} title="Import / Export">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                  <path d="M12 18v-6M9 15l3 3 3-3"/>
                </svg>
                Import / Export
              </button>
              {exportMenuOpen && (
                <div style={{ position: 'absolute', right: 0, top: 'calc(100% + 6px)', zIndex: 200, background: 'var(--cds-layer-01)', border: '1px solid var(--cds-border-subtle-01)', borderRadius: 6, minWidth: 220, boxShadow: '0 4px 16px rgba(0,0,0,.12)' }}
                  onMouseLeave={() => setExportMenuOpen(false)}>
                  <div style={{ padding: '6px 16px 4px', fontSize: 11, fontWeight: 600, color: 'var(--cds-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Excel</div>
                  <button onClick={() => { setImportOpen(true); setExportMenuOpen(false) }}
                    style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                    Import / Export Excel
                  </button>
                  <div style={{ height: 1, background: 'var(--cds-border-subtle-01)', margin: '4px 0' }} />
                  <div style={{ padding: '6px 16px 4px', fontSize: 11, fontWeight: 600, color: 'var(--cds-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Export CSV</div>
                  <button onClick={() => { downloadCsvEpd(buildEpdCsv(filtered), todayFilenameEpd('filtered')); setExportMenuOpen(false) }}
                    style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                    Current view ({filtered.length})
                  </button>
                  <button onClick={() => { downloadCsvEpd(buildEpdCsv(asignadas), todayFilenameEpd('all')); setExportMenuOpen(false) }}
                    style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px 12px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                    All assigned ({asignadas.length})
                  </button>
                </div>
              )}
            </div>
          )}
          {canWrite && (
            <button className="btn btn-primary" onClick={() => { setSelected(null); setModalMode('create'); setCreateOpen(true) }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              Add employee
            </button>
          )}
        </div>
      </div>

      {/* Stats — pie charts row */}
      {(() => {
        // --- Chart 1: by country ---
        const countrySlices = [
          { n: countMexico,      l: 'Mexico',       f: 'MEXICO'      as CountryFilter, color: '#0f62fe' },
          { n: countPhilippines, l: 'Philippines',  f: 'PHILIPPINES' as CountryFilter, color: '#24a148' },
          { n: countIndia,       l: 'India',        f: 'INDIA'       as CountryFilter, color: '#f1c21b' },
        ]
        const totalCountry = countMexico + countPhilippines + countIndia
        const cx = 90, cy = 90, r = 70

        let cumAngle = -Math.PI / 2
        const countryPaths = countrySlices.map(s => {
          const pct = totalCountry > 0 ? s.n / totalCountry : 1 / 3
          const angle = Math.min(pct * 2 * Math.PI, 2 * Math.PI - 0.0001)
          const x1 = cx + r * Math.cos(cumAngle)
          const y1 = cy + r * Math.sin(cumAngle)
          cumAngle += angle
          const x2 = cx + r * Math.cos(cumAngle)
          const y2 = cy + r * Math.sin(cumAngle)
          const largeArc = angle > Math.PI ? 1 : 0
          const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`
          return { ...s, d, pct }
        })

        // --- Chart 2: by owner IBM vs USAA ---
        const countIBM  = asignadas.filter(l => l.owner?.toUpperCase() !== 'USAA').length
        const countUSAA = asignadas.filter(l => l.owner?.toUpperCase() === 'USAA').length
        // --- Chart 3: by usage ---
        const countExclusiveIBM    = asignadas.filter(l => l.usage === 'Exclusive IBM').length
        const countIBMClient       = asignadas.filter(l => l.usage === 'IBM Client').length
        const countExclusiveClient = asignadas.filter(l => l.usage === 'Exclusive Client').length
        const countUsageUnassigned = asignadas.filter(l => !l.usage || l.usage === '').length
        const usageSlices = [
          { n: countExclusiveIBM,    l: 'Exclusive IBM',    f: 'Exclusive IBM'    as UsageFilter, color: '#0f62fe' },
          { n: countIBMClient,       l: 'IBM Client',       f: 'IBM Client'       as UsageFilter, color: '#24a148' },
          { n: countExclusiveClient, l: 'Exclusive Client', f: 'Exclusive Client' as UsageFilter, color: '#ff832b' },
          { n: countUsageUnassigned, l: 'Unassigned',       f: 'Unassigned'       as UsageFilter, color: '#8d8d8d' },
        ]
        const totalUsage = countExclusiveIBM + countIBMClient + countExclusiveClient + countUsageUnassigned

        let cumAngleU = -Math.PI / 2
        const usagePaths = usageSlices.map(s => {
          const pct = totalUsage > 0 ? s.n / totalUsage : 0.25
          const angle = Math.min(pct * 2 * Math.PI, 2 * Math.PI - 0.0001)
          const x1 = cx + r * Math.cos(cumAngleU)
          const y1 = cy + r * Math.sin(cumAngleU)
          cumAngleU += angle
          const x2 = cx + r * Math.cos(cumAngleU)
          const y2 = cy + r * Math.sin(cumAngleU)
          const largeArc = angle > Math.PI ? 1 : 0
          const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`
          return { ...s, d, pct }
        })
        const ownerSlices = [
          { n: countIBM,  l: 'IBM',  f: 'IBM'  as OwnerFilter, color: '#0f62fe' },
          { n: countUSAA, l: 'USAA', f: 'USAA' as OwnerFilter, color: '#fa4d56' },
        ]
        const totalOwner = countIBM + countUSAA

        let cumAngleO = -Math.PI / 2
        const ownerPaths = ownerSlices.map(s => {
          const pct = totalOwner > 0 ? s.n / totalOwner : 0.5
          const angle = Math.min(pct * 2 * Math.PI, 2 * Math.PI - 0.0001)
          const x1 = cx + r * Math.cos(cumAngleO)
          const y1 = cy + r * Math.sin(cumAngleO)
          cumAngleO += angle
          const x2 = cx + r * Math.cos(cumAngleO)
          const y2 = cy + r * Math.sin(cumAngleO)
          const largeArc = angle > Math.PI ? 1 : 0
          const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`
          return { ...s, d, pct }
        })

        const dividerStyle: React.CSSProperties = {
          width: 1, alignSelf: 'stretch', background: 'var(--border, #e5e7eb)', flexShrink: 0,
        }

        return (
          <div className="panel" style={{ marginTop: 0, display: 'flex', alignItems: 'stretch', padding: '16px 16px', gap: 0 }}>

            {/* ── Chart 1: by Country ── */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, paddingRight: 16 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                Devices by Country
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <svg width="140" height="140" viewBox="0 0 180 180" style={{ flexShrink: 0 }}>
                  {countryPaths.map(p => (
                    <path
                      key={p.l}
                      d={p.d}
                      fill={p.color}
                      opacity={countryFilter === p.f ? 1 : countryFilter === 'ALL' ? 1 : 0.35}
                      stroke="#fff"
                      strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'opacity .15s' }}
                      onClick={() => setCountryFilter(countryFilter === p.f ? 'ALL' : p.f)}
                    >
                      <title>{p.l}: {p.n} device{p.n !== 1 ? 's' : ''}</title>
                    </path>
                  ))}
                  <circle cx={cx} cy={cy} r={36} fill="var(--surface, #fff)" />
                  <text x={cx} y={cy - 6} textAnchor="middle" fontSize="18" fontWeight="700" fill="var(--text-primary, #1f2328)">{total}</text>
                  <text x={cx} y={cy + 12} textAnchor="middle" fontSize="9" fill="var(--text-muted, #57606a)">assigned</text>
                </svg>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {countryPaths.map(p => (
                    <button
                      key={p.l}
                      onClick={() => setCountryFilter(countryFilter === p.f ? 'ALL' : p.f)}
                      title={`Filter: ${p.l}`}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
                        opacity: countryFilter === p.f ? 1 : countryFilter === 'ALL' ? 1 : 0.45,
                        transition: 'opacity .15s',
                      }}
                    >
                      <span style={{ width: 10, height: 10, borderRadius: '50%', background: p.color, display: 'inline-block', flexShrink: 0 }} />
                      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary, #1f2328)', minWidth: 72, textAlign: 'left' }}>{p.l}</span>
                      <span style={{ fontSize: 12, color: 'var(--text-muted, #57606a)', fontVariantNumeric: 'tabular-nums' }}>
                        {p.n}
                        <span style={{ marginLeft: 3, fontSize: 10 }}>
                          ({totalCountry > 0 ? Math.round(p.pct * 100) : 0}%)
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Divider */}
            <div style={dividerStyle} />

            {/* ── Chart 2: by Owner ── */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, paddingLeft: 16, paddingRight: 16 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                Devices by Owner
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <svg width="140" height="140" viewBox="0 0 180 180" style={{ flexShrink: 0 }}>
                  {ownerPaths.map(p => (
                    <path
                      key={p.l}
                      d={p.d}
                      fill={p.color}
                      opacity={ownerFilter === p.f ? 1 : ownerFilter === 'ALL' ? 1 : 0.35}
                      stroke="#fff"
                      strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'opacity .15s' }}
                      onClick={() => setOwnerFilter(ownerFilter === p.f ? 'ALL' : p.f)}
                    >
                      <title>{p.l}: {p.n} device{p.n !== 1 ? 's' : ''}</title>
                    </path>
                  ))}
                  <circle cx={cx} cy={cy} r={36} fill="var(--surface, #fff)" />
                  <text x={cx} y={cy - 6} textAnchor="middle" fontSize="18" fontWeight="700" fill="var(--text-primary, #1f2328)">{total}</text>
                  <text x={cx} y={cy + 12} textAnchor="middle" fontSize="9" fill="var(--text-muted, #57606a)">assigned</text>
                </svg>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {ownerPaths.map(p => (
                    <button
                      key={p.l}
                      onClick={() => setOwnerFilter(ownerFilter === p.f ? 'ALL' : p.f)}
                      title={`Filter: ${p.l}`}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
                        opacity: ownerFilter === p.f ? 1 : ownerFilter === 'ALL' ? 1 : 0.45,
                        transition: 'opacity .15s',
                      }}
                    >
                      <span style={{ width: 10, height: 10, borderRadius: '50%', background: p.color, display: 'inline-block', flexShrink: 0 }} />
                      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary, #1f2328)', minWidth: 40, textAlign: 'left' }}>{p.l}</span>
                      <span style={{ fontSize: 12, color: 'var(--text-muted, #57606a)', fontVariantNumeric: 'tabular-nums' }}>
                        {p.n}
                        <span style={{ marginLeft: 3, fontSize: 10 }}>
                          ({totalOwner > 0 ? Math.round(p.pct * 100) : 0}%)
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>
  
            {/* Divider */}
            <div style={dividerStyle} />
  
            {/* ── Chart 3: by Usage ── */}
            <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, paddingLeft: 16 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                Devices by Usage
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <svg width="140" height="140" viewBox="0 0 180 180" style={{ flexShrink: 0 }}>
                  {usagePaths.map(p => (
                    <path
                      key={p.l}
                      d={p.d}
                      fill={p.color}
                      opacity={usageFilter === p.f ? 1 : usageFilter === 'ALL' ? 1 : 0.35}
                      stroke="#fff"
                      strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'opacity .15s' }}
                      onClick={() => setUsageFilter(usageFilter === p.f ? 'ALL' : p.f)}
                    >
                      <title>{p.l}: {p.n} device{p.n !== 1 ? 's' : ''}</title>
                    </path>
                  ))}
                  <circle cx={cx} cy={cy} r={36} fill="var(--surface, #fff)" />
                  <text x={cx} y={cy - 6} textAnchor="middle" fontSize="18" fontWeight="700" fill="var(--text-primary, #1f2328)">{total}</text>
                  <text x={cx} y={cy + 12} textAnchor="middle" fontSize="9" fill="var(--text-muted, #57606a)">assigned</text>
                </svg>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {usagePaths.map(p => (
                    <button
                      key={p.l}
                      onClick={() => setUsageFilter(usageFilter === p.f ? 'ALL' : p.f)}
                      title={`Filter: ${p.l}`}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
                        opacity: usageFilter === p.f ? 1 : usageFilter === 'ALL' ? 1 : 0.45,
                        transition: 'opacity .15s',
                      }}
                    >
                      <span style={{ width: 10, height: 10, borderRadius: '50%', background: p.color, display: 'inline-block', flexShrink: 0 }} />
                      <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-primary, #1f2328)', minWidth: 90, textAlign: 'left' }}>{p.l}</span>
                      <span style={{ fontSize: 12, color: 'var(--text-muted, #57606a)', fontVariantNumeric: 'tabular-nums' }}>
                        {p.n}
                        <span style={{ marginLeft: 3, fontSize: 10 }}>
                          ({totalUsage > 0 ? Math.round(p.pct * 100) : 0}%)
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>
  
          </div>
        )
      })()}

      {/* Table */}
      {loading ? (
        <div className="panel"><div className="empty-state">Loading…</div></div>
      ) : error ? (
        <div className="panel"><div className="empty-state">{error}</div></div>
      ) : filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">
            {search
              ? 'No devices match the search.'
              : 'No devices assigned to employees in the inventory.'}
          </div>
        </div>
      ) : (
        <div className="panel">
          <table className="epd-table">
            <thead>
              <tr>
                <th>Employee</th>
                <th>IBM email</th>
                <th>Device</th>
                <th>Hostname</th>
                <th>Location</th>
                <th>Owner</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(l => (
                <tr key={l.id}>

                  {/* Employee */}
                  <td>
                    <div className="row-employee">{l.employee_name}</div>
                    {l.employee_talent_id && (
                      <div className="row-variant">ID: {l.employee_talent_id}</div>
                    )}
                  </td>

                  {/* Email */}
                  <td>
                    <span className={l.employee_email ? 'epd-email' : 'row-employee empty'}>
                      {l.employee_email || '—'}
                    </span>
                  </td>

                  {/* Device: serial + model */}
                  <td>
                    <span
                      className="tag"
                      style={{ cursor: 'pointer' }}
                      onClick={() => openView(l)}
                      title="View device details"
                    >
                      {l.serial}
                    </span>
                    {(l.model && l.model !== 'Unknown') && (
                      <div className="row-model" style={{ marginTop: 2 }}>{[l.brand !== 'Unknown' ? l.brand : '', l.model].filter(Boolean).join(' ')}</div>
                    )}
                    {l.variant && <div className="row-variant">{l.variant}</div>}
                  </td>

                  {/* Hostname */}
                  <td>
                    {l.hostname
                      ? <span className="tag">{l.hostname}</span>
                      : <span className="row-employee empty">—</span>
                    }
                  </td>

                  {/* Location */}
                  <td>
                    <span className={l.geography ? '' : 'row-employee empty'}>
                      {l.geography || '—'}
                    </span>
                  </td>

                  {/* Owner */}
                  <td>{ownerBadge(l.owner)}</td>

                  {/* Actions */}
                  <td>
                    <div className="row-actions">
                      <button
                        className="icon-btn"
                        title="View details"
                        onClick={() => openView(l)}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                          <circle cx="12" cy="12" r="3"/>
                        </svg>
                      </button>
                      {canWrite && (
                        <>
                          <button
                            className="icon-btn"
                            title="Edit"
                            onClick={() => openEdit(l)}
                          >
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M12 20h9"/>
                              <path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                            </svg>
                          </button>
                          <button
                            className="icon-btn danger"
                            title="Delete"
                            onClick={() => setConfirmTarget(l)}
                          >
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M3 6h18"/>
                              <path d="M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/>
                            </svg>
                          </button>
                        </>
                      )}
                    </div>
                  </td>

                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {modalOpen && selected && (
        <LaptopModal
          mode={modalMode}
          laptop={selected}
          role={user?.role ?? 'viewer'}
          onClose={() => setModalOpen(false)}
          onCreate={handleCreate}
          onUpdate={handleUpdate}
          onDelete={l => { setModalOpen(false); setConfirmTarget(l) }}
        />
      )}

      {createOpen && (
        <LaptopModal
          mode="create"
          laptop={null}
          role={user?.role ?? 'viewer'}
          onClose={() => setCreateOpen(false)}
          onCreate={async (payload) => { await handleCreate(payload); setCreateOpen(false) }}
          onUpdate={async () => {}}
          onDelete={() => {}}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete device ${confirmTarget.brand} ${confirmTarget.model} (${confirmTarget.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed}
          onCancel={() => setConfirmTarget(null)}
        />
      )}

      {importOpen && (
        <InventoryImportModal
          view="epd"
          onClose={() => setImportOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — inventory refreshed') }}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
