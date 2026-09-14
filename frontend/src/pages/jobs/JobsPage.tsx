import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { jobsApi } from '../../api/endpoints'
import { JobStatusBadge, PageSpinner, ErrorState, EmptyState, Pagination } from '../../components/ui'
import { useAuth } from '../../context/AuthContext'
import { Play, ExternalLink } from 'lucide-react'
import { formatDate, formatDuration } from '../../utils/format'

export default function JobsPage() {
  const [page, setPage] = useState(1)
  const [scanDir, setScanDir] = useState('')
  const [scanMsg, setScanMsg] = useState('')
  const { isAdmin } = useAuth()

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['jobs', page],
    queryFn: () => jobsApi.list({ page, page_size: 20 }),
    refetchInterval: 5000, // poll while jobs may be running
  })

  const scanMutation = useMutation({
    mutationFn: () => jobsApi.triggerScan(scanDir || undefined),
    onSuccess: (res) => {
      setScanMsg(`اسکن شروع شد — Job ID: ${res.job_id}`)
      refetch()
    },
    onError: (err: unknown) => {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setScanMsg(msg ?? 'خطا در شروع اسکن')
    },
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-900">پردازش‌های اسکن</h1>
          <p className="text-sm text-gray-500 mt-0.5">تاریخچه و وضعیت اسکن‌های FITS</p>
        </div>

        {isAdmin && (
          <div className="flex items-center gap-2">
            <input
              className="input w-56"
              placeholder="مسیر اسکن (اختیاری)"
              value={scanDir}
              onChange={(e) => setScanDir(e.target.value)}
            />
            <button
              onClick={() => scanMutation.mutate()}
              disabled={scanMutation.isPending}
              className="btn-primary"
            >
              <Play className="w-4 h-4" />
              شروع اسکن
            </button>
          </div>
        )}
      </div>

      {scanMsg && (
        <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-700">
          {scanMsg}
        </div>
      )}

      <div className="card overflow-hidden">
        {isLoading ? (
          <PageSpinner />
        ) : isError ? (
          <ErrorState message="بارگذاری جاب‌ها با خطا مواجه شد" onRetry={refetch} />
        ) : !data?.data?.length ? (
          <EmptyState title="هیچ اسکنی یافت نشد" description="روی دکمه شروع اسکن کلیک کنید" />
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr>
                  <th className="table-th">ID</th>
                  <th className="table-th">وضعیت</th>
                  <th className="table-th">مسیر</th>
                  <th className="table-th">فایل‌ها</th>
                  <th className="table-th">شروع</th>
                  <th className="table-th">مدت</th>
                  <th className="table-th"></th>
                </tr>
              </thead>
              <tbody>
                {data.data.map((job) => {
                  const duration = job.started_at && job.finished_at
                    ? formatDuration(new Date(job.finished_at).getTime() - new Date(job.started_at).getTime())
                    : job.status === 'running' ? 'در حال اجرا…' : '—'
                  return (
                    <tr key={job.id} className="hover:bg-gray-50">
                      <td className="table-td font-mono text-gray-400">#{job.id}</td>
                      <td className="table-td"><JobStatusBadge status={job.status} /></td>
                      <td className="table-td text-gray-500 max-w-xs truncate font-mono text-xs">{job.scan_dir}</td>
                      <td className="table-td">
                        <span className="text-green-700">{job.done_files}</span>
                        <span className="text-gray-400"> / </span>
                        <span>{job.total_files}</span>
                        {job.error_files > 0 && (
                          <span className="text-red-600 ml-1">({job.error_files} خطا)</span>
                        )}
                      </td>
                      <td className="table-td text-gray-500 text-xs">{formatDate(job.started_at)}</td>
                      <td className="table-td text-gray-500 text-xs">{duration}</td>
                      <td className="table-td">
                        <Link to={`/jobs/${job.id}`} className="text-brand-600 hover:text-brand-700 inline-flex items-center gap-1 text-sm font-medium">
                          جزئیات <ExternalLink className="w-3 h-3" />
                        </Link>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
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
    </div>
  )
}
