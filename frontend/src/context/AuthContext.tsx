import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react'
import { authApi, usersApi } from '../api/endpoints'
import { tokenStore, silentRefresh, setSessionExpiredHandler } from '../api/client'
import type { SafeUser, TokenPair } from '../types'

interface AuthContextValue {
  user: SafeUser | null
  /** false while we try to restore the session after a page reload */
  ready: boolean
  isAuthenticated: boolean
  isAdmin: boolean
  isEditor: boolean
  can: (permission: string) => boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  logoutAll: () => Promise<void>
  /** re-read the current user (e.g. after the forced password change) */
  refreshUser: () => Promise<void>
  setTokenPair: (pair: TokenPair) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SafeUser | null>(null)
  const [ready, setReady] = useState(false)

  const setTokenPair = useCallback((pair: TokenPair) => {
    tokenStore.setAccessToken(pair.access_token)
    setUser(pair.user)
  }, [])

  // Restore the session from the httpOnly refresh cookie (so a page reload doesn't log you out)
  useEffect(() => {
    let alive = true
    silentRefresh()
      .then((pair) => { if (alive) setUser(pair.user) })
      .catch(() => { /* no valid session */ })
      .finally(() => { if (alive) setReady(true) })
    setSessionExpiredHandler(() => setUser(null))
    return () => { alive = false }
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    setTokenPair(await authApi.login(email, password))
  }, [setTokenPair])

  const logout = useCallback(async () => {
    try { await authApi.logout() } catch { /* best effort */ }
    tokenStore.clear()
    setUser(null)
  }, [])

  const logoutAll = useCallback(async () => {
    try { await authApi.logoutAll() } catch { /* best effort */ }
    tokenStore.clear()
    setUser(null)
  }, [])

  const refreshUser = useCallback(async () => {
    try {
      const me = await usersApi.me()
      setUser((prev) => (prev ? { ...prev, ...me } : me))
    } catch { /* ignore */ }
  }, [])

  const can = useCallback(
    (p: string) => !!user?.permissions?.includes(p),
    [user],
  )

  return (
    <AuthContext.Provider value={{
      user, ready,
      isAuthenticated: !!user,
      isAdmin: user?.role === 'admin',
      isEditor: user?.role === 'admin' || user?.role === 'editor',
      can, login, logout, logoutAll, refreshUser, setTokenPair,
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
