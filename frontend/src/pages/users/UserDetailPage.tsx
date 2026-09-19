import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { usersApi } from '../../api/endpoints'
import { RoleBadge, PageSpinner, ErrorState, Modal, Spinner } from '../../components/ui'
import { useToast } from '../../context/ToastContext'
import { useAuth } from '../../context/AuthContext'
import { ArrowRight, Pencil, KeyRound, Trash2 } from 'lucide-react'
import { formatDate } from '../../utils/format'
import type { SafeUser } from '../../types'

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const userId = Number(id)
  const { user: me } = useAuth()
  const toast = useToast()
  const qc = useQueryClient()

  const [showEdit, setShowEdit]     = useState(false)
  const [showReset, setShowReset]   = useState(false)
  const [showDelete, setShowDelete] = useState(false)

  const { data: user, isLoading, isError } = useQuery({
    queryKey: ['user', userId],
    queryFn: () => usersApi.getById(userId),
  })

  const deleteMutation = useMutation({
    mutationFn: () => usersApi.delete(userId),
    onSuccess: () => {
      toast.success('کاربر حذف شد')
      window.history.back()
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      toast.error(msg ?? 'خطا در حذف')
    },
  })

  if (isLoading) return <PageSpinner />
  if (isError || !user) return <ErrorState message="کاربر یافت نشد" />

  const isSelf = user.id === me?.id

  return (
    <div className="max-w-2xl space-y-5">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/users" className="text-text-muted hover:text-text">
          <ArrowRight className="w-5 h-5" />
        </Link>
        <h1 className="text-xl font-bold text-text flex-1">جزئیات کاربر</h1>
        <div className="flex gap-2">
          <button onClick={() => setShowEdit(true)} className="btn-secondary">
            <Pencil className="w-4 h-4" /> ویرایش
          </button>
          <button onClick={() => setShowReset(true)} className="btn-secondary">
            <KeyRound className="w-4 h-4" /> ریست رمز
          </button>
          {!isSelf && (
            <button onClick={() => setShowDelete(true)} className="btn-danger">
              <Trash2 className="w-4 h-4" /> حذف
            </button>
          )}
        </div>
      </div>

      {/* Profile card */}
      <div className="card p-6">
        <div className="flex items-center gap-5 pb-5 border-b border-border">
          <div className="w-16 h-16 rounded-full bg-accent-100 flex items-center justify-center shrink-0">
            <span className="text-2xl font-bold text-accent-700">
              {(user.full_name || user.email).charAt(0).toUpperCase()}
            </span>
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-text">{user.full_name || '—'}</h2>
              {isSelf && (
                <span className="text-xs bg-accent-50 text-accent-600 px-2 py-0.5 rounded-full font-medium">
                  شما
                </span>
              )}
            </div>
            <p className="text-sm text-text-secondary">{user.email}</p>
            <div className="flex items-center gap-2 mt-2">
              <RoleBadge role={user.role} />
              <span className={user.is_active ? 'badge-green' : 'badge-gray'}>
                {user.is_active ? 'فعال' : 'غیرفعال'}
              </span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-5 pt-5">
          {[
            { label: 'شناسه کاربر',   value: `#${user.id}` },
            { label: 'ایمیل',          value: user.email },
            { label: 'آخرین ورود',     value: formatDate(user.last_login_at) },
            { label: 'تاریخ عضویت',    value: formatDate(user.created_at) },
          ].map(({ label, value }) => (
            <div key={label}>
              <p className="text-xs text-text-muted font-medium mb-1">{label}</p>
              <p className="text-sm text-text font-medium">{value || '—'}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Permissions info */}
      <div className="card p-5">
        <h3 className="text-sm font-semibold text-text mb-4">دسترسی‌ها</h3>
        <div className="space-y-2">
          {[
            { label: 'مشاهده فایل‌های FITS',     allowed: true },
            { label: 'مشاهده هدرها و متادیتا',   allowed: true },
            { label: 'ویرایش متادیتا',            allowed: user.role === 'editor' || user.role === 'admin' },
            { label: 'مدیریت کاربران',            allowed: user.role === 'admin' },
            { label: 'حذف فایل‌ها',               allowed: user.role === 'admin' },
            { label: 'اجرای اسکن جدید',           allowed: user.role === 'admin' },
            { label: 'مشاهده لاگ‌های ممیزی',      allowed: user.role === 'admin' },
          ].map(({ label, allowed }) => (
            <div key={label} className="flex items-center gap-3">
              <div className={`w-2 h-2 rounded-full shrink-0 ${allowed ? 'bg-green-500' : 'bg-border-strong'}`} />
              <span className={`text-sm ${allowed ? 'text-text' : 'text-text-muted'}`}>{label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Modals */}
      {showEdit  && <EditModal  user={user} onClose={() => { setShowEdit(false);  qc.invalidateQueries({ queryKey: ['user', userId] }) }} />}
      {showReset && <ResetModal user={user} onClose={() => setShowReset(false)} />}
      {showDelete && (
        <Modal title="حذف کاربر" onClose={() => setShowDelete(false)}>
          <p className="text-sm text-text-secondary mb-5">
            آیا مطمئنید؟ این عملیات غیرقابل بازگشت است.
          </p>
          <div className="flex gap-3">
            <button
              onClick={() => deleteMutation.mutate()}
              disabled={deleteMutation.isPending}
              className="btn-danger flex-1"
            >
              {deleteMutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'حذف'}
            </button>
            <button onClick={() => setShowDelete(false)} className="btn-secondary">انصراف</button>
          </div>
        </Modal>
      )}
    </div>
  )
}

// ── Inline edit modal ─────────────────────────────────────────────────────────

function EditModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const toast = useToast()
  const [form, setForm] = useState({
    full_name: user.full_name,
    role: user.role,
    is_active: user.is_active,
  })
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.update(user.id, form),
    onSuccess: () => { toast.success('بروزرسانی شد'); onClose() },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا')
    },
  })

  return (
    <Modal title="ویرایش کاربر" onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="label">نام کامل</label>
          <input className="input" value={form.full_name}
            onChange={(e) => setForm({ ...form, full_name: e.target.value })} />
        </div>
        <div>
          <label className="label">نقش</label>
          <select className="input" value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value as SafeUser['role'] })}>
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <label className="flex items-center gap-3 cursor-pointer p-3 rounded-lg border border-border hover:bg-surface2">
          <input type="checkbox" checked={form.is_active}
            onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
            className="w-4 h-4 rounded accent-accent-600" />
          <span className="text-sm text-text">حساب فعال باشد</span>
        </label>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-3">
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending} className="btn-primary flex-1">
            {mutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'ذخیره'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}

function ResetModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const toast = useToast()
  const [pwd, setPwd]       = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError]   = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.adminResetPassword(user.id, pwd),
    onSuccess: () => { toast.success('رمز تغییر کرد'); onClose() },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا')
    },
  })

  const submit = () => {
    if (pwd.length < 10) { setError('حداقل ۱۰ کاراکتر (ترکیب حروف و عدد)'); return }
    if (pwd !== confirm) { setError('تکرار مطابقت ندارد'); return }
    mutation.mutate()
  }

  return (
    <Modal title="ریست رمز" onClose={onClose}>
      <div className="space-y-4">
        <div><label className="label">رمز جدید</label>
          <input className="input" type="password" value={pwd} onChange={(e) => setPwd(e.target.value)} autoFocus />
        </div>
        <div><label className="label">تکرار</label>
          <input className="input" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        </div>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-3">
          <button onClick={submit} disabled={mutation.isPending || !pwd} className="btn-primary flex-1">
            {mutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'تغییر'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}
