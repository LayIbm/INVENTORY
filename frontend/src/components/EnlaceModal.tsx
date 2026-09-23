import { useState, type ReactNode } from 'react'
import { Enlace, CreateEnlacePayload, UpdateEnlacePayload } from '../hooks/useEnlaces'

type Mode = 'view' | 'edit' | 'create'

interface Props {
  mode: Mode
  enlace: Enlace | null
  role: string
  onClose: () => void
  onCreate: (payload: CreateEnlacePayload) => Promise<void>
  onUpdate: (id: string, payload: UpdateEnlacePayload) => Promise<void>
  onDelete: (enlace: Enlace) => void
}

const emptyForm: CreateEnlacePayload = {
  type: 'Primary',
  company: '',
  public_ip: '',
  ip: '',
  velocity: '',
  end_date_contract: '',
  months_of_contract: '',
  additional_service: '',
  contract_number: '',
  client_number: '',
  identifier_link: '',
  name_contact: '',
  phone_contact: '',
  email_contact: '',
  support_phone: '',
  support_clave: '',
  current_po: '',
  comments: '',
}

function enlaceToForm(enlace: Enlace): CreateEnlacePayload {
  return {
    type: enlace.type,
    company: enlace.company,
    public_ip: enlace.public_ip,
    ip: enlace.ip,
    velocity: enlace.velocity,
    end_date_contract: enlace.end_date_contract,
    months_of_contract: enlace.months_of_contract,
    additional_service: enlace.additional_service,
    contract_number: enlace.contract_number,
    client_number: enlace.client_number,
    identifier_link: enlace.identifier_link,
    name_contact: enlace.name_contact,
    phone_contact: enlace.phone_contact,
    email_contact: enlace.email_contact,
    support_phone: enlace.support_phone,
    support_clave: enlace.support_clave,
    current_po: enlace.current_po,
    comments: enlace.comments,
  }
}

export function typeBadge(type: string) {
  if (type === 'Primary') return <span className="badge ok">Primary</span>
  return <span className="badge warn">Secondary</span>
}

export default function EnlaceModal({ mode, enlace, role, onClose, onCreate, onUpdate, onDelete }: Props) {
  const canWrite = role === 'manager' || role === 'admin'
  const [localMode, setLocalMode] = useState<Mode>(mode)
  const [form, setForm] = useState<CreateEnlacePayload>(
    mode === 'create' ? emptyForm : enlaceToForm(enlace!)
  )
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function set<K extends keyof CreateEnlacePayload>(key: K, value: CreateEnlacePayload[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSave() {
    if (!form.company.trim()) {
      setError('Company is required')
      return
    }
    setError('')
    setLoading(true)
    try {
      if (localMode === 'create') {
        await onCreate(form)
      } else if (enlace) {
        await onUpdate(enlace.id, form as UpdateEnlacePayload)
      }
      onClose()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error saving link'
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
      <div className="modal" style={{ maxWidth: 980 }}>
        <div className="modal-head">
          <div className="modal-head-top">
            <div>
              <div className="modal-title-row">
                <h3>{isCreate ? 'New link' : enlace?.company || 'Link details'}</h3>
                {!isCreate && enlace && typeBadge(enlace.type)}
              </div>
              <div className="modal-sub">
                {isCreate
                  ? 'Complete the WAN / ISP link details.'
                  : enlace?.identifier_link || enlace?.public_ip || 'No identifier'}
              </div>
            </div>
            <button className="modal-close" onClick={onClose}>&times;</button>
          </div>
        </div>

        <div className="modal-body">
          {error && <p style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>{error}</p>}

          {isView && enlace ? (
            <div className="spec-grid">
              <SpecView label="Type" value={typeBadge(enlace.type)} />
              <SpecView label="Company" value={enlace.company || '—'} />
              <SpecView label="Public IP" value={enlace.public_ip || '—'} />
              <SpecView label="IP" value={enlace.ip || '—'} />
              <SpecView label="Velocity" value={enlace.velocity || '—'} />
              <SpecView label="End Date Contract" value={enlace.end_date_contract || '—'} />
              <SpecView label="Months of Contract" value={enlace.months_of_contract || '—'} />
              <SpecView label="Additional Services" value={enlace.additional_service || '—'} />
              <SpecView label="Contract Number" value={enlace.contract_number || '—'} />
              <SpecView label="Client Number" value={enlace.client_number || '—'} />
              <SpecView label="Identifier Link" value={enlace.identifier_link || '—'} />
              <SpecView label="Name Contact" value={enlace.name_contact || '—'} />
              <SpecView label="Phone Contact" value={enlace.phone_contact || '—'} />
              <SpecView label="Email Contact" value={enlace.email_contact || '—'} />
              <SpecView label="Support Phone" value={enlace.support_phone || '—'} />
              <SpecView label="Support Clave" value={enlace.support_clave || '—'} />
              <SpecView label="Current PO" value={enlace.current_po || '—'} />
              <SpecView label="Created by" value={enlace.created_by || '—'} />
            </div>
          ) : (
            <div className="spec-grid">
              <SpecSelect
                label="Type"
                value={form.type}
                options={['Primary', 'Secondary']}
                onChange={(v) => set('type', v as CreateEnlacePayload['type'])}
              />
              <SpecField label="Company" value={form.company} onChange={(v) => set('company', v)} placeholder="Provider" />
              <SpecField label="Public IP" value={form.public_ip} onChange={(v) => set('public_ip', v)} placeholder="Ex. 189.1.1.10" />
              <SpecField label="IP" value={form.ip} onChange={(v) => set('ip', v)} placeholder="Ex. 10.10.10.1" />
              <SpecField label="Velocity" value={form.velocity} onChange={(v) => set('velocity', v)} placeholder="Ex. 200 Mbps" />
              <SpecField label="End Date Contract" value={form.end_date_contract} onChange={(v) => set('end_date_contract', v)} placeholder="DD/MM/YYYY" />
              <SpecField label="Months of Contract" value={form.months_of_contract} onChange={(v) => set('months_of_contract', v)} placeholder="Ex. 24" />
              <SpecField label="Additional Services" value={form.additional_service} onChange={(v) => set('additional_service', v)} placeholder="Extra services" />
              <SpecField label="Contract Number" value={form.contract_number} onChange={(v) => set('contract_number', v)} placeholder="Contract number" />
              <SpecField label="Client Number" value={form.client_number} onChange={(v) => set('client_number', v)} placeholder="Client number" />
              <SpecField label="Identifier Link" value={form.identifier_link} onChange={(v) => set('identifier_link', v)} placeholder="Link identifier" />
              <SpecField label="Name Contact" value={form.name_contact} onChange={(v) => set('name_contact', v)} placeholder="Contact name" />
              <SpecField label="Phone Contact" value={form.phone_contact} onChange={(v) => set('phone_contact', v)} placeholder="Contact phone" />
              <SpecField label="Email Contact" value={form.email_contact} onChange={(v) => set('email_contact', v)} placeholder="provider@email.com" />
              <SpecField label="Support Phone" value={form.support_phone} onChange={(v) => set('support_phone', v)} placeholder="Support phone" />
              <SpecField label="Support Clave" value={form.support_clave} onChange={(v) => set('support_clave', v)} placeholder="Support PIN or key" />
              <SpecField label="Current PO" value={form.current_po} onChange={(v) => set('current_po', v)} placeholder="Purchase order actual" />
            </div>
          )}

          <div className="spec-note">
            <div className="l">Comments</div>
            {isView
              ? <p>{enlace?.comments || '—'}</p>
              : <textarea value={form.comments} onChange={(e) => set('comments', e.target.value)} placeholder="Additional comments…" />
            }
          </div>
        </div>

        <div className="modal-foot">
          <div className="left">
            {isView && canWrite && (
              <>
                <button className="btn btn-ghost" onClick={() => setLocalMode('edit')}>Edit</button>
                <button className="btn btn-danger-ghost" onClick={() => enlace && onDelete(enlace)}>Delete</button>
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
                {loading ? 'Saving…' : isCreate ? 'Create link' : 'Save changes'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function SpecView({ label, value }: { label: string; value: ReactNode }) {
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
  label, value, options, onChange,
}: {
  label: string; value: string; options: string[]; onChange: (v: string) => void
}) {
  return (
    <div className="spec-item">
      <div className="l">{label}</div>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((option) => <option key={option} value={option}>{option}</option>)}
      </select>
    </div>
  )
}
