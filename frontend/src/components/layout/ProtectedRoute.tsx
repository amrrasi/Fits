import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { Lock } from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { PageSpinner } from '../ui'
import type { Role } from '../../types'

interface Props {
  requiredRole?: Role
}

export default function ProtectedRoute({ requiredRole }: Props) {
  const { isAuthenticated, user, ready } = useAuth()
  const location = useLocation()

  if (!ready) {
    return <div className="min-h-screen flex items-center justify-center bg-bg"><PageSpinner /></div>
  }
  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  if (requiredRole && user?.role !== requiredRole) {
    return (
      <div className="flex items-center justify-center h-64 enter">
        <div className="text-center max-w-xs">
          <div className="mx-auto mb-3 w-12 h-12 rounded-2xl bg-accent-100 text-accent-600 flex items-center justify-center">
            <Lock className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-bold text-text">این بخش مخصوص مدیران است</h2>
          <p className="text-text-secondary mt-1 text-sm">برای دسترسی به این صفحه با مدیر سیستم هماهنگ کنید.</p>
        </div>
      </div>
    )
  }
  return <Outlet />
}
