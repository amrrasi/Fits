import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { jobsApi } from '../../api/endpoints'
import { JobStatusBadge, PageSpinner, ErrorState, EmptyState, Pagination } from '../../components/ui'
import { ArrowLeft, RefreshCw } from 'lucide-react'
import { formatDate, formatDuration } from '../../utils/format'

export default function JobDetailPage() {
  const { id } = useParams<{ id: string }>()
  const jobId = Number(id)
  const [errPage, setErrPage] = useState(1)

  // Poll job status while it's running
  const jobQ = useQuery({
    queryKey: ['job', jobId],
    queryFn: () => jobsApi.getById(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'running' ? 2000 : false
    },
  })

  const errorsQ = useQuery({
    queryKey: ['job-errors', jobId, errPage],
    queryFn: () => jobsApi.errors(jobId, { page: errPage, page_size: 20 }),
    enabled: !!jobQ.data && jobQ.data.status !== 'running',
  })

  const job = jobQ.data

  if (jobQ.isLoading) return <PageSpinner />
  if (jobQ.isError || !job) return <ErrorState message="جاب یافت نشد" />

  const pct = job.total_files > 0
    ? Math.round(((job.done_files + job.error_files) / job.total_files) * 100)
    : 0

  const durationMs = job.started_at && job.finished_at
    ? new Date(job.finished_at).getTime() - new Date(job.started_at).getTime()
    : job.started_at
      ? Date.now() - new Date(job.started_at).getTime()
      : 0

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/jobs" className="text-gray-400 dark:text-gray-500 hover:text-gray-600">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">اسکن #{job.id}</h1>
            <JobStatusBadge status={job.status} />
            {job.status === 'running' && (
              <RefreshCw className="w-4 h-4 text-brand-500 animate-spin" />
            )}
          </div>
          <p className="text-sm text-gray-400 dark:text-gray-500 font-mono mt-0.5">{job.scan_dir}</p>
        </div>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: 'کل فایل‌ها',     value: job.total_files, color: 'text-gray-900 dark:text-gray-100' },
          { label: 'موفق',            value: job.done_files,  color: 'text-green-700' },
          { label: 'خطا',             value: job.error_files, color: 'text-red-600' },
          { label: 'مدت زمان',        value: formatDuration(durationMs), color: 'text-gray-700 dark:text-gray-300' },
        ].map(({ label, value, color }) => (
          <div key={label} className="card px-5 py-4">
            <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
            <p className={`text-2xl font-bold mt-1 ${color}`}>{value}</p>
          </div>
        ))}
      </div>

      {/* Progress bar */}
      <div className="card p-5">
        <div className="flex justify-between text-sm mb-2">
          <span className="font-medium text-gray-700 dark:text-gray-300">پیشرفت پردازش</span>
          <span className="font-bold text-gray-900 dark:text-gray-100">{pct}%</span>
        </div>
        <div className="w-full bg-gray-100 dark:bg-white/[0.07] rounded-full h-3 overflow-hidden">
          <div
            className={`h-3 rounded-full transition-all duration-500 ${
              job.status === 'failed' ? 'bg-red-500' :
              job.status === 'completed' ? 'bg-green-500' :
              job.status === 'partially_failed' ? 'bg-yellow-500' : 'bg-brand-500'
            }`}
            style={{ width: `${pct}%` }}
          />
        </div>

        <div className="grid grid-cols-2 gap-4 mt-4 text-sm text-gray-500 dark:text-gray-400">
          <div><span className="text-gray-400 dark:text-gray-500">شروع: </span>{formatDate(job.started_at)}</div>
          <div><span className="text-gray-400 dark:text-gray-500">پایان: </span>{formatDate(job.finished_at)}</div>
        </div>

        {job.error_message && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {job.error_message}
          </div>
        )}
      </div>

      {/* Errors table */}
      {job.status !== 'running' && (
        <div className="card overflow-hidden">
          <div className="px-5 py-4 border-b border-gray-200 dark:border-white/10">
            <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
              خطاهای پردازش
              {errorsQ.data?.total ? (
                <span className="ml-2 text-xs font-normal text-red-600 bg-red-50 px-2 py-0.5 rounded-full">
                  {errorsQ.data.total} خطا
                </span>
              ) : null}
            </h2>
          </div>

          {errorsQ.isLoading ? (
            <PageSpinner />
          ) : !errorsQ.data?.data?.length ? (
            <EmptyState title="هیچ خطایی ثبت نشده" description="همه فایل‌ها با موفقیت پردازش شدند" />
          ) : (
            <>
              <table className="w-full">
                <thead>
                  <tr>
                    <th className="table-th">مرحله</th>
                    <th className="table-th">مسیر فایل</th>
                    <th className="table-th">پیام خطا</th>
                    <th className="table-th">زمان</th>
                  </tr>
                </thead>
                <tbody>
                  {errorsQ.data.data.map((e) => (
                    <tr key={e.id} className="hover:bg-gray-50">
                      <td className="table-td">
                        <span className={`badge ${
                          e.stage === 'parse' ? 'badge-yellow' :
                          e.stage === 'insert' ? 'badge-red' : 'badge-gray'
                        }`}>
                          {e.stage}
                        </span>
                      </td>
                      <td className="table-td max-w-xs">
                        <span className="text-xs font-mono text-gray-500 dark:text-gray-400 truncate block" title={e.file_path}>
                          {e.file_path.split('/').pop()}
                        </span>
                        <span className="text-xs text-gray-300 dark:text-gray-600 truncate block">{e.file_path}</span>
                      </td>
                      <td className="table-td max-w-sm">
                        <span className="text-xs text-red-700 line-clamp-2" title={e.message}>
                          {e.message}
                        </span>
                      </td>
                      <td className="table-td text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap">
                        {formatDate(e.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <Pagination
                page={errPage}
                totalPages={errorsQ.data.total_pages}
                total={errorsQ.data.total}
                pageSize={20}
                onPageChange={setErrPage}
              />
            </>
          )}
        </div>
      )}
    </div>
  )
}
