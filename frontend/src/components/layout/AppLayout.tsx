import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutDashboard, Files, Cpu, Users, LogOut, Orbit, ChevronLeft, Shield, Sun, Moon
} from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { useTheme } from '../../context/ThemeContext'
import clsx from 'clsx'

const navItems = [
  { to: '/dashboard', label: 'داشبورد',      icon: LayoutDashboard },
  { to: '/files',     label: 'فایل‌های FITS', icon: Files },
  { to: '/jobs',      label: 'اسکن‌ها',       icon: Cpu },
]

const adminItems = [
  { to: '/users',      label: 'کاربران',  icon: Users },
  { to: '/audit-logs', label: 'گزارش‌ها', icon: Shield },
]

const breadcrumbMap: Record<string, string> = {
  '/dashboard':  'داشبورد',
  '/files':      'فایل‌های FITS',
  '/jobs':       'اسکن‌ها',
  '/users':      'کاربران',
  '/audit-logs': 'گزارش‌ها',
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
      className="relative w-7 h-7 grid place-items-center rounded-md text-text-secondary
                 hover:bg-surface2 hover:text-text transition-colors"
    >
      <Sun className={clsx('w-[15px] h-[15px] absolute transition-all duration-200',
        isDark ? 'opacity-0 -rotate-90 scale-50' : 'opacity-100 rotate-0 scale-100')} />
      <Moon className={clsx('w-[15px] h-[15px] absolute transition-all duration-200',
        isDark ? 'opacity-100 rotate-0 scale-100' : 'opacity-0 rotate-90 scale-50')} />
    </button>
  )
}

function NavItem({ to, label, icon: Icon }: { to: string; label: string; icon: typeof Files }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        clsx(
          'group relative flex items-center gap-2.5 px-3 py-1.5 rounded-md text-sm transition-colors',
          isActive
            ? 'text-accent-700 dark:text-accent-300 bg-accent-50 dark:bg-accent-900/40'
            : 'text-text-secondary hover:text-text hover:bg-surface2'
        )
      }
    >
      {({ isActive }) => (
        <>
          <span className={clsx(
            'absolute inset-y-1 start-0 w-0.5 rounded-full transition-colors',
            isActive ? 'bg-accent-600' : 'bg-transparent'
          )} />
          <Icon className="w-4 h-4 shrink-0" strokeWidth={1.75} />
          <span className={isActive ? 'font-medium' : ''}>{label}</span>
        </>
      )}
    </NavLink>
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

  const crumb = Object.entries(breadcrumbMap)
    .filter(([path]) => location.pathname.startsWith(path))
    .sort((a, b) => b[0].length - a[0].length)[0]?.[1] ?? 'رصدخانه ملی ایران'

  return (
    <div className="flex h-screen bg-bg">
      {/* Sidebar — same theme tokens as the rest of the app, no more permanently-dark shell */}
      <aside className="w-56 flex flex-col sidebar-surface border-e border-border shrink-0">
        <div className="flex items-center gap-2.5 px-4 h-14 border-b border-border">
          <Orbit className="w-[18px] h-[18px] text-accent-600 dark:text-accent-400" strokeWidth={1.75} />
          <div className="text-[13px] font-semibold text-text tracking-tight leading-tight">
            رصدخانه ملی ایران
            <div className="text-[10px] font-normal text-text-muted">پردازش داده‌های FITS</div>
          </div>
        </div>

        <nav className="flex-1 px-2.5 py-3 space-y-0.5 overflow-y-auto">
          {navItems.map((item) => <NavItem key={item.to} {...item} />)}

          {isAdmin && (
            <>
              <div className="pt-4 pb-1.5 px-3">
                <div className="text-[10px] font-semibold text-text-muted uppercase tracking-wider">مدیریت</div>
              </div>
              {adminItems.map((item) => <NavItem key={item.to} {...item} />)}
            </>
          )}
        </nav>

        <div className="border-t border-border p-2.5">
          <NavLink
            to="/profile"
            className={({ isActive }) =>
              clsx('flex items-center gap-2.5 px-2 py-1.5 rounded-md transition-colors',
                isActive ? 'bg-surface2' : 'hover:bg-surface2')
            }
          >
            <div className="w-6 h-6 rounded-full bg-accent-600 flex items-center justify-center shrink-0">
              <span className="text-[10px] font-semibold text-white">
                {(user?.full_name || user?.email || '?').charAt(0).toUpperCase()}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-[13px] font-medium text-text truncate">{user?.full_name || user?.email}</div>
              <div className="text-[11px] text-text-muted">{user?.role}</div>
            </div>
          </NavLink>
          <button
            onClick={handleLogout}
            className="flex items-center gap-2.5 w-full px-2 py-1.5 mt-0.5 rounded-md text-[13px] text-text-muted
                       hover:text-text hover:bg-surface2 transition-colors"
          >
            <LogOut className="w-3.5 h-3.5" strokeWidth={1.75} />
            خروج
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="h-11 bg-surface border-b border-border px-4 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-1.5 text-xs text-text-secondary">
            <span>رصدخانه ملی ایران</span>
            <ChevronLeft className="w-3 h-3 text-text-muted" />
            <span className="text-text font-medium">{crumb}</span>
          </div>
          <ThemeToggle />
        </header>

        <main className="flex-1 overflow-auto p-5">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
