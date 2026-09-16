import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutDashboard, Files, Cpu, Users, LogOut, Star, ChevronLeft, UserCircle, Shield
} from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
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
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="w-60 flex flex-col bg-gray-900 text-white shrink-0">
        {/* Logo */}
        <div className="flex items-center gap-3 px-5 py-5 border-b border-gray-800">
          <div className="w-8 h-8 bg-brand-500 rounded-lg flex items-center justify-center shrink-0">
            <Star className="w-4 h-4 text-white" />
          </div>
          <div>
            <div className="font-semibold text-sm">FITS Processor</div>
            <div className="text-xs text-gray-400">Astronomical Data</div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} className={({ isActive }) =>
              clsx('flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                isActive ? 'bg-brand-600 text-white' : 'text-gray-400 hover:bg-gray-800 hover:text-white')
            }>
              <Icon className="w-4 h-4 shrink-0" />
              {label}
            </NavLink>
          ))}

          {isAdmin && (
            <>
              <div className="pt-4 pb-1 px-3">
                <div className="text-xs font-semibold text-gray-600 uppercase tracking-wider">مدیریت</div>
              </div>
              {adminItems.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} className={({ isActive }) =>
                  clsx('flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                    isActive ? 'bg-brand-600 text-white' : 'text-gray-400 hover:bg-gray-800 hover:text-white')
                }>
                  <Icon className="w-4 h-4 shrink-0" />
                  {label}
                </NavLink>
              ))}
            </>
          )}
        </nav>

        {/* User footer */}
        <div className="border-t border-gray-800 p-3">
          <NavLink
            to="/profile"
            className={({ isActive }) =>
              clsx('flex items-center gap-3 px-2 py-2 rounded-lg transition-colors cursor-pointer',
                isActive ? 'bg-gray-800' : 'hover:bg-gray-800')
            }
          >
            <div className="w-8 h-8 bg-brand-700 rounded-full flex items-center justify-center shrink-0">
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
            className="flex items-center gap-2 w-full px-3 py-2 mt-1 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            خروج
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Topbar */}
        <header className="bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span>FITS Processor</span>
            <ChevronLeft className="w-3.5 h-3.5 text-gray-300" />
            <span className="text-gray-900 font-medium">{crumb}</span>
          </div>
          <NavLink
            to="/profile"
            className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 transition-colors"
          >
            <UserCircle className="w-4 h-4" />
            {user?.full_name || user?.email}
          </NavLink>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
