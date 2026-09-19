import React, { createContext, useContext, useEffect, useState, useCallback } from 'react'

export type ThemePref = 'light' | 'dark' | 'system'

interface ThemeContextValue {
  /** what the user picked */
  pref: ThemePref
  /** what is actually applied right now */
  theme: 'light' | 'dark'
  setPref: (p: ThemePref) => void
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)
const STORAGE_KEY = 'fits-theme'
const mq = () => window.matchMedia('(prefers-color-scheme: dark)')

function readPref(): ThemePref {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch { /* storage blocked */ }
  return 'system'
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [pref, setPrefState] = useState<ThemePref>(readPref)
  const [systemDark, setSystemDark] = useState(() => mq().matches)

  useEffect(() => {
    const m = mq()
    const on = (e: MediaQueryListEvent) => setSystemDark(e.matches)
    m.addEventListener('change', on)
    return () => m.removeEventListener('change', on)
  }, [])

  const theme: 'light' | 'dark' = pref === 'system' ? (systemDark ? 'dark' : 'light') : pref

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  const setPref = useCallback((p: ThemePref) => {
    setPrefState(p)
    try { localStorage.setItem(STORAGE_KEY, p) } catch { /* ignore */ }
  }, [])

  const toggleTheme = useCallback(() => setPref(theme === 'dark' ? 'light' : 'dark'), [theme, setPref])

  return (
    <ThemeContext.Provider value={{ pref, theme, setPref, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
