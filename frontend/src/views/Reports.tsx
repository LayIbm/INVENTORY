import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import { Laptop, fetchLaptops } from '../hooks/useLaptops'
import { Equipment, fetchEquipment } from '../hooks/useEquipment'

// ── Types ─────────────────────────────────────────────────────────────────────

type SourceFilter  = 'ALL' | 'Laptop' | 'Desktop' | 'Monitor' | 'Mouse' | 'Keyboard' | 'Headset' | 'Cable' | 'Adapter'
type AvailFilter   = 'ALL' | 'DISPONIBLE' | 'ASIGNADA' | 'NO_DISPONIBLE' | 'SCRAP'
type OwnerFilter   = 'ALL' | 'IBM' | 'USAA'
type CountryFilter = 'ALL' | 'MEXICO' | 'PHILIPPINES' | 'INDIA'
type UsageFilter   = 'ALL' | 'Exclusive IBM' | 'IBM Client' | 'Exclusive Client' | 'Unassigned'

type UnifiedRow =
  | { source: 'laptop';    data: Laptop }
  | { source: 'equipment'; data: Equipment }

// ── Filter definitions ────────────────────────────────────────────────────────

const deviceTypeFilters: { k: SourceFilter; l: string }[] = [
  { k: 'ALL',      l: 'All' },
  { k: 'Laptop',   l: 'Laptops' },
  { k: 'Desktop',  l: 'Tiny Desktops' },
  { k: 'Monitor',  l: 'Monitors' },
  { k: 'Mouse',    l: 'Mouse' },
  { k: 'Keyboard', l: 'Keyboard' },
  { k: 'Headset',  l: 'Headset' },
  { k: 'Cable',    l: 'Cable' },
  { k: 'Adapter',  l: 'Adapter' },
]

const ownerFilters: { k: OwnerFilter; l: string }[] = [
  { k: 'ALL',  l: 'All' },
  { k: 'IBM',  l: 'IBM' },
  { k: 'USAA', l: 'USAA' },
]

const availFilters: { k: AvailFilter; l: string }[] = [
  { k: 'ALL',           l: 'All' },
  { k: 'DISPONIBLE',    l: 'Available' },
  { k: 'ASIGNADA',      l: 'Assigned' },
  { k: 'NO_DISPONIBLE', l: 'Unavailable' },
  { k: 'SCRAP',         l: 'Scrap' },
]

const countryFilters: { k: CountryFilter; l: string }[] = [
  { k: 'ALL',         l: 'All' },
  { k: 'MEXICO',      l: 'Mexico' },
  { k: 'PHILIPPINES', l: 'Philippines' },
  { k: 'INDIA',       l: 'India' },
]

const usageFilters: { k: UsageFilter; l: string }[] = [
  { k: 'ALL',              l: 'All' },
  { k: 'Exclusive IBM',    l: 'Exclusive IBM' },
  { k: 'IBM Client',       l: 'IBM Client' },
  { k: 'Exclusive Client', l: 'Exclusive Client' },
  { k: 'Unassigned',       l: 'Unassigned' },
]

// ── Row helpers ───────────────────────────────────────────────────────────────

function rowDeviceType(r: UnifiedRow): SourceFilter {
  if (r.source === 'laptop') return 'Laptop'
  return r.data.device_type as SourceFilter
}

function rowAvailability(r: UnifiedRow): string {
  return r.data.availability
}

function rowOwner(r: UnifiedRow): string {
  return (r.data.owner ?? '').toUpperCase()
}

function rowCountry(r: UnifiedRow): CountryFilter {
  const g = (r.data.geography ?? '').toLowerCase()
  if (g.includes('mexico'))  return 'MEXICO'
  if (g.includes('philip'))  return 'PHILIPPINES'
  if (g.includes('india'))   return 'INDIA'
  return 'ALL'
}

// ── Badges ────────────────────────────────────────────────────────────────────

function availBadge(avail: string) {
  if (avail === 'DISPONIBLE')    return <span className="badge ok">Available</span>
  if (avail === 'ASIGNADA')      return <span className="badge neutral">Assigned</span>
  if (avail === 'SCRAP')         return <span className="badge" style={{ background: '#e8e8e8', color: '#525252' }}>Scrap</span>
  return <span className="badge danger">Unavailable</span>
}

// ── CSV export ────────────────────────────────────────────────────────────────

function csvEscape(value: string | number | boolean | null | undefined): string {
  const str = value == null ? '' : String(value)
  return `"${str.replace(/"/g, '""')}"`
}

const availLabel: Record<string, string> = {
  DISPONIBLE: 'Available', ASIGNADA: 'Assigned',
  NO_DISPONIBLE: 'Unavailable', SCRAP: 'Scrap',
}
const prepLabel: Record<string, string> = {
  LISTA: 'Ready', NECESITA_PREP: 'Needs prep', NO_FUNCIONAL: 'Non-functional',
}
const comodatoLabel: Record<string, string> = {
  FIRMADO: 'Signed', PENDIENTE: 'Pending', 'N/A': 'N/A',
}

function buildCsv(rows: UnifiedRow[]): string {
  const headers = [
    'Type', 'Serial', 'Brand', 'Model', 'Variant', 'Condition',
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

  const csvRows = rows.map(r => {
    const d = r.data
    const isLaptop = r.source === 'laptop'
    const l = isLaptop ? (d as Laptop) : null
    const prep = isLaptop ? l!.prep : (d as Equipment).assignability
    return [
      csvEscape(isLaptop ? 'Laptop' : (d as Equipment).device_type),
      csvEscape(d.serial),
      csvEscape(d.brand),
      csvEscape(d.model),
      csvEscape(d.variant),
      csvEscape(d.condition),
      csvEscape(availLabel[d.availability] ?? d.availability),
      csvEscape(prepLabel[prep ?? ''] ?? prep),
      csvEscape(comodatoLabel[d.comodato ?? ''] ?? d.comodato),
      csvEscape(d.owner),
      csvEscape(d.geography),
      csvEscape(d.hostname),
      csvEscape(l?.epd_status ?? ''),
      csvEscape(l?.ipv6 ?? ''),
      csvEscape(d.employee_name),
      csvEscape(d.employee_email),
      csvEscape(d.employee_talent_id),
      csvEscape(l?.employee_manager_email ?? ''),
      csvEscape(d.expected_return_date),
      csvEscape(d.os),
      csvEscape(d.powers_on),
      csvEscape(l != null ? (l.win11_ready ? 'Yes' : 'No') : ''),
      csvEscape(d.bios_password),
      csvEscape(d.wifi),
      csvEscape(d.bluetooth),
      csvEscape(l != null ? (l.bluetooth_disabled_bios ? 'Yes' : 'No') : ''),
      csvEscape(String(d.charger_included ? 'Yes' : 'No')),
      csvEscape(l?.lcd_ok ?? ''),
      csvEscape(d.last_format_date),
      csvEscape(l?.last_bios_update ?? ''),
      csvEscape(l?.bios_details ?? ''),
      csvEscape(d.notes),
    ].join(',')
  })

  const lines = [headers.map(csvEscape).join(','), ...csvRows]
  return '\uFEFF' + lines.join('\r\n')
}

function downloadCsv(content: string, filename: string) {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.style.display = 'none'
  document.body.appendChild(a); a.click()
  document.body.removeChild(a); URL.revokeObjectURL(url)
}

function todayFilename(): string {
  const d = new Date()
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `inventory-report-${yyyy}-${mm}-${dd}.csv`
}

// ── Main component ────────────────────────────────────────────────────────────

export default function Reports() {
  const { user } = useAuth()
  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const [laptops,   setLaptops]   = useState<Laptop[]>([])
  const [equipment, setEquipment] = useState<Equipment[]>([])
  const [loading,   setLoading]   = useState(true)
  const [error,     setError]     = useState<string | null>(null)

  // Filters
  const [search,        setSearch]        = useState('')
  const [typeFilter,    setTypeFilter]    = useState<SourceFilter>('ALL')
  const [availFilter,   setAvailFilter]   = useState<AvailFilter>('ALL')
  const [ownerFilter,   setOwnerFilter]   = useState<OwnerFilter>('ALL')
  const [countryFilter, setCountryFilter] = useState<CountryFilter>('ALL')
  const [usageFilter,   setUsageFilter]   = useState<UsageFilter>('ALL')

  const load = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      const [l, e] = await Promise.all([fetchLaptops(), fetchEquipment()])
      setLaptops(l)
      setEquipment(e)
    } catch {
      setError('Could not load inventory. Check your connection and try again.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  // ── Unified rows ─────────────────────────────────────────────────────────

  const COMPUTER_TYPES = new Set(['Desktop', 'Monitor'])
  const PERIPHERAL_TYPES = new Set(['Mouse', 'Keyboard', 'Headset', 'Cable', 'Adapter'])

  const allRows: UnifiedRow[] = [
    ...laptops.map(l => ({ source: 'laptop' as const, data: l })),
    ...equipment
      .filter(e => COMPUTER_TYPES.has(e.device_type) || PERIPHERAL_TYPES.has(e.device_type))
      .map(e => ({ source: 'equipment' as const, data: e })),
  ]

  // ── Filtered rows ─────────────────────────────────────────────────────────

  function matchesSearch(r: UnifiedRow): boolean {
    const q = search.toLowerCase()
    if (!q) return true
    const d = r.data
    return [
      d.serial, d.brand, d.model, d.variant,
      d.employee_name ?? '', d.employee_email, d.employee_talent_id,
      d.owner, d.hostname, d.geography, d.availability, d.notes,
      r.source === 'laptop' ? (d as Laptop).prep : (d as Equipment).assignability,
      r.source === 'equipment' ? (d as Equipment).device_type : 'Laptop',
    ].join(' ').toLowerCase().includes(q)
  }

  const filtered = allRows.filter(r => {
    if (typeFilter    !== 'ALL' && rowDeviceType(r)   !== typeFilter)    return false
    if (ownerFilter   !== 'ALL' && rowOwner(r)        !== ownerFilter)   return false
    if (countryFilter !== 'ALL' && rowCountry(r)      !== countryFilter) return false
    if (availFilter   !== 'ALL' && rowAvailability(r) !== availFilter)   return false
    if (usageFilter !== 'ALL') {
      if (r.source !== 'laptop') return false
      const usage = (r.data as Laptop).usage ?? ''
      if (usageFilter === 'Unassigned' ? (usage !== '') : (usage !== usageFilter)) return false
    }
    return matchesSearch(r)
  })

  // ── Chart base sets ────────────────────────────────────────────────────────
  // Each chart ignores its OWN filter so its slices always reflect a meaningful
  // total, but respects all other active filters + search.

  // Status chart: ignore availFilter → scoped by type + owner + country + search
  const rowsForStatusChart = allRows.filter(r => {
    if (typeFilter    !== 'ALL' && rowDeviceType(r) !== typeFilter)    return false
    if (ownerFilter   !== 'ALL' && rowOwner(r)      !== ownerFilter)   return false
    if (countryFilter !== 'ALL' && rowCountry(r)    !== countryFilter) return false
    return matchesSearch(r)
  })

  // Type chart: ignore typeFilter → scoped by avail + owner + country + search
  const rowsForTypeChart = allRows.filter(r => {
    if (ownerFilter   !== 'ALL' && rowOwner(r)        !== ownerFilter)   return false
    if (countryFilter !== 'ALL' && rowCountry(r)      !== countryFilter) return false
    if (availFilter   !== 'ALL' && rowAvailability(r) !== availFilter)   return false
    return matchesSearch(r)
  })

  const total         = rowsForStatusChart.length
  const countAvail    = rowsForStatusChart.filter(r => rowAvailability(r) === 'DISPONIBLE').length
  const countAssigned = rowsForStatusChart.filter(r => rowAvailability(r) === 'ASIGNADA').length
  const countUnavail  = rowsForStatusChart.filter(r => rowAvailability(r) === 'NO_DISPONIBLE').length
  const countScrap    = rowsForStatusChart.filter(r => rowAvailability(r) === 'SCRAP').length

  // ── Handlers ──────────────────────────────────────────────────────────────

  function handleExport() {
    downloadCsv(buildCsv(filtered), todayFilename())
  }

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <>
      {/* Topbar */}
      <div className="topbar">
        <div>
          <h2>Inventory Reports</h2>
          <p>Full inventory: computers, peripherals and network devices. Filter and export as CSV.</p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>
            </svg>
            <input
              type="text"
              placeholder="Search by serial, model, employee, hostname…"
              value={search}
              onChange={e => setSearch(e.target.value)}
            />
          </div>
          <button
            className="btn btn-primary"
            onClick={handleExport}
            disabled={loading || filtered.length === 0}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
              <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Export CSV ({filtered.length})
          </button>
        </div>
      </div>

      {/* Stats — pie charts */}
      {(() => {
        const cx = 90, cy = 90, r = 70

        function buildPaths<T extends { n: number; color: string }>(slices: T[]) {
          const tot = slices.reduce((s, p) => s + p.n, 0)
          let cum = -Math.PI / 2
          return slices.map(s => {
            const pct = tot > 0 ? s.n / tot : 1 / slices.length
            // SVG arcs can't draw a full 360° — cap at just under 2π
            const angle = Math.min(pct * 2 * Math.PI, 2 * Math.PI - 0.0001)
            const x1 = cx + r * Math.cos(cum)
            const y1 = cy + r * Math.sin(cum)
            cum += angle
            const x2 = cx + r * Math.cos(cum)
            const y2 = cy + r * Math.sin(cum)
            const largeArc = angle > Math.PI ? 1 : 0
            const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`
            return { ...s, d, pct, tot }
          })
        }

        // Chart 1 — by status
        const statusSlices = [
          { n: countAvail,    l: 'Available',   f: 'DISPONIBLE'    as AvailFilter, color: '#24a148' },
          { n: countAssigned, l: 'Assigned',    f: 'ASIGNADA'      as AvailFilter, color: '#0f62fe' },
          { n: countUnavail,  l: 'Unavailable', f: 'NO_DISPONIBLE' as AvailFilter, color: '#da1e28' },
          { n: countScrap,    l: 'Scrap',       f: 'SCRAP'         as AvailFilter, color: '#8d8d8d' },
        ]
        const statusPaths = buildPaths(statusSlices)

        // Chart 2 — by device type (ignores typeFilter, respects all others)
        const countLaptops   = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Laptop').length
        const countDesktops  = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Desktop').length
        const countMonitors  = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Monitor').length
        const countMouse     = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Mouse').length
        const countKeyboard  = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Keyboard').length
        const countHeadset   = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Headset').length
        const countCable     = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Cable').length
        const countAdapter   = rowsForTypeChart.filter(r => rowDeviceType(r) === 'Adapter').length
        const typeSlices = [
          { n: countLaptops,  l: 'Laptops',      f: 'Laptop'   as SourceFilter, color: '#0f62fe' },
          { n: countDesktops, l: 'Tiny Desktops', f: 'Desktop'  as SourceFilter, color: '#24a148' },
          { n: countMonitors, l: 'Monitors',      f: 'Monitor'  as SourceFilter, color: '#8a3ffc' },
          { n: countMouse,    l: 'Mouse',         f: 'Mouse'    as SourceFilter, color: '#ff832b' },
          { n: countKeyboard, l: 'Keyboard',      f: 'Keyboard' as SourceFilter, color: '#f1c21b' },
          { n: countHeadset,  l: 'Headset',       f: 'Headset'  as SourceFilter, color: '#fa4d56' },
          { n: countCable,    l: 'Cable',         f: 'Cable'    as SourceFilter, color: '#08bdba' },
          { n: countAdapter,  l: 'Adapter',       f: 'Adapter'  as SourceFilter, color: '#be95ff' },
        ]
        const typePaths = buildPaths(typeSlices)

        const dividerStyle: React.CSSProperties = {
          width: 1, alignSelf: 'stretch', background: 'var(--border, #e5e7eb)', flexShrink: 0,
        }

        return (
          <div className="panel" style={{ marginTop: 0, marginBottom: 'var(--cds-spacing-06)', display: 'flex', alignItems: 'stretch', padding: '16px 24px', gap: 0 }}>

            {/* ── Chart 1: by Status ── */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, paddingRight: 24 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                Devices by Status
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                <svg width="180" height="180" viewBox="0 0 180 180" style={{ flexShrink: 0 }}>
                  {statusPaths.map(p => (
                    <path key={p.l} d={p.d} fill={p.color}
                      opacity={availFilter === p.f ? 1 : availFilter === 'ALL' ? 1 : 0.35}
                      stroke="#fff" strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'opacity .15s' }}
                      onClick={() => setAvailFilter(availFilter === p.f ? 'ALL' : p.f)}
                    >
                      <title>{p.l}: {p.n} device{p.n !== 1 ? 's' : ''}</title>
                    </path>
                  ))}
                  <circle cx={cx} cy={cy} r={36} fill="var(--surface, #fff)" />
                  <text x={cx} y={cy - 6} textAnchor="middle" fontSize="18" fontWeight="700" fill="var(--text-primary, #1f2328)">{total}</text>
                  <text x={cx} y={cy + 12} textAnchor="middle" fontSize="9" fill="var(--text-muted, #57606a)">total</text>
                </svg>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  {statusPaths.map(p => (
                    <button key={p.l}
                      onClick={() => setAvailFilter(availFilter === p.f ? 'ALL' : p.f)}
                      title={`Filter: ${p.l}`}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
                        opacity: availFilter === p.f ? 1 : availFilter === 'ALL' ? 1 : 0.45,
                        transition: 'opacity .15s',
                      }}
                    >
                      <span style={{ width: 12, height: 12, borderRadius: '50%', background: p.color, display: 'inline-block', flexShrink: 0 }} />
                      <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary, #1f2328)', minWidth: 84, textAlign: 'left' }}>{p.l}</span>
                      <span style={{ fontSize: 13, color: 'var(--text-muted, #57606a)', fontVariantNumeric: 'tabular-nums' }}>
                        {p.n}
                        <span style={{ marginLeft: 4, fontSize: 11 }}>
                          ({p.tot > 0 ? Math.round(p.pct * 100) : 0}%)
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Divider */}
            <div style={dividerStyle} />

            {/* ── Chart 2: by Device Type ── */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, paddingLeft: 24 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                Devices by Type
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
                <svg width="180" height="180" viewBox="0 0 180 180" style={{ flexShrink: 0 }}>
                  {typePaths.map(p => (
                    <path key={p.l} d={p.d} fill={p.color}
                      opacity={typeFilter === p.f ? 1 : typeFilter === 'ALL' ? 1 : 0.35}
                      stroke="#fff" strokeWidth="2"
                      style={{ cursor: 'pointer', transition: 'opacity .15s' }}
                      onClick={() => setTypeFilter(typeFilter === p.f ? 'ALL' : p.f)}
                    >
                      <title>{p.l}: {p.n} device{p.n !== 1 ? 's' : ''}</title>
                    </path>
                  ))}
                  <circle cx={cx} cy={cy} r={36} fill="var(--surface, #fff)" />
                  <text x={cx} y={cy - 6} textAnchor="middle" fontSize="18" fontWeight="700" fill="var(--text-primary, #1f2328)">{total}</text>
                  <text x={cx} y={cy + 12} textAnchor="middle" fontSize="9" fill="var(--text-muted, #57606a)">total</text>
                </svg>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                  {typePaths.map(p => (
                    <button key={p.l}
                      onClick={() => setTypeFilter(typeFilter === p.f ? 'ALL' : p.f)}
                      title={`Filter: ${p.l}`}
                      style={{
                        display: 'flex', alignItems: 'center', gap: 10,
                        background: 'none', border: 'none', cursor: 'pointer', padding: 0,
                        opacity: typeFilter === p.f ? 1 : typeFilter === 'ALL' ? 1 : 0.45,
                        transition: 'opacity .15s',
                      }}
                    >
                      <span style={{ width: 12, height: 12, borderRadius: '50%', background: p.color, display: 'inline-block', flexShrink: 0 }} />
                      <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-primary, #1f2328)', minWidth: 90, textAlign: 'left' }}>{p.l}</span>
                      <span style={{ fontSize: 13, color: 'var(--text-muted, #57606a)', fontVariantNumeric: 'tabular-nums' }}>
                        {p.n}
                        <span style={{ marginLeft: 4, fontSize: 11 }}>
                          ({p.tot > 0 ? Math.round(p.pct * 100) : 0}%)
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

      {/* Filters */}
      <div className="filter-row">
        <div className="filter-group">
          <span className="filter-group-label">Device type</span>
          <div className="filter-group-chips">
            {deviceTypeFilters.map(f => (
              <button key={f.k} className={`chip${typeFilter === f.k ? ' active' : ''}`}
                onClick={() => setTypeFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
        <div className="filter-divider" />
        <div className="filter-group">
          <span className="filter-group-label">Owner</span>
          <div className="filter-group-chips">
            {ownerFilters.map(f => (
              <button key={f.k} className={`chip${ownerFilter === f.k ? ' active' : ''}`}
                onClick={() => setOwnerFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
        <div className="filter-divider" />
        <div className="filter-group">
          <span className="filter-group-label">Status</span>
          <div className="filter-group-chips">
            {availFilters.map(f => (
              <button key={f.k} className={`chip${availFilter === f.k ? ' active' : ''}`}
                onClick={() => setAvailFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
        <div className="filter-divider" />
        <div className="filter-group">
          <span className="filter-group-label">Country</span>
          <div className="filter-group-chips">
            {countryFilters.map(f => (
              <button key={f.k} className={`chip${countryFilter === f.k ? ' active' : ''}`}
                onClick={() => setCountryFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
        <div className="filter-divider" />
        <div className="filter-group">
          <span className="filter-group-label">Usage (laptops)</span>
          <div className="filter-group-chips">
            {usageFilters.map(f => (
              <button key={f.k} className={`chip${usageFilter === f.k ? ' active' : ''}`}
                onClick={() => setUsageFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
      </div>

      {/* Table */}
      {loading ? (
        <div className="panel"><div className="empty-state">Loading inventory…</div></div>
      ) : error ? (
        <div className="panel"><div className="empty-state reports-error">{error}</div></div>
      ) : filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No devices match the current filters.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="reports-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Serial</th>
                <th>Brand / Model</th>
                <th>Owner</th>
                <th>Geography</th>
                <th>Availability</th>
                <th>Employee</th>
                <th>Notes</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(r => {
                const d = r.data
                const typeTag = r.source === 'laptop' ? 'Laptop' : (d as Equipment).device_type
                const employeeName = d.employee_name
                const notesText = d.notes && d.notes.trim().length > 0 ? d.notes : null

                return (
                  <tr key={`${r.source}-${d.id}`}>
                    {/* Type */}
                    <td>
                      <span className="badge neutral">{typeTag}</span>
                    </td>

                    {/* Serial */}
                    <td>
                      <span className="tag">{d.serial}</span>
                    </td>

                    {/* Brand / Model */}
                    <td>
                      <div className="row-model">{[d.brand !== 'Unknown' ? d.brand : '', d.model].filter(Boolean).join(' ')}</div>
                      {d.variant && <div className="row-variant">{d.variant}</div>}
                    </td>

                    {/* Owner */}
                    <td>
                      <span className={`badge ${d.owner?.toUpperCase() === 'USAA' ? 'warn' : 'neutral'}`}>
                        {d.owner || 'IBM'}
                      </span>
                    </td>

                    {/* Geography */}
                    <td>
                      <span className={d.geography ? '' : 'row-employee empty'}>
                        {d.geography || '—'}
                      </span>
                    </td>

                    {/* Availability */}
                    <td>{availBadge(d.availability)}</td>

                    {/* Employee */}
                    <td>
                      <span className={`row-employee${employeeName ? '' : ' empty'}`}>
                        {employeeName ?? 'Unassigned'}
                      </span>
                      {d.employee_email && (
                        <div className="row-variant">{d.employee_email}</div>
                      )}
                    </td>

                    {/* Notes */}
                    <td className="col-notes">
                      {notesText ? (
                        <span className="note-preview-text" title={notesText}>
                          {notesText.length > 60 ? notesText.slice(0, 60) + '…' : notesText}
                        </span>
                      ) : (
                        <span className="report-notes-empty">—</span>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {!canWrite && (
        <p style={{ marginTop: 8, fontSize: 12, color: 'var(--text-muted, #57606a)', paddingLeft: 2 }}>
          Notes are read-only for viewers.
        </p>
      )}
    </>
  )
}
