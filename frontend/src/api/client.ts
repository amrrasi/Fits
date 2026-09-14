import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type { TokenPair } from '../types'

// ── Token storage (memory only — never localStorage) ─────────────────────────
let accessToken: string | null = null
let refreshToken: string | null = null

export const tokenStore = {
  setTokens(pair: TokenPair) {
    accessToken = pair.access_token
    refreshToken = pair.refresh_token
  },
  clearTokens() {
    accessToken = null
    refreshToken = null
  },
  getAccessToken: () => accessToken,
  getRefreshToken: () => refreshToken,
}

// ── Axios instance ────────────────────────────────────────────────────────────
export const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
  timeout: 30_000,
})

// Attach bearer token to every request
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

// Auto-refresh on 401
let isRefreshing = false
let pendingQueue: Array<{ resolve: (v: string) => void; reject: (e: unknown) => void }> = []

api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error)
    }

    // Don't retry auth endpoints themselves
    if (original.url?.includes('/auth/')) {
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        pendingQueue.push({ resolve, reject })
      }).then((token) => {
        original.headers.Authorization = `Bearer ${token}`
        return api(original)
      })
    }

    original._retry = true
    isRefreshing = true

    try {
      const rt = tokenStore.getRefreshToken()
      if (!rt) throw new Error('no refresh token')

      const { data } = await axios.post<TokenPair>('/api/auth/refresh', {
        refresh_token: rt,
      })
      tokenStore.setTokens(data)

      pendingQueue.forEach((p) => p.resolve(data.access_token))
      pendingQueue = []

      original.headers.Authorization = `Bearer ${data.access_token}`
      return api(original)
    } catch (err) {
      pendingQueue.forEach((p) => p.reject(err))
      pendingQueue = []
      tokenStore.clearTokens()
      window.location.href = '/login'
      return Promise.reject(err)
    } finally {
      isRefreshing = false
    }
  },
)

export default api
