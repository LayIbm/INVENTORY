import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import client from '../api/client'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import ibmLogoSrc from '../assets/ibm-logo.svg'

export default function ChangePassword() {
  const { refreshMustChange } = useAuth()
  const navigate = useNavigate()
  const { toasts, addToast } = useToast()

  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (next !== confirm) {
      addToast('New passwords do not match', 'danger')
      return
    }
    if (next.length < 8) {
      addToast('Password must be at least 8 characters', 'danger')
      return
    }
    setLoading(true)
    try {
      await client.post('/auth/change-password', {
        current_password: current,
        new_password: next,
      })
      refreshMustChange(false)
      addToast('Password updated')
      navigate('/')
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        'Error changing password'
      addToast(msg, 'danger')
    } finally {
      setLoading(false)
    }
  }

  const logoFilter = 'brightness(0) saturate(100%) invert(65%) sepia(50%) saturate(500%) hue-rotate(190deg) brightness(105%)'

  return (
    <section className="view-login">

      {/* ── Left: dark form panel ── */}
      <div className="login-left">
        <div className="login-logo">
          <img src={ibmLogoSrc} alt="IBM" width={96} style={{ filter: logoFilter, display: 'block' }} />
        </div>

        <div className="login-form-area">
          <div className="login-card">
            <h1>Change your password</h1>
            <p className="sub">
              This is your first sign-in.<br />
              Set a new password to continue.
            </p>

            <form onSubmit={handleSubmit}>
              <div className="field">
                <label className="field-required">Current password</label>
                <input
                  type="password"
                  placeholder="••••••••"
                  value={current}
                  onChange={(e) => setCurrent(e.target.value)}
                  required
                />
              </div>
              <div className="field">
                <label className="field-required">New password</label>
                <input
                  type="password"
                  placeholder="At least 8 characters"
                  value={next}
                  onChange={(e) => setNext(e.target.value)}
                  required
                />
              </div>
              <div className="field">
                <label className="field-required">Confirm new password</label>
                <input
                  type="password"
                  placeholder="Repeat the new password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  required
                />
              </div>
              <button className="btn btn-primary" type="submit" disabled={loading}>
                {loading ? 'Saving…' : 'Save password'}
              </button>
            </form>
          </div>
        </div>
      </div>

      {/* ── Right: IBM Blue hero panel ── */}
      <div className="login-right">
        <div className="login-hero">
          <h2>
            Device
            <strong>Inventory</strong>
          </h2>
          <p>
            Track, manage, and assign corporate devices
            across the IBM–USAA project team in real time.
          </p>
        </div>
      </div>

      <ToastContainer toasts={toasts} />
    </section>
  )
}
