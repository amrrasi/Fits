import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { Spinner } from '../../components/ui'
import { Orbit } from 'lucide-react'

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
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="flex flex-col items-center mb-8">
          <div className="w-11 h-11 rounded-lg flex items-center justify-center mb-3 bg-accent-600">
            <Orbit className="w-5 h-5 text-white" strokeWidth={1.75} />
          </div>
          <h1 className="text-lg font-semibold text-white">سامانه پردازش فایل‌های FITS</h1>
          <p className="text-zinc-500 text-sm mt-1">رصدخانه ملی ایران (IPM)</p>
        </div>

        <div className="bg-zinc-900 rounded-xl border border-zinc-800 p-6">
          <h2 className="text-sm font-semibold text-white mb-5">ورود به حساب کاربری</h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-zinc-400 mb-1.5">ایمیل</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@fits.local"
                required
                autoFocus
                className="w-full px-3 py-2 rounded-md bg-zinc-800/60 border border-zinc-700 text-white
                           placeholder:text-zinc-500 text-sm transition-colors
                           focus:outline-none focus:ring-2 focus:ring-accent-500/40 focus:border-accent-500"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-zinc-400 mb-1.5">رمز عبور</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                className="w-full px-3 py-2 rounded-md bg-zinc-800/60 border border-zinc-700 text-white
                           placeholder:text-zinc-500 text-sm transition-colors
                           focus:outline-none focus:ring-2 focus:ring-accent-500/40 focus:border-accent-500"
              />
            </div>

            {error && (
              <div className="p-2.5 bg-red-500/10 border border-red-500/20 rounded-md text-sm text-red-400">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 bg-accent-600 hover:bg-accent-700 active:bg-accent-800
                         text-white rounded-md font-medium text-sm transition-colors
                         disabled:opacity-60 disabled:cursor-not-allowed
                         flex items-center justify-center gap-2"
            >
              {loading ? <><Spinner className="w-4 h-4 text-white" /> در حال ورود...</> : 'ورود'}
            </button>
          </form>
        </div>

        <p className="text-center text-xs text-zinc-600 mt-6">
          تمامی حقوق متعلق به رصدخانه ملی ایران (IPM) می‌باشد — {new Date().getFullYear()}
        </p>
      </div>
    </div>
  )
}
