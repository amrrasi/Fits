import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { CheckCircle2, XCircle, AlertTriangle, X } from 'lucide-react'
import clsx from 'clsx'

// ── Types ─────────────────────────────────────────────────────────────────────

type ToastType = 'success' | 'error' | 'warning'

interface Toast {
  id: string
  type: ToastType
  message: string
}

interface ToastContextValue {
  success: (message: string) => void
  error:   (message: string) => void
  warning: (message: string) => void
}

// ── Context ───────────────────────────────────────────────────────────────────

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const add = useCallback((type: ToastType, message: string) => {
    const id = Math.random().toString(36).slice(2)
    setToasts((prev) => [...prev, { id, type, message }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 4000)
  }, [])

  const remove = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{
      success: (m) => add('success', m),
      error:   (m) => add('error', m),
      warning: (m) => add('warning', m),
    }}>
      {children}

      {/* Toast container */}
      <div className="fixed bottom-5 end-5 z-[60] flex flex-col gap-2 pointer-events-none" role="status" aria-live="polite">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={clsx(
              'flex items-start gap-3 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium',
              'pointer-events-auto max-w-sm enter',
              toast.type === 'success' && 'bg-success-bg border-success/40 text-success',
              toast.type === 'error'   && 'bg-danger-bg   border-danger/40   text-danger',
              toast.type === 'warning' && 'bg-warning-bg border-warning/40 text-warning',
            )}
          >
            {toast.type === 'success' && <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5 text-success" />}
            {toast.type === 'error'   && <XCircle      className="w-4 h-4 shrink-0 mt-0.5 text-danger" />}
            {toast.type === 'warning' && <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5 text-warning" />}
            <span className="flex-1">{toast.message}</span>
            <button onClick={() => remove(toast.id)} className="shrink-0 opacity-60 hover:opacity-100 transition-opacity">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}
