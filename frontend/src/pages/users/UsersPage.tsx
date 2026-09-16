import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { usersApi } from '../../api/endpoints'
import {
  RoleBadge, PageSpinner, ErrorState, EmptyState, Pagination, Modal, Spinner
} from '../../components/ui'
import { useAuth } from '../../context/AuthContext'
import { useToast } from '../../context/ToastContext'
import { useDebounce } from '../../hooks/useDebounce'
import { UserPlus, Pencil, Trash2, KeyRound, Search, Eye } from 'lucide-react'
import { formatDate } from '../../utils/format'
import type { SafeUser } from '../../types'

export default function UsersPage() {
  const { user: me } = useAuth()
  const toast = useToast()
  const qc = useQueryClient()

  const [page, setPage]     = useState(1)
  const [search, setSearch] = useState('')
  const [role, setRole]     = useState('')
  const [active, setActive] = useState('')

  const debouncedSearch = useDebounce(search, 400)

  const [showCreate, setShowCreate] = useState(false)
  const [editUser, setEditUser]     = useState<SafeUser | null>(null)
  const [deleteUser, setDeleteUser] = useState<SafeUser | null>(null)
  const [resetUser, setResetUser]   = useState<SafeUser | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['users', page, debouncedSearch, role, active],
    queryFn: () => usersApi.list({ page, page_size: 20, search: debouncedSearch, role, active }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => usersApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users'] })
      setDeleteUser(null)
      toast.success('کاربر با موفقیت حذف شد')
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      toast.error(msg ?? 'خطا در حذف کاربر')
    },
  })

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">مدیریت کاربران</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            {isLoading ? '...' : `${data?.total ?? 0} کاربر`}
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className="btn-primary">
          <UserPlus className="w-4 h-4" />
          کاربر جدید
        </button>
      </div>

      {/* Filters */}
      <div className="card p-4">
        <div className="flex flex-wrap gap-3 items-center">
          <div className="relative flex-1 min-w-52">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500" />
            <input
              className="input pl-9"
              placeholder="جستجو در ایمیل یا نام..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            />
          </div>
          <select className="input w-36" value={role} onChange={(e) => { setRole(e.target.value); setPage(1) }}>
            <option value="">همه نقش‌ها</option>
            <option value="admin">Admin</option>
            <option value="editor">Editor</option>
            <option value="viewer">Viewer</option>
          </select>
          <select className="input w-36" value={active} onChange={(e) => { setActive(e.target.value); setPage(1) }}>
            <option value="">همه وضعیت‌ها</option>
            <option value="true">فعال</option>
            <option value="false">غیرفعال</option>
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="card overflow-hidden">
        {isLoading ? <PageSpinner /> : isError ? (
          <ErrorState message="بارگذاری کاربران با خطا مواجه شد" onRetry={refetch} />
        ) : !data?.data?.length ? (
          <EmptyState
            title="کاربری یافت نشد"
            description={search || role || active ? 'فیلترها را تغییر دهید' : 'اولین کاربر را ایجاد کنید'}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr>
                    <th className="table-th">کاربر</th>
                    <th className="table-th">نقش</th>
                    <th className="table-th">وضعیت</th>
                    <th className="table-th">آخرین ورود</th>
                    <th className="table-th">تاریخ ثبت</th>
                    <th className="table-th w-28">عملیات</th>
                  </tr>
                </thead>
                <tbody>
                  {data.data.map((u) => (
                    <tr key={u.id} className="hover:bg-gray-50 transition-colors">
                      <td className="table-td">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-brand-100 flex items-center justify-center shrink-0">
                            <span className="text-xs font-bold text-brand-700">
                              {(u.full_name || u.email).charAt(0).toUpperCase()}
                            </span>
                          </div>
                          <div>
                            <div className="font-medium text-gray-900 dark:text-gray-100 text-sm">
                              {u.full_name || <span className="text-gray-400 dark:text-gray-500 italic">بدون نام</span>}
                              {u.id === me?.id && (
                                <span className="ml-2 text-xs bg-brand-50 text-brand-600 px-1.5 py-0.5 rounded">شما</span>
                              )}
                            </div>
                            <div className="text-xs text-gray-400 dark:text-gray-500">{u.email}</div>
                          </div>
                        </div>
                      </td>
                      <td className="table-td"><RoleBadge role={u.role} /></td>
                      <td className="table-td">
                        <span className={u.is_active ? 'badge-green' : 'badge-gray'}>
                          {u.is_active ? 'فعال' : 'غیرفعال'}
                        </span>
                      </td>
                      <td className="table-td text-gray-500 dark:text-gray-400 text-xs">{formatDate(u.last_login_at)}</td>
                      <td className="table-td text-gray-500 dark:text-gray-400 text-xs">{formatDate(u.created_at)}</td>
                      <td className="table-td">
                        <div className="flex items-center gap-1">
                          <Link
                            to={`/users/${u.id}`}
                            className="btn-ghost p-1.5 text-gray-400 dark:text-gray-500 hover:text-brand-600"
                            title="مشاهده جزئیات"
                          >
                            <Eye className="w-3.5 h-3.5" />
                          </Link>
                          <button
                            onClick={() => setEditUser(u)}
                            className="btn-ghost p-1.5 text-gray-400 dark:text-gray-500 hover:text-blue-600"
                            title="ویرایش"
                          >
                            <Pencil className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => setResetUser(u)}
                            className="btn-ghost p-1.5 text-gray-400 dark:text-gray-500 hover:text-yellow-600"
                            title="ریست رمز"
                          >
                            <KeyRound className="w-3.5 h-3.5" />
                          </button>
                          {u.id !== me?.id && (
                            <button
                              onClick={() => setDeleteUser(u)}
                              className="btn-ghost p-1.5 text-gray-400 dark:text-gray-500 hover:text-red-600 hover:bg-red-50"
                              title="حذف"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination
              page={page}
              totalPages={data.total_pages}
              total={data.total}
              pageSize={20}
              onPageChange={setPage}
            />
          </>
        )}
      </div>

      {/* Modals */}
      {showCreate  && <CreateUserModal  onClose={() => setShowCreate(false)} />}
      {editUser    && <EditUserModal    user={editUser}   onClose={() => setEditUser(null)} />}
      {resetUser   && <ResetPasswordModal user={resetUser} onClose={() => setResetUser(null)} />}
      {deleteUser  && (
        <Modal title="حذف کاربر" onClose={() => setDeleteUser(null)}>
          <div className="mb-5">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              آیا مطمئنید می‌خواهید کاربر زیر را حذف کنید؟
            </p>
            <div className="mt-3 flex items-center gap-3 p-3 bg-gray-50 dark:bg-white/[0.03] rounded-lg">
              <div className="w-8 h-8 rounded-full bg-red-100 flex items-center justify-center shrink-0">
                <span className="text-xs font-bold text-red-700">
                  {deleteUser.email.charAt(0).toUpperCase()}
                </span>
              </div>
              <div>
                <div className="text-sm font-medium text-gray-900 dark:text-gray-100">{deleteUser.full_name || deleteUser.email}</div>
                <div className="text-xs text-gray-400 dark:text-gray-500">{deleteUser.email}</div>
              </div>
            </div>
            <p className="text-xs text-red-600 mt-3">این عملیات غیرقابل بازگشت است.</p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => deleteMutation.mutate(deleteUser.id)}
              disabled={deleteMutation.isPending}
              className="btn-danger flex-1"
            >
              {deleteMutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'بله، حذف شود'}
            </button>
            <button onClick={() => setDeleteUser(null)} className="btn-secondary">انصراف</button>
          </div>
        </Modal>
      )}
    </div>
  )
}

// ── Create modal ──────────────────────────────────────────────────────────────

function CreateUserModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const [form, setForm] = useState({ email: '', password: '', full_name: '', role: 'viewer' })
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.create(form),
    onSuccess: (user) => {
      qc.invalidateQueries({ queryKey: ['users'] })
      toast.success(`کاربر ${user.email} ایجاد شد`)
      onClose()
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا در ایجاد کاربر')
    },
  })

  const valid = form.email && form.password.length >= 8

  return (
    <Modal title="ایجاد کاربر جدید" onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="label">ایمیل <span className="text-red-500">*</span></label>
          <input
            className="input"
            type="email"
            placeholder="user@example.com"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            autoFocus
          />
        </div>
        <div>
          <label className="label">نام کامل</label>
          <input
            className="input"
            placeholder="نام کاربر"
            value={form.full_name}
            onChange={(e) => setForm({ ...form, full_name: e.target.value })}
          />
        </div>
        <div>
          <label className="label">رمز عبور <span className="text-red-500">*</span></label>
          <input
            className="input"
            type="password"
            placeholder="حداقل ۸ کاراکتر"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
          />
          {form.password.length > 0 && form.password.length < 8 && (
            <p className="text-xs text-red-500 mt-1">رمز عبور باید حداقل ۸ کاراکتر باشد</p>
          )}
        </div>
        <div>
          <label className="label">نقش</label>
          <select className="input" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            <option value="viewer">Viewer — فقط مشاهده</option>
            <option value="editor">Editor — مشاهده + ویرایش متادیتا</option>
            <option value="admin">Admin — دسترسی کامل</option>
          </select>
        </div>
        {error && <p className="text-sm text-red-600 bg-red-50 p-2 rounded-lg">{error}</p>}
        <div className="flex gap-3 pt-1">
          <button
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending || !valid}
            className="btn-primary flex-1"
          >
            {mutation.isPending ? <><Spinner className="w-4 h-4 text-white" />در حال ایجاد...</> : 'ایجاد کاربر'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}

// ── Edit modal ────────────────────────────────────────────────────────────────

function EditUserModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const qc = useQueryClient()
  const toast = useToast()
  const [form, setForm] = useState({
    full_name: user.full_name,
    role: user.role,
    is_active: user.is_active,
  })
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.update(user.id, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users'] })
      toast.success('اطلاعات کاربر بروزرسانی شد')
      onClose()
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا در بروزرسانی')
    },
  })

  return (
    <Modal title={`ویرایش — ${user.email}`} onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="label">نام کامل</label>
          <input
            className="input"
            value={form.full_name}
            onChange={(e) => setForm({ ...form, full_name: e.target.value })}
          />
        </div>
        <div>
          <label className="label">نقش</label>
          <select
            className="input"
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value as SafeUser['role'] })}
          >
            <option value="viewer">Viewer — فقط مشاهده</option>
            <option value="editor">Editor — مشاهده + ویرایش متادیتا</option>
            <option value="admin">Admin — دسترسی کامل</option>
          </select>
        </div>
        <label className="flex items-center gap-3 cursor-pointer p-3 rounded-lg hover:bg-gray-50 border border-gray-200 dark:border-white/10">
          <input
            type="checkbox"
            checked={form.is_active}
            onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
            className="w-4 h-4 rounded accent-brand-600"
          />
          <div>
            <div className="text-sm font-medium text-gray-800 dark:text-gray-200">حساب فعال</div>
            <div className="text-xs text-gray-400 dark:text-gray-500">کاربر غیرفعال نمی‌تواند وارد شود</div>
          </div>
        </label>
        {error && <p className="text-sm text-red-600 bg-red-50 p-2 rounded-lg">{error}</p>}
        <div className="flex gap-3 pt-1">
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending} className="btn-primary flex-1">
            {mutation.isPending ? <><Spinner className="w-4 h-4 text-white" />در حال ذخیره...</> : 'ذخیره تغییرات'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}

// ── Reset password modal ──────────────────────────────────────────────────────

function ResetPasswordModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const toast = useToast()
  const [password, setPassword] = useState('')
  const [confirm, setConfirm]   = useState('')
  const [error, setError]       = useState('')
  const [done, setDone]         = useState(false)

  const mutation = useMutation({
    mutationFn: () => usersApi.adminResetPassword(user.id, password),
    onSuccess: () => {
      setDone(true)
      toast.success(`رمز عبور ${user.email} تغییر کرد`)
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'خطا در تغییر رمز')
    },
  })

  const handleSubmit = () => {
    setError('')
    if (password.length < 8) { setError('رمز باید حداقل ۸ کاراکتر باشد'); return }
    if (password !== confirm) { setError('تکرار رمز مطابقت ندارد'); return }
    mutation.mutate()
  }

  return (
    <Modal title={`ریست رمز — ${user.email}`} onClose={onClose}>
      {done ? (
        <div className="text-center py-6">
          <div className="w-14 h-14 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-3">
            <span className="text-2xl">✅</span>
          </div>
          <p className="text-sm font-medium text-gray-800 dark:text-gray-200">رمز با موفقیت تغییر کرد</p>
          <button onClick={onClose} className="btn-secondary mt-4">بستن</button>
        </div>
      ) : (
        <div className="space-y-4">
          <div>
            <label className="label">رمز جدید</label>
            <input
              className="input"
              type="password"
              placeholder="حداقل ۸ کاراکتر"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoFocus
            />
          </div>
          <div>
            <label className="label">تکرار رمز جدید</label>
            <input
              className="input"
              type="password"
              placeholder="تکرار کنید"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
          </div>
          {error && <p className="text-sm text-red-600 bg-red-50 p-2 rounded-lg">{error}</p>}
          <div className="flex gap-3 pt-1">
            <button
              onClick={handleSubmit}
              disabled={mutation.isPending || !password || !confirm}
              className="btn-primary flex-1"
            >
              {mutation.isPending ? <><Spinner className="w-4 h-4 text-white" />در حال تغییر...</> : 'تغییر رمز'}
            </button>
            <button onClick={onClose} className="btn-secondary">انصراف</button>
          </div>
        </div>
      )}
    </Modal>
  )
}
