import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard, Files, Cpu, Users, LogOut, Star, ChevronRight
} from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import clsx from 'clsx'

const navItems = [
  { to: '/files', label: 'FITS Files', icon: Files },
  { to: '/jobs', label: 'Processing Jobs', icon: Cpu },
]

const adminItems = [
  { to: '/users', label: 'User Management', icon: Users },
]

export default function AppLayout() {
  const { user, logout, isAdmin } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="w-64 flex flex-col bg-gray-900 text-white shrink-0">
        {/* Logo */}
        <div className="flex items-center gap-3 px-5 py-5 border-b border-gray-700">
          <div className="w-8 h-8 bg-brand-500 rounded-lg flex items-center justify-center shrink-0">
            <Star className="w-4 h-4 text-white" />
          </div>
          <div>
            <div className="font-semibold text-sm">FITS Processor</div>
            <div className="text-xs text-gray-400">Astronomical Data</div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          <NavLink to="/dashboard" className={({ isActive }) =>
            clsx('flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
              isActive ? 'bg-brand-600 text-white' : 'text-gray-300 hover:bg-gray-800 hover:text-white')
          }>
            <LayoutDashboard className="w-4 h-4 shrink-0" />
            Dashboard
          </NavLink>

          <div className="pt-3 pb-1">
            <div className="px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Data</div>
          </div>
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} className={({ isActive }) =>
              clsx('flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                isActive ? 'bg-brand-600 text-white' : 'text-gray-300 hover:bg-gray-800 hover:text-white')
            }>
              <Icon className="w-4 h-4 shrink-0" />
              {label}
            </NavLink>
          ))}

          {isAdmin && (
            <>
              <div className="pt-3 pb-1">
                <div className="px-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Admin</div>
              </div>
              {adminItems.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} className={({ isActive }) =>
                  clsx('flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                    isActive ? 'bg-brand-600 text-white' : 'text-gray-300 hover:bg-gray-800 hover:text-white')
                }>
                  <Icon className="w-4 h-4 shrink-0" />
                  {label}
                </NavLink>
              ))}
            </>
          )}
        </nav>

        {/* User footer */}
        <div className="border-t border-gray-700 p-3">
          <div className="flex items-center gap-3 px-2 py-2">
            <div className="w-8 h-8 bg-brand-700 rounded-full flex items-center justify-center shrink-0">
              <span className="text-xs font-semibold text-white">
                {user?.full_name?.charAt(0)?.toUpperCase() ?? user?.email?.charAt(0)?.toUpperCase() ?? '?'}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-white truncate">{user?.full_name || user?.email}</div>
              <div className="text-xs text-gray-400 capitalize">{user?.role}</div>
            </div>
            <button onClick={handleLogout} className="text-gray-400 hover:text-white transition-colors p-1 rounded" title="Logout">
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Topbar */}
        <header className="bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span>FITS Processor</span>
            <ChevronRight className="w-3 h-3" />
            <span className="text-gray-900 font-medium">Dashboard</span>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
