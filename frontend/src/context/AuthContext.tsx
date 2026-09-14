import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { authApi } from '../api/endpoints'
import { tokenStore } from '../api/client'
import type { SafeUser, TokenPair } from '../types'

interface AuthContextValue {
  user: SafeUser | null
  isAuthenticated: boolean
  isAdmin: boolean
  isEditor: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  setTokenPair: (pair: TokenPair) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SafeUser | null>(null)

  const setTokenPair = useCallback((pair: TokenPair) => {
    tokenStore.setTokens(pair)
    setUser(pair.user)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const pair = await authApi.login(email, password)
    setTokenPair(pair)
  }, [setTokenPair])

  const logout = useCallback(async () => {
    const rt = tokenStore.getRefreshToken()
    if (rt) {
      try { await authApi.logout(rt) } catch { /* best effort */ }
    }
    tokenStore.clearTokens()
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      isAdmin: user?.role === 'admin',
      isEditor: user?.role === 'admin' || user?.role === 'editor',
      login,
      logout,
      setTokenPair,
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
