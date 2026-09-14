import { Loader2, AlertCircle, Inbox, ChevronLeft, ChevronRight } from 'lucide-react'
import type { FileStatus, JobStatus } from '../../types'
import clsx from 'clsx'

// ── Spinner ───────────────────────────────────────────────────────────────────

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={clsx('animate-spin', className ?? 'w-5 h-5 text-brand-600')} />
}

export function PageSpinner() {
  return (
    <div className="flex items-center justify-center h-64">
      <Spinner className="w-8 h-8 text-brand-600" />
    </div>
  )
}

// ── Status Badges ─────────────────────────────────────────────────────────────

const fileStatusMap: Record<FileStatus, { label: string; className: string }> = {
  done:       { label: 'Done',       className: 'badge-green' },
  processing: { label: 'Processing', className: 'badge-blue' },
  pending:    { label: 'Pending',    className: 'badge-yellow' },
  error:      { label: 'Error',      className: 'badge-red' },
  skipped:    { label: 'Skipped',    className: 'badge-gray' },
}

const jobStatusMap: Record<JobStatus, { label: string; className: string }> = {
  completed:        { label: 'Completed',        className: 'badge-green' },
  running:          { label: 'Running',          className: 'badge-blue' },
  partially_failed: { label: 'Partial',          className: 'badge-yellow' },
  failed:           { label: 'Failed',           className: 'badge-red' },
  cancelled:        { label: 'Cancelled',        className: 'badge-gray' },
}

export function FileStatusBadge({ status }: { status: FileStatus }) {
  const cfg = fileStatusMap[status] ?? { label: status, className: 'badge-gray' }
  return <span className={cfg.className}>{cfg.label}</span>
}

export function JobStatusBadge({ status }: { status: JobStatus }) {
  const cfg = jobStatusMap[status] ?? { label: status, className: 'badge-gray' }
  return <span className={cfg.className}>{cfg.label}</span>
}

export function RoleBadge({ role }: { role: string }) {
  const cls = role === 'admin' ? 'badge-red' : role === 'editor' ? 'badge-blue' : 'badge-gray'
  return <span className={cls}>{role}</span>
}

// ── Empty / Error states ──────────────────────────────────────────────────────

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Inbox className="w-12 h-12 text-gray-300 mb-3" />
      <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      {description && <p className="text-sm text-gray-500 mt-1 max-w-xs">{description}</p>}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <AlertCircle className="w-10 h-10 text-red-400 mb-3" />
      <h3 className="text-sm font-semibold text-gray-900">خطایی رخ داده</h3>
      <p className="text-sm text-gray-500 mt-1 max-w-sm">{message}</p>
      {onRetry && (
        <button onClick={onRetry} className="btn-secondary mt-4">
          تلاش مجدد
        </button>
      )}
    </div>
  )
}

// ── Pagination ────────────────────────────────────────────────────────────────

interface PaginationProps {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPageChange: (p: number) => void
}

export function Pagination({ page, totalPages, total, pageSize, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null

  const from = (page - 1) * pageSize + 1
  const to   = Math.min(page * pageSize, total)

  return (
    <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200 bg-white">
      <p className="text-sm text-gray-500">
        نمایش <span className="font-medium">{from}</span> تا <span className="font-medium">{to}</span> از <span className="font-medium">{total}</span>
      </p>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="p-1.5 rounded-lg hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
        {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
          const p = totalPages <= 7 ? i + 1 : i + Math.max(1, page - 3)
          if (p > totalPages) return null
          return (
            <button
              key={p}
              onClick={() => onPageChange(p)}
              className={clsx(
                'w-8 h-8 text-sm rounded-lg font-medium',
                p === page ? 'bg-brand-600 text-white' : 'hover:bg-gray-100 text-gray-700'
              )}
            >
              {p}
            </button>
          )
        })}
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="p-1.5 rounded-lg hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}

// ── Modal ─────────────────────────────────────────────────────────────────────

export function Modal({ title, onClose, children }: {
  title: string
  onClose: () => void
  children: React.ReactNode
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
        <h2 className="text-base font-semibold text-gray-900 mb-4">{title}</h2>
        {children}
      </div>
    </div>
  )
}
