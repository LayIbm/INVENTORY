import React, { createContext, useContext, useState, useCallback } from 'react'

export interface UserClaims {
  user_id: string
  username: string
  name: string
  role: 'viewer' | 'manager' | 'admin'
  must_change_password: boolean
}

interface AuthState {
  token: string | null
  user: UserClaims | null
}

interface AuthContextValue extends AuthState {
  login: (token: string, user: UserClaims) => void
  logout: () => void
  refreshMustChange: (value: boolean) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

// Module-level token ref so axios interceptor can read it without hooks
let _token: string | null = null
export const getToken = () => _token

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({ token: null, user: null })

  const login = useCallback((token: string, user: UserClaims) => {
    _token = token
    setState({ token, user })
  }, [])

  const logout = useCallback(() => {
    _token = null
    setState({ token: null, user: null })
  }, [])

  const refreshMustChange = useCallback((value: boolean) => {
    setState((prev) =>
      prev.user ? { ...prev, user: { ...prev.user, must_change_password: value } } : prev,
    )
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, login, logout, refreshMustChange }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
