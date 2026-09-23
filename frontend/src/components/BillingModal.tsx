import { useState } from 'react'
import { Invoice, CreateInvoicePayload, UpdateInvoicePayload } from '../hooks/useBilling'

type Mode = 'view' | 'edit' | 'create'

interface Props {
  mode: Mode
  invoice: Invoice | null
  role: string
  onClose: () => void
  onCreate: (payload: CreateInvoicePayload) => Promise<void>
  onUpdate: (id: string, payload: UpdateInvoicePayload) => Promise<void>
  onDelete: (inv: Invoice) => void
}

function todayStr() {
  const d = new Date()
  return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`
}

const emptyForm: CreateInvoicePayload = {
  number: '',
  client: '',
  concept: '',
  amount: 0,
  status: 'PENDIENTE',
  issue_date: todayStr(),
  due_date: '',
  notes: '',
}

function invoiceToForm(inv: Invoice): CreateInvoicePayload {
  return {
    number: inv.number,
    client: inv.client,
    concept: inv.concept,
    amount: inv.amount,
    status: inv.status,
    issue_date: inv.issue_date,
    due_date: inv.due_date,
    notes: inv.notes,
  }
}

export function statusBadge(inv: Invoice) {
  if (inv.status === 'PAGADA')    return <span className="badge ok">Paid</span>
  if (inv.status === 'PENDIENTE') return <span className="badge warn">Pending</span>
  return <span className="badge danger">Cancelled</span>
}

export default function BillingModal({ mode, invoice, role, onClose, onCreate, onUpdate, onDelete }: Props) {
  const canWrite = role === 'manager' || role === 'admin'
  const [localMode, setLocalMode] = useState<Mode>(mode)
  const [form, setForm] = useState<CreateInvoicePayload>(
    mode === 'create' ? emptyForm : invoiceToForm(invoice!)
  )
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function set<K extends keyof CreateInvoicePayload>(k: K, v: CreateInvoicePayload[K]) {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  async function handleSave() {
    if (!form.number.trim() || !form.client.trim()) {
      setError('Invoice number and client are required')
      return
    }
    setError('')
    setLoading(true)
    try {
      if (localMode === 'create') {
        await onCreate({ ...form, number: form.number.toUpperCase().trim() })
      } else if (invoice) {
        const { number: _n, ...rest } = form
        await onUpdate(invoice.id, rest as UpdateInvoicePayload)
      }
      onClose()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error saving'
      setError(msg)
    } finally {
      setLoading(false)
    }
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
                <h3>{isCreate ? 'New invoice' : `Invoice ${invoice?.number}`}</h3>
                {!isCreate && invoice && statusBadge(invoice)}
              </div>
              <div className="modal-sub">
                {isCreate
                  ? 'Fill in the invoice details.'
                  : invoice?.client
                }
              </div>
            </div>
            <button className="modal-close" onClick={onClose}>&times;</button>
          </div>
        </div>

        <div className="modal-body">
          {error && <p style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>{error}</p>}

          <div className="spec-grid">
            {isCreate && (
              <SpecField
                label="Invoice number"
                value={form.number}
                onChange={(v) => set('number', v)}
                placeholder="e.g. INV-0001"
              />
            )}

            {isView && invoice ? (
              <>
                <SpecView label="Client" value={invoice.client} />
                <SpecView label="Concept" value={invoice.concept || '—'} />
                <SpecView label="Amount" value={`$${invoice.amount.toLocaleString('en-US', { minimumFractionDigits: 2 })}`} />
                <SpecView label="Status" value={statusBadge(invoice)} />
                <SpecView label="Issue date" value={invoice.issue_date || '—'} />
                <SpecView label="Due date" value={invoice.due_date || '—'} />
                <SpecView label="Created by" value={invoice.created_by || '—'} />
              </>
            ) : (
              <>
                <SpecField label="Client" value={form.client} onChange={(v) => set('client', v)} placeholder="Name or company" />
                <SpecField label="Concept" value={form.concept} onChange={(v) => set('concept', v)} placeholder="Service description" />
                <SpecField
                  label="Amount"
                  value={String(form.amount)}
                  onChange={(v) => set('amount', parseFloat(v) || 0)}
                  placeholder="0.00"
                />
                <SpecSelect
                  label="Status"
                  value={form.status}
                  options={['PENDIENTE', 'PAGADA', 'CANCELADA']}
                  labels={['Pending', 'Paid', 'Cancelled']}
                  onChange={(v) => set('status', v as CreateInvoicePayload['status'])}
                />
                <SpecField label="Issue date" value={form.issue_date} onChange={(v) => set('issue_date', v)} placeholder="DD/MM/YYYY" />
                <SpecField label="Due date" value={form.due_date} onChange={(v) => set('due_date', v)} placeholder="DD/MM/YYYY" />
              </>
            )}
          </div>

          <div className="spec-note">
            <div className="l">Notes</div>
            {isView
              ? <p>{invoice?.notes || '—'}</p>
              : <textarea value={form.notes} onChange={(e) => set('notes', e.target.value)} placeholder="Additional notes…" />
            }
          </div>
        </div>

        <div className="modal-foot">
          <div className="left">
            {isView && canWrite && (
              <>
                <button className="btn btn-ghost" onClick={() => setLocalMode('edit')}>Edit</button>
                <button className="btn btn-danger-ghost" onClick={() => invoice && onDelete(invoice)}>Delete</button>
              </>
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
              <button className="btn btn-primary" onClick={handleSave} disabled={loading}>
                {loading ? 'Saving…' : isCreate ? 'Create invoice' : 'Save changes'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function SpecView({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <div className="v">{value}</div>
    </div>
  )
}

function SpecField({
  label, value, onChange, placeholder,
}: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
    </div>
  )
}

function SpecSelect({
  label, value, options, labels, onChange,
}: {
  label: string; value: string; options: string[]; labels?: string[]; onChange: (v: string) => void
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((o, i) => <option key={o} value={o}>{labels?.[i] ?? o}</option>)}
      </select>
    </div>
  )
}
