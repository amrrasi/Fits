import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import ProtectedRoute from './components/layout/ProtectedRoute'
import AppLayout from './components/layout/AppLayout'
import LoginPage from './pages/auth/LoginPage'
import FilesPage from './pages/files/FilesPage'
import FileDetailPage from './pages/files/FileDetailPage'
import JobsPage from './pages/jobs/JobsPage'
import UsersPage from './pages/users/UsersPage'

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute />}>
          <Route element={<AppLayout />}>
            <Route path="/" element={<Navigate to="/files" replace />} />
            <Route path="/files" element={<FilesPage />} />
            <Route path="/files/:id" element={<FileDetailPage />} />
            <Route path="/jobs" element={<JobsPage />} />
            <Route element={<ProtectedRoute requiredRole="admin" />}>
              <Route path="/users" element={<UsersPage />} />
            </Route>
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/files" replace />} />
      </Routes>
    </AuthProvider>
  )
}
