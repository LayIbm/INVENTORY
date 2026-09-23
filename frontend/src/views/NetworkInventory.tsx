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

type NetworkType = 'Switch' | 'Router' | 'Firewall' | 'AccessPoint' | 'License' | 'OtherNetwork'
type TypeFilter   = 'ALL' | NetworkType
type ModalMode    = 'view' | 'edit' | 'create'

const NETWORK_TYPES: NetworkType[] = ['Switch', 'Router', 'Firewall', 'AccessPoint', 'License', 'OtherNetwork']

const typeFilters: { k: TypeFilter; l: string }[] = [
  { k: 'ALL',          l: 'All' },
  { k: 'Switch',       l: 'Switch' },
  { k: 'Router',       l: 'Router' },
  { k: 'Firewall',     l: 'Firewall' },
  { k: 'AccessPoint',  l: 'Access Point' },
  { k: 'License',      l: 'License' },
  { k: 'OtherNetwork', l: 'Other network device' },
]

const typeLabel: Record<NetworkType, string> = {
  Switch:       'Switch',
  Router:       'Router',
  Firewall:     'Firewall',
  AccessPoint:  'Access Point',
  License:      'License',
  OtherNetwork: 'Other network device',
}

function availBadge(e: Equipment) {
  if (e.availability === 'DISPONIBLE') return <span className="badge ok">Available</span>
  if (e.availability === 'ASIGNADA')   return <span className="badge neutral">Assigned</span>
  return <span className="badge danger">Unavailable</span>
}

function comodatoBadge(e: Equipment) {
  if (e.comodato === 'FIRMADO')   return <span className="badge ok">Signed</span>
  if (e.comodato === 'PENDIENTE') return <span className="badge warn">Pending</span>
  return <span className="badge neutral">N/A</span>
}

export default function NetworkInventory() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()

  const [items, setItems]           = useState<Equipment[]>([])
  const [loading, setLoading]       = useState(true)
  const [error, setError]           = useState<string | null>(null)
  const [search, setSearch]         = useState('')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('ALL')

  const [modalMode, setModalMode]         = useState<ModalMode>('view')
  const [selected, setSelected]           = useState<Equipment | null>(null)
  const [modalOpen, setModalOpen]         = useState(false)
  const [confirmTarget, setConfirmTarget] = useState<Equipment | null>(null)
  const [importOpen, setImportOpen]       = useState(false)

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const all = await fetchEquipment()
      setItems(all.filter(e => (NETWORK_TYPES as string[]).includes(e.device_type)))
    } catch {
      setError('Error loading network inventory.')
      addToast('Error loading network infrastructure', 'danger')
    } finally {
      setLoading(false)
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const filtered = items.filter(e => {
    const matchType = typeFilter === 'ALL' ? true : e.device_type === typeFilter
    const q = search.toLowerCase()
    const matchSearch = !q || [e.serial, e.brand, e.model, e.employee_name ?? ''].join(' ').toLowerCase().includes(q)
    return matchType && matchSearch
  })

  async function handleCreate(payload: CreateEquipmentPayload) {
    try {
      await createEquipment(payload)
      addToast(`Device ${payload.serial} added`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating device'
      addToast(msg, 'danger'); throw err
    }
  }

  async function handleUpdate(id: string, payload: UpdateEquipmentPayload) {
    try {
      await updateEquipment(id, payload)
      addToast('Device updated')
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
      addToast(`Device ${confirmTarget.serial} deleted`, 'danger')
      await load()
      if (selected?.id === confirmTarget.id) setModalOpen(false)
    } catch { addToast('Error deleting device', 'danger') }
    finally { setConfirmTarget(null) }
  }

  const totalSwitches = items.filter(e => e.device_type === 'Switch').length
  const totalRouters  = items.filter(e => e.device_type === 'Router').length
  const totalOthers   = items.filter(e =>
    e.device_type === 'Firewall' || e.device_type === 'AccessPoint' ||
    e.device_type === 'License'  || e.device_type === 'OtherNetwork'
  ).length

  const topbarSub = canWrite
    ? 'Manage inventory: switches, routers, firewalls and more.'
    : 'View the network infrastructure inventory status.'

  return (
    <>
      {/* Topbar */}
      <div className="topbar">
        <div>
          <h2>Network Infrastructure</h2>
          <p>{topbarSub}</p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/>
              <path d="M21 21l-4.3-4.3"/>
            </svg>
            <input type="text" placeholder="Search by serial, brand, model or employee"
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
              Add device
            </button>
          )}
        </div>
      </div>

      {/* Stats */}
      <div className="stats-row" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
        <div className="stat-card" style={{ '--stat-color': 'var(--cds-blue-60)' } as React.CSSProperties}>
          <div className="n">{String(items.length).padStart(2, '0')}</div>
          <div className="l">Total</div>
        </div>
        <div className="stat-card" style={{ '--stat-color': 'var(--cds-support-success)' } as React.CSSProperties}>
          <div className="n">{String(totalSwitches).padStart(2, '0')}</div>
          <div className="l">Switches</div>
        </div>
        <div className="stat-card" style={{ '--stat-color': 'var(--cds-blue-50)' } as React.CSSProperties}>
          <div className="n">{String(totalRouters).padStart(2, '0')}</div>
          <div className="l">Routers</div>
        </div>
        <div className="stat-card" style={{ '--stat-color': 'var(--cds-support-warning)' } as React.CSSProperties}>
          <div className="n">{String(totalOthers).padStart(2, '0')}</div>
          <div className="l">Other devices</div>
        </div>
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
      </div>

      {/* Loading / Error / Table */}
      {loading ? (
        <div className="panel">
          <div className="empty-state">Loading…</div>
        </div>
      ) : error ? (
        <div className="panel">
          <div className="empty-state">{error}</div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No network devices match this filter.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="peripherals-table">
            <thead>
              <tr>
                <th>Serial</th>
                <th>Type</th>
                <th>Brand / Model</th>
                <th>Condition</th>
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
                    <span className="badge neutral">{typeLabel[e.device_type as NetworkType] ?? e.device_type}</span>
                  </td>
                  <td>
                    <div className="row-model">{[e.brand !== 'Unknown' ? e.brand : '', e.model].filter(Boolean).join(' ')}</div>
                    {e.variant && <div className="row-variant">{e.variant}</div>}
                  </td>
                  <td>{e.condition}</td>
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
          context="network"
          onClose={() => setModalOpen(false)}
          onCreate={handleCreate} onUpdate={handleUpdate}
          onDelete={e => { setModalOpen(false); setConfirmTarget(e) }}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete device ${confirmTarget.model} (${confirmTarget.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed} onCancel={() => setConfirmTarget(null)}
        />
      )}

      {importOpen && (
        <InventoryImportModal
          view="network"
          onClose={() => setImportOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — inventory refreshed') }}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
