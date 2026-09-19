import { errorMessage } from '../../api/client'
import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { filesApi } from '../../api/endpoints'
import {
  FileStatusBadge, PageSpinner, ErrorState, EmptyState, Pagination, Modal, Spinner
} from '../../components/ui'
import { useAuth } from '../../context/AuthContext'
import { ArrowRight, Pencil } from 'lucide-react'
import { formatBytes, formatDate } from '../../utils/format'
import type { FITSMetadata } from '../../types'

type Tab = 'metadata' | 'headers' | 'history'

export default function FileDetailPage() {
  const { id } = useParams<{ id: string }>()
  const fileId = Number(id)
  const { isEditor } = useAuth()
  const qc = useQueryClient()

  const [tab, setTab]             = useState<Tab>('metadata')
  const [headerPage, setHeaderPage] = useState(1)
  const [headerSearch, setHeaderSearch] = useState('')
  const [editField, setEditField] = useState<{ name: string; current: string } | null>(null)

  const fileQ    = useQuery({ queryKey: ['file', fileId], queryFn: () => filesApi.getById(fileId) })
  const metaQ    = useQuery({ queryKey: ['metadata', fileId], queryFn: () => filesApi.metadata(fileId) })
  const headersQ = useQuery({
    queryKey: ['headers', fileId, headerPage, headerSearch],
    queryFn: () => filesApi.headers(fileId, { page: headerPage, page_size: 50, search: headerSearch }),
    enabled: tab === 'headers',
  })
  const historyQ = useQuery({
    queryKey: ['meta-history', fileId],
    queryFn: () => filesApi.metadataHistory(fileId),
    enabled: tab === 'history',
  })

  const editMutation = useMutation({
    mutationFn: ({ field, value, reason }: { field: string; value: string; reason: string }) =>
      filesApi.editMetadata(fileId, field, value, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['metadata', fileId] })
      qc.invalidateQueries({ queryKey: ['meta-history', fileId] })
      setEditField(null)
    },
  })

  if (fileQ.isLoading) return <PageSpinner />
  if (fileQ.isError || !fileQ.data) return <ErrorState message="فایل یافت نشد" />

  const file = fileQ.data

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to="/files" className="text-text-muted hover:text-text">
          <ArrowRight className="w-5 h-5" />
        </Link>
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold text-text truncate">{file.file_name}</h1>
          <div className="flex items-center gap-3 mt-1">
            <FileStatusBadge status={file.status} />
            <span className="text-sm text-text-muted">{formatBytes(file.file_size)}</span>
            <span className="text-sm text-text-muted">{file.hdu_count} HDU</span>
            <span className="text-xs text-text-muted font-mono">{file.checksum.slice(0, 20)}…</span>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-border">
        {(['metadata', 'headers', 'history'] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              tab === t
                ? 'border-accent-600 text-accent-700'
                : 'border-transparent text-text-secondary hover:text-text'
            }`}
          >
            {t === 'metadata' ? 'متادیتا' : t === 'headers' ? 'هدرها' : 'تاریخچه ویرایش'}
          </button>
        ))}
      </div>

      {/* Metadata tab */}
      {tab === 'metadata' && (
        <div className="card overflow-hidden">
          {metaQ.isLoading ? <PageSpinner /> : !metaQ.data ? (
            <EmptyState title="متادیتایی یافت نشد" />
          ) : (
            <MetadataTable
              meta={metaQ.data}
              canEdit={isEditor}
              onEdit={(name, current) => setEditField({ name, current })}
            />
          )}
        </div>
      )}

      {/* Headers tab */}
      {tab === 'headers' && (
        <div className="card overflow-hidden">
          <div className="p-4 border-b border-border">
            <input
              className="input max-w-sm"
              placeholder="جستجو در keyword یا value..."
              value={headerSearch}
              onChange={(e) => { setHeaderSearch(e.target.value); setHeaderPage(1) }}
            />
          </div>
          {headersQ.isLoading ? <PageSpinner /> : !headersQ.data?.data?.length ? (
            <EmptyState title="هدری یافت نشد" />
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr>
                      <th className="table-th w-8">HDU</th>
                      <th className="table-th w-32">Keyword</th>
                      <th className="table-th">Value</th>
                      <th className="table-th">Comment</th>
                      <th className="table-th w-20">Type</th>
                    </tr>
                  </thead>
                  <tbody>
                    {headersQ.data.data.map((h) => (
                      <tr key={h.id} className="hover:bg-surface2">
                        <td className="table-td text-center text-text-muted">{h.hdu_index}</td>
                        <td className="table-td font-mono text-xs font-semibold text-accent-700">{h.keyword}</td>
                        <td className="table-td font-mono text-xs text-text max-w-xs truncate">{h.value}</td>
                        <td className="table-td text-xs text-text-muted max-w-xs truncate">{h.comment}</td>
                        <td className="table-td text-xs text-text-muted">{h.value_type}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pagination
                page={headerPage}
                totalPages={headersQ.data.total_pages}
                total={headersQ.data.total}
                pageSize={50}
                onPageChange={setHeaderPage}
              />
            </>
          )}
        </div>
      )}

      {/* History tab */}
      {tab === 'history' && (
        <div className="card overflow-hidden">
          {historyQ.isLoading ? <PageSpinner /> : !historyQ.data?.length ? (
            <EmptyState title="هیچ ویرایشی ثبت نشده" description="هنگامی که فیلدی ویرایش شود اینجا نمایش داده می‌شود" />
          ) : (
            <table className="w-full">
              <thead>
                <tr>
                  <th className="table-th">فیلد</th>
                  <th className="table-th">مقدار قبلی</th>
                  <th className="table-th">مقدار جدید</th>
                  <th className="table-th">دلیل</th>
                  <th className="table-th">تاریخ</th>
                </tr>
              </thead>
              <tbody>
                {historyQ.data.map((o) => (
                  <tr key={o.id} className="hover:bg-surface2">
                    <td className="table-td font-mono text-xs font-semibold text-accent-700">{o.field_name}</td>
                    <td className="table-td text-xs text-text-muted">{o.original_value ?? '—'}</td>
                    <td className="table-td text-xs font-medium text-text">{o.new_value}</td>
                    <td className="table-td text-xs text-text-muted">{o.reason ?? '—'}</td>
                    <td className="table-td text-xs text-text-muted">{formatDate(o.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Edit modal */}
      {editField && (
        <EditMetadataModal
          fieldName={editField.name}
          currentValue={editField.current}
          loading={editMutation.isPending}
          error={editMutation.error ? errorMessage(editMutation.error) : undefined}
          onClose={() => setEditField(null)}
          onSave={(value, reason) => editMutation.mutate({ field: editField.name, value, reason })}
        />
      )}
    </div>
  )
}

// ── Metadata table ────────────────────────────────────────────────────────────

const metadataGroups: Array<{ title: string; fields: Array<{ key: keyof FITSMetadata; label: string; unit?: string }> }> = [
  {
    title: 'هندسه تصویر', fields: [
      { key: 'naxis', label: 'NAXIS' }, { key: 'naxis1', label: 'NAXIS1', unit: 'px' },
      { key: 'naxis2', label: 'NAXIS2', unit: 'px' }, { key: 'naxis3', label: 'NAXIS3' },
      { key: 'bitpix', label: 'BITPIX' },
    ],
  },
  {
    title: 'دوربین و ابزار', fields: [
      { key: 'instrume', label: 'Instrument' }, { key: 'telescop', label: 'Telescope' },
      { key: 'observer', label: 'Observer' }, { key: 'object', label: 'Object' },
      { key: 'origin', label: 'Origin' }, { key: 'software', label: 'Software' },
    ],
  },
  {
    title: 'نوردهی و زمان', fields: [
      { key: 'exptime', label: 'Exposure Time', unit: 's' }, { key: 'gain', label: 'Gain' },
      { key: 'rdnoise', label: 'Read Noise' }, { key: 'date_obs', label: 'Date OBS' },
      { key: 'time_obs', label: 'Time OBS' }, { key: 'mjd_obs', label: 'MJD OBS' },
    ],
  },
  {
    title: 'فیلتر', fields: [
      { key: 'filter', label: 'Filter' }, { key: 'filter_id', label: 'Filter ID' },
    ],
  },
  {
    title: 'دما', fields: [
      { key: 'set_temp', label: 'Set Temp', unit: '°C' }, { key: 'ccd_temp', label: 'CCD Temp', unit: '°C' },
      { key: 'amb_temp', label: 'Ambient Temp', unit: '°C' },
    ],
  },
  {
    title: 'موقعیت هدف', fields: [
      { key: 'ra', label: 'RA', unit: '°' }, { key: 'dec', label: 'DEC', unit: '°' },
      { key: 'airmass', label: 'Airmass' },
    ],
  },
  {
    title: 'موقعیت رصدخانه', fields: [
      { key: 'site_lat', label: 'Latitude', unit: '°' }, { key: 'site_lon', label: 'Longitude', unit: '°' },
      { key: 'site_elev', label: 'Elevation', unit: 'm' },
    ],
  },
]

const editableFields = new Set([
  'object', 'observer', 'telescop', 'instrume', 'filter', 'filter_id',
  'origin', 'software', 'equip_id', 'ra', 'dec', 'airmass',
  'site_lat', 'site_lon', 'site_elev', 'exptime', 'gain', 'rdnoise',
  'set_temp', 'ccd_temp', 'amb_temp', 'date_obs', 'time_obs',
])

function MetadataTable({ meta, canEdit, onEdit }: {
  meta: FITSMetadata
  canEdit: boolean
  onEdit: (name: string, current: string) => void
}) {
  return (
    <div className="divide-y divide-border">
      {metadataGroups.map((group) => (
        <div key={group.title} className="p-4">
          <h3 className="text-xs font-semibold text-text-secondary uppercase tracking-wider mb-3">{group.title}</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-x-6 gap-y-3">
            {group.fields.map(({ key, label, unit }) => {
              const rawVal = meta[key]
              const val = rawVal !== null && rawVal !== undefined ? String(rawVal) : null
              const isEditable = canEdit && editableFields.has(key as string)
              return (
                <div key={key} className="flex items-start justify-between gap-2 group">
                  <div className="min-w-0 flex-1">
                    <div className="text-xs text-text-muted">{label}{unit ? ` (${unit})` : ''}</div>
                    <div className={`text-sm font-medium mt-0.5 ${val ? 'text-text' : 'text-text-muted'}`}>
                      {val ?? '—'}
                    </div>
                  </div>
                  {isEditable && (
                    <button
                      onClick={() => onEdit(key as string, val ?? '')}
                      className="opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-surface2 shrink-0 mt-0.5"
                      title="ویرایش"
                    >
                      <Pencil className="w-3 h-3 text-text-muted" />
                    </button>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}

// ── Edit modal ────────────────────────────────────────────────────────────────

function EditMetadataModal({ fieldName, currentValue, loading, error, onClose, onSave }: {
  fieldName: string
  currentValue: string
  loading: boolean
  error?: string
  onClose: () => void
  onSave: (value: string, reason: string) => void
}) {
  const [value, setValue]   = useState(currentValue)
  const [reason, setReason] = useState('')

  return (
    <Modal title={`ویرایش ${fieldName}`} onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="label">مقدار جدید</label>
          <input className="input" value={value} onChange={(e) => setValue(e.target.value)} autoFocus />
        </div>
        <div>
          <label className="label">دلیل تغییر (اختیاری)</label>
          <input className="input" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="مثال: اصلاح نام شیء" />
        </div>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-3 pt-2">
          <button onClick={() => onSave(value, reason)} disabled={loading || !value} className="btn-primary flex-1">
            {loading ? <><Spinner className="w-4 h-4 text-white" />در حال ذخیره...</> : 'ذخیره'}
          </button>
          <button onClick={onClose} className="btn-secondary">انصراف</button>
        </div>
      </div>
    </Modal>
  )
}
