import { createBrowserRouter, Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import ProtectedRoute from './ProtectedRoute'
import Layout from '../components/Layout'
import Login from '../views/Login'
import ChangePassword from '../views/ChangePassword'
import ComputerInventory from '../views/ComputerInventory'
import Peripherals from '../views/Peripherals'
import NetworkInventory from '../views/NetworkInventory'
import BIOSInventory from '../views/BIOSInventory'
import EPDInventory from '../views/EPDInventory'
import Users from '../views/Users'
import Reports from '../views/Reports'
import Billing from '../views/Billing'
import Enlaces from '../views/Enlaces'

// Guard for /change-password: requires a token but allows must_change_password=true
function ChangePasswordGuard() {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  return <ChangePassword />
}

export const router = createBrowserRouter([
  { path: '/login', element: <Login /> },
  { path: '/change-password', element: <ChangePasswordGuard /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <Layout />,
        children: [
          { path: '/', element: <Navigate to="/computers" replace /> },
          { path: '/computers', element: <ComputerInventory /> },
          { path: '/peripherals', element: <Peripherals /> },
          { path: '/network', element: <NetworkInventory /> },
          { path: '/bios', element: <BIOSInventory /> },
          { path: '/epd', element: <EPDInventory /> },
          { path: '/reports', element: <Reports /> },
          { path: '/billing', element: <Billing /> },
          { path: '/enlaces', element: <Enlaces /> },
          {
            element: <ProtectedRoute requiredRole="admin" />,
            children: [{ path: '/users', element: <Users /> }],
          },
        ],
      },
    ],
  },
])
