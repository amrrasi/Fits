import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutDashboard, Files, Cpu, Users, LogOut, Star, ChevronLeft, UserCircle, Shield, Sun, Moon
} from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { useTheme } from '../../context/ThemeContext'
import clsx from 'clsx'

const navItems = [
  { to: '/dashboard', label: 'داشبورد',          icon: LayoutDashboard },
  { to: '/files',     label: 'فایل‌های FITS',     icon: Files },
  { to: '/jobs',      label: 'اسکن‌ها',            icon: Cpu },
]

const adminItems = [
  { to: '/users',      label: 'مدیریت کاربران', icon: Users },
  { to: '/audit-logs', label: 'لاگ ممیزی',       icon: Shield },
]

const breadcrumbMap: Record<string, string> = {
  '/dashboard':  'داشبورد',
  '/files':      'فایل‌های FITS',
  '/jobs':       'اسکن‌ها',
  '/users':      'مدیریت کاربران',
  '/audit-logs': 'لاگ ممیزی',
  '/profile':    'پروفایل من',
}

function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  const isDark = theme === 'dark'
  return (
    <button
      onClick={toggleTheme}
      aria-label={isDark ? 'رفتن به حالت روشن' : 'رفتن به حالت تاریک'}
      title={isDark ? 'حالت روشن' : 'حالت تاریک'}
      className="relative w-9 h-9 grid place-items-center rounded-xl border border-black/[0.08] bg-white/60
                 text-gray-500 hover:text-brand-600 hover:border-brand-300/50 backdrop-blur-md
                 transition-all duration-200 hover:-translate-y-px
                 dark:bg-white/[0.05] dark:border-white/10 dark:text-gray-400 dark:hover:text-brand-300"
    >
      <Sun className={clsx('w-4 h-4 absolute transition-all duration-300',
        isDark ? 'opacity-0 -rotate-90 scale-50' : 'opacity-100 rotate-0 scale-100')} />
      <Moon className={clsx('w-4 h-4 absolute transition-all duration-300',
        isDark ? 'opacity-100 rotate-0 scale-100' : 'opacity-0 rotate-90 scale-50')} />
    </button>
  )
}

export default function AppLayout() {
  const { user, logout, isAdmin } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  // Breadcrumb: find the best match for current path
  const crumb = Object.entries(breadcrumbMap)
    .filter(([path]) => location.pathname.startsWith(path))
    .sort((a, b) => b[0].length - a[0].length)[0]?.[1] ?? 'FITS Processor'

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 flex flex-col shrink-0 relative
                         bg-space-950 text-white
                         border-l border-white/[0.06]">
        {/* subtle ambient glow */}
        <div className="pointer-events-none absolute -top-24 -left-24 w-56 h-56 rounded-full
                         bg-brand-500/20 blur-[80px]" />
        <div className="pointer-events-none absolute bottom-0 -right-16 w-48 h-48 rounded-full
                         bg-aurora-500/10 blur-[70px]" />

        {/* Logo */}
        <div className="relative flex items-center gap-3 px-5 py-5 border-b border-white/[0.06]">
          <div className="w-8 h-8 rounded-lg grid place-items-center shrink-0
                           bg-gradient-to-br from-brand-400 to-brand-600 shadow-glow">
            <Star className="w-4 h-4 text-white" fill="currentColor" strokeWidth={0} />
          </div>
          <div>
            <div className="font-semibold text-sm leading-tight">FITS Processor</div>
            <div className="text-[11px] text-gray-400">داده‌های نجومی</div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="relative flex-1 px-3 py-4 space-y-0.5 overflow-y-auto scroll-thin">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} className={({ isActive }) =>
              clsx('group flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-all duration-200',
                isActive
                  ? 'bg-gradient-to-l from-brand-600/90 to-brand-600/60 text-white shadow-glow'
                  : 'text-gray-400 hover:bg-white/[0.06] hover:text-white')
            }>
              <Icon className="w-4 h-4 shrink-0 transition-transform duration-200 group-hover:scale-110" />
              {label}
            </NavLink>
          ))}

          {isAdmin && (
            <>
              <div className="pt-4 pb-1 px-3">
                <div className="text-[11px] font-semibold text-gray-500 tracking-wide">مدیریت</div>
              </div>
              {adminItems.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} className={({ isActive }) =>
                  clsx('group flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-all duration-200',
                    isActive
                      ? 'bg-gradient-to-l from-brand-600/90 to-brand-600/60 text-white shadow-glow'
                      : 'text-gray-400 hover:bg-white/[0.06] hover:text-white')
                }>
                  <Icon className="w-4 h-4 shrink-0 transition-transform duration-200 group-hover:scale-110" />
                  {label}
                </NavLink>
              ))}
            </>
          )}
        </nav>

        {/* User footer */}
        <div className="relative border-t border-white/[0.06] p-3">
          <NavLink
            to="/profile"
            className={({ isActive }) =>
              clsx('flex items-center gap-3 px-2 py-2 rounded-xl transition-colors duration-200 cursor-pointer',
                isActive ? 'bg-white/[0.08]' : 'hover:bg-white/[0.06]')
            }
          >
            <div className="w-8 h-8 rounded-full grid place-items-center shrink-0
                             bg-gradient-to-br from-brand-500 to-brand-800 ring-1 ring-white/10">
              <span className="text-xs font-semibold text-white">
                {(user?.full_name || user?.email || '?').charAt(0).toUpperCase()}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-white truncate">{user?.full_name || user?.email}</div>
              <div className="text-xs text-gray-400 capitalize">{user?.role}</div>
            </div>
          </NavLink>
          <button
            onClick={handleLogout}
            className="flex items-center gap-2 w-full px-3 py-2 mt-1 rounded-xl text-sm text-gray-400
                       hover:text-white hover:bg-white/[0.06] transition-colors duration-200"
          >
            <LogOut className="w-4 h-4" />
            خروج
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Topbar */}
        <header className="glass-panel px-6 py-3 flex items-center justify-between shrink-0 z-10">
          <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <span>FITS Processor</span>
            <ChevronLeft className="w-3.5 h-3.5 text-gray-300 dark:text-gray-600" />
            <span className="text-gray-900 dark:text-gray-100 font-medium">{crumb}</span>
          </div>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            <NavLink
              to="/profile"
              className="flex items-center gap-2 text-sm text-gray-500 hover:text-brand-600 transition-colors duration-200
                         dark:text-gray-400 dark:hover:text-brand-300"
            >
              <UserCircle className="w-4 h-4" />
              {user?.full_name || user?.email}
            </NavLink>
          </div>
        </header>

        {/* Content */}
        <main key={location.pathname} className="flex-1 overflow-auto p-6 enter scroll-thin">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
