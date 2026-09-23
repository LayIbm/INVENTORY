import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useTheme } from '../context/ThemeContext'
import client from '../api/client'
import ToastContainer from '../components/ToastContainer'
import { useToast } from '../hooks/useToast'
import ibmLogoSrc from '../assets/ibm-logo.svg'

function IBMLogo({ width = 96, color = '#78a9ff' }: { width?: number; color?: string }) {
  return (
    <img
      src={ibmLogoSrc}
      alt="IBM"
      width={width}
      style={{ filter: colorToFilter(color), display: 'block' }}
    />
  )
}

function colorToFilter(color: string): string {
  if (color === '#78a9ff') return 'brightness(0) saturate(100%) invert(65%) sepia(50%) saturate(500%) hue-rotate(190deg) brightness(105%)'
  if (color === '#0f62fe') return 'brightness(0) saturate(100%) invert(20%) sepia(90%) saturate(2000%) hue-rotate(210deg) brightness(100%)'
  if (color === '#ffffff') return 'brightness(0) invert(1)'
  return 'none'
}

function ThemeToggleButton() {
  const { theme, toggleTheme } = useTheme()
  return (
    <button
      onClick={toggleTheme}
      title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
      style={{
        position: 'absolute',
        top: 'var(--cds-spacing-05)',
        right: 'var(--cds-spacing-05)',
        background: 'transparent',
        border: '1px solid currentColor',
        borderRadius: 'var(--cds-radius-sm)',
        padding: '6px',
        cursor: 'pointer',
        color: 'var(--cds-text-secondary)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        opacity: 0.7,
        zIndex: 10,
      }}
    >
      {theme === 'dark' ? (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <circle cx="12" cy="12" r="4"/>
          <path d="M12 2v2M12 20v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M2 12h2M20 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
        </svg>
      ) : (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <path d="M21 12.79A9 9 0 1111.21 3a7 7 0 009.79 9.79z"/>
        </svg>
      )}
    </button>
  )
}

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const { toasts, addToast } = useToast()

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await client.post('/auth/login', { username, password })
      const data = res.data
      login(data.token, {
        user_id: data.user_id,
        username: data.username,
        name: data.name,
        role: data.role,
        must_change_password: data.must_change_password,
      })
      if (data.must_change_password) {
        navigate('/change-password')
      } else {
        navigate('/')
      }
    } catch {
      addToast('Invalid credentials', 'danger')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="view-login" style={{ position: 'relative' }}>
      <ThemeToggleButton />

      {/* ── Left: dark form panel ── */}
      <div className="login-left">
        <div className="login-logo">
          <IBMLogo width={96} color="#78a9ff" />
        </div>

        <div className="login-form-area">
          <div className="login-card">
            <h1>Sign in</h1>
            <p className="sub">
              Device Inventory Management<br />
              USAA — IBM Project
            </p>

            <form onSubmit={handleSubmit}>
              <div className="field">
                <label htmlFor="u" className="field-required">IBMid / Username</label>
                <input
                  id="u"
                  type="text"
                  placeholder="first.last"
                  autoComplete="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                />
              </div>
              <div className="field">
                <label htmlFor="p" className="field-required">Password</label>
                <input
                  id="p"
                  type="password"
                  placeholder="••••••••"
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>
              <button className="btn btn-primary" type="submit" disabled={loading}>
                {loading ? 'Signing in…' : 'Sign in'}
              </button>
            </form>

            <div className="login-foot">Having trouble? Contact IT.</div>
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
