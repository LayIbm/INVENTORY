import { useState } from 'react'
import { User, CreateUserPayload, UpdateUserPayload } from '../hooks/useUsers'

interface Props {
  user: User | null
  onClose: () => void
  onSave: (payload: CreateUserPayload | UpdateUserPayload) => Promise<void>
}

export default function UserModal({ user: editUser, onClose, onSave }: Props) {
  const isEdit = editUser !== null

  const [name, setName] = useState(editUser?.name ?? '')
  const [username, setUsername] = useState(editUser?.username ?? '')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState(editUser?.role ?? 'viewer')
  const [active, setActive] = useState(editUser?.active ?? true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSave() {
    if (!name.trim()) {
      setError('Full name is required')
      return
    }
    if (!username.trim()) {
      setError('Username is required')
      return
    }
    if (/\s/.test(username)) {
      setError('Username cannot contain spaces')
      return
    }
    if (!isEdit) {
      if (!password) {
        setError('Password is required')
        return
      }
      if (/\s/.test(password)) {
        setError('Password cannot contain spaces')
        return
      }
      if (password.length < 8) {
        setError('Password must be at least 8 characters')
        return
      }
    }

    setLoading(true)
    setError('')
    try {
      if (isEdit) {
        await onSave({ name: name.trim(), username: username.trim(), role, active } as UpdateUserPayload)
      } else {
        await onSave({ name: name.trim(), username: username.trim(), password, role } as CreateUserPayload)
      }
      onClose()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Error saving'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="overlay" style={{ display: 'flex' }}>
      <div className="modal" style={{ maxWidth: 420 }}>
        <div className="modal-head" style={{ paddingBottom: 16 }}>
          <div className="modal-head-top">
            <h3>{isEdit ? 'Edit user' : 'Create user'}</h3>
            <button className="modal-close" onClick={onClose}>&times;</button>
          </div>
          <p className="modal-sub">
            {isEdit ? 'Update the account details.' : 'The user will be required to change their password on first login.'}
          </p>
        </div>
        <div className="modal-body">
          {error && (
            <p style={{ color: 'var(--cds-support-error)', fontSize: 13, marginBottom: 12, display: 'flex', alignItems: 'center', gap: 6 }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flex: 'none' }}>
                <circle cx="12" cy="12" r="10"/><path d="M12 8v4m0 4h.01"/>
              </svg>
              {error}
            </p>
          )}
          <div className="field">
            <label className="field-required">Full name</label>
            <input
              type="text"
              placeholder="e.g. Jane Smith"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="field">
            <label className="field-required">Username</label>
            <input
              type="text"
              placeholder="jane.smith"
              value={username}
              onChange={(e) => setUsername(e.target.value.replace(/\s/g, ''))}
            />
          </div>
          <div className="field">
            <label className={!isEdit ? 'field-required' : undefined}>
              {isEdit ? 'New password (optional)' : 'Temporary password'}
            </label>
            <input
              type="password"
              placeholder={isEdit ? 'Leave blank to keep current' : 'At least 8 characters, no spaces'}
              value={password}
              onChange={(e) => setPassword(e.target.value.replace(/\s/g, ''))}
            />
            {!isEdit && (
              <p style={{ fontSize: '0.75rem', color: 'var(--cds-text-helper)', marginTop: 4 }}>
                At least 8 characters · No spaces · User must change it on first login
              </p>
            )}
          </div>
          <div className="field">
            <label>Role</label>
            <select value={role} onChange={(e) => setRole(e.target.value as 'viewer' | 'manager' | 'admin')}>
              <option value="manager">Manager — can manage the inventory</option>
              <option value="viewer">Viewer — read-only access</option>
            </select>
          </div>
          {isEdit && (
            <div className="field">
              <label>Status</label>
              <select value={active ? 'true' : 'false'} onChange={(e) => setActive(e.target.value === 'true')}>
                <option value="true">Active</option>
                <option value="false">Inactive</option>
              </select>
            </div>
          )}
        </div>
        <div className="modal-foot" style={{ justifyContent: 'flex-end', gap: 8 }}>
          <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" onClick={handleSave} disabled={loading}>
            {loading ? 'Saving…' : isEdit ? 'Save changes' : 'Create user'}
          </button>
        </div>
      </div>
    </div>
  )
}
