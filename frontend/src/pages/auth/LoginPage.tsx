import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { Spinner } from '../../components/ui'
import { Star } from 'lucide-react'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail]       = useState('')
  const [password, setPassword] = useState('')
  const [error, setError]       = useState('')
  const [loading, setLoading]   = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email.trim(), password)
      navigate('/files')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا در ورود. لطفاً مجدداً تلاش کنید.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 bg-brand-600 rounded-xl flex items-center justify-center mb-3 shadow-lg">
            <Star className="w-6 h-6 text-white" />
          </div>
          <h1 className="text-xl font-bold text-white">FITS Processor</h1>
          <p className="text-gray-400 text-sm mt-1">سیستم مدیریت داده‌های نجومی</p>
        </div>

        {/* Card */}
        <div className="bg-gray-900 rounded-2xl border border-gray-800 p-6 shadow-xl">
          <h2 className="text-base font-semibold text-white mb-5">ورود به حساب کاربری</h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">ایمیل</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@fits.local"
                required
                autoFocus
                className="w-full px-3 py-2.5 rounded-lg bg-gray-800 border border-gray-700 text-white
                           placeholder:text-gray-500 text-sm focus:outline-none focus:ring-2
                           focus:ring-brand-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">رمز عبور</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                className="w-full px-3 py-2.5 rounded-lg bg-gray-800 border border-gray-700 text-white
                           placeholder:text-gray-500 text-sm focus:outline-none focus:ring-2
                           focus:ring-brand-500 focus:border-transparent"
              />
            </div>

            {error && (
              <div className="p-3 bg-red-900/40 border border-red-800 rounded-lg text-sm text-red-300">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-brand-600 hover:bg-brand-700 text-white rounded-lg
                         font-medium text-sm transition-colors disabled:opacity-60 disabled:cursor-not-allowed
                         flex items-center justify-center gap-2"
            >
              {loading ? <><Spinner className="w-4 h-4 text-white" /> در حال ورود...</> : 'ورود'}
            </button>
          </form>
        </div>

        <p className="text-center text-xs text-gray-600 mt-6">
          FITS Processor v0.4 — Astronomical Data Management
        </p>
      </div>
    </div>
  )
}
