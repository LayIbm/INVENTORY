import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import {
  Invoice,
  fetchInvoices, createInvoice, updateInvoice, deleteInvoice,
  CreateInvoicePayload, UpdateInvoicePayload,
} from '../hooks/useBilling'
import BillingModal, { statusBadge } from '../components/BillingModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'

type StatusFilter = 'ALL' | 'PENDIENTE' | 'PAGADA' | 'CANCELADA'
type ModalMode = 'view' | 'edit' | 'create'

const filters: { k: StatusFilter; l: string }[] = [
  { k: 'ALL',       l: 'All' },
  { k: 'PENDIENTE', l: 'Pending' },
  { k: 'PAGADA',    l: 'Paid' },
  { k: 'CANCELADA', l: 'Cancelled' },
]

export default function Billing() {
  const { user } = useAuth()
  const { toasts, addToast } = useToast()
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<StatusFilter>('ALL')

  const [modalMode, setModalMode] = useState<ModalMode>('view')
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const [confirmTarget, setConfirmTarget] = useState<Invoice | null>(null)

  const load = useCallback(async () => {
    try {
      setInvoices(await fetchInvoices())
    } catch {
      addToast('Error loading invoices', 'danger')
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  const filtered = invoices.filter((inv) => {
    const matchFilter = filter === 'ALL' ? true : inv.status === filter
    const q = search.toLowerCase()
    const matchSearch = !q || [inv.number, inv.client, inv.concept].join(' ').toLowerCase().includes(q)
    return matchFilter && matchSearch
  })

  const total     = invoices.length
  const pendiente = invoices.filter((i) => i.status === 'PENDIENTE').length
  const pagada    = invoices.filter((i) => i.status === 'PAGADA').length
  const cancelada = invoices.filter((i) => i.status === 'CANCELADA').length

  const canWrite = user?.role === 'manager' || user?.role === 'admin'

  async function handleCreate(payload: CreateInvoicePayload) {
    try {
      await createInvoice(payload)
      addToast(`Invoice ${payload.number} created`)
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error creating invoice'
      addToast(msg, 'danger')
      throw err
    }
  }

  async function handleUpdate(id: string, payload: UpdateInvoicePayload) {
    try {
      await updateInvoice(id, payload)
      addToast('Invoice updated')
      await load()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error updating invoice'
      addToast(msg, 'danger')
      throw err
    }
  }

  async function handleDeleteConfirmed() {
    if (!confirmTarget) return
    try {
      await deleteInvoice(confirmTarget.id)
      addToast(`Invoice ${confirmTarget.number} deleted`, 'danger')
      await load()
      if (selectedInvoice?.id === confirmTarget.id) setModalOpen(false)
    } catch {
      addToast('Error deleting invoice', 'danger')
    } finally {
      setConfirmTarget(null)
    }
  }

  return (
    <>
      <div className="topbar">
        <div>
          <h2>Billing</h2>
          <p>
            {canWrite
              ? 'Manage invoices: create, edit and cancel records.'
              : 'View the billing records.'}
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
              placeholder="Search by number, client or concept"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          {canWrite && (
            <button className="btn btn-primary" onClick={() => {
              setModalMode('create')
              setSelectedInvoice(null)
              setModalOpen(true)
            }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
              New invoice
            </button>
          )}
        </div>
      </div>

      {/* Stats */}
      <div className="stats-row">
        {[
          { n: total,     l: 'Total invoices', c: 'var(--cds-blue-60)' },
          { n: pendiente, l: 'Pending',         c: 'var(--cds-support-warning)' },
          { n: pagada,    l: 'Paid',            c: 'var(--cds-support-success)' },
          { n: cancelada, l: 'Cancelled',       c: 'var(--cds-support-error)' },
        ].map((s) => (
          <div key={s.l} className="stat-card" style={{ '--stat-color': s.c } as React.CSSProperties}>
            <div className="n">{String(s.n).padStart(2, '0')}</div>
            <div className="l">{s.l}</div>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="filter-row">
        {filters.map((f) => (
          <button key={f.k} className={`chip${filter === f.k ? ' active' : ''}`} onClick={() => setFilter(f.k)}>
            {f.l}
          </button>
        ))}
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="panel">
          <div className="empty-state">No invoices match this filter.</div>
        </div>
      ) : (
        <div className="panel">
          <table className="billing-table">
            <thead>
              <tr>
                <th>Number</th>
                <th>Client</th>
                <th>Concept</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Issue date</th>
                <th>Due date</th>
                <th style={{ textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((inv) => (
                <tr key={inv.id}>
                  <td>
                    <span className="tag" onClick={() => { setSelectedInvoice(inv); setModalMode('view'); setModalOpen(true) }} style={{ cursor: 'pointer' }}>
                      {inv.number}
                    </span>
                  </td>
                  <td style={{ fontWeight: 600 }}>{inv.client}</td>
                  <td style={{ color: 'var(--cds-text-secondary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{inv.concept || '—'}</td>
                  <td style={{ fontFamily: "'IBM Plex Mono', monospace", fontSize: 13 }}>
                    ${inv.amount.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </td>
                  <td>{statusBadge(inv)}</td>
                  <td style={{ color: 'var(--cds-text-secondary)', fontSize: 13 }}>{inv.issue_date || '—'}</td>
                  <td style={{ color: 'var(--cds-text-secondary)', fontSize: 13 }}>{inv.due_date || '—'}</td>
                  <td>
                    <div className="row-actions">
                      <button className="icon-btn" title="View detail" onClick={() => { setSelectedInvoice(inv); setModalMode('view'); setModalOpen(true) }}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M1.5 12S5 5 12 5s10.5 7 10.5 7-3.5 7-10.5 7-10.5-7-10.5-7z"/>
                          <circle cx="12" cy="12" r="3"/>
                        </svg>
                      </button>
                      {canWrite && (
                        <>
                          <button className="icon-btn" title="Edit" onClick={() => { setSelectedInvoice(inv); setModalMode('edit'); setModalOpen(true) }}>
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                              <path d="M12 20h9"/>
                              <path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                            </svg>
                          </button>
                          <button className="icon-btn danger" title="Delete" onClick={() => setConfirmTarget(inv)}>
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
        <BillingModal
          mode={modalMode}
          invoice={selectedInvoice}
          role={user?.role ?? 'viewer'}
          onClose={() => setModalOpen(false)}
          onCreate={handleCreate}
          onUpdate={handleUpdate}
          onDelete={(inv) => { setModalOpen(false); setConfirmTarget(inv) }}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete invoice ${confirmTarget.number} (${confirmTarget.client})? This action cannot be undone.`}
          onConfirm={handleDeleteConfirmed}
          onCancel={() => setConfirmTarget(null)}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
