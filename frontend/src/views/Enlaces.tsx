import { useState, useEffect, useCallback, type CSSProperties } from 'react'
import { useAuth } from '../context/AuthContext'
import {
  Enlace,
  fetchEnlaces,
  createEnlace,
  updateEnlace,
  deleteEnlace,
  CreateEnlacePayload,
  UpdateEnlacePayload,
} from '../hooks/useEnlaces'
import EnlaceModal, { typeBadge } from '../components/EnlaceModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import InventoryImportModal from '../components/InventoryImportModal'

type TypeFilter = 'ALL' | 'Primary' | 'Secondary'
type ModalMode = 'view' | 'edit' | 'create'

const filters: { k: TypeFilter; l: string }[] = [
  { k: 'ALL', l: 'All' },
  { k: 'Primary', l: 'Primary' },
  { k: 'Secondary', l: 'Secondary' },
]

export default function Enlaces() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()
  const [enlaces, setEnlaces] = useState<Enlace[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<TypeFilter>('ALL')

  const [modalMode, setModalMode] = useState<ModalMode>('view')
  const [selectedEnlace, setSelectedEnlace] = useState<Enlace | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const [confirmTarget, setConfirmTarget] = useState<Enlace | null>(null)
  const [importOpen, setImportOpen] = useState(false)

  const load = useCallback(async () => {
    try {
      setEnlaces(await fetchEnlaces())
    } catch {
      addToast('Error loading links', 'danger')
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const filtered = enlaces.filter((enlace) => {
    const matchFilter = filter === 'ALL' ? true : enlace.type === filter
    const q = search.toLowerCase()
    const matchSearch = !q || [
      enlace.company,
      enlace.public_ip,
      enlace.ip,
      enlace.identifier_link,
      enlace.contract_number,
      enlace.client_number,
      enlace.name_contact,
      enlace.email_contact,
    ].join(' ').toLowerCase().includes(q)
    return matchFilter && matchSearch
  })

  const total = enlaces.length
  const primary = enlaces.filter((item) => item.type === 'Primary').length
  const secondary = enlaces.filter((item) => item.type === 'Secondary').length
  const withPublicIp = enlaces.filter((item) => item.public_ip.trim()).length

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  async function handleCreate(payload: CreateEnlacePayload) {
    try {
      await createEnlace(payload)
      addToast(`Link ${payload.company} created`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating link'
      addToast(msg, 'danger')
      throw err
    }
  }

  async function handleUpdate(id: string, payload: UpdateEnlacePayload) {
    try {
      await updateEnlace(id, payload)
      addToast('Link updated')
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error updating link'
      addToast(msg, 'danger')
      throw err
    }
  }

  async function handleDeleteConfirmed() {
    if (!confirmTarget) return
    try {
      await deleteEnlace(confirmTarget.id)
      addToast(`Link ${confirmTarget.company} deleted`, 'danger')
      await load()
      if (selectedEnlace?.id === confirmTarget.id) setModalOpen(false)
    } catch {
      addToast('Error deleting link', 'danger')
    } finally {
      setConfirmTarget(null)
    }
  }

  return (
    <>
      <div className="topbar">
        <div>
          <h2>Links</h2>
          <p>
            {canWrite
              ? 'Manage WAN / ISP links, contacts, and contract details.'
              : 'View the WAN / ISP links inventory.'}
          </p>
        </div>
        <div className="topbar-actions">
          <div className="search-box">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7"/>
              <path d="M21 21l-4.3-4.3"/>
            </svg>
            <input
              type="text"
              placeholder="Search by company, IP, contract, or contact"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
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
              setModalMode('create')
              setSelectedEnlace(null)
              setModalOpen(true)
            }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              Add link
            </button>
          )}
        </div>
      </div>

      <div className="stats-row" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
        {[
          { n: total, l: 'Total links', c: 'var(--cds-blue-60)' },
          { n: primary, l: 'Primary', c: 'var(--cds-support-success)' },
          { n: secondary, l: 'Secondary', c: 'var(--cds-support-warning)' },
          { n: withPublicIp, l: 'With public IP', c: 'var(--cds-purple-60)' },
        ].map((s) => (
          <div key={s.l} className="stat-card" style={{ '--stat-color': s.c } as CSSProperties}>
            <div className="n">{String(s.n).padStart(2, '0')}</div>
            <div className="l">{s.l}</div>
          </div>
        ))}
      </div>

      <div className="filter-row">
        {filters.map((item) => (
          <button key={item.k} className={`chip${filter === item.k ? ' active' : ''}`} onClick={() => setFilter(item.k)}>
            {item.l}
          </button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No links match this filter.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="links-table">
            <thead>
              <tr>
                <th>Company</th>
                <th>Type</th>
                <th>Public IP</th>
                <th>IP</th>
                <th>Velocity</th>
                <th>Contract</th>
                <th>Contact</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((enlace) => (
                <tr key={enlace.id}>
                  <td>
                    <span
                      className="tag"
                      onClick={() => {
                        setSelectedEnlace(enlace)
                        setModalMode('view')
                        setModalOpen(true)
                      }}
                      style={{ cursor: 'pointer' }}
                    >
                      {enlace.company}
                    </span>
                  </td>
                  <td>{typeBadge(enlace.type)}</td>
                  <td style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 13 }}>{enlace.public_ip || '—'}</td>
                  <td style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 13 }}>{enlace.ip || '—'}</td>
                  <td>{enlace.velocity || '—'}</td>
                  <td>
                    <div className="row-model">{enlace.contract_number || '—'}</div>
                    <div className="row-variant">{enlace.end_date_contract || 'No end date'}</div>
                  </td>
                  <td>
                    <div className="row-model">{enlace.name_contact || '—'}</div>
                    <div className="row-variant">{enlace.email_contact || enlace.phone_contact || 'No contact'}</div>
                  </td>
                  <td>
                    <div className="row-actions">
                      <button className="icon-btn" title="View details" onClick={() => {
                        setSelectedEnlace(enlace)
                        setModalMode('view')
                        setModalOpen(true)
                      }}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                          <circle cx="12" cy="12" r="3"/>
                        </svg>
                      </button>
                      {canWrite && (
                        <>
                          <button className="icon-btn" title="Edit" onClick={() => {
                            setSelectedEnlace(enlace)
                            setModalMode('edit')
                            setModalOpen(true)
                          }}>
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M12 20h9"/>
                              <path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                            </svg>
                          </button>
                          <button className="icon-btn danger" title="Delete" onClick={() => setConfirmTarget(enlace)}>
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
        <EnlaceModal
          mode={modalMode}
          enlace={selectedEnlace}
          role={user?.role ?? 'viewer'}
          onClose={() => setModalOpen(false)}
          onCreate={handleCreate}
          onUpdate={handleUpdate}
          onDelete={(enlace) => {
            setModalOpen(false)
            setConfirmTarget(enlace)
          }}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete link ${confirmTarget.company}${confirmTarget.identifier_link ? ` (${confirmTarget.identifier_link})` : ''}? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed}
          onCancel={() => setConfirmTarget(null)}
        />
      )}

      {importOpen && (
        <InventoryImportModal
          view="links"
          onClose={() => setImportOpen(false)}
          onImportDone={() => { void load(); addToast('Import complete — links refreshed') }}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
