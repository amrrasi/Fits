import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import type { Role } from '../../types'

interface Props {
  requiredRole?: Role
}

export default function ProtectedRoute({ requiredRole }: Props) {
  const { isAuthenticated, user } = useAuth()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (requiredRole && user?.role !== requiredRole) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl mb-2">🔒</div>
          <h2 className="text-lg font-semibold text-gray-900">دسترسی محدود</h2>
          <p className="text-gray-500 mt-1">شما دسترسی به این صفحه را ندارید</p>
        </div>
      </div>
    )
  }

  return <Outlet />
}
