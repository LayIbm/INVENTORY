import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import { Laptop, fetchLaptops, updateLaptop, deleteLaptop, UpdateLaptopPayload } from '../hooks/useLaptops'
import LaptopModal from '../components/LaptopModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import InventoryImportModal from '../components/InventoryImportModal'

type BIOSFilter = 'ALL' | 'WITH_CHANGE' | 'NO_CHANGE' | 'BT_DISABLED'
type ModalMode  = 'view' | 'edit' | 'create'

const biosFilters: { k: BIOSFilter; l: string }[] = [
  { k: 'ALL',         l: 'All' },
  { k: 'WITH_CHANGE', l: 'Change recorded' },
  { k: 'NO_CHANGE',   l: 'No change (pending)' },
  { k: 'BT_DISABLED', l: 'BT disabled in BIOS' },
]

export default function BIOSInventory() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()

  const [laptops, setLaptops]         = useState<Laptop[]>([])
  const [loading, setLoading]         = useState(true)
  const [error, setError]             = useState<string | null>(null)
  const [search, setSearch]           = useState('')
  const [biosFilter, setBiosFilter]   = useState<BIOSFilter>('ALL')

  const [modalMode, setModalMode]         = useState<ModalMode>('view')
  const [selected, setSelected]           = useState<Laptop | null>(null)
  const [modalOpen, setModalOpen]         = useState(false)
  const [initialTab, setInitialTab]       = useState<'tech' | 'loans' | 'log' | 'bios'>('bios')
  const [confirmTarget, setConfirmTarget] = useState<Laptop | null>(null)
  const [importOpen, setImportOpen] = useState(false)

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setLaptops(await fetchLaptops())
    } catch {
      setError('Error loading laptop inventory.')
      addToast('Error loading laptops', 'danger')
    } finally {
      setLoading(false)
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const filtered = laptops.filter(l => {
    const matchBIOS =
      biosFilter === 'ALL'         ? true :
      biosFilter === 'WITH_CHANGE' ? l.last_bios_update !== '' :
      biosFilter === 'NO_CHANGE'   ? l.last_bios_update === '' :
      /* BT_DISABLED */               l.bluetooth_disabled_bios

    const q = search.toLowerCase()
    const matchSearch = !q || [
      l.serial, l.model, l.variant, l.brand,
      l.employee_name ?? '', l.employee_email, l.employee_talent_id,
      l.bios_password, l.last_bios_update, l.bios_details,
      l.owner, l.hostname, l.geography,
    ].join(' ').toLowerCase().includes(q)

    return matchBIOS && matchSearch
  })

  const total      = laptops.length
  const withChange = laptops.filter(l => l.last_bios_update !== '').length
  const noChange   = laptops.filter(l => l.last_bios_update === '').length
  const btDisabled = laptops.filter(l => l.bluetooth_disabled_bios).length

  async function handleUpdate(id: string, payload: UpdateLaptopPayload) {
    try {
      await updateLaptop(id, payload)
      addToast('BIOS updated')
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
    } catch { addToast('Error deleting', 'danger') }
    finally { setConfirmTarget(null) }
  }

  function openView(l: Laptop) {
    setSelected(l); setModalMode('view'); setInitialTab('bios'); setModalOpen(true)
  }
  function openEdit(l: Laptop) {
    setSelected(l); setModalMode('edit'); setInitialTab('bios'); setModalOpen(true)
  }

  return (
    <>
      <div className="topbar">
        <div>
          <h2>BIOS Control</h2>
          <p>{canWrite
            ? 'Manage BIOS passwords, change dates and Bluetooth status for all laptops.'
            : 'View BIOS status and passwords for inventory laptops.'
          }</p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>
            </svg>
            <input type="text" placeholder="Search by serial, model, employee, password…"
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
        </div>
      </div>

      {/* Stats */}
      <div className="stats-row" style={{ gridTemplateColumns: 'repeat(4,1fr)' }}>
        {[
          { n: total,      l: 'Total laptops',          c: 'var(--cds-blue-60)' },
          { n: withChange, l: 'Change recorded',         c: 'var(--cds-support-success)' },
          { n: noChange,   l: 'No change (pending)',     c: 'var(--cds-support-warning)' },
          { n: btDisabled, l: 'BT disabled in BIOS',    c: 'var(--cds-blue-50)' },
        ].map(s => (
          <div key={s.l} className="stat-card" style={{ '--stat-color': s.c } as React.CSSProperties}>
            <div className="n">{String(s.n).padStart(2, '0')}</div>
            <div className="l">{s.l}</div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="filter-row">
        <div className="filter-group">
          <span className="filter-group-label">BIOS status</span>
          <div className="filter-group-chips">
            {biosFilters.map(f => (
              <button key={f.k} className={`chip${biosFilter === f.k ? ' active' : ''}`}
                onClick={() => setBiosFilter(f.k)}>{f.l}</button>
            ))}
          </div>
        </div>
      </div>

      {/* Table */}
      {loading ? (
        <div className="panel"><div className="empty-state">Loading…</div></div>
      ) : error ? (
        <div className="panel"><div className="empty-state">{error}</div></div>
      ) : filtered.length === 0 ? (
        <div className="panel"><div className="empty-state">No laptops match this filter.</div></div>
      ) : (
        <div className="panel">
          <table className="bios-table">
            <thead>
              <tr>
                <th>Serial</th>
                <th>Model</th>
                <th>Employee</th>
                <th>BIOS password</th>
                <th>BT in BIOS</th>
                <th>IPv6</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(l => (
                <tr key={l.id}>
                  <td>
                    <span className="tag" style={{ cursor: 'pointer' }} onClick={() => openView(l)}>
                      {l.serial}
                    </span>
                  </td>
                  <td>
                    <div className="row-model">{[l.brand !== 'Unknown' ? l.brand : '', l.model].filter(Boolean).join(' ')}</div>
                    {l.variant && <div className="row-variant">{l.variant}</div>}
                  </td>
                  <td>
                    <span className={l.employee_name ? 'row-employee' : 'row-employee empty'}>
                      {l.employee_name ?? 'Unassigned'}
                    </span>
                  </td>
                  <td>
                    <span style={{ fontFamily: 'var(--cds-code-01-font-family, monospace)', fontSize: 12,
                      background: 'var(--cds-layer-02)', padding: '2px 6px', borderRadius: 4,
                      display: 'inline-block', maxWidth: '100%', overflow: 'hidden',
                      textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {l.bios_password || 'NO PASSWORD'}
                    </span>
                  </td>
                  <td>
                    <span className={`badge ${l.bluetooth_disabled_bios ? 'ok' : 'danger'}`}>
                      {l.bluetooth_disabled_bios ? 'Disabled' : 'Enabled'}
                    </span>
                  </td>
                  <td style={{ fontFamily: 'var(--cds-code-01-font-family, monospace)', fontSize: 12,
                    color: l.ipv6 ? 'var(--cds-text-primary)' : 'var(--cds-text-placeholder)',
                    fontStyle: l.ipv6 ? 'normal' : 'italic' }}>
                    {l.ipv6 || '—'}
                  </td>
                  <td>
                    <div className="row-actions">
                      <button className="icon-btn" title="View BIOS details" onClick={() => openView(l)}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                          <circle cx="12" cy="12" r="3"/>
                        </svg>
                      </button>
                      {canWrite && (
                        <>
                          <button className="icon-btn" title="Edit BIOS" onClick={() => openEdit(l)}>
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                            </svg>
                          </button>
                          <button className="icon-btn danger" title="Delete laptop" onClick={() => setConfirmTarget(l)}>
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
          mode={modalMode} laptop={selected} role={user?.role ?? 'viewer'}
          initialTab={initialTab}
          onClose={() => setModalOpen(false)}
          onCreate={async () => {}}
          onUpdate={handleUpdate}
          onDelete={l => { setModalOpen(false); setConfirmTarget(l) }}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete laptop ${confirmTarget.model} (${confirmTarget.serial})? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed} onCancel={() => setConfirmTarget(null)}
        />
      )}

      {importOpen && (
        <InventoryImportModal
          view="bios"
          onClose={() => setImportOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — inventory refreshed') }}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
