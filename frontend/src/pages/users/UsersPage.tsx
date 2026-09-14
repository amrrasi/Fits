import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { usersApi } from '../../api/endpoints'
import {
  RoleBadge, PageSpinner, ErrorState, EmptyState, Pagination, Modal, Spinner
} from '../../components/ui'
import { useAuth } from '../../context/AuthContext'
import { UserPlus, Pencil, Trash2, KeyRound, Search } from 'lucide-react'
import { formatDate } from '../../utils/format'
import type { SafeUser } from '../../types'

export default function UsersPage() {
  const { user: me } = useAuth()
  const qc = useQueryClient()
  const [page, setPage]     = useState(1)
  const [search, setSearch] = useState('')
  const [role, setRole]     = useState('')

  const [showCreate, setShowCreate]       = useState(false)
  const [editUser, setEditUser]           = useState<SafeUser | null>(null)
  const [deleteUser, setDeleteUser]       = useState<SafeUser | null>(null)
  const [resetUser, setResetUser]         = useState<SafeUser | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['users', page, search, role],
    queryFn: () => usersApi.list({ page, page_size: 20, search, role }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => usersApi.delete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); setDeleteUser(null) },
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-900">مدیریت کاربران</h1>
          <p className="text-sm text-gray-500 mt-0.5">{data ? `${data.total} کاربر` : '...'}</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="btn-primary">
          <UserPlus className="w-4 h-4" /> کاربر جدید
        </button>
      </div>

      {/* Filters */}
      <div className="card p-4 flex gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
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
      </div>

      <div className="card overflow-hidden">
        {isLoading ? <PageSpinner /> : isError ? (
          <ErrorState message="بارگذاری کاربران با خطا مواجه شد" onRetry={refetch} />
        ) : !data?.data?.length ? (
          <EmptyState title="کاربری یافت نشد" />
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr>
                  <th className="table-th">کاربر</th>
                  <th className="table-th">نقش</th>
                  <th className="table-th">وضعیت</th>
                  <th className="table-th">آخرین ورود</th>
                  <th className="table-th">تاریخ ثبت</th>
                  <th className="table-th"></th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((u) => (
                  <tr key={u.id} className="hover:bg-gray-50">
                    <td className="table-td">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-brand-100 flex items-center justify-center shrink-0">
                          <span className="text-xs font-semibold text-brand-700">
                            {(u.full_name || u.email).charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div>
                          <div className="font-medium text-gray-900">{u.full_name || '—'}</div>
                          <div className="text-xs text-gray-400">{u.email}</div>
                        </div>
                      </div>
                    </td>
                    <td className="table-td"><RoleBadge role={u.role} /></td>
                    <td className="table-td">
                      <span className={u.is_active ? 'badge-green' : 'badge-gray'}>
                        {u.is_active ? 'فعال' : 'غیرفعال'}
                      </span>
                    </td>
                    <td className="table-td text-gray-500 text-xs">{formatDate(u.last_login_at)}</td>
                    <td className="table-td text-gray-500 text-xs">{formatDate(u.created_at)}</td>
                    <td className="table-td">
                      <div className="flex items-center gap-1">
                        <button onClick={() => setEditUser(u)} className="btn-ghost p-1.5" title="ویرایش">
                          <Pencil className="w-3.5 h-3.5" />
                        </button>
                        <button onClick={() => setResetUser(u)} className="btn-ghost p-1.5" title="ریست رمز">
                          <KeyRound className="w-3.5 h-3.5" />
                        </button>
                        {u.id !== me?.id && (
                          <button onClick={() => setDeleteUser(u)} className="btn-ghost p-1.5 text-red-400 hover:text-red-600 hover:bg-red-50" title="حذف">
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <Pagination page={page} totalPages={data.total_pages} total={data.total} pageSize={20} onPageChange={setPage} />
          </>
        )}
      </div>

      {showCreate && <CreateUserModal onClose={() => setShowCreate(false)} />}
      {editUser   && <EditUserModal   user={editUser}   onClose={() => setEditUser(null)} />}
      {resetUser  && <ResetPasswordModal user={resetUser} onClose={() => setResetUser(null)} />}
      {deleteUser && (
        <Modal title="حذف کاربر" onClose={() => setDeleteUser(null)}>
          <p className="text-sm text-gray-600 mb-5">
            آیا مطمئنید می‌خواهید <span className="font-semibold">{deleteUser.email}</span> را حذف کنید؟
          </p>
          <div className="flex gap-3">
            <button
              onClick={() => deleteMutation.mutate(deleteUser.id)}
              disabled={deleteMutation.isPending}
              className="btn-danger flex-1"
            >
              {deleteMutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'حذف کاربر'}
            </button>
            <button onClick={() => setDeleteUser(null)} className="btn-secondary">انصراف</button>
          </div>
        </Modal>
      )}
    </div>
  )
}

// ── Create user modal ─────────────────────────────────────────────────────────

function CreateUserModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient()
  const [form, setForm] = useState({ email: '', password: '', full_name: '', role: 'viewer' })
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.create(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); onClose() },
    onError: (err: unknown) => {
      setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'خطا')
    },
  })

  return (
    <Modal title="ایجاد کاربر جدید" onClose={onClose}>
      <div className="space-y-3">
        <div><label className="label">ایمیل</label><input className="input" type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
        <div><label className="label">نام کامل</label><input className="input" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} /></div>
        <div><label className="label">رمز عبور</label><input className="input" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></div>
        <div>
          <label className="label">نقش</label>
          <select className="input" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex gap-3 pt-2">
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending} className="btn-primary flex-1">
            {mutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'ایجاد'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}

// ── Edit user modal ───────────────────────────────────────────────────────────

function EditUserModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const qc = useQueryClient()
  const [form, setForm] = useState({ full_name: user.full_name, role: user.role, is_active: user.is_active })
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: () => usersApi.update(user.id, form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); onClose() },
    onError: (err: unknown) => {
      setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'خطا')
    },
  })

  return (
    <Modal title="ویرایش کاربر" onClose={onClose}>
      <div className="space-y-3">
        <div><label className="label">نام کامل</label><input className="input" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} /></div>
        <div>
          <label className="label">نقش</label>
          <select className="input" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value as SafeUser['role'] })}>
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <label className="flex items-center gap-2 cursor-pointer">
          <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} className="rounded" />
          <span className="text-sm text-gray-700">حساب فعال باشد</span>
        </label>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex gap-3 pt-2">
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending} className="btn-primary flex-1">
            {mutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'ذخیره'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}

// ── Reset password modal ──────────────────────────────────────────────────────

function ResetPasswordModal({ user, onClose }: { user: SafeUser; onClose: () => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  const mutation = useMutation({
    mutationFn: () => usersApi.adminResetPassword(user.id, password),
    onSuccess: () => setDone(true),
    onError: (err: unknown) => {
      setError((err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'خطا')
    },
  })

  return (
    <Modal title={`ریست رمز — ${user.email}`} onClose={onClose}>
      {done ? (
        <div className="text-center py-4">
          <div className="text-3xl mb-2">✅</div>
          <p className="text-sm text-gray-700">رمز با موفقیت تغییر کرد</p>
          <button onClick={onClose} className="btn-secondary mt-4">بستن</button>
        </div>
      ) : (
        <div className="space-y-3">
          <div><label className="label">رمز جدید</label>
            <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <div className="flex gap-3 pt-2">
            <button onClick={() => mutation.mutate()} disabled={mutation.isPending || password.length < 8} className="btn-primary flex-1">
              {mutation.isPending ? <Spinner className="w-4 h-4 text-white" /> : 'تغییر رمز'}
            </button>
            <button onClick={onClose} className="btn-secondary">انصراف</button>
          </div>
        </div>
      )}
    </Modal>
  )
}
