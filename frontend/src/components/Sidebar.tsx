import { NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useTheme } from '../context/ThemeContext'
import ibmLogoSrc from '../assets/ibm-logo.svg'

function IBMWordmark({ width = 56 }: { width?: number }) {
  const filter = 'brightness(0) saturate(100%) invert(65%) sepia(50%) saturate(500%) hue-rotate(190deg) brightness(105%)'
  return (
    <img src={ibmLogoSrc} alt="IBM" width={width} style={{ filter, display: 'block' }} />
  )
}

function initials(name: string) {
  const parts = name.trim().split(' ')
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase()
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase()
}

const roleLabels: Record<string, string> = {
  viewer: 'View only',
  manager: 'Manager',
  admin: 'Administrator',
}

function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  return (
    <button
      className="icon-btn"
      onClick={toggleTheme}
      title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
      style={{ marginBottom: 'var(--cds-spacing-03)' }}
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

export default function Sidebar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/login')
  }

  if (!user) return null

  return (
    <aside className="sidebar">
      <div className="brand">
        <IBMWordmark width={48} />
      </div>

      <div className="nav-group-label">General</div>
      <NavLink
        to="/computers"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <rect x="2" y="3" width="20" height="14" rx="1"/>
          <path d="M8 21h8M12 17v4"/>
        </svg>
        Computer Inventory
      </NavLink>
      <NavLink
        to="/peripherals"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <path d="M3 18h18M3 6h18M3 12h18"/>
          <circle cx="8" cy="12" r="2" fill="currentColor" stroke="none"/>
        </svg>
        Peripherals
      </NavLink>
      <NavLink
        to="/network"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <rect x="2" y="14" width="5" height="5" rx="1"/>
          <rect x="9.5" y="5" width="5" height="5" rx="1"/>
          <rect x="17" y="14" width="5" height="5" rx="1"/>
          <path d="M12 10v2.5M4.5 14V12h15v2"/>
        </svg>
        Network Infrastructure
      </NavLink>
      <NavLink
        to="/enlaces"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <path d="M8 12h8"/>
          <path d="M12 8h4a4 4 0 010 8h-4"/>
          <path d="M12 16H8a4 4 0 010-8h4"/>
        </svg>
        Links
      </NavLink>
      <NavLink
        to="/epd"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <circle cx="9" cy="7" r="3"/>
          <path d="M3 21v-1.5A4.5 4.5 0 017.5 15h3A4.5 4.5 0 0115 19.5V21"/>
          <path d="M17 11h4M17 15h4M19 8v10"/>
        </svg>
        EPD by Employee
      </NavLink>
      <NavLink
        to="/bios"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0110 0v4"/>
        </svg>
        BIOS Control
      </NavLink>
      <NavLink
        to="/reports"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" y1="13" x2="8" y2="13"/>
          <line x1="16" y1="17" x2="8" y2="17"/>
          <polyline points="10 9 9 9 8 9"/>
        </svg>
        Reports
      </NavLink>
      <NavLink
        to="/billing"
        className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
          <rect x="2" y="5" width="20" height="14" rx="2"/>
          <line x1="2" y1="10" x2="22" y2="10"/>
          <line x1="6" y1="15" x2="10" y2="15"/>
        </svg>
        Billing
      </NavLink>

      {user.role === 'admin' && (
        <>
          <div className="nav-group-label">Administration</div>
          <NavLink
            to="/users"
            className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
              <circle cx="9" cy="8" r="3.2"/>
              <path d="M2.5 20c0-3.6 2.9-6 6.5-6s6.5 2.4 6.5 6"/>
              <circle cx="18" cy="7" r="2.4"/>
              <path d="M15.8 13.3c2.6.5 4.2 2.4 4.2 5.2"/>
            </svg>
            Users
          </NavLink>
        </>
      )}

      <div className="sidebar-foot">
        <ThemeToggle />
        <div className="user-chip">
          <div className="user-avatar">{initials(user.name)}</div>
          <div className="user-meta">
            <div className="name">{user.name}</div>
            <div className="role">{roleLabels[user.role]}</div>
          </div>
        </div>
        <button className="logout-link" onClick={handleLogout}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
            <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/>
            <path d="M16 17l5-5-5-5"/>
            <path d="M21 12H9"/>
          </svg>
          Sign out
        </button>
      </div>
    </aside>
  )
}
