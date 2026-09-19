import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type { TokenPair } from '../types'

// ── Access token: memory only (never localStorage). The refresh token is an
//    httpOnly cookie managed by the server, so scripts can never read it. ──────
let accessToken: string | null = null

export const tokenStore = {
  setAccessToken(t: string) { accessToken = t },
  clear() { accessToken = null },
  getAccessToken: () => accessToken,
}

const CSRF_HEADER = { 'X-Requested-With': 'fits' }

export const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json', ...CSRF_HEADER },
  timeout: 30_000,
  withCredentials: true,
})

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  return config
})

/** Turns any error into a short, user-friendly Persian message. */
export function errorMessage(err: unknown, fallback = 'مشکلی پیش آمد؛ دوباره تلاش کنید'): string {
  const e = err as AxiosError<{ error?: string }>
  if (e?.response?.data?.error) return e.response.data.error
  if (e?.code === 'ECONNABORTED') return 'پاسخی از سرور نیامد؛ اتصال خود را بررسی کنید'
  if (e?.request && !e.response) return 'ارتباط با سرور برقرار نشد'
  return fallback
}

// Shared silent-refresh (also used at app start to restore the session after a page reload)
let refreshing: Promise<TokenPair> | null = null
export function silentRefresh(): Promise<TokenPair> {
  if (!refreshing) {
    refreshing = axios
      .post<TokenPair>('/api/auth/refresh', undefined, { headers: CSRF_HEADER, withCredentials: true })
      .then((r) => {
        accessToken = r.data.access_token
        return r.data
      })
      .finally(() => { refreshing = null })
  }
  return refreshing
}

let onSessionExpired: (() => void) | null = null
export function setSessionExpiredHandler(fn: () => void) { onSessionExpired = fn }

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
    if (error.response?.status !== 401 || !original || original._retry || original.url?.includes('/auth/')) {
      return Promise.reject(error)
    }
    original._retry = true
    try {
      const pair = await silentRefresh()
      original.headers.Authorization = `Bearer ${pair.access_token}`
      return api(original)
    } catch (err) {
      tokenStore.clear()
      onSessionExpired?.()
      return Promise.reject(err)
    }
  },
)

export default api
