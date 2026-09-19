import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { filesApi } from '../../api/endpoints'
import { FileStatusBadge, PageSpinner, ErrorState, EmptyState, Pagination } from '../../components/ui'
import { useDebounce } from '../../hooks/useDebounce'
import { Search, SlidersHorizontal, ExternalLink } from 'lucide-react'
import { formatBytes, formatDate } from '../../utils/format'
import type { FileStatus } from '../../types'

export default function FilesPage() {
  const [searchParams] = useSearchParams()
  const [page, setPage]     = useState(1)
  const [search, setSearch] = useState(searchParams.get('search') ?? '')
  const [status, setStatus] = useState<string>(searchParams.get('status') ?? '')
  const [sort, setSort]     = useState('created_at')
  const [order, setOrder]   = useState('desc')

  const debouncedSearch = useDebounce(search, 400)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['files', page, debouncedSearch, status, sort, order],
    queryFn: () => filesApi.list({
      page,
      page_size: 20,
      search: debouncedSearch,
      status,
      sort,
      order,
    }),
  })

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-text">فایل‌های FITS</h1>
          <p className="text-sm text-text-secondary mt-0.5">
            {isLoading ? 'در حال بارگذاری...' : `${data?.total ?? 0} فایل`}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="card p-4">
        <div className="flex flex-wrap gap-3 items-center">
          <div className="relative flex-1 min-w-52">
            <Search className="absolute start-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
            <input
              type="text"
              placeholder="جستجو در نام فایل..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1) }}
              className="input ps-9"
            />
          </div>

          <select
            value={status}
            onChange={(e) => { setStatus(e.target.value as FileStatus | ''); setPage(1) }}
            className="input w-44"
          >
            <option value="">همه وضعیت‌ها</option>
            <option value="done">Done</option>
            <option value="processing">Processing</option>
            <option value="pending">Pending</option>
            <option value="error">Error</option>
            <option value="skipped">Skipped</option>
          </select>

          <div className="flex items-center gap-2">
            <SlidersHorizontal className="w-4 h-4 text-text-muted shrink-0" />
            <select value={sort} onChange={(e) => setSort(e.target.value)} className="input w-40">
              <option value="created_at">تاریخ ثبت</option>
              <option value="file_name">نام فایل</option>
              <option value="file_size">حجم</option>
              <option value="processed_at">تاریخ پردازش</option>
            </select>
            <select value={order} onChange={(e) => setOrder(e.target.value)} className="input w-28">
              <option value="desc">نزولی</option>
              <option value="asc">صعودی</option>
            </select>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="card overflow-hidden">
        {isLoading ? (
          <PageSpinner />
        ) : isError ? (
          <ErrorState message="بارگذاری فایل‌ها با خطا مواجه شد" onRetry={refetch} />
        ) : !data?.data?.length ? (
          <EmptyState
            title="هیچ فایلی یافت نشد"
            description={search || status ? 'فیلترها را تغییر دهید' : 'ابتدا یک اسکن را اجرا کنید'}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr>
                    <th className="table-th">نام فایل</th>
                    <th className="table-th">وضعیت</th>
                    <th className="table-th">حجم</th>
                    <th className="table-th">HDUs</th>
                    <th className="table-th">پردازش‌شده</th>
                    <th className="table-th">ثبت‌شده</th>
                    <th className="table-th w-20"></th>
                  </tr>
                </thead>
                <tbody>
                  {data.data.map((file) => (
                    <tr key={file.id} className="hover:bg-surface2 transition-colors">
                      <td className="table-td">
                        <div className="font-medium text-text truncate max-w-xs" title={file.file_path}>
                          {file.file_name}
                        </div>
                        <div className="text-xs text-text-muted font-mono mt-0.5 truncate max-w-xs">
                          {file.checksum.slice(0, 16)}…
                        </div>
                      </td>
                      <td className="table-td"><FileStatusBadge status={file.status} /></td>
                      <td className="table-td text-text-secondary">{formatBytes(file.file_size)}</td>
                      <td className="table-td text-text-secondary">{file.hdu_count}</td>
                      <td className="table-td text-text-secondary text-xs">{formatDate(file.processed_at)}</td>
                      <td className="table-td text-text-secondary text-xs">{formatDate(file.created_at)}</td>
                      <td className="table-td">
                        <Link
                          to={`/files/${file.id}`}
                          className="text-accent-600 hover:text-accent-700 inline-flex items-center gap-1 text-sm font-medium"
                        >
                          جزئیات <ExternalLink className="w-3 h-3" />
                        </Link>
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
              pageSize={20}
              onPageChange={setPage}
            />
          </>
        )}
      </div>
    </div>
  )
}
