import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import {
  Laptop, fetchLaptops, createLaptop, updateLaptop, deleteLaptop,
  CreateLaptopPayload, UpdateLaptopPayload,
} from '../hooks/useLaptops'
import {
  Equipment, fetchEquipment, createEquipment, updateEquipment, deleteEquipment,
  CreateEquipmentPayload, UpdateEquipmentPayload,
} from '../hooks/useEquipment'
import LaptopModal from '../components/LaptopModal'
import EquipmentModal from '../components/EquipmentModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import InventoryImportModal from '../components/InventoryImportModal'
import { useToast } from '../hooks/useToast'

// Unified row discriminated union
type LaptopRow   = { source: 'laptop';    data: Laptop }
type EquipRow    = { source: 'equipment'; data: Equipment }
type UnifiedRow  = LaptopRow | EquipRow

type TypeFilter    = 'ALL' | 'Laptop' | 'Desktop' | 'Monitor'
type AvailFilter   = 'ALL' | 'DISPONIBLE' | 'ASIGNADA' | 'NO_DISPONIBLE' | 'NECESITA_PREP' | 'SCRAP'
type OwnerFilter   = 'ALL' | 'IBM' | 'USAA'
type CountryFilter = 'ALL' | 'MEXICO' | 'PHILIPPINES' | 'INDIA'
type ModalMode     = 'view' | 'edit' | 'create'

const typeFilters: { k: TypeFilter; l: string }[] = [
  { k: 'ALL',     l: 'All' },
  { k: 'Laptop',  l: 'Laptops' },
  { k: 'Desktop', l: 'Tiny Desktops' },
  { k: 'Monitor', l: 'Monitors' },
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
  { k: 'NECESITA_PREP', l: 'Needs prep' },
  { k: 'SCRAP',         l: 'Scrap' },
]

const countryFilters: { k: CountryFilter; l: string }[] = [
  { k: 'ALL',         l: 'All' },
  { k: 'MEXICO',      l: 'Mexico' },
  { k: 'PHILIPPINES', l: 'Philippines' },
  { k: 'INDIA',       l: 'India' },
]

function rowType(r: UnifiedRow): TypeFilter {
  if (r.source === 'laptop') return 'Laptop'
  return r.data.device_type as TypeFilter
}

function rowAvailability(r: UnifiedRow): string {
  return r.data.availability
}

function rowOwner(r: UnifiedRow): string {
  return (r.data.owner ?? '').toUpperCase()
}

function rowCountry(r: UnifiedRow): CountryFilter {
  const geography = (r.data.geography ?? '').toLowerCase()
  if (geography.includes('mexico')) return 'MEXICO'
  if (geography.includes('philip')) return 'PHILIPPINES'
  if (geography.includes('india')) return 'INDIA'
  return 'ALL'
}

function rowSerial(r: UnifiedRow)       { return r.source === 'laptop' ? r.data.serial       : r.data.serial }
function rowBrand(r: UnifiedRow)        { return r.source === 'laptop' ? r.data.brand        : r.data.brand }
function rowModel(r: UnifiedRow)        { return r.source === 'laptop' ? r.data.model        : r.data.model }
function rowVariant(r: UnifiedRow)      { return r.source === 'laptop' ? r.data.variant      : r.data.variant }
function rowEmployee(r: UnifiedRow)     { return r.source === 'laptop' ? r.data.employee_name : r.data.employee_name }
function rowPrep(r: UnifiedRow)         { return r.source === 'laptop' ? r.data.prep         : r.data.assignability }
function rowComodato(r: UnifiedRow)     { return r.source === 'laptop' ? r.data.comodato     : r.data.comodato }

function availBadge(avail: string) {
  if (avail === 'DISPONIBLE')    return <span className="badge ok">Available</span>
  if (avail === 'ASIGNADA')      return <span className="badge neutral">Assigned</span>
  if (avail === 'SCRAP')         return <span className="badge" style={{background:'#e8e8e8',color:'#525252'}}>Scrap</span>
  return <span className="badge danger">Unavailable</span>
}

function prepBadge(prep: string) {
  if (prep === 'LISTA')          return <span className="badge ok">Ready</span>
  if (prep === 'NECESITA_PREP')  return <span className="badge warn">Needs prep.</span>
  return <span className="badge danger">Non-functional</span>
}

function comodatoBadge(c: string) {
  if (c === 'FIRMADO')   return <span className="badge ok">Signed</span>
  if (c === 'PENDIENTE') return <span className="badge warn">Pending</span>
  return <span className="badge neutral">N/A</span>
}

const typeLabel: Record<TypeFilter, string> = {
  ALL: 'All', Laptop: 'Laptop', Desktop: 'Tiny Desktop', Monitor: 'Monitor',
}

export default function ComputerInventory() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()

  const [laptops, setLaptops]     = useState<Laptop[]>([])
  const [equipment, setEquipment] = useState<Equipment[]>([])
  const [search, setSearch]             = useState('')
  const [typeFilter, setTypeFilter]     = useState<TypeFilter>('ALL')
  const [availFilter, setAvailFilter]   = useState<AvailFilter>('ALL')
  const [ownerFilter, setOwnerFilter]   = useState<OwnerFilter>('ALL')
  const [countryFilter, setCountryFilter] = useState<CountryFilter>('ALL')

  // Modal state
  const [modalMode, setModalMode]         = useState<ModalMode>('view')
  const [selectedLaptop, setSelectedLaptop]   = useState<Laptop | null>(null)
  const [selectedEquip, setSelectedEquip]     = useState<Equipment | null>(null)
  const [laptopModalOpen, setLaptopModalOpen] = useState(false)
  const [equipModalOpen, setEquipModalOpen]   = useState(false)

  // Create type picker
  const [createPickerOpen, setCreatePickerOpen] = useState(false)
  const [createEquipType,  setCreateEquipType]  = useState<'Desktop' | 'Monitor'>('Desktop')
  const [exportMenuOpen,   setExportMenuOpen]   = useState(false)

  // Import/Export modal
  const [importModalOpen, setImportModalOpen] = useState(false)

  // Confirm delete
  const [confirmLaptop, setConfirmLaptop]   = useState<Laptop | null>(null)
  const [confirmEquip, setConfirmEquip]     = useState<Equipment | null>(null)

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const load = useCallback(async () => {
    try {
      const [l, e] = await Promise.all([
        fetchLaptops(),
        fetchEquipment(),
      ])
      setLaptops(l)
      setEquipment(e.filter(eq => eq.device_type === 'Desktop' || eq.device_type === 'Monitor'))
    } catch { addToast('Error loading inventory', 'danger') }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  // Build unified rows
  const allRows: UnifiedRow[] = [
    ...laptops.map(l => ({ source: 'laptop' as const, data: l })),
    ...equipment.map(e => ({ source: 'equipment' as const, data: e })),
  ]

  const filtered = allRows.filter(r => {
    const matchType = typeFilter === 'ALL' ? true : rowType(r) === typeFilter
    const matchOwner = ownerFilter === 'ALL' ? true : rowOwner(r) === ownerFilter
    const matchCountry = countryFilter === 'ALL' ? true : rowCountry(r) === countryFilter
    // NECESITA_PREP is a special filter on prep status, not availability
    const matchAvail = availFilter === 'ALL' ? true
      : availFilter === 'NECESITA_PREP'
        ? rowPrep(r) === 'NECESITA_PREP'
        : rowAvailability(r) === availFilter
    const q = search.toLowerCase()
    const searchFields = r.source === 'laptop'
      ? [
          r.data.serial, r.data.model, r.data.variant, r.data.brand,
          r.data.employee_name ?? '', r.data.employee_email,
          r.data.employee_talent_id, r.data.employee_manager_email,
          r.data.bios_password, r.data.os, r.data.notes,
          r.data.availability, r.data.prep, r.data.comodato,
          r.data.owner, r.data.hostname, r.data.geography,
        ]
      : [
          r.data.serial, r.data.model, r.data.variant, r.data.brand,
          r.data.employee_name ?? '', r.data.employee_email,
          r.data.employee_talent_id,
          r.data.bios_password, r.data.os, r.data.notes,
          r.data.availability, r.data.assignability, r.data.comodato,
          r.data.device_type,
          r.data.owner, r.data.hostname, r.data.geography,
        ]
    const matchSearch = !q || searchFields.join(' ').toLowerCase().includes(q)
    return matchType && matchOwner && matchCountry && matchAvail && matchSearch
  })

  const matchesSearch = (r: UnifiedRow) => {
    const q = search.toLowerCase()
    if (!q) return true
    const searchFields = r.source === 'laptop'
      ? [
          r.data.serial, r.data.model, r.data.variant, r.data.brand,
          r.data.employee_name ?? '', r.data.employee_email,
          r.data.employee_talent_id, r.data.employee_manager_email,
          r.data.bios_password, r.data.os, r.data.notes,
          r.data.availability, r.data.prep, r.data.comodato,
          r.data.owner, r.data.hostname, r.data.geography,
        ]
      : [
          r.data.serial, r.data.model, r.data.variant, r.data.brand,
          r.data.employee_name ?? '', r.data.employee_email,
          r.data.employee_talent_id,
          r.data.bios_password, r.data.os, r.data.notes,
          r.data.availability, r.data.assignability, r.data.comodato,
          r.data.device_type,
          r.data.owner, r.data.hostname, r.data.geography,
        ]
    return searchFields.join(' ').toLowerCase().includes(q)
  }

  const rowsByBaseFilters = allRows
    .filter(r => typeFilter === 'ALL' || rowType(r) === typeFilter)
    .filter(r => ownerFilter === 'ALL' || rowOwner(r) === ownerFilter)
    .filter(r => countryFilter === 'ALL' || rowCountry(r) === countryFilter)

  const rowsForTypeCards = allRows
    .filter(r => ownerFilter === 'ALL' || rowOwner(r) === ownerFilter)
    .filter(r => countryFilter === 'ALL' || rowCountry(r) === countryFilter)
    .filter(r => availFilter === 'ALL'
      ? true
      : availFilter === 'NECESITA_PREP'
        ? rowPrep(r) === 'NECESITA_PREP'
        : rowAvailability(r) === availFilter)
    .filter(matchesSearch)

  const total = filtered.length
  const totalLaptops = rowsForTypeCards.filter(r => rowType(r) === 'Laptop').length
  const totalDesktops = rowsForTypeCards.filter(r => rowType(r) === 'Desktop').length
  const totalMonitors = rowsForTypeCards.filter(r => rowType(r) === 'Monitor').length

  // Stats — row 2: by status, scoped to active type + owner + country filters
  const countDisponibles = rowsByBaseFilters.filter(r => rowAvailability(r) === 'DISPONIBLE').length
  const countAsignadas = rowsByBaseFilters.filter(r => rowAvailability(r) === 'ASIGNADA').length
  const countNoDisp = rowsByBaseFilters.filter(r => rowAvailability(r) === 'NO_DISPONIBLE').length
  const countScrap = rowsByBaseFilters.filter(r => rowAvailability(r) === 'SCRAP').length

  // ── Handlers ──────────────────────────────────────────────────────────────

  async function handleCreateLaptop(payload: CreateLaptopPayload) {
    try {
      await createLaptop(payload)
      addToast(`Laptop ${payload.serial} added`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating laptop'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleUpdateLaptop(id: string, payload: UpdateLaptopPayload) {
    try {
      await updateLaptop(id, payload)
      addToast('Laptop updated')
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error updating'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleDeleteLaptopConfirmed() {
    if (!confirmLaptop) return
    try {
      await deleteLaptop(confirmLaptop.id)
      addToast(`Laptop ${confirmLaptop.serial} deleted`, 'danger')
      await load()
      if (selectedLaptop?.id === confirmLaptop.id) setLaptopModalOpen(false)
    } catch { addToast('Error deleting laptop', 'danger') }
    finally { setConfirmLaptop(null) }
  }

  async function handleCreateEquip(payload: CreateEquipmentPayload) {
    try {
      await createEquipment(payload)
      addToast(`Device ${payload.serial} added`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating device'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleUpdateEquip(id: string, payload: UpdateEquipmentPayload) {
    try {
      await updateEquipment(id, payload)
      addToast('Device updated')
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error updating'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleDeleteEquipConfirmed() {
    if (!confirmEquip) return
    try {
      await deleteEquipment(confirmEquip.id)
      addToast(`Device ${confirmEquip.serial} deleted`, 'danger')
      await load()
      if (selectedEquip?.id === confirmEquip.id) setEquipModalOpen(false)
    } catch { addToast('Error deleting device', 'danger') }
    finally { setConfirmEquip(null) }
  }

  function openRow(r: UnifiedRow, mode: ModalMode) {
    if (r.source === 'laptop') {
      setSelectedLaptop(r.data); setModalMode(mode); setLaptopModalOpen(true)
    } else {
      setSelectedEquip(r.data); setModalMode(mode); setEquipModalOpen(true)
    }
  }

  function deleteRow(r: UnifiedRow) {
    if (r.source === 'laptop') setConfirmLaptop(r.data)
    else setConfirmEquip(r.data)
  }

  function handleCreatePick(type: 'Laptop' | 'Desktop' | 'Monitor') {
    setCreatePickerOpen(false)
    if (type === 'Laptop') {
      setSelectedLaptop(null); setModalMode('create'); setLaptopModalOpen(true)
    } else {
      setCreateEquipType(type)
      setSelectedEquip(null); setModalMode('create'); setEquipModalOpen(true)
    }
  }

  const topbarSub = canWrite
    ? 'Manage the inventory: laptops, tiny desktops and monitors.'
    : 'View the current state of the computer equipment inventory.'

  return (
    <>
      {/* Topbar */}
      <div className="topbar">
        <div>
          <h2>Computer Inventory</h2>
          <p>{topbarSub}</p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/>
              <path d="M21 21l-4.3-4.3"/>
            </svg>
            <input type="text" placeholder="Search by serial, model, employee, BIOS, OS…"
              value={search} onChange={e => setSearch(e.target.value)} />
          </div>

          {canWrite && (() => {
            const csvEsc = (v: string | number | boolean | null | undefined) => {
              const s = v == null ? '' : String(v); return `"${s.replace(/"/g, '""')}"`
            }
            const aLbl: Record<string, string> = { DISPONIBLE: 'Available', ASIGNADA: 'Assigned', NO_DISPONIBLE: 'Unavailable', SCRAP: 'Scrap' }
            const pLbl: Record<string, string> = { LISTA: 'Ready', NECESITA_PREP: 'Needs prep', NO_FUNCIONAL: 'Non-functional' }
            const cLbl: Record<string, string> = { FIRMADO: 'Signed', PENDIENTE: 'Pending', 'N/A': 'N/A' }
            const tLabel: Record<string, string> = { Laptop: 'Laptop', Desktop: 'Tiny Desktop', Monitor: 'Monitor' }
            function buildRows(rows: UnifiedRow[]) {
              const headers = [
                'Type','Serial','Brand','Model','Variant','Condition',
                'Availability','Preparation','Comodato',
                'Owner','Geography','Hostname','EPD Status','IPv6',
                'Employee','Employee email','Employee talent ID','Manager email',
                'Expected return date',
                'OS','Powers on','Win11 ready','BIOS password',
                'WiFi','Bluetooth','Bluetooth disabled (BIOS)',
                'Charger included','LCD OK',
                'Last format date','Last BIOS update','BIOS details',
                'Notes',
              ]
              const body = rows.map(r => {
                const d = r.data; const isL = r.source === 'laptop'
                const l = isL ? (d as Laptop) : null
                const prep = isL ? l!.prep : (d as Equipment).assignability
                return [
                  csvEsc(isL ? tLabel['Laptop'] : tLabel[(d as Equipment).device_type] ?? (d as Equipment).device_type),
                  csvEsc(d.serial), csvEsc(d.brand), csvEsc(d.model), csvEsc(d.variant), csvEsc(d.condition),
                  csvEsc(aLbl[d.availability] ?? d.availability),
                  csvEsc(pLbl[prep ?? ''] ?? prep),
                  csvEsc(cLbl[d.comodato ?? ''] ?? d.comodato),
                  csvEsc(d.owner), csvEsc(d.geography), csvEsc(d.hostname),
                  csvEsc(l?.epd_status ?? ''), csvEsc(l?.ipv6 ?? ''),
                  csvEsc(d.employee_name), csvEsc(d.employee_email),
                  csvEsc(d.employee_talent_id),
                  csvEsc(l?.employee_manager_email ?? ''),
                  csvEsc(d.expected_return_date),
                  csvEsc(d.os), csvEsc(d.powers_on),
                  csvEsc(l != null ? (l.win11_ready ? 'Yes' : 'No') : ''),
                  csvEsc(d.bios_password),
                  csvEsc(d.wifi), csvEsc(d.bluetooth),
                  csvEsc(l != null ? (l.bluetooth_disabled_bios ? 'Yes' : 'No') : ''),
                  csvEsc(d.charger_included ? 'Yes' : 'No'),
                  csvEsc(l?.lcd_ok ?? ''),
                  csvEsc(d.last_format_date),
                  csvEsc(l?.last_bios_update ?? ''), csvEsc(l?.bios_details ?? ''),
                  csvEsc(d.notes),
                ].join(',')
              })
              return '\uFEFF' + [headers.map(csvEsc).join(','), ...body].join('\r\n')
            }
            function dl(content: string, name: string) {
              const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
              const url = URL.createObjectURL(blob)
              const a = document.createElement('a'); a.href = url; a.download = name; a.style.display = 'none'
              document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url)
            }
            const today = () => { const d = new Date(); return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}` }
            return (
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
                    <button onClick={() => { setImportModalOpen(true); setExportMenuOpen(false) }}
                      style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                      Import / Export Excel
                    </button>
                    <div style={{ height: 1, background: 'var(--cds-border-subtle-01)', margin: '4px 0' }} />
                    <div style={{ padding: '6px 16px 4px', fontSize: 11, fontWeight: 600, color: 'var(--cds-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Export CSV</div>
                    <button onClick={() => { dl(buildRows(filtered), `computer-inventory-filtered-${today()}.csv`); setExportMenuOpen(false) }}
                      style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                      Current view ({filtered.length})
                    </button>
                    <button onClick={() => { dl(buildRows(allRows), `computer-inventory-all-${today()}.csv`); setExportMenuOpen(false) }}
                      style={{ display: 'block', width: '100%', textAlign: 'left', padding: '8px 16px 12px', background: 'none', border: 'none', fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer' }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'none')}>
                      All devices ({allRows.length})
                    </button>
                  </div>
                )}
              </div>
            )
          })()}
          {canWrite && (
            <div style={{ position: 'relative' }}>
              <button className="btn btn-primary" onClick={() => setCreatePickerOpen(p => !p)}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                  <path d="M12 5v14M5 12h14"/>
                </svg>
                Add device
              </button>
              {createPickerOpen && (
                <div style={{
                  position: 'absolute', right: 0, top: 'calc(100% + 6px)', zIndex: 200,
                  background: 'var(--cds-layer-01)', border: '1px solid var(--cds-border-subtle-01)',
                  borderRadius: 6, minWidth: 170, boxShadow: '0 4px 16px rgba(0,0,0,.12)',
                }}>
                  {(['Laptop', 'Desktop', 'Monitor'] as const).map(t => (
                    <button key={t} onClick={() => handleCreatePick(t)} style={{
                      display: 'block', width: '100%', textAlign: 'left',
                      padding: '10px 16px', background: 'none', border: 'none',
                      fontSize: 13, color: 'var(--cds-text-primary)', cursor: 'pointer',
                    }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--cds-layer-hover-01)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'none')}
                    >
                      {typeLabel[t]}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Stats — row 1: by type */}
      <div style={{ paddingLeft: 4, marginBottom: 6 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.07em' }}>
          Devices
        </span>
      </div>
      <div className="stats-row">
        {[
          { n: total, l: 'Total devices', c: 'var(--cds-blue-60)', onClick: () => setTypeFilter('ALL'), active: typeFilter === 'ALL' },
          { n: totalLaptops, l: 'Laptops', c: 'var(--cds-blue-50)', onClick: () => setTypeFilter(typeFilter === 'Laptop' ? 'ALL' : 'Laptop'), active: typeFilter === 'Laptop' },
          { n: totalDesktops, l: 'Tiny Desktops', c: 'var(--cds-support-success)', onClick: () => setTypeFilter(typeFilter === 'Desktop' ? 'ALL' : 'Desktop'), active: typeFilter === 'Desktop' },
          { n: totalMonitors, l: 'Monitors', c: 'var(--cds-purple-60, #8a3ffc)', onClick: () => setTypeFilter(typeFilter === 'Monitor' ? 'ALL' : 'Monitor'), active: typeFilter === 'Monitor' },
        ].map(s => (
          <div
            key={s.l}
            className="stat-card"
            style={{
              '--stat-color': s.c,
              cursor: 'pointer',
              transition: 'opacity .15s',
              opacity: s.active || typeFilter === 'ALL' ? 1 : 0.55,
            } as React.CSSProperties}
            onClick={s.onClick}
            title={`Filter: ${s.l}`}
          >
            <div className="n">{String(s.n).padStart(2, '0')}</div>
            <div className="l">{s.l}</div>
          </div>
        ))}
      </div>

      {/* Stats — row 2: availability cards */}
      <div style={{ paddingLeft: 4, marginBottom: 6 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted, #57606a)', textTransform: 'uppercase', letterSpacing: '0.07em' }}>
          Status
        </span>
      </div>
      <div className="stats-row" style={{ marginTop: 0, marginBottom: 'var(--cds-spacing-07)' }}>
        {[
          { n: countDisponibles, l: 'Available', c: 'var(--cds-support-success)', filter: 'DISPONIBLE' as AvailFilter },
          { n: countAsignadas, l: 'Assigned', c: 'var(--cds-blue-60)', filter: 'ASIGNADA' as AvailFilter },
          { n: countNoDisp, l: 'Unavailable', c: 'var(--cds-support-error)', filter: 'NO_DISPONIBLE' as AvailFilter },
          { n: countScrap, l: 'Scrap', c: '#8d8d8d', filter: 'SCRAP' as AvailFilter },
        ].map(s => (
          <div key={s.l} className="stat-card"
            style={{ '--stat-color': s.c, cursor: 'pointer', transition: 'opacity .15s',
              opacity: availFilter === s.filter ? 1 : availFilter === 'ALL' ? 1 : 0.5 } as React.CSSProperties}
            onClick={() => setAvailFilter(availFilter === s.filter ? 'ALL' : s.filter)}
            title={`Filter: ${s.l}`}
          >
            <div className="n">{String(s.n).padStart(2, '0')}</div>
            <div className="l">{s.l}</div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="filter-row">
        <div className="filter-group">
          <span className="filter-group-label">Device type</span>
          <div className="filter-group-chips">
            {typeFilters.map(f => (
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
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No devices match this filter.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="computers-table">
            <thead>
              <tr>
                <th>Serial</th>
                <th>Type</th>
                <th>Brand / Model</th>
                <th>Employee</th>
                <th>Availability</th>
                <th>Readiness</th>
                <th>Loan agr.</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(r => {
                const serial   = rowSerial(r)
                const employee = rowEmployee(r)
                return (
                  <tr key={`${r.source}-${serial}`}>
                    <td>
                      <span className="tag" style={{ cursor: 'pointer' }}
                        onClick={() => openRow(r, 'view')}>{serial}</span>
                    </td>
                    <td>
                      <span className="badge neutral">{typeLabel[rowType(r)]}</span>
                    </td>
                    <td>
                      <div className="row-model">{[rowBrand(r) !== 'Unknown' ? rowBrand(r) : '', rowModel(r)].filter(Boolean).join(' ')}</div>
                      {rowVariant(r) && <div className="row-variant">{rowVariant(r)}</div>}
                    </td>
                    <td>
                      <span className={`row-employee${employee ? '' : ' empty'}`}>
                        {employee ?? 'Unassigned'}
                      </span>
                    </td>
                    <td>{availBadge(rowAvailability(r))}</td>
                    <td>{prepBadge(rowPrep(r))}</td>
                    <td>{comodatoBadge(rowComodato(r))}</td>
                    <td>
                      <div className="row-actions">
                        <button className="icon-btn" title="View detail" onClick={() => openRow(r, 'view')}>
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                            <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                            <circle cx="12" cy="12" r="3"/>
                          </svg>
                        </button>
                        {canWrite && (
                          <>
                            <button className="icon-btn" title="Edit" onClick={() => openRow(r, 'edit')}>
                              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                                <path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                              </svg>
                            </button>
                            <button className="icon-btn danger" title="Delete" onClick={() => deleteRow(r)}>
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
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Laptop modal */}
      {laptopModalOpen && (
        <LaptopModal
          mode={modalMode} laptop={selectedLaptop} role={user?.role ?? 'viewer'}
          onClose={() => setLaptopModalOpen(false)}
          onCreate={handleCreateLaptop} onUpdate={handleUpdateLaptop}
          onDelete={l => { setLaptopModalOpen(false); setConfirmLaptop(l) }}
        />
      )}

      {/* Equipment modal (Desktop / Monitor) */}
      {equipModalOpen && (
        <EquipmentModal
          mode={modalMode} equipment={selectedEquip} role={user?.role ?? 'viewer'}
          allowedTypes={modalMode === 'create' ? [createEquipType] : undefined}
          onClose={() => setEquipModalOpen(false)}
          onCreate={handleCreateEquip} onUpdate={handleUpdateEquip}
          onDelete={e => { setEquipModalOpen(false); setConfirmEquip(e) }}
        />
      )}

      {confirmLaptop && (
        <ConfirmDialog
          message={`Delete laptop ${confirmLaptop.model} (${confirmLaptop.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteLaptopConfirmed} onCancel={() => setConfirmLaptop(null)}
        />
      )}
      {confirmEquip && (
        <ConfirmDialog
          message={`Delete device ${confirmEquip.model} (${confirmEquip.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteEquipConfirmed} onCancel={() => setConfirmEquip(null)}
        />
      )}

      {importModalOpen && (
        <InventoryImportModal
          view="computer"
          onClose={() => setImportModalOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — inventory refreshed') }}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
