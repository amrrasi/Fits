import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import api from '../../api/client'
import { PageSpinner, ErrorState, EmptyState, Pagination } from '../../components/ui'
import { useDebounce } from '../../hooks/useDebounce'
import type { ApiResponse, PagedResponse } from '../../types'
import { formatDate } from '../../utils/format'
import { Shield } from 'lucide-react'

interface AuditLog {
  id: number
  user_id?: number
  user_email?: string
  action: string
  entity_type: string
  entity_id?: string
  old_value?: unknown
  new_value?: unknown
  ip_address?: string
  request_id?: string
  created_at: string
}

const ACTION_COLORS: Record<string, string> = {
  'user.create':      'badge-green',
  'user.update':      'badge-blue',
  'user.delete':      'badge-red',
  'user.login':       'badge-gray',
  'user.logout':      'badge-gray',
  'metadata.edit':    'badge-blue',
  'file.delete':      'badge-red',
  'scan.trigger':     'badge-yellow',
  'password.change':  'badge-yellow',
  'password.reset':   'badge-yellow',
}

function actionBadge(action: string) {
  const cls = ACTION_COLORS[action] ?? 'badge-gray'
  return <span className={cls}>{action}</span>
}

export default function AuditLogPage() {
  const [page, setPage]           = useState(1)
  const [action, setAction]       = useState('')
  const [entityType, setEntityType] = useState('')
  const [dateFrom, setDateFrom]   = useState('')
  const [dateTo, setDateTo]       = useState('')

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['audit-logs', page, action, entityType, dateFrom, dateTo],
    queryFn: () =>
      api.get<PagedResponse<AuditLog>>('/audit-logs', {
        params: { page, page_size: 50, action, entity_type: entityType, date_from: dateFrom, date_to: dateTo },
      }).then((r) => r.data),
  })

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 bg-purple-100 rounded-lg flex items-center justify-center">
          <Shield className="w-4 h-4 text-purple-600" />
        </div>
        <div>
          <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">لاگ‌های ممیزی</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
            {isLoading ? '...' : `${data?.total ?? 0} رویداد`}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="card p-4">
        <div className="flex flex-wrap gap-3">
          <select
            className="input w-48"
            value={action}
            onChange={(e) => { setAction(e.target.value); setPage(1) }}
          >
            <option value="">همه اقدامات</option>
            <option value="user.create">user.create</option>
            <option value="user.update">user.update</option>
            <option value="user.delete">user.delete</option>
            <option value="user.login">user.login</option>
            <option value="metadata.edit">metadata.edit</option>
            <option value="file.delete">file.delete</option>
            <option value="scan.trigger">scan.trigger</option>
            <option value="password.change">password.change</option>
            <option value="password.reset">password.reset</option>
          </select>

          <select
            className="input w-44"
            value={entityType}
            onChange={(e) => { setEntityType(e.target.value); setPage(1) }}
          >
            <option value="">همه موجودیت‌ها</option>
            <option value="user">user</option>
            <option value="fits_file">fits_file</option>
            <option value="metadata">metadata</option>
            <option value="scan">scan</option>
          </select>

          <div className="flex items-center gap-2">
            <input
              type="date"
              className="input w-40"
              value={dateFrom}
              onChange={(e) => { setDateFrom(e.target.value); setPage(1) }}
              placeholder="از تاریخ"
            />
            <span className="text-gray-400 dark:text-gray-500 text-sm">تا</span>
            <input
              type="date"
              className="input w-40"
              value={dateTo}
              onChange={(e) => { setDateTo(e.target.value); setPage(1) }}
              placeholder="تا تاریخ"
            />
          </div>

          {(action || entityType || dateFrom || dateTo) && (
            <button
              className="btn-ghost text-sm"
              onClick={() => { setAction(''); setEntityType(''); setDateFrom(''); setDateTo(''); setPage(1) }}
            >
              پاک کردن فیلترها
            </button>
          )}
        </div>
      </div>

      {/* Table */}
      <div className="card overflow-hidden">
        {isLoading ? <PageSpinner /> : isError ? (
          <ErrorState message="بارگذاری لاگ‌ها با خطا مواجه شد" onRetry={refetch} />
        ) : !data?.data?.length ? (
          <EmptyState title="رویدادی یافت نشد" description="هنوز هیچ فعالیتی ثبت نشده یا فیلترها نتیجه‌ای ندارند" />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr>
                    <th className="table-th">اقدام</th>
                    <th className="table-th">کاربر</th>
                    <th className="table-th">موجودیت</th>
                    <th className="table-th">شناسه</th>
                    <th className="table-th">IP</th>
                    <th className="table-th">زمان</th>
                    <th className="table-th w-20">جزئیات</th>
                  </tr>
                </thead>
                <tbody>
                  {data.data.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50 transition-colors">
                      <td className="table-td">{actionBadge(log.action)}</td>
                      <td className="table-td">
                        {log.user_email ? (
                          <span className="text-sm text-gray-700 dark:text-gray-300">{log.user_email}</span>
                        ) : (
                          <span className="text-xs text-gray-400 dark:text-gray-500 italic">سیستم</span>
                        )}
                      </td>
                      <td className="table-td">
                        <span className="text-xs font-mono bg-gray-100 dark:bg-white/[0.07] px-1.5 py-0.5 rounded text-gray-600 dark:text-gray-400">
                          {log.entity_type}
                        </span>
                      </td>
                      <td className="table-td text-xs text-gray-400 dark:text-gray-500 font-mono">
                        {log.entity_id ?? '—'}
                      </td>
                      <td className="table-td text-xs text-gray-400 dark:text-gray-500 font-mono">
                        {log.ip_address ?? '—'}
                      </td>
                      <td className="table-td text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap">
                        {formatDate(log.created_at)}
                      </td>
                      <td className="table-td">
                        {(log.old_value || log.new_value) && (
                          <ChangeDiff old={log.old_value} next={log.new_value} />
                        )}
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
              pageSize={50}
              onPageChange={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}

// ── Change diff popover ───────────────────────────────────────────────────────

function ChangeDiff({ old: oldVal, next: newVal }: { old: unknown; next: unknown }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="text-xs text-brand-600 hover:text-brand-800 font-medium"
      >
        مشاهده
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-6 z-50 bg-white dark:bg-white/[0.04] border border-gray-200 dark:border-white/10 rounded-xl shadow-xl p-4 w-80">
            <p className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-2">تغییرات</p>
            {oldVal !== undefined && oldVal !== null && (
              <div className="mb-2">
                <p className="text-xs text-red-500 font-medium mb-1">قبل:</p>
                <pre className="text-xs bg-red-50 p-2 rounded-lg overflow-auto max-h-24 text-red-700">
                  {JSON.stringify(oldVal, null, 2)}
                </pre>
              </div>
            )}
            {newVal !== undefined && newVal !== null && (
              <div>
                <p className="text-xs text-green-500 font-medium mb-1">بعد:</p>
                <pre className="text-xs bg-green-50 p-2 rounded-lg overflow-auto max-h-24 text-green-700">
                  {JSON.stringify(newVal, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
