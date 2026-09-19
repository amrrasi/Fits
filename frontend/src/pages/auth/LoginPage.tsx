import { useState, FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Orbit, Eye, EyeOff, Mail, Lock } from 'lucide-react'
import { useAuth } from '../../context/AuthContext'
import { errorMessage } from '../../api/client'
import { Spinner } from '../../components/ui'

export default function LoginPage() {
  const { login, isAuthenticated, ready } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/dashboard'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [show, setShow] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (ready && isAuthenticated) return <Navigate to={from} replace />

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(email.trim(), password)
      navigate(from, { replace: true })
    } catch (err) {
      setError(errorMessage(err, 'ورود انجام نشد؛ دوباره تلاش کنید'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen relative flex items-center justify-center p-4 overflow-hidden bg-gradient-to-br from-[#0b0e1a] via-[#14193a] to-[#221c55]">
      <div className="stars absolute inset-0 pointer-events-none" aria-hidden />
      <div className="absolute -top-32 -start-32 w-96 h-96 rounded-full bg-accent-500/25 blur-3xl pointer-events-none" aria-hidden />
      <div className="absolute -bottom-40 -end-32 w-[28rem] h-[28rem] rounded-full bg-violet-500/20 blur-3xl pointer-events-none" aria-hidden />

      <div className="relative w-full max-w-sm enter-pop">
        <div className="text-center mb-7">
          <div className="mx-auto mb-4 w-14 h-14 rounded-2xl bg-gradient-to-br from-accent-400 to-accent-700 flex items-center justify-center shadow-lg shadow-accent-600/40">
            <Orbit className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-2xl font-extrabold text-white">سلام</h1>
          <p className="text-sm text-indigo-200/80 mt-1.5">برای دیدن آسمانِ داده‌هایتان وارد شوید</p>
        </div>

        <form onSubmit={handleSubmit} className="rounded-3xl p-6 space-y-4 bg-white/[0.07] border border-white/15 backdrop-blur-xl shadow-2xl">
          {error && (
            <div role="alert" className="text-sm rounded-xl px-3.5 py-2.5 bg-red-500/15 border border-red-400/30 text-red-200">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="email" className="block text-xs font-medium text-indigo-100/90 mb-1.5">ایمیل</label>
            <div className="relative">
              <Mail className="w-4 h-4 text-indigo-200/60 absolute start-3 top-1/2 -translate-y-1/2" />
              <input id="email" type="email" required autoFocus autoComplete="username" value={email} maxLength={254}
                onChange={(e) => setEmail(e.target.value)} dir="ltr"
                className="w-full ps-9 pe-3 py-2.5 rounded-xl text-sm text-white placeholder:text-indigo-200/40 bg-white/10 border border-white/15 focus:outline-none focus:ring-2 focus:ring-accent-400/60 focus:border-accent-400"
                placeholder="you@example.com" />
            </div>
          </div>

          <div>
            <label htmlFor="password" className="block text-xs font-medium text-indigo-100/90 mb-1.5">رمز عبور</label>
            <div className="relative">
              <Lock className="w-4 h-4 text-indigo-200/60 absolute start-3 top-1/2 -translate-y-1/2" />
              <input id="password" type={show ? 'text' : 'password'} required autoComplete="current-password" value={password}
                onChange={(e) => setPassword(e.target.value)} dir="ltr" maxLength={200}
                className="w-full ps-9 pe-10 py-2.5 rounded-xl text-sm text-white placeholder:text-indigo-200/40 bg-white/10 border border-white/15 focus:outline-none focus:ring-2 focus:ring-accent-400/60 focus:border-accent-400"
                placeholder="••••••••••" />
              <button type="button" onClick={() => setShow((s) => !s)} aria-label={show ? 'مخفی کردن رمز' : 'نمایش رمز'}
                className="absolute end-3 top-1/2 -translate-y-1/2 text-indigo-200/70 hover:text-white">
                {show ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <button type="submit" disabled={loading || !email || !password}
            className="w-full py-2.5 rounded-xl font-semibold text-sm text-white bg-gradient-to-l from-accent-500 to-accent-700 hover:opacity-95 active:scale-[.99] disabled:opacity-50 transition-all flex items-center justify-center gap-2 shadow-lg shadow-accent-700/30">
            {loading ? <Spinner className="w-4 h-4 text-white" /> : 'ورود به حساب'}
          </button>
        </form>

        <p className="text-center text-xs text-indigo-200/50 mt-5">ورود / لاگین</p>
      </div>
    </div>
  )
}
