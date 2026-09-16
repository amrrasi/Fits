import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { Spinner } from '../../components/ui'

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
    <div className="min-h-screen bg-space-950 flex items-center justify-center p-4 relative overflow-hidden">
      {/* Ambient glow */}
      <div className="absolute -top-40 -start-40 w-96 h-96 bg-brand-600/20 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute -bottom-40 -end-40 w-96 h-96 bg-aurora-500/10 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-sm relative enter-pop">
        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-3
                           bg-gradient-to-br from-brand-400 to-brand-600 shadow-glow">
            <div><img src="/media/image/logo.png" alt="رصدخانه ملی ایران"/></div>
          </div>
          <h1 className="text-xl font-bold text-white">سامانه پردازش فایل‌های FITS</h1>
          <p className="text-gray-400 text-sm mt-1">مدیریت داده‌های رصد خانه ملی ایران</p>
        </div>

        {/* Card */}
        <div className="bg-white/[0.05] backdrop-blur-xl rounded-2xl border border-white/10 p-6 shadow-glass-dark">
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
                className="w-full px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/10 text-white
                           placeholder:text-gray-500 text-sm transition-all duration-200
                           focus:outline-none focus:ring-2 focus:ring-brand-500/40 focus:border-brand-400/60 focus:bg-white/[0.07]"
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
                className="w-full px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/10 text-white
                           placeholder:text-gray-500 text-sm transition-all duration-200
                           focus:outline-none focus:ring-2 focus:ring-brand-500/40 focus:border-brand-400/60 focus:bg-white/[0.07]"
              />
            </div>

            {error && (
              <div className="p-3 bg-red-400/10 border border-red-400/20 rounded-xl text-sm text-red-300 enter-pop">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-gradient-to-b from-brand-500 to-brand-600 hover:from-brand-400 hover:to-brand-600
                         text-white rounded-xl font-medium text-sm transition-all duration-200 shadow-glow
                         hover:-translate-y-px active:scale-[.98]
                         disabled:opacity-60 disabled:cursor-not-allowed disabled:translate-y-0
                         flex items-center justify-center gap-2"
            >
              {loading ? <><Spinner className="w-4 h-4 text-white" /> در حال ورود...</> : 'ورود'}
            </button>
          </form>
        </div>

        <p className="text-center text-xs text-gray-500 mt-6">
           تمامی حقوق متعلق به رصدخانه ملی ایران (IPM) میباشد - {new Date().getFullYear()}
        </p>
      </div>
    </div>
  )
}
