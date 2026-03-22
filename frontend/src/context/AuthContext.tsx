import React, { createContext, useContext, useEffect, useState } from 'react'

export interface AuthUser {
  email: string
  name: string
  avatarUrl: string
  provider: string
  role: string
}

interface AuthContextValue {
  user: AuthUser | null
  loading: boolean
  logout: () => Promise<void>
  can: (permission: string) => boolean
}

const ROLE_PERMS: Record<string, string[]> = {
  admin:     ['*'],
  developer: ['view', 'upload'],
  viewer:    ['view'],
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/auth/me')
      .then(res => (res.ok ? res.json() : null))
      .then((data: AuthUser | null) => setUser(data))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  async function logout() {
    await fetch('/auth/logout', {
      method: 'POST',
      headers: { 'X-Requested-With': 'XMLHttpRequest' },
    })
    setUser(null)
    window.location.href = '/login'
  }

  function can(permission: string): boolean {
    if (!user) return false
    const perms = ROLE_PERMS[user.role] ?? []
    return perms.includes('*') || perms.includes(permission)
  }

  return (
    <AuthContext.Provider value={{ user, loading, logout, can }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
