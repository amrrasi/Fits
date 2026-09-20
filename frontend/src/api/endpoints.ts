import api from './client'
import type {
  TokenPair, SafeUser, FITSFile, FITSHeader, FITSMetadata,
  MetadataOverride, ProcessingJob, ProcessingError,
  ApiResponse, PagedResponse, SessionInfo } from '../types'

// ── Auth ──────────────────────────────────────────────────────────────────────

export const authApi = {
  login: (email: string, password: string) =>
    api.post<TokenPair>('/auth/login', { email, password }).then((r) => r.data),

  // The refresh token lives in an httpOnly cookie — JS never sees it.
  logout: () => api.post('/auth/logout'),
  logoutAll: () => api.post('/auth/logout-all'),
  sessions: () => api.get<ApiResponse<SessionInfo[]>>('/auth/sessions').then((r) => r.data.data),
  revokeSession: (id: string) => api.delete(`/auth/sessions/${id}`),
  refresh: () => api.post<TokenPair>('/auth/refresh').then((r) => r.data),
}

// ── Users ─────────────────────────────────────────────────────────────────────

export interface ListUsersParams {
  page?: number
  page_size?: number
  search?: string
  role?: string
  active?: string
}

export const usersApi = {
  me: () =>
    api.get<ApiResponse<SafeUser>>('/users/me').then((r) => r.data.data),

  list: (params?: ListUsersParams) =>
    api.get<PagedResponse<SafeUser>>('/users', { params }).then((r) => r.data),

  getById: (id: number) =>
    api.get<ApiResponse<SafeUser>>(`/users/${id}`).then((r) => r.data.data),

  create: (payload: { email: string; password: string; full_name: string; role: string }) =>
    api.post<ApiResponse<SafeUser>>('/users', payload).then((r) => r.data.data),

  update: (id: number, payload: { full_name: string; role: string; is_active: boolean }) =>
    api.put<ApiResponse<SafeUser>>(`/users/${id}`, payload).then((r) => r.data.data),

  delete: (id: number) =>
    api.delete(`/users/${id}`),

  changeMyPassword: (oldPassword: string, newPassword: string) =>
    api.put('/users/me/password', { old_password: oldPassword, new_password: newPassword }),

  adminResetPassword: (id: number, newPassword: string) =>
    api.put(`/users/${id}/password`, { new_password: newPassword }),
}

// ── FITS Files ────────────────────────────────────────────────────────────────

export interface ListFilesParams {
  page?: number
  page_size?: number
  search?: string
  status?: string
  sort?: string
  order?: string
  date_from?: string
  date_to?: string
}

export const filesApi = {
  list: (params?: ListFilesParams) =>
    api.get<PagedResponse<FITSFile>>('/files', { params }).then((r) => r.data),

  getById: (id: number) =>
    api.get<ApiResponse<FITSFile>>(`/files/${id}`).then((r) => r.data.data),

  delete: (id: number) =>
    api.delete(`/files/${id}`),

  headers: (id: number, params?: { page?: number; page_size?: number; search?: string; hdu_index?: number }) =>
    api.get<PagedResponse<FITSHeader>>(`/files/${id}/headers`, { params }).then((r) => r.data),

  metadata: (id: number) =>
    api.get<ApiResponse<FITSMetadata>>(`/files/${id}/metadata`).then((r) => r.data.data),

  editMetadata: (id: number, field_name: string, new_value: string, reason?: string) =>
    api.put<ApiResponse<FITSMetadata>>(`/files/${id}/metadata`, { field_name, new_value, reason })
      .then((r) => r.data.data),

  metadataHistory: (id: number) =>
    api.get<ApiResponse<MetadataOverride[]>>(`/files/${id}/metadata/history`).then((r) => r.data.data),
}

// ── Stats ─────────────────────────────────────────────────────────────────────

export interface DashboardStats {
  total_files: number
  done_files: number
  error_files: number
  pending_files: number
  total_jobs: number
  running_jobs: number
  total_headers: number
}

export const statsApi = {
  get: () => api.get<ApiResponse<DashboardStats>>('/stats').then((r) => r.data.data),
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

export const jobsApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    api.get<PagedResponse<ProcessingJob>>('/jobs', { params }).then((r) => r.data),

  getById: (id: number) =>
    api.get<ApiResponse<ProcessingJob>>(`/jobs/${id}`).then((r) => r.data.data),

  status: (id: number) =>
    api.get<ApiResponse<{ id: number; status: string; total_files: number; done_files: number; error_files: number; finished_at?: string }>>(`/jobs/${id}/status`).then((r) => r.data.data),

  errors: (id: number, params?: { page?: number; page_size?: number }) =>
    api.get<PagedResponse<ProcessingError>>(`/jobs/${id}/errors`, { params }).then((r) => r.data),

  triggerScan: (scanDir?: string) =>
    api.post<ApiResponse<{ job_id: number; scan_dir: string; message: string }>>('/scan', { scan_dir: scanDir })
      .then((r) => r.data.data),
}
