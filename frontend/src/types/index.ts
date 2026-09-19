// ── Auth ──────────────────────────────────────────────────────────────────────

export type Role = 'admin' | 'editor' | 'viewer'

export interface SafeUser {
  id: number
  email: string
  full_name: string
  role: Role
  is_active: boolean
  last_login_at?: string
  created_at: string
  permissions?: string[]
}

export interface TokenPair {
  access_token: string
  expires_at: string
  user: SafeUser
}

// ── FITS Files ────────────────────────────────────────────────────────────────

export type FileStatus = 'pending' | 'processing' | 'done' | 'error' | 'skipped'

export interface FITSFile {
  id: number
  file_path: string
  file_name: string
  file_size: number
  checksum: string
  hdu_count: number
  status: FileStatus
  error_message?: string
  processed_at?: string
  created_at: string
  updated_at: string
}

// ── Headers ───────────────────────────────────────────────────────────────────

export interface FITSHeader {
  id: number
  file_id: number
  hdu_index: number
  hdu_name: string
  keyword: string
  value: string
  comment: string
  value_type: string
  created_at: string
}

// ── Metadata ──────────────────────────────────────────────────────────────────

export interface FITSMetadata {
  id: number
  file_id: number
  naxis?: number
  naxis1?: number
  naxis2?: number
  naxis3?: number
  bitpix?: number
  exptime?: number
  date_obs?: string
  time_obs?: string
  mjd_obs?: number
  mjd_end?: number
  gain?: number
  rdnoise?: number
  instrume?: string
  telescop?: string
  observer?: string
  object?: string
  origin?: string
  software?: string
  equip_id?: string
  set_temp?: number
  ccd_temp?: number
  amb_temp?: number
  filter?: string
  filter_id?: string
  site_lat?: number
  site_lon?: number
  site_elev?: number
  ra?: number
  dec?: number
  airmass?: number
  crval1?: number
  crval2?: number
  crpix1?: number
  crpix2?: number
  cd1_1?: number
  cd1_2?: number
  cd2_1?: number
  cd2_2?: number
  ctype1?: string
  ctype2?: string
  bscale?: number
  bzero?: number
  created_at: string
  updated_at: string
}

export interface MetadataOverride {
  id: number
  file_id: number
  field_name: string
  original_value?: string
  new_value: string
  reason?: string
  edited_by?: number
  created_at: string
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

export type JobStatus = 'running' | 'completed' | 'partially_failed' | 'failed' | 'cancelled'

export interface ProcessingJob {
  id: number
  scan_dir: string
  status: JobStatus
  total_files: number
  done_files: number
  error_files: number
  started_at: string
  finished_at?: string
  error_message?: string
}

export interface ProcessingError {
  id: number
  job_id: number
  file_id?: number
  file_path: string
  stage: string
  message: string
  created_at: string
}

// ── API shapes ────────────────────────────────────────────────────────────────

export interface ApiResponse<T> {
  data: T
}

export interface PagedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface ApiError {
  error: string
  code: number
}
