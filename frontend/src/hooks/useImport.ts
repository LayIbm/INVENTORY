import { useState, useCallback, useRef } from 'react'
import client from '../api/client'

export interface ImportIssue {
  field: string
  original_value: string
  normalized_value?: string
  severity: 'ERROR' | 'WARNING' | 'INFO'
  message: string
}

export interface ImportRow {
  file: string
  sheet: string
  row: number
  entity: 'laptop' | 'equipment' | 'unknown'
  proposed_action: 'INSERT' | 'UPDATE' | 'SKIP' | 'ERROR' | 'UNKNOWN'
  executed_action?: string
  issues?: ImportIssue[]
  fields?: Record<string, string>
}

export interface SheetInfo {
  file: string
  sheet: string
  row_count: number
  columns: string[]
}

export interface ImportSummary {
  files: number
  sheets: number
  rows_read: number
  rows_ignored: number
  valid: number
  insertable: number
  updatable: number
  duplicates: number
  with_warnings: number
  with_errors: number
  inserted: number
  updated: number
  skipped: number
  errored: number
  duration_ms: number
  laptop_rows: number
  equipment_rows: number
  unknown_rows: number
}

export interface PreviewResult {
  summary: ImportSummary
  sheets: SheetInfo[]
  sample: ImportRow[]
  errors: ImportRow[]
  warnings: ImportRow[]
  can_apply: boolean
}

export interface ApplyResult {
  summary: ImportSummary
  errors_file?: string
  finished_at: string
}

interface LegacyLinkImportSummary {
  laptops_inserted: number
  laptops_updated: number
  equipment_inserted: number
  equipment_updated: number
  links_inserted: number
  links_updated: number
  errors: string[]
}

// ── Low-level API functions (used directly by InventoryImportModal) ────────────

export async function previewImport(
  file: File,
  view?: string,
  signal?: AbortSignal,
): Promise<PreviewResult> {
  const form = new FormData()
  form.append('file', file)
  const params = new URLSearchParams()
  if (view) params.set('view', view)
  const url = `/import/inventory/preview${params.toString() ? '?' + params.toString() : ''}`
  const res = await client.post<PreviewResult>(url, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    signal,
  })
  return res.data
}

export async function applyImport(
  file: File,
  view?: string,
  options?: { upsert?: boolean; atomic?: boolean },
  signal?: AbortSignal,
): Promise<ApplyResult> {
  const form = new FormData()
  form.append('file', file)
  const params = new URLSearchParams()
  if (view) params.set('view', view)
  if (options?.upsert) params.set('upsert', 'true')
  if (options?.atomic) params.set('atomic', 'true')
  const url = `/import/inventory/apply${params.toString() ? '?' + params.toString() : ''}`
  const res = await client.post<ApplyResult>(url, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    signal,
  })
  return res.data
}

export async function applyLegacyLinksImport(
  file: File,
  signal?: AbortSignal,
): Promise<ApplyResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await client.post<LegacyLinkImportSummary>('/import/links', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    signal,
  })
  const inserted = res.data.links_inserted
  const updated = res.data.links_updated
  const errored = res.data.errors.length
  return {
    finished_at: new Date().toISOString(),
    summary: {
      files: 1,
      sheets: 1,
      rows_read: inserted + updated + errored,
      rows_ignored: 0,
      valid: inserted + updated,
      insertable: inserted,
      updatable: updated,
      duplicates: 0,
      with_warnings: 0,
      with_errors: errored,
      inserted,
      updated,
      skipped: 0,
      errored,
      duration_ms: 0,
      laptop_rows: 0,
      equipment_rows: 0,
      unknown_rows: 0,
    },
  }
}

// ── React hook encapsulating import state ──────────────────────────────────────

export interface UseImportState {
  loading: boolean
  error: string
  preview: PreviewResult | null
  result: ApplyResult | null
  runPreview: (file: File) => Promise<PreviewResult | null>
  runApply: (file: File, options?: { upsert?: boolean }) => Promise<ApplyResult | null>
  cancel: () => void
  reset: () => void
}

export function useImport(): UseImportState {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [preview, setPreview] = useState<PreviewResult | null>(null)
  const [result, setResult] = useState<ApplyResult | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const cancel = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setLoading(false)
  }, [])

  const reset = useCallback(() => {
    cancel()
    setError('')
    setPreview(null)
    setResult(null)
  }, [cancel])

  const runPreview = useCallback(async (file: File): Promise<PreviewResult | null> => {
    cancel()
    const ctrl = new AbortController()
    abortRef.current = ctrl
    setLoading(true)
    setError('')
    try {
      const res = await previewImport(file, undefined, ctrl.signal)
      setPreview(res)
      return res
    } catch (err: unknown) {
      if ((err as { name?: string })?.name === 'CanceledError') return null
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        'Error al analizar el archivo'
      setError(msg)
      return null
    } finally {
      setLoading(false)
    }
  }, [cancel])

  const runApply = useCallback(
    async (file: File, options?: { upsert?: boolean }): Promise<ApplyResult | null> => {
      cancel()
      const ctrl = new AbortController()
      abortRef.current = ctrl
      setLoading(true)
      setError('')
      try {
        const res = await applyImport(file, undefined, options, ctrl.signal)
        setResult(res)
        return res
      } catch (err: unknown) {
        if ((err as { name?: string })?.name === 'CanceledError') return null
        const msg =
          (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
          'Error durante la importación'
        setError(msg)
        return null
      } finally {
        setLoading(false)
      }
    },
    [cancel],
  )

  return { loading, error, preview, result, runPreview, runApply, cancel, reset }
}
