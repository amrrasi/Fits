import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { ToastProvider } from './context/ToastContext'
import ProtectedRoute from './components/layout/ProtectedRoute'
import AppLayout from './components/layout/AppLayout'
import LoginPage from './pages/auth/LoginPage'
import DashboardPage from './pages/dashboard/DashboardPage'
import FilesPage from './pages/files/FilesPage'
import FileDetailPage from './pages/files/FileDetailPage'
import JobsPage from './pages/jobs/JobsPage'
import JobDetailPage from './pages/jobs/JobDetailPage'
import UsersPage from './pages/users/UsersPage'
import UserDetailPage from './pages/users/UserDetailPage'
import AuditLogPage from './pages/users/AuditLogPage'
import ProfilePage from './pages/profile/ProfilePage'

export default function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <Routes>
          {/* Public */}
          <Route path="/login" element={<LoginPage />} />

          {/* All authenticated users */}
          <Route element={<ProtectedRoute />}>
            <Route element={<AppLayout />}>
              <Route path="/"           element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard"  element={<DashboardPage />} />
              <Route path="/files"      element={<FilesPage />} />
              <Route path="/files/:id"  element={<FileDetailPage />} />
              <Route path="/jobs"       element={<JobsPage />} />
              <Route path="/jobs/:id"   element={<JobDetailPage />} />
              <Route path="/profile"    element={<ProfilePage />} />

              {/* Admin only */}
              <Route element={<ProtectedRoute requiredRole="admin" />}>
                <Route path="/users"          element={<UsersPage />} />
                <Route path="/users/:id"      element={<UserDetailPage />} />
                <Route path="/audit-logs"     element={<AuditLogPage />} />
              </Route>
            </Route>
          </Route>

          {/* Catch-all */}
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </ToastProvider>
    </AuthProvider>
  )
}
