import { useState, FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { KeyRound, CheckCircle2, MonitorSmartphone, ShieldAlert, LogOut } from 'lucide-react'
import { usersApi, authApi } from '../../api/endpoints'
import { errorMessage } from '../../api/client'
import { useAuth } from '../../context/AuthContext'
import { useToast } from '../../context/ToastContext'
import { RoleBadge, Spinner, PasswordStrength, passwordChecks, Avatar } from '../../components/ui'
import { formatDate } from '../../utils/format'

/** "Chrome · Windows" from a raw User-Agent string. */
function deviceLabel(ua: string): string {
  if (!ua) return 'دستگاه ناشناس'
  const browser = /Edg\//.test(ua) ? 'Edge' : /Firefox\//.test(ua) ? 'Firefox' : /Chrome\//.test(ua) ? 'Chrome' : /Safari\//.test(ua) ? 'Safari' : 'مرورگر'
  const os = /Windows/.test(ua) ? 'Windows' : /Android/.test(ua) ? 'Android' : /iPhone|iPad/.test(ua) ? 'iOS' : /Mac OS/.test(ua) ? 'macOS' : /Linux/.test(ua) ? 'Linux' : ''
  return os ? `${browser} · ${os}` : browser
}

export default function ProfilePage() {
  const { user, refreshUser } = useAuth()
  const toast = useToast()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [pwdError, setPwdError] = useState('')
  const [pwdSuccess, setPwdSuccess] = useState(false)

  const forced = !!user?.must_change_password

  const sessionsQ = useQuery({ queryKey: ['sessions'], queryFn: authApi.sessions, enabled: !forced })
  const revokeMut = useMutation({
    mutationFn: (id: string) => authApi.revokeSession(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['sessions'] }); toast.success('دستگاه از حساب شما خارج شد') },
    onError: (e) => toast.error(errorMessage(e)),
  })

  const pwdMutation = useMutation({
    mutationFn: () => usersApi.changeMyPassword(oldPwd, newPwd),
    onSuccess: async () => {
      setPwdSuccess(true)
      setOldPwd(''); setNewPwd(''); setConfirmPwd(''); setPwdError('')
      await refreshUser()
      qc.invalidateQueries({ queryKey: ['sessions'] })
      if (forced) {
        toast.success('رمز عبور شما تغییر کرد؛ خوش آمدید!')
        navigate('/dashboard', { replace: true })
      }
    },
    onError: (err: unknown) => setPwdError(errorMessage(err, 'خطا در تغییر رمز')),
  })

  const handlePwdSubmit = (e: FormEvent) => {
    e.preventDefault()
    setPwdSuccess(false)
    setPwdError('')
    if (!passwordChecks(newPwd, user?.email).every((c) => c.ok)) {
      setPwdError('رمز جدید شرایط امنیتی را ندارد؛ موارد بالای فرم را ببینید')
      return
    }
    if (newPwd !== confirmPwd) {
      setPwdError('تکرار رمز جدید مطابقت ندارد')
      return
    }
    pwdMutation.mutate()
  }

  if (!user) return null

  return (
    <div className="max-w-xl space-y-6">
      <div>
        <h1 className="text-xl font-bold text-text">پروفایل من</h1>
        <p className="text-sm text-text-secondary mt-0.5">اطلاعات حساب و امنیت شما</p>
      </div>

      {forced && (
        <div role="alert" className="flex gap-3 p-4 rounded-2xl bg-warning-bg border border-warning/40 text-warning">
          <ShieldAlert className="w-5 h-5 shrink-0 mt-0.5" />
          <div className="text-sm">
            <p className="font-bold">لطفاً رمز عبور جدید انتخاب کنید</p>
            <p className="mt-0.5 opacity-90">رمز فعلی شما موقتی است. تا زمانی که آن را عوض نکنید، بخش‌های دیگر برنامه در دسترس نیستند.</p>
          </div>
        </div>
      )}

      {/* User info */}
      <div className="card p-6">
        <div className="flex items-center gap-4 mb-5">
          <Avatar name={user.full_name || user.email} size={56} />
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-text">{user.full_name || '—'}</h2>
            <p className="text-sm text-text-secondary truncate ltr">{user.email}</p>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          {[
            { label: 'نقش کاربری', value: <RoleBadge role={user.role} /> },
            { label: 'وضعیت', value: <span className={user.is_active ? 'badge-green' : 'badge-gray'}>{user.is_active ? 'فعال' : 'غیرفعال'}</span> },
            { label: 'آخرین ورود', value: formatDate(user.last_login_at) },
            { label: 'تاریخ عضویت', value: formatDate(user.created_at) },
          ].map(({ label, value }) => (
            <div key={label} className="flex flex-col gap-1">
              <span className="text-xs text-text-muted font-medium">{label}</span>
              <span className="text-text">{value}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Change password */}
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-5">
          <KeyRound className="w-4 h-4 text-text-secondary" />
          <h2 className="text-sm font-semibold text-text">تغییر رمز عبور</h2>
        </div>
        {pwdSuccess && !forced && (
          <div className="flex items-center gap-2 p-3 bg-success-bg border border-success/40 rounded-lg mb-4 text-sm text-success">
            <CheckCircle2 className="w-4 h-4 shrink-0" />
            رمز عبور با موفقیت تغییر کرد؛ بقیه‌ی دستگاه‌ها از حساب شما خارج شدند
          </div>
        )}
        <form onSubmit={handlePwdSubmit} className="space-y-4" autoComplete="off">
          <div>
            <label className="label">رمز فعلی</label>
            <input type="password" className="input" value={oldPwd} autoComplete="current-password"
              onChange={(e) => setOldPwd(e.target.value)} required maxLength={200} />
          </div>
          <div>
            <label className="label">رمز جدید</label>
            <input type="password" className="input" value={newPwd} autoComplete="new-password"
              onChange={(e) => setNewPwd(e.target.value)} required minLength={10} maxLength={72} />
            <PasswordStrength value={newPwd} email={user.email} />
          </div>
          <div>
            <label className="label">تکرار رمز جدید</label>
            <input type="password" className="input" value={confirmPwd} autoComplete="new-password"
              onChange={(e) => setConfirmPwd(e.target.value)} required maxLength={72} />
          </div>
          {pwdError && <p className="text-sm text-danger" role="alert">{pwdError}</p>}
          <button type="submit" disabled={pwdMutation.isPending} className="btn-primary w-full">
            {pwdMutation.isPending ? <><Spinner className="w-4 h-4 text-white" /> در حال ذخیره...</> : 'تغییر رمز عبور'}
          </button>
        </form>
      </div>

      {/* Active sessions */}
      {!forced && (
        <div className="card p-6">
          <div className="flex items-center gap-2 mb-1">
            <MonitorSmartphone className="w-4 h-4 text-text-secondary" />
            <h2 className="text-sm font-semibold text-text">دستگاه‌های وارد‌شده</h2>
          </div>
          <p className="text-xs text-text-muted mb-4">اگر دستگاهی را نمی‌شناسید، آن را خارج کنید و رمز عبور را تغییر دهید.</p>
          {sessionsQ.isLoading ? (
            <div className="py-4 flex justify-center"><Spinner className="w-5 h-5 text-accent-600" /></div>
          ) : (
            <ul className="divide-y divide-border">
              {(sessionsQ.data ?? []).map((s) => (
                <li key={s.id} className="py-3 flex items-center gap-3">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-text flex items-center gap-2">
                      {deviceLabel(s.user_agent)}
                      {s.current && <span className="pill-success">این دستگاه</span>}
                    </p>
                    <p className="text-xs text-text-muted mt-0.5">
                      <span className="ltr inline-block">{s.ip_address || '—'}</span> · {formatDate(s.created_at)}
                    </p>
                  </div>
                  {!s.current && (
                    <button className="btn-ghost text-danger hover:bg-danger-bg !px-2.5" disabled={revokeMut.isPending}
                      onClick={() => revokeMut.mutate(s.id)} title="خروج این دستگاه">
                      <LogOut className="w-4 h-4" /> خروج
                    </button>
                  )}
                </li>
              ))}
              {sessionsQ.data?.length === 0 && <li className="py-3 text-sm text-text-muted">نشست فعالی پیدا نشد</li>}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
