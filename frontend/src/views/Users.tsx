import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import {
  User, fetchUsers, createUser, updateUser, toggleUserActive, deleteUser,
  CreateUserPayload, UpdateUserPayload,
} from '../hooks/useUsers'
import UserModal from '../components/UserModal'
import ConfirmDialog from '../components/ConfirmDialog'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'

export default function Users() {
  const { user: me } = useAuth()
  const { toasts, addToast } = useToast()
  const [users, setUsers] = useState<User[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [confirmTarget, setConfirmTarget] = useState<User | null>(null)
  const [editUser, setEditUser] = useState<User | null>(null)

  const load = useCallback(async () => {
    try {
      setUsers(await fetchUsers())
    } catch {
      addToast('Error loading users', 'danger')
    }
  }, [addToast])

  useEffect(() => { void load() }, [load])

  async function handleSave(payload: CreateUserPayload | UpdateUserPayload) {
    if (editUser) {
      await updateUser(editUser.id, payload as UpdateUserPayload)
      addToast(`User ${editUser.name} updated`)
    } else {
      const p = payload as CreateUserPayload
      await createUser(p)
      addToast(`User ${p.name} created`)
    }
    await load()
  }

  async function handleToggle(u: User) {
    try {
      await toggleUserActive(u.id)
      addToast(`${u.name} ${u.active ? 'deactivated' : 'activated'}`)
      await load()
    } catch {
      addToast('Error changing status', 'danger')
    }
  }

  async function handleDeleteConfirmed() {
    if (!confirmTarget) return
    try {
      await deleteUser(confirmTarget.id)
      addToast(`User ${confirmTarget.name} deleted`, 'danger')
      await load()
    } catch {
      addToast('Error deleting user', 'danger')
    } finally {
      setConfirmTarget(null)
    }
  }

  const roleLabel: Record<string, string> = { admin: 'Administrator', manager: 'Manager', viewer: 'Viewer' }

  return (
    <>
      <div className="topbar" style={{ marginTop: 36 }}>
        <div>
          <h2 style={{ fontSize: 19 }}>System users</h2>
          <p>Create and manage manager and viewer accounts.</p>
        </div>
        <button className="btn btn-primary" onClick={() => { setEditUser(null); setModalOpen(true) }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          Create user
        </button>
      </div>

      <div className="panel">
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Username</th>
              <th>Role</th>
              <th>Status</th>
              <th style={{ textAlign: 'right' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => {
              const isSelf = u.id === me?.user_id
              return (
                <tr key={u.id}>
                  <td style={{ fontWeight: 600 }}>{u.name}</td>
                  <td>
                    <span style={{ fontFamily: 'var(--mono)', color: 'var(--ink-soft)', fontSize: 12.5 }}>
                      {u.username}
                    </span>
                  </td>
                  <td>
                    <span className={`role-pill ${u.role}`}>{roleLabel[u.role]}</span>
                  </td>
                  <td>
                    {u.active
                      ? <span className="badge ok">Active</span>
                      : <span className="badge neutral">Inactive</span>
                    }
                  </td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="icon-btn"
                        title="Edit"
                        disabled={isSelf}
                        style={isSelf ? { opacity: .35, cursor: 'not-allowed' } : undefined}
                        onClick={() => { setEditUser(u); setModalOpen(true) }}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M12 20h9"/>
                          <path d="M16.5 3.5a2.1 2.1 0 013 3L7 19l-4 1 1-4 12.5-12.5z"/>
                        </svg>
                      </button>
                      <button
                        className="icon-btn"
                        title={u.active ? 'Deactivate' : 'Activate'}
                        disabled={isSelf}
                        style={isSelf ? { opacity: .35, cursor: 'not-allowed' } : undefined}
                        onClick={() => handleToggle(u)}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <circle cx="12" cy="12" r="9"/>
                          <path d="M8 12h8"/>
                        </svg>
                      </button>
                      <button
                        className="icon-btn danger"
                        title="Delete user"
                        disabled={isSelf}
                        style={isSelf ? { opacity: .35, cursor: 'not-allowed' } : undefined}
                        onClick={() => setConfirmTarget(u)}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                          <path d="M3 6h18"/>
                          <path d="M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/>
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {modalOpen && (
        <UserModal
          user={editUser}
          onClose={() => setModalOpen(false)}
          onSave={handleSave}
        />
      )}

      {confirmTarget && (
        <ConfirmDialog
          message={`Delete user "${confirmTarget.name}" (${confirmTarget.username})? This action cannot be undone.`}
          confirmLabel="Delete user"
          onConfirm={handleDeleteConfirmed}
          onCancel={() => setConfirmTarget(null)}
        />
      )}

      <ToastContainer toasts={toasts} />
    </>
  )
}
