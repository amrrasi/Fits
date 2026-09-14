import { useState, FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { usersApi } from '../../api/endpoints'
import { useAuth } from '../../context/AuthContext'
import { RoleBadge, Spinner } from '../../components/ui'
import { formatDate } from '../../utils/format'
import { User, KeyRound, CheckCircle2 } from 'lucide-react'

export default function ProfilePage() {
  const { user } = useAuth()
  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [pwdError, setPwdError] = useState('')
  const [pwdSuccess, setPwdSuccess] = useState(false)

  const pwdMutation = useMutation({
    mutationFn: () => usersApi.changeMyPassword(oldPwd, newPwd),
    onSuccess: () => {
      setPwdSuccess(true)
      setOldPwd('')
      setNewPwd('')
      setConfirmPwd('')
      setPwdError('')
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setPwdError(msg ?? 'خطا در تغییر رمز')
    },
  })

  const handlePwdSubmit = (e: FormEvent) => {
    e.preventDefault()
    setPwdSuccess(false)
    setPwdError('')
    if (newPwd.length < 8) {
      setPwdError('رمز جدید باید حداقل ۸ کاراکتر باشد')
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
        <h1 className="text-xl font-bold text-gray-900">پروفایل من</h1>
        <p className="text-sm text-gray-500 mt-0.5">اطلاعات حساب کاربری شما</p>
      </div>

      {/* User info card */}
      <div className="card p-6">
        <div className="flex items-center gap-4 mb-5">
          <div className="w-14 h-14 rounded-full bg-brand-100 flex items-center justify-center shrink-0">
            <span className="text-xl font-bold text-brand-700">
              {(user.full_name || user.email).charAt(0).toUpperCase()}
            </span>
          </div>
          <div>
            <h2 className="text-base font-semibold text-gray-900">{user.full_name || '—'}</h2>
            <p className="text-sm text-gray-500">{user.email}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          {[
            { label: 'نقش کاربری',   value: <RoleBadge role={user.role} /> },
            { label: 'وضعیت',         value: <span className={user.is_active ? 'badge-green' : 'badge-gray'}>{user.is_active ? 'فعال' : 'غیرفعال'}</span> },
            { label: 'آخرین ورود',    value: formatDate(user.last_login_at) },
            { label: 'تاریخ عضویت',   value: formatDate(user.created_at) },
          ].map(({ label, value }) => (
            <div key={label} className="flex flex-col gap-1">
              <span className="text-xs text-gray-400 font-medium">{label}</span>
              <span className="text-gray-800">{value}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Change password */}
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-5">
          <KeyRound className="w-4 h-4 text-gray-500" />
          <h2 className="text-sm font-semibold text-gray-900">تغییر رمز عبور</h2>
        </div>

        {pwdSuccess && (
          <div className="flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-lg mb-4 text-sm text-green-700">
            <CheckCircle2 className="w-4 h-4 shrink-0" />
            رمز عبور با موفقیت تغییر کرد
          </div>
        )}

        <form onSubmit={handlePwdSubmit} className="space-y-4">
          <div>
            <label className="label">رمز فعلی</label>
            <input
              type="password"
              className="input"
              value={oldPwd}
              onChange={(e) => setOldPwd(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="label">رمز جدید</label>
            <input
              type="password"
              className="input"
              value={newPwd}
              onChange={(e) => setNewPwd(e.target.value)}
              required
              minLength={8}
            />
          </div>
          <div>
            <label className="label">تکرار رمز جدید</label>
            <input
              type="password"
              className="input"
              value={confirmPwd}
              onChange={(e) => setConfirmPwd(e.target.value)}
              required
            />
          </div>

          {pwdError && (
            <p className="text-sm text-red-600">{pwdError}</p>
          )}

          <button
            type="submit"
            disabled={pwdMutation.isPending}
            className="btn-primary w-full"
          >
            {pwdMutation.isPending
              ? <><Spinner className="w-4 h-4 text-white" /> در حال ذخیره...</>
              : 'تغییر رمز عبور'
            }
          </button>
        </form>
      </div>
    </div>
  )
}
