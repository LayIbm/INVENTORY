import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import {
  Equipment,
  fetchEquipment, createEquipment, updateEquipment, deleteEquipment,
  CreateEquipmentPayload, UpdateEquipmentPayload,
} from '../hooks/useEquipment'
import EquipmentModal from '../components/EquipmentModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import InventoryImportModal from '../components/InventoryImportModal'

type PeripheralType = 'Mouse' | 'Keyboard' | 'Headset' | 'Cable' | 'Adapter'
type TypeFilter     = 'ALL' | PeripheralType
type AvailFilter    = 'ALL' | 'DISPONIBLE' | 'ASIGNADA' | 'NO_DISPONIBLE' | 'SCRAP'
type ModalMode      = 'view' | 'edit' | 'create'

const PERIPHERAL_TYPES: PeripheralType[] = ['Mouse', 'Keyboard', 'Headset', 'Cable', 'Adapter']

const typeFilters: { k: TypeFilter; l: string }[] = [
  { k: 'ALL',      l: 'All' },
  { k: 'Mouse',    l: 'Mouse' },
  { k: 'Keyboard', l: 'Keyboard' },
  { k: 'Headset',  l: 'Headset' },
  { k: 'Cable',    l: 'Cable' },
  { k: 'Adapter',  l: 'Adapter' },
]

const availFilters: { k: AvailFilter; l: string }[] = [
  { k: 'ALL',           l: 'All' },
  { k: 'DISPONIBLE',    l: 'Available' },
  { k: 'ASIGNADA',      l: 'Assigned' },
  { k: 'NO_DISPONIBLE', l: 'Unavailable' },
  { k: 'SCRAP',         l: 'Scrap' },
]

const typeLabel: Record<PeripheralType, string> = {
  Mouse: 'Mouse', Keyboard: 'Keyboard', Headset: 'Headset', Cable: 'Cable', Adapter: 'Adapter',
}

const typeColor: Record<PeripheralType, string> = {
  Mouse:    'var(--cds-blue-60)',
  Keyboard: 'var(--cds-support-success)',
  Headset:  'var(--cds-purple-60, #8a3ffc)',
  Cable:    'var(--cds-support-warning)',
  Adapter:  'var(--cds-blue-50)',
}

function availBadge(e: Equipment) {
  if (e.availability === 'DISPONIBLE') return <span className="badge ok">Available</span>
  if (e.availability === 'ASIGNADA')   return <span className="badge neutral">Assigned</span>
  if (e.availability === 'SCRAP')      return <span className="badge" style={{background:'#e8e8e8',color:'#525252'}}>Scrap</span>
  return <span className="badge danger">Unavailable</span>
}

function comodatoBadge(e: Equipment) {
  if (e.comodato === 'FIRMADO')   return <span className="badge ok">Signed</span>
  if (e.comodato === 'PENDIENTE') return <span className="badge warn">Pending</span>
  return <span className="badge neutral">N/A</span>
}

export default function Peripherals() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()

  const [items, setItems]             = useState<Equipment[]>([])
  const [search, setSearch]           = useState('')
  const [typeFilter, setTypeFilter]   = useState<TypeFilter>('ALL')
  const [availFilter, setAvailFilter] = useState<AvailFilter>('ALL')

  const [modalMode, setModalMode]         = useState<ModalMode>('view')
  const [selected, setSelected]           = useState<Equipment | null>(null)
  const [modalOpen, setModalOpen]         = useState(false)
  const [confirmTarget, setConfirmTarget] = useState<Equipment | null>(null)
  const [importOpen, setImportOpen]       = useState(false)

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const load = useCallback(async () => {
    try {
      const all = await fetchEquipment()
      setItems(all.filter(e => (PERIPHERAL_TYPES as string[]).includes(e.device_type)))
    } catch { addToast('Error loading peripherals', 'danger') }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const filtered = items.filter(e => {
    const matchType  = typeFilter  === 'ALL' ? true : e.device_type === typeFilter
    const matchAvail = availFilter === 'ALL' ? true : e.availability === availFilter
    const q = search.toLowerCase()
    const matchSearch = !q || [
      e.serial, e.model, e.variant, e.brand,
      e.employee_name ?? '', e.employee_email, e.employee_talent_id,
      e.notes, e.device_type, e.availability, e.comodato,
      e.owner, e.hostname, e.geography,
    ].join(' ').toLowerCase().includes(q)
    return matchType && matchAvail && matchSearch
  })

  async function handleCreate(payload: CreateEquipmentPayload) {
    try {
      await createEquipment(payload)
      addToast(`Peripheral ${payload.serial} added`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating peripheral'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleUpdate(id: string, payload: UpdateEquipmentPayload) {
    try {
      await updateEquipment(id, payload)
      addToast('Peripheral updated')
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error updating'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleDeleteConfirmed() {
    if (!confirmTarget) return
    try {
      await deleteEquipment(confirmTarget.id)
      addToast(`Peripheral ${confirmTarget.serial} deleted`, 'danger')
      await load()
      if (selected?.id === confirmTarget.id) setModalOpen(false)
    } catch { addToast('Error deleting peripheral', 'danger') }
    finally { setConfirmTarget(null) }
  }

  const topbarSub = canWrite
    ? 'Manage inventory: mice, keyboards, headsets, cables and adapters.'
    : 'View the peripheral inventory status.'

  return (
    <>
      {/* Topbar */}
      <div className="topbar">
        <div>
          <h2>Peripherals</h2>
          <p>{topbarSub}</p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/>
              <path d="M21 21l-4.3-4.3"/>
            </svg>
            <input type="text" placeholder="Search by serial, model, employee, type…"
              value={search} onChange={e => setSearch(e.target.value)} />
          </div>
          {canWrite && (
            <button className="btn btn-ghost" onClick={() => setImportOpen(true)} title="Import / Export Excel">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                <polyline points="14 2 14 8 20 8"/>
                <path d="M12 18v-6M9 15l3 3 3-3"/>
              </svg>
              Import / Export
            </button>
          )}
          {canWrite && (
            <button className="btn btn-primary" onClick={() => {
              setModalMode('create'); setSelected(null); setModalOpen(true)
            }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              Add peripheral
            </button>
          )}
        </div>
      </div>

      {/* Stats — one card per peripheral type */}
      <div className="stats-row" style={{ gridTemplateColumns: 'repeat(5, 1fr)' }}>
        {PERIPHERAL_TYPES.map(t => {
          const total    = items.filter(e => e.device_type === t).length
          const prestados = items.filter(e => e.device_type === t && e.availability === 'ASIGNADA').length
          return (
            <div key={t} className="stat-card" style={{ '--stat-color': typeColor[t] } as React.CSSProperties}>
              <div className="n">{String(total).padStart(2, '0')}</div>
              <div className="l">{typeLabel[t]}</div>
              <div style={{ fontSize: 11, color: 'var(--cds-text-secondary)', marginTop: 4, fontFamily: 'var(--cds-code-01-font-family, monospace)' }}>
                {prestados > 0 ? `${prestados} assigned` : 'none assigned'}
              </div>
            </div>
          )
        })}
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
          <span className="filter-group-label">Status</span>
          <div className="filter-group-chips">
            {availFilters.map(f => (
              <button key={f.k} className={`chip${availFilter === f.k ? ' active' : ''}`}
                onClick={() => setAvailFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No peripherals match this filter.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="peripherals-table">
            <thead>
              <tr>
                <th>Serial</th>
                <th>Type</th>
                <th>Brand / Model</th>
                <th>Employee</th>
                <th>Availability</th>
                <th>Loan agreement</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(e => (
                <tr key={e.id}>
                  <td>
                    <span className="tag" style={{ cursor: 'pointer' }}
                      onClick={() => { setSelected(e); setModalMode('view'); setModalOpen(true) }}>
                      {e.serial}
                    </span>
                  </td>
                  <td>
                    <span className="badge neutral">{typeLabel[e.device_type as PeripheralType] ?? e.device_type}</span>
                  </td>
                  <td>
                    <div className="row-model">{[e.brand !== 'Unknown' ? e.brand : '', e.model].filter(Boolean).join(' ')}</div>
                    {e.variant && <div className="row-variant">{e.variant}</div>}
                  </td>
                  <td>
                    <span className={`row-employee${e.employee_name ? '' : ' empty'}`}>
                      {e.employee_name ?? 'Unassigned'}
                    </span>
                  </td>
                  <td>{availBadge(e)}</td>
                  <td>{comodatoBadge(e)}</td>
                  <td>
                    <div className="row-actions">
                      <button className="icon-btn" title="View details"
                        onClick={() => { setSelected(e); setModalMode('view'); setModalOpen(true) }}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                          <circle cx="12" cy="12" r="3"/>
                        </svg>
                      </button>
                      {canWrite && (
                        <>
                          <button className="icon-btn" title="Edit"
                            onClick={() => { setSelected(e); setModalMode('edit'); setModalOpen(true) }}>
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                            </svg>
                          </button>
                          <button className="icon-btn danger" title="Delete"
                            onClick={() => setConfirmTarget(e)}>
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

      {modalOpen && (
        <EquipmentModal
          mode={modalMode} equipment={selected} role={user?.role ?? 'viewer'}
          allowedTypes={['Mouse', 'Keyboard', 'Headset', 'Cable', 'Adapter']}
          onClose={() => setModalOpen(false)}
          onCreate={handleCreate} onUpdate={handleUpdate}
          onDelete={e => { setModalOpen(false); setConfirmTarget(e) }}
        />
      )}

      {importOpen && (
        <InventoryImportModal
          view="peripherals"
          onClose={() => setImportOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — inventory refreshed') }}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete peripheral ${confirmTarget.model} (${confirmTarget.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed} onCancel={() => setConfirmTarget(null)}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
