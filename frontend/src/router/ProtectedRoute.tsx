import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

interface Props {
  /** If set, only users with this role (or higher) can access */
  requiredRole?: 'viewer' | 'manager' | 'admin'
}

const roleRank: Record<string, number> = { viewer: 0, manager: 1, admin: 2 }

export default function ProtectedRoute({ requiredRole }: Props) {
  const { token, user } = useAuth()

  if (!token || !user) {
    return <Navigate to="/login" replace />
  }

  if (user.must_change_password) {
    return <Navigate to="/change-password" replace />
  }

  if (requiredRole && roleRank[user.role] < roleRank[requiredRole]) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}
