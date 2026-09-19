import React from 'react'
import { Loader2, AlertCircle, Inbox, ChevronLeft, ChevronRight } from 'lucide-react'
import type { FileStatus, JobStatus } from '../../types'
import clsx from 'clsx'

// ── Spinner ───────────────────────────────────────────────────────────────────

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={clsx('animate-spin', className ?? 'w-5 h-5 text-accent-600')} />
}

export function PageSpinner() {
  return (
    <div className="flex items-center justify-center h-64">
      <Spinner className="w-6 h-6 text-accent-600" />
    </div>
  )
}

// ── Status pills — subtle dot + tinted text, not loud filled badges ──────────

const fileStatusMap: Record<FileStatus, { label: string; cls: string }> = {
  done:       { label: 'تکمیل‌شده', cls: 'pill-success' },
  processing: { label: 'در حال پردازش', cls: 'pill-info' },
  pending:    { label: 'در صف', cls: 'pill-warning' },
  error:      { label: 'خطا', cls: 'pill-danger' },
  skipped:    { label: 'رد شده (تکراری)', cls: 'pill-neutral' },
}

const jobStatusMap: Record<JobStatus, { label: string; cls: string }> = {
  completed:        { label: 'تکمیل‌شده', cls: 'pill-success' },
  running:          { label: 'در حال اجرا', cls: 'pill-info' },
  partially_failed: { label: 'ناقص', cls: 'pill-warning' },
  failed:           { label: 'ناموفق', cls: 'pill-danger' },
  cancelled:        { label: 'لغوشده', cls: 'pill-neutral' },
}

const dotColor: Record<string, string> = {
  'pill-success': 'bg-success',
  'pill-info':    'bg-info',
  'pill-warning': 'bg-warning',
  'pill-danger':  'bg-danger',
  'pill-neutral': 'bg-text-muted',
}

function StatusPill({ label, cls }: { label: string; cls: string }) {
  return (
    <span className={cls}>
      <span className={clsx('pill-dot', dotColor[cls])} />
      {label}
    </span>
  )
}

export function FileStatusBadge({ status }: { status: FileStatus }) {
  const cfg = fileStatusMap[status] ?? { label: status, cls: 'pill-neutral' }
  return <StatusPill {...cfg} />
}

export function JobStatusBadge({ status }: { status: JobStatus }) {
  const cfg = jobStatusMap[status] ?? { label: status, cls: 'pill-neutral' }
  return <StatusPill {...cfg} />
}

const roleLabel: Record<string, string> = { admin: 'مدیر', editor: 'ویرایشگر', viewer: 'مشاهده‌گر' }

export function RoleBadge({ role }: { role: string }) {
  const cls = role === 'admin' ? 'pill-danger' : role === 'editor' ? 'pill-info' : 'pill-neutral'
  return <StatusPill label={roleLabel[role] ?? role} cls={cls} />
}

// ── Page structure primitives ────────────────────────────────────────────────

export function PageHeader({
  title, description, actions,
}: { title: string; description?: string; actions?: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 mb-5">
      <div>
        <h1 className="text-lg font-semibold text-text">{title}</h1>
        {description && <p className="text-sm text-text-secondary mt-0.5">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </div>
  )
}

export function Section({
  title, description, actions, children, className,
}: { title?: string; description?: string; actions?: React.ReactNode; children: React.ReactNode; className?: string }) {
  return (
    <section className={clsx('panel', className)}>
      {(title || actions) && (
        <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border">
          <div>
            {title && <h2 className="text-sm font-semibold text-text">{title}</h2>}
            {description && <p className="text-xs text-text-secondary mt-0.5">{description}</p>}
          </div>
          {actions}
        </div>
      )}
      {children}
    </section>
  )
}

export function Stat({ label, value, hint }: { label: string; value: React.ReactNode; hint?: string }) {
  return (
    <div className="panel px-4 py-3">
      <p className="text-xs text-text-secondary">{label}</p>
      <p className="text-xl font-semibold text-text mt-1 font-mono">{value}</p>
      {hint && <p className="text-xs text-text-muted mt-0.5">{hint}</p>}
    </div>
  )
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={clsx('animate-pulse rounded bg-surface2', className ?? 'h-4 w-full')} />
}

export function TableSkeleton({ rows = 6, cols = 5 }: { rows?: number; cols?: number }) {
  return (
    <div className="divide-y divide-border">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex items-center gap-4 px-3 py-3">
          {Array.from({ length: cols }).map((__, c) => (
            <Skeleton key={c} className={clsx('h-3.5', c === 0 ? 'w-1/4' : 'flex-1')} />
          ))}
        </div>
      ))}
    </div>
  )
}

// ── Empty / Error states ──────────────────────────────────────────────────────

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center px-4">
      <Inbox className="w-9 h-9 text-text-muted mb-3" strokeWidth={1.5} />
      <h3 className="text-sm font-semibold text-text">{title}</h3>
      {description && <p className="text-sm text-text-secondary mt-1 max-w-xs">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center px-4">
      <AlertCircle className="w-8 h-8 text-danger mb-3" strokeWidth={1.5} />
      <h3 className="text-sm font-semibold text-text">خطایی رخ داده</h3>
      <p className="text-sm text-text-secondary mt-1 max-w-sm">{message}</p>
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
    <div className="flex items-center justify-between px-3 py-2.5 border-t border-border bg-surface">
      <p className="text-xs text-text-secondary font-mono">
        {from}–{to} از {total}
      </p>
      <div className="flex items-center gap-0.5">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="p-1.5 rounded hover:bg-surface2 disabled:opacity-30 disabled:cursor-not-allowed text-text-secondary"
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
                'w-7 h-7 text-xs rounded font-medium font-mono',
                p === page ? 'bg-accent-600 text-white' : 'hover:bg-surface2 text-text-secondary'
              )}
            >
              {p}
            </button>
          )
        })}
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="p-1.5 rounded hover:bg-surface2 disabled:opacity-30 disabled:cursor-not-allowed text-text-secondary"
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
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative bg-surface rounded-lg shadow-popover border border-border w-full max-w-md mx-4 p-5">
        <h2 className="text-sm font-semibold text-text mb-4">{title}</h2>
        {children}
      </div>
    </div>
  )
}
