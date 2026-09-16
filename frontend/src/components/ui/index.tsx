import React from 'react'
import { Loader2, AlertCircle, Inbox, ChevronLeft, ChevronRight, X } from 'lucide-react'
import type { FileStatus, JobStatus } from '../../types'
import clsx from 'clsx'

// ── Spinner ───────────────────────────────────────────────────────────────────

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={clsx('animate-spin', className ?? 'w-5 h-5 text-brand-600 dark:text-brand-400')} />
}

export function PageSpinner() {
  return (
    <div className="flex items-center justify-center h-64">
      <Spinner className="w-8 h-8 text-brand-600 dark:text-brand-400" />
    </div>
  )
}

// ── Status Badges ─────────────────────────────────────────────────────────────

const fileStatusMap: Record<FileStatus, { label: string; className: string }> = {
  done:       { label: 'انجام‌شده',   className: 'badge-green' },
  processing: { label: 'در حال پردازش', className: 'badge-blue' },
  pending:    { label: 'در صف',       className: 'badge-yellow' },
  error:      { label: 'خطا',         className: 'badge-red' },
  skipped:    { label: 'رد شده',      className: 'badge-gray' },
}

const jobStatusMap: Record<JobStatus, { label: string; className: string }> = {
  completed:        { label: 'تکمیل‌شده',  className: 'badge-green' },
  running:          { label: 'در حال اجرا', className: 'badge-blue' },
  partially_failed: { label: 'ناقص',       className: 'badge-yellow' },
  failed:           { label: 'ناموفق',     className: 'badge-red' },
  cancelled:        { label: 'لغوشده',     className: 'badge-gray' },
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
    <div className="flex flex-col items-center justify-center py-16 text-center enter-pop">
      <div className="w-14 h-14 rounded-2xl grid place-items-center mb-3
                      bg-gray-100 dark:bg-white/[0.05]">
        <Inbox className="w-6 h-6 text-gray-400 dark:text-gray-500" />
      </div>
      <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{title}</h3>
      {description && <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-xs">{description}</p>}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center enter-pop">
      <div className="w-14 h-14 rounded-2xl grid place-items-center mb-3
                      bg-red-50 dark:bg-red-400/10">
        <AlertCircle className="w-6 h-6 text-red-500 dark:text-red-400" />
      </div>
      <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">خطایی رخ داده</h3>
      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm">{message}</p>
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
    <div className="flex items-center justify-between px-4 py-3 border-t border-black/[0.06] dark:border-white/[0.06]">
      <p className="text-sm text-gray-500 dark:text-gray-400">
        نمایش <span className="font-medium text-gray-700 dark:text-gray-300">{from}</span> تا <span className="font-medium text-gray-700 dark:text-gray-300">{to}</span> از <span className="font-medium text-gray-700 dark:text-gray-300">{total}</span>
      </p>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="p-1.5 rounded-lg text-gray-500 hover:bg-black/[0.04] hover:text-gray-900
                     disabled:opacity-40 disabled:cursor-not-allowed transition-colors
                     dark:text-gray-400 dark:hover:bg-white/[0.06] dark:hover:text-gray-100"
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
                'w-8 h-8 text-sm rounded-lg font-medium transition-all duration-150',
                p === page
                  ? 'bg-gradient-to-b from-brand-500 to-brand-600 text-white shadow-glow'
                  : 'text-gray-600 hover:bg-black/[0.05] dark:text-gray-300 dark:hover:bg-white/[0.07]'
              )}
            >
              {p}
            </button>
          )
        })}
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="p-1.5 rounded-lg text-gray-500 hover:bg-black/[0.04] hover:text-gray-900
                     disabled:opacity-40 disabled:cursor-not-allowed transition-colors
                     dark:text-gray-400 dark:hover:bg-white/[0.06] dark:hover:text-gray-100"
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-space-950/40 backdrop-blur-sm animate-[fade-up_.2s_ease-out]"
        onClick={onClose}
      />
      <div className="relative w-full max-w-md rounded-2xl p-6 enter-pop
                       bg-white/90 backdrop-blur-2xl border border-black/[0.06] shadow-glass
                       dark:bg-space-900/90 dark:border-white/10 dark:shadow-glass-dark">
        <div className="flex items-start justify-between mb-4">
          <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
          <button
            onClick={onClose}
            className="p-1 -m-1 rounded-lg text-gray-400 hover:text-gray-700 hover:bg-black/[0.04]
                       transition-colors dark:hover:text-gray-200 dark:hover:bg-white/[0.06]"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}
