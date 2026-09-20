import { useEffect, useMemo, useRef, useState, FormEvent } from 'react'
import { NavLink, Outlet, Link, useNavigate, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  LayoutDashboard, Files, Cpu, Users, LogOut, Orbit, Shield, Sun, Moon, Monitor,
  Menu, X, Search, ChevronLeft, UserCircle, KeyRound, MonitorSmartphone, CalendarDays, Home,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../../context/AuthContext'
import { useTheme, ThemePref } from '../../context/ThemeContext'
import { useToast } from '../../context/ToastContext'
import { statsApi } from '../../api/endpoints'
import { Avatar } from '../ui'

const navItems = [
  { to: '/dashboard', label: 'داشبورد',       icon: LayoutDashboard },
  { to: '/files',     label: 'فایل‌های FITS',  icon: Files, perm: 'files.view' },
  { to: '/jobs',      label: 'اسکن‌ها',        icon: Cpu, perm: 'jobs.view' },
]
const adminItems = [
  { to: '/users',      label: 'کاربران',  icon: Users, perm: 'users.view' },
  { to: '/audit-logs', label: 'گزارش‌ها', icon: Shield, perm: 'audit.view' },
]
const crumbs: Record<string, string> = {
  dashboard: 'داشبورد', files: 'فایل‌های FITS', jobs: 'اسکن‌ها',
  users: 'کاربران', 'audit-logs': 'گزارش‌ها', profile: 'پروفایل من',
}
const roleLabel: Record<string, string> = { admin: 'مدیر', editor: 'ویرایشگر', viewer: 'بیننده' }

function useClickOutside(ref: React.RefObject<HTMLElement>, onOut: () => void) {
  useEffect(() => {
    const h = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) onOut() }
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') onOut() }
    document.addEventListener('mousedown', h)
    document.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', h)
      document.removeEventListener('keydown', esc)
    }
  }, [ref, onOut])
}

function NavItem({ to, label, icon: Icon, onClick }: { to: string; label: string; icon: typeof Files; onClick?: () => void }) {
  return (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) => clsx(
        'flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-150',
        isActive
          ? 'bg-accent-600 text-white shadow-sm'
          : 'text-text-secondary hover:bg-surface2 hover:text-text',
      )}
    >
      <Icon className="w-[18px] h-[18px] shrink-0" />
      {label}
    </NavLink>
  )
}

function ThemeMenu() {
  const { pref, setPref, theme } = useTheme()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))
  const opts: { v: ThemePref; label: string; icon: typeof Sun }[] = [
    { v: 'light', label: 'روشن', icon: Sun },
    { v: 'dark', label: 'تاریک', icon: Moon },
    { v: 'system', label: 'مطابق سیستم', icon: Monitor },
  ]
  return (
    <div className="relative" ref={ref}>
      <button onClick={() => setOpen((o) => !o)} className="btn-ghost !p-2" aria-label="تغییر تم" aria-expanded={open}>
        {theme === 'dark' ? <Moon className="w-[18px] h-[18px]" /> : <Sun className="w-[18px] h-[18px]" />}
      </button>
      {open && (
        <div className="menu absolute end-0 top-full mt-2 z-50 !min-w-[10rem]">
          {opts.map(({ v, label, icon: Icon }) => (
            <button key={v} className={clsx('menu-item', pref === v && 'bg-accent-500/10 text-accent-600 font-semibold')}
              onClick={() => { setPref(v); setOpen(false) }}>
              <Icon className="w-4 h-4" /> {label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function UserMenu() {
  const { user, logout, logoutAll } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const go = (to: string) => { setOpen(false); navigate(to) }
  const doLogout = async () => { await logout(); navigate('/login', { replace: true }) }
  const doLogoutAll = async () => {
    setOpen(false)
    await logoutAll()
    toast.success('از همه‌ی دستگاه‌ها خارج شدید')
    navigate('/login', { replace: true })
  }

  return (
    <div className="relative" ref={ref}>
      <button onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-2 ps-1 pe-2.5 py-1 rounded-full hover:bg-surface2 transition-colors" aria-expanded={open}>
        <Avatar name={user?.full_name} size={30} />
        <span className="hidden sm:block text-sm font-medium text-text max-w-[9rem] truncate">{user?.full_name}</span>
      </button>
      {open && (
        <div className="menu absolute end-0 top-full mt-2 z-50">
          <div className="px-2.5 py-2 mb-1 border-b border-border">
            <p className="text-sm font-bold text-text truncate">{user?.full_name}</p>
            <p className="text-xs text-text-muted truncate ltr">{user?.email}</p>
            <span className="pill-info mt-1.5">{roleLabel[user?.role ?? ''] ?? user?.role}</span>
          </div>
          <button className="menu-item" onClick={() => go('/profile')}><UserCircle className="w-4 h-4" /> پروفایل من</button>
          <button className="menu-item" onClick={() => go('/profile')}><KeyRound className="w-4 h-4" /> تغییر رمز عبور</button>
          <button className="menu-item" onClick={doLogoutAll}><MonitorSmartphone className="w-4 h-4" /> خروج از همه‌ی دستگاه‌ها</button>
          <div className="my-1 border-t border-border" />
          <button className="menu-item-danger" onClick={doLogout}><LogOut className="w-4 h-4" /> خروج از حساب</button>
        </div>
      )}
    </div>
  )
}

export default function AppLayout() {
  const { user, can } = useAuth()
  const visibleNav = navItems.filter((i) => !i.perm || can(i.perm))
  const visibleAdmin = adminItems.filter((i) => can(i.perm))
  const location = useLocation()
  const navigate = useNavigate()
  const [drawer, setDrawer] = useState(false)
  const [q, setQ] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  useEffect(() => { setDrawer(false) }, [location.pathname])

  // Ctrl/⌘ + K focuses the quick search
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); searchRef.current?.focus() }
    }
    document.addEventListener('keydown', h)
    return () => document.removeEventListener('keydown', h)
  }, [])

  const statsQ = useQuery({ queryKey: ['stats'], queryFn: statsApi.get, refetchInterval: 15_000 })
  const running = statsQ.data?.running_jobs ?? 0

  const trail = useMemo(() => {
    const parts = location.pathname.split('/').filter(Boolean)
    if (parts.length === 0) return []
    const first = crumbs[parts[0]] ?? parts[0]
    const out = [{ label: first, to: '/' + parts[0] }]
    if (parts[1]) out.push({ label: parts[0] === 'files' ? 'جزئیات فایل' : parts[0] === 'jobs' ? 'جزئیات اسکن' : 'جزئیات', to: location.pathname })
    return out
  }, [location.pathname])

  const today = useMemo(() => new Intl.DateTimeFormat('fa-IR', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date()), [])

  const onSearch = (e: FormEvent) => {
    e.preventDefault()
    const v = q.trim()
    navigate(v ? `/files?search=${encodeURIComponent(v)}` : '/files')
    searchRef.current?.blur()
  }

  const sidebar = (
    <>
      <Link to="/dashboard" className="flex items-center gap-2.5 px-4 h-16 shrink-0">
        <span className="w-9 h-9 rounded-xl bg-gradient-to-br from-accent-400 to-accent-700 flex items-center justify-center shadow-sm">
          <Orbit className="w-5 h-5 text-white" />
        </span>
        <span className="leading-tight">
          <span className="block text-sm font-extrabold text-text">FITS Processor</span>
          <span className="block text-[11px] text-text-muted">مدیریت داده‌های رصدی</span>
        </span>
      </Link>
      <nav className="flex-1 overflow-y-auto px-3 py-2 space-y-1">
        {visibleNav.map(({ perm: _p, ...i }) => <NavItem key={i.to} {...i} />)}
        {visibleAdmin.length > 0 && (
          <>
            <p className="px-3 pt-5 pb-1.5 text-[11px] font-semibold text-text-muted">مدیریت</p>
            {visibleAdmin.map(({ perm: _p, ...i }) => <NavItem key={i.to} {...i} />)}
          </>
        )}
      </nav>
      <div className="p-3 border-t border-border shrink-0">
        <div className="flex items-center gap-2.5 p-2 rounded-xl bg-surface2">
          <Avatar name={user?.full_name} size={34} />
          <div className="min-w-0">
            <p className="text-sm font-semibold text-text truncate">{user?.full_name}</p>
            <p className="text-xs text-text-muted">{roleLabel[user?.role ?? ''] ?? user?.role}</p>
          </div>
        </div>
      </div>
    </>
  )

  return (
    <div className="flex h-screen bg-bg text-text overflow-hidden">
      {/* Desktop sidebar */}
      <aside className="hidden lg:flex w-64 shrink-0 flex-col bg-sidebar border-e border-border">{sidebar}</aside>

      {/* Mobile drawer */}
      {drawer && (
        <div className="lg:hidden fixed inset-0 z-50">
          <div className="absolute inset-0 bg-black/50 backdrop-blur-[2px]" onClick={() => setDrawer(false)} aria-hidden />
          <aside className="absolute inset-y-0 start-0 w-72 max-w-[85%] flex flex-col bg-sidebar border-e border-border enter">
            <button className="btn-ghost !p-2 absolute top-3 end-3" onClick={() => setDrawer(false)} aria-label="بستن منو"><X className="w-5 h-5" /></button>
            {sidebar}
          </aside>
        </div>
      )}

      <div className="flex-1 flex flex-col min-w-0">
        <header className="relative z-30 h-16 bg-surface/95 backdrop-blur border-b border-border px-3 sm:px-5 flex items-center gap-3 shrink-0">
          <button className="lg:hidden btn-ghost !p-2" onClick={() => setDrawer(true)} aria-label="باز کردن منو"><Menu className="w-5 h-5" /></button>

          {/* Breadcrumbs */}
          <nav className="hidden md:flex items-center gap-1.5 text-sm min-w-0" aria-label="مسیر">
            <Link to="/dashboard" className="text-text-muted hover:text-text"><Home className="w-4 h-4" /></Link>
            {trail.map((c, i) => (
              <span key={c.to} className="flex items-center gap-1.5 min-w-0">
                <ChevronLeft className="w-3.5 h-3.5 text-text-muted shrink-0" />
                {i === trail.length - 1
                  ? <span className="font-semibold text-text truncate">{c.label}</span>
                  : <Link to={c.to} className="text-text-secondary hover:text-text truncate">{c.label}</Link>}
              </span>
            ))}
          </nav>

          {/* Quick search */}
          <form onSubmit={onSearch} className="relative flex-1 max-w-md mx-auto">
            <Search className="w-4 h-4 text-text-muted absolute start-3 top-1/2 -translate-y-1/2 pointer-events-none" />
            <input ref={searchRef} value={q} onChange={(e) => setQ(e.target.value)} maxLength={100}
              className="input ps-9 pe-14 !rounded-full !bg-surface2 !border-transparent focus:!bg-surface"
              placeholder="جستجوی سریع فایل‌ها…" aria-label="جستجوی سریع" />
            <span className="kbd absolute end-3 top-1/2 -translate-y-1/2 pointer-events-none hidden sm:inline-flex ltr">Ctrl K</span>
          </form>

          <div className="flex items-center gap-1 sm:gap-2 shrink-0">
            {running > 0 && (
              <Link to="/jobs" className="pill-info hover:opacity-80 transition-opacity" title="اسکن در حال اجراست">
                <span className="pill-dot bg-info animate-pulse" />
                <span className="hidden sm:inline">{running} اسکن در حال اجرا</span>
                <span className="sm:hidden">{running}</span>
              </Link>
            )}
            <span className="hidden xl:flex items-center gap-1.5 text-xs text-text-secondary px-2">
              <CalendarDays className="w-4 h-4" /> {today}
            </span>
            <ThemeMenu />
            <UserMenu />
          </div>
        </header>

        <main className="relative z-0 flex-1 overflow-y-auto p-4 sm:p-6">
          <div className="max-w-7xl mx-auto enter"><Outlet /></div>
        </main>
      </div>
    </div>
  )
}
