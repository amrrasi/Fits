import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { statsApi, jobsApi } from '../../api/endpoints'
import { PageSpinner, JobStatusBadge } from '../../components/ui'
import { Files, CheckCircle2, AlertCircle, Clock, Cpu, Hash, Play } from 'lucide-react'
import { formatDate } from '../../utils/format'
import { useAuth } from '../../context/AuthContext'

export default function DashboardPage() {
  const { isAdmin } = useAuth()

  const statsQ = useQuery({
    queryKey: ['stats'],
    queryFn: statsApi.get,
    refetchInterval: 10_000,
  })

  const jobsQ = useQuery({
    queryKey: ['jobs-recent'],
    queryFn: () => jobsApi.list({ page: 1, page_size: 5 }),
    refetchInterval: 5_000,
  })

  const stats = statsQ.data

  const cards = [
    { label: 'کل فایل‌ها',    value: stats?.total_files ?? '—',               icon: Files,         color: 'bg-blue-50 text-blue-600',   to: '/files' },
    { label: 'پردازش‌شده',    value: stats?.done_files ?? '—',                icon: CheckCircle2,  color: 'bg-green-50 text-green-600', to: '/files?status=done' },
    { label: 'خطا',            value: stats?.error_files ?? '—',               icon: AlertCircle,   color: 'bg-red-50 text-red-600',     to: '/files?status=error' },
    { label: 'در انتظار',      value: stats?.pending_files ?? '—',             icon: Clock,         color: 'bg-yellow-50 text-yellow-600', to: '/files?status=pending' },
    { label: 'کل اسکن‌ها',    value: stats?.total_jobs ?? '—',                icon: Cpu,           color: 'bg-purple-50 text-purple-600', to: '/jobs' },
    { label: 'کل هدرها',       value: stats?.total_headers?.toLocaleString() ?? '—', icon: Hash,   color: 'bg-gray-50 text-gray-600',   to: '/files' },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-gray-900">داشبورد</h1>
        <p className="text-sm text-gray-500 mt-0.5">خلاصه وضعیت سیستم پردازش FITS</p>
      </div>

      {/* Running job banner */}
      {stats?.running_jobs ? (
        <div className="flex items-center gap-3 p-4 bg-blue-50 border border-blue-200 rounded-xl">
          <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse shrink-0" />
          <p className="text-sm text-blue-800 font-medium">{stats.running_jobs} اسکن در حال اجرا است</p>
          <Link to="/jobs" className="ml-auto text-sm text-blue-700 font-semibold hover:underline">مشاهده →</Link>
        </div>
      ) : null}

      {/* Stats grid */}
      {statsQ.isLoading ? <PageSpinner /> : (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {cards.map(({ label, value, icon: Icon, color, to }) => (
            <Link key={label} to={to} className="card p-5 hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-xs text-gray-500 font-medium">{label}</p>
                  <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
                </div>
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 ${color}`}>
                  <Icon className="w-5 h-5" />
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Recent jobs */}
      <div className="card overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-200">
          <h2 className="text-sm font-semibold text-gray-900">آخرین اسکن‌ها</h2>
          <div className="flex items-center gap-3">
            {isAdmin && <ScanButton />}
            <Link to="/jobs" className="text-sm text-brand-600 hover:text-brand-700 font-medium">همه اسکن‌ها →</Link>
          </div>
        </div>

        {jobsQ.isLoading ? <PageSpinner /> : !jobsQ.data?.data?.length ? (
          <div className="px-5 py-8 text-center text-sm text-gray-400">هیچ اسکنی یافت نشد</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr>
                <th className="table-th">ID</th>
                <th className="table-th">وضعیت</th>
                <th className="table-th">پیشرفت</th>
                <th className="table-th">شروع</th>
                <th className="table-th"></th>
              </tr>
            </thead>
            <tbody>
              {jobsQ.data?.data.map((job) => {
                const pct = job.total_files > 0
                  ? Math.round(((job.done_files + job.error_files) / job.total_files) * 100) : 0
                return (
                  <tr key={job.id} className="hover:bg-gray-50">
                    <td className="table-td font-mono text-gray-400 text-xs">#{job.id}</td>
                    <td className="table-td"><JobStatusBadge status={job.status} /></td>
                    <td className="table-td">
                      <div className="flex items-center gap-2">
                        <div className="flex-1 bg-gray-100 rounded-full h-1.5 min-w-16">
                          <div
                            className={`h-1.5 rounded-full transition-all ${
                              job.status === 'failed' ? 'bg-red-500' :
                              job.status === 'completed' ? 'bg-green-500' : 'bg-brand-500'
                            }`}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        <span className="text-xs text-gray-500 shrink-0">{job.done_files}/{job.total_files}</span>
                      </div>
                    </td>
                    <td className="table-td text-gray-500 text-xs">{formatDate(job.started_at)}</td>
                    <td className="table-td">
                      <Link to={`/jobs/${job.id}`} className="text-brand-600 hover:text-brand-700 text-sm font-medium">جزئیات →</Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function ScanButton() {
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')

  const trigger = async () => {
    setLoading(true)
    setMsg('')
    try {
      const res = await jobsApi.triggerScan()
      setMsg(`Job #${res.job_id} شروع شد`)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setMsg(err?.response?.data?.error ?? 'خطا')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex items-center gap-2">
      {msg && <span className="text-xs text-gray-500">{msg}</span>}
      <button onClick={trigger} disabled={loading} className="btn-primary py-1.5 px-3 text-xs">
        <Play className="w-3 h-3" />
        شروع اسکن
      </button>
    </div>
  )
}
