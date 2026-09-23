import { useEffect, useRef, useState } from 'react'
import {
  PreviewResult,
  ApplyResult,
  ImportRow,
  previewImport,
  applyImport,
  applyLegacyLinksImport,
} from '../hooks/useImport'
import client from '../api/client'
import { downloadBlob } from '../utils/downloadBlob'

// ── Types ──────────────────────────────────────────────────────────────────────

type Step =
  | 'select'      // initial — no file chosen
  | 'selected'    // file chosen, not yet analysed
  | 'analysing'   // calling preview
  | 'preview'     // preview result available
  | 'importing'   // calling apply
  | 'result'      // apply done
  | 'fatal'       // unrecoverable error

type ViewKind = 'computer' | 'peripherals' | 'network' | 'links' | 'epd' | 'bios'

const VIEW_LABELS: Record<ViewKind, string> = {
  computer:    'Computer Inventory',
  peripherals: 'Peripherals',
  network:     'Network Infrastructure',
  links:       'Links',
  epd:         'EPD by Employee',
  bios:        'BIOS Control',
}

interface Props {
  onClose: () => void
  onImportDone?: () => void
  view?: ViewKind
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const ALLOWED_EXTS = ['xlsx', 'csv']

function detectFileType(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() ?? ''
  if (ext === 'xlsx') return 'Excel (.xlsx)'
  if (ext === 'csv')  return 'CSV (.csv)'
  return ext ? `.${ext}` : 'unknown'
}

// ── Main component ─────────────────────────────────────────────────────────────

export default function InventoryImportModal({ onClose, onImportDone, view }: Props) {
  const [step, setStep]       = useState<Step>('select')
  const [file, setFile]       = useState<File | null>(null)
  const [preview, setPreview] = useState<PreviewResult | null>(null)
  const [result, setResult]   = useState<ApplyResult | null>(null)
  const [error, setError]       = useState<string>('')
  const [dragging, setDragging] = useState(false)
  const [upsert, setUpsert]     = useState(false)
  const [downloading, setDownloading] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const fileInputRef = useRef<HTMLInputElement>(null)

  // ── File validation & selection ────────────────────────────────────────────

  function acceptFile(f: File) {
    const ext = f.name.split('.').pop()?.toLowerCase() ?? ''
    if (!ALLOWED_EXTS.includes(ext)) {
      setError(
        ext === 'xls'
          ? 'The .xls format is not supported. Please convert the file to .xlsx or .csv first.'
          : `File type not allowed: .${ext || 'unknown'}. Only .xlsx and .csv files are accepted.`,
      )
      setStep('fatal')
      return
    }
    setFile(f)
    setError('')
    setStep('selected')
  }

  function handleFileInput(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    if (f) acceptFile(f)
    // Reset input so the same file can be re-selected after removal
    e.target.value = ''
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragging(false)
    const f = e.dataTransfer.files?.[0]
    if (f) acceptFile(f)
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    setDragging(true)
  }

  function handleDragLeave(e: React.DragEvent) {
    // Only clear dragging when leaving the drop zone entirely
    if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
      setDragging(false)
    }
  }

  function removeFile() {
    abortRef.current?.abort()
    abortRef.current = null
    setFile(null)
    setPreview(null)
    setError('')
    setStep('select')
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  // ── Analyse (preview / dry-run) ───────────────────────────────────────────

  async function handleAnalyse() {
    if (!file) return
    if (view === 'links') {
      await handleImport()
      return
    }
    abortRef.current?.abort()
    const ctrl = new AbortController()
    abortRef.current = ctrl
    setStep('analysing')
    setError('')
    try {
      const res = await previewImport(file, view, ctrl.signal)
      setPreview(res)
      setStep('preview')
    } catch (err: unknown) {
      if ((err as { name?: string })?.name === 'CanceledError') return
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        'Error analysing the file'
      setError(msg)
      setStep('fatal')
    }
  }

  // ── Apply (import) ────────────────────────────────────────────────────────

  async function handleImport() {
    if (!file || step === 'importing') return
    abortRef.current?.abort()
    const ctrl = new AbortController()
    abortRef.current = ctrl
    setStep('importing')
    setError('')
    try {
      const res = view === 'links'
        ? await applyLegacyLinksImport(file, ctrl.signal)
        : await applyImport(file, view, { upsert }, ctrl.signal)
      setResult(res)
      setStep('result')
      // Notify parent to refresh data — but do NOT close the modal yet.
      // The user will close from the ResultPane ("Close" button).
      onImportDone?.()
    } catch (err: unknown) {
      if ((err as { name?: string })?.name === 'CanceledError') return
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        'Error during import'
      setError(msg)
      // Keep modal open and preserve preview on apply failure
      setStep('preview')
    }
  }

  function handleCancel() {
    abortRef.current?.abort()
    abortRef.current = null
    if (step === 'analysing') setStep('selected')
    else if (step === 'importing' && preview) setStep('preview')
    else setStep(file ? 'selected' : 'select')
  }

  function handleReset() {
    abortRef.current?.abort()
    abortRef.current = null
    setFile(null)
    setPreview(null)
    setResult(null)
    setError('')
    setStep('select')
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  // ── Downloads ─────────────────────────────────────────────────────────────

  async function handleDownloadTemplate() {
    setDownloading(true)
    try {
      const params = view ? `?view=${view}` : ''
      const label  = view ? VIEW_LABELS[view].toLowerCase().replace(/ /g, '_') : 'inventory'
      const res = await client.get(`/export/template${params}`, { responseType: 'blob' })
      downloadBlob(res.data as Blob, `${label}_template.xlsx`)
    } catch {
      setError('Could not download template.')
    } finally {
      setDownloading(false)
    }
  }

  async function handleDownloadExport() {
    setDownloading(true)
    try {
      const params = view ? `?view=${view}` : ''
      const label  = view ? VIEW_LABELS[view].toLowerCase().replace(/ /g, '_') : 'inventory'
      const res = await client.get(`/export/excel${params}`, { responseType: 'blob' })
      downloadBlob(res.data as Blob, `${label}_export.xlsx`)
    } catch {
      setError('Could not download export.')
    } finally {
      setDownloading(false)
    }
  }

  const isBusy = step === 'analysing' || step === 'importing'

  // ── Render ────────────────────────────────────────────────────────────────

  // Escape closes the modal when not busy
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !isBusy) onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [isBusy, onClose])

  return (
    <div
      className="overlay"
      style={{ display: 'flex' }}
      onClick={(e) => !isBusy && e.target === e.currentTarget && onClose()}
    >
      <div className="modal" style={{ maxWidth: 640 }}>
        {/* ── Header ── */}
        <div className="modal-head">
          <div className="modal-head-top">
            <div>
              <h3>Import inventory from Excel</h3>
              <p className="modal-sub">
                Upload an .xlsx or .csv file to import laptops and devices into the inventory.
              </p>
            </div>
            <button
              type="button"
              className="modal-close"
              onClick={onClose}
              title="Close"
              aria-label="Close modal"
              disabled={isBusy}
            >
              ✕
            </button>
          </div>

          {/* Step progress indicator */}
          <StepIndicator step={step} />
        </div>

        {/* ── Body ── */}
        <div className="modal-body" style={{ minHeight: 220, maxHeight: '60vh' }}>

          {/* Apply error banner (shown over preview pane) */}
          {error && step === 'preview' && (
            <div style={{
              marginBottom: '1rem',
              padding: '0.75rem',
              background: 'var(--cds-support-error-bg)',
              borderLeft: '4px solid var(--cds-support-error)',
            }}>
              <p style={{ color: 'var(--cds-support-error)', fontWeight: 600, margin: 0, fontSize: '0.8rem' }}>
                Error during import
              </p>
              <p style={{ margin: '0.25rem 0 0', fontSize: '0.8rem', color: 'var(--cds-text-primary)' }}>
                {error}
              </p>
            </div>
          )}

          {/* ── select / selected ── */}
          {(step === 'select' || step === 'selected') && (
            <>
              {/* ── Download strip ── */}
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap',
                padding: '10px 14px', marginBottom: 16,
                background: 'var(--cds-layer-02)',
                borderRadius: 6, border: '1px solid var(--cds-border-subtle-01)',
              }}>
                <span style={{ fontSize: 12, color: 'var(--cds-text-secondary)', flex: 1 }}>
                  Download:
                </span>
                <button className="btn btn-ghost" style={{ fontSize: 12 }}
                  onClick={handleDownloadTemplate} disabled={downloading}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                    <polyline points="14 2 14 8 20 8"/>
                  </svg>
                  Template
                </button>
                <button className="btn btn-ghost" style={{ fontSize: 12 }}
                  onClick={handleDownloadExport} disabled={downloading}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9">
                    <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                  </svg>
                  Export current data
                </button>
              </div>

              <DropZone
                dragging={dragging}
                file={file}
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onClick={() => fileInputRef.current?.click()}
                onRemove={removeFile}
              />
              <input
                ref={fileInputRef}
                type="file"
                accept=".xlsx,.csv"
                style={{ display: 'none' }}
                onChange={handleFileInput}
              />
              {step === 'selected' && file && (
                <div style={{ marginTop: '1rem', display: 'flex', gap: '0.5rem' }}>
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={handleAnalyse}
                  >
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
                      <circle cx="11" cy="11" r="7"/>
                      <path d="M21 21l-4.3-4.3"/>
                    </svg>
                    Analyse file
                  </button>
                  <button type="button" className="btn btn-ghost" onClick={removeFile}>
                    Remove file
                  </button>
                </div>
              )}
            </>
          )}

          {/* ── analysing ── */}
          {step === 'analysing' && (
            <div style={{ textAlign: 'center', padding: '2.5rem 0' }}>
              <Spinner />
              <p style={{ marginTop: '0.75rem', color: 'var(--cds-text-secondary)', fontSize: '0.875rem' }}>
                Analysing file, please wait…
              </p>
              <button
                type="button"
                className="btn btn-ghost"
                style={{ marginTop: '1rem' }}
                onClick={handleCancel}
              >
                Cancel
              </button>
            </div>
          )}

          {/* ── preview ── */}
          {step === 'preview' && preview && (
            <PreviewPane
              preview={preview}
              upsert={upsert}
              onUpsertChange={setUpsert}
              onImport={handleImport}
              onBack={removeFile}
            />
          )}

          {/* ── importing ── */}
          {step === 'importing' && (
            <div style={{ textAlign: 'center', padding: '2.5rem 0' }}>
              <Spinner />
              <p style={{ marginTop: '0.75rem', color: 'var(--cds-text-secondary)', fontSize: '0.875rem' }}>
                Importing valid records…
              </p>
              <p style={{ fontSize: '0.75rem', color: 'var(--cds-text-placeholder)', marginTop: '0.25rem' }}>
                Do not close this window until the import finishes.
              </p>
            </div>
          )}

          {/* ── result ── */}
          {step === 'result' && result && (
            <ResultPane result={result} previewErrors={preview?.errors} onClose={onClose} onReset={handleReset} />
          )}

          {/* ── fatal (file type error or preview error) ── */}
          {step === 'fatal' && (
            <div style={{
              padding: '1rem',
              background: 'var(--cds-support-error-bg)',
              borderLeft: '4px solid var(--cds-support-error)',
              borderRadius: 0,
            }}>
              <p style={{ color: 'var(--cds-support-error)', fontWeight: 600, marginBottom: '0.5rem', fontSize: '0.875rem' }}>
                Cannot continue
              </p>
              <p style={{ fontSize: '0.875rem', color: 'var(--cds-text-primary)', marginBottom: '1rem' }}>
                {error}
              </p>
              <button type="button" className="btn btn-secondary" onClick={handleReset}>
                Try another file
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Sub-components ─────────────────────────────────────────────────────────────

// Step progress bar
function StepIndicator({ step }: { step: Step }) {
  const steps: { key: Step[]; label: string }[] = [
    { key: ['select', 'selected'], label: 'Select file' },
    { key: ['analysing', 'preview'], label: 'Review' },
    { key: ['importing', 'result'], label: 'Import' },
  ]
  const activeIndex =
    ['select', 'selected'].includes(step) ? 0
    : ['analysing', 'preview'].includes(step) ? 1
    : 2

  return (
    <div style={{ display: 'flex', gap: 0, marginTop: '0.75rem' }}>
      {steps.map((s, i) => (
        <div
          key={s.label}
          style={{
            flex: 1,
            height: 3,
            background: i <= activeIndex ? 'var(--cds-interactive)' : 'var(--cds-border-subtle-00)',
            borderRight: i < steps.length - 1 ? '2px solid var(--cds-background)' : undefined,
            transition: 'background 0.2s',
          }}
          title={s.label}
        />
      ))}
    </div>
  )
}

// Drop zone
interface DropZoneProps {
  dragging: boolean
  file: File | null
  onDrop: (e: React.DragEvent) => void
  onDragOver: (e: React.DragEvent) => void
  onDragLeave: (e: React.DragEvent) => void
  onClick: () => void
  onRemove: () => void
}

function DropZone({ dragging, file, onDrop, onDragOver, onDragLeave, onClick, onRemove }: DropZoneProps) {
  return (
    <div
      role={file ? undefined : 'button'}
      tabIndex={file ? undefined : 0}
      aria-label={file ? undefined : 'Select file to import'}
      style={{
        border: `2px dashed ${dragging ? 'var(--cds-interactive)' : 'var(--cds-border-subtle-01)'}`,
        padding: '2rem 1.5rem',
        textAlign: 'center',
        background: dragging ? 'var(--cds-blue-10)' : 'var(--cds-layer-01)',
        cursor: file ? 'default' : 'pointer',
        transition: 'border-color 0.12s, background 0.12s',
      }}
      onDrop={onDrop}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onClick={file ? undefined : onClick}
      onKeyDown={file ? undefined : (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
    >
      {!file ? (
        <>
          <svg
            width="32" height="32" viewBox="0 0 24 24"
            fill="none" stroke="var(--cds-text-placeholder)" strokeWidth="1.5"
            style={{ display: 'block', margin: '0 auto 0.75rem' }}
          >
            <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
            <polyline points="17 8 12 3 7 8"/>
            <line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          <p style={{ color: 'var(--cds-text-secondary)', fontSize: '0.875rem', marginBottom: '0.25rem' }}>
            Drag and drop a file here, or{' '}
            <span style={{ color: 'var(--cds-interactive)', textDecoration: 'underline', cursor: 'pointer' }}>
              select from your computer
            </span>
          </p>
          <p style={{ color: 'var(--cds-text-placeholder)', fontSize: '0.75rem' }}>
            Allowed formats: .xlsx, .csv
          </p>
        </>
      ) : (
        <div
          style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', justifyContent: 'center' }}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--cds-support-success)" strokeWidth="2">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
          </svg>
          <div style={{ textAlign: 'left' }}>
            <p style={{ fontWeight: 600, fontSize: '0.875rem', margin: 0 }}>{file.name}</p>
            <p style={{ color: 'var(--cds-text-secondary)', fontSize: '0.75rem', margin: '0.1rem 0 0' }}>
              {formatBytes(file.size)} · {detectFileType(file.name)}
            </p>
          </div>
          <button
            type="button"
            className="icon-btn"
            title="Remove file"
            aria-label="Remove selected file"
            onClick={(e) => { e.stopPropagation(); onRemove() }}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12"/>
            </svg>
          </button>
        </div>
      )}
    </div>
  )
}

// Preview pane
interface PreviewPaneProps {
  preview: PreviewResult
  upsert: boolean
  onUpsertChange: (v: boolean) => void
  onImport: () => void
  onBack: () => void
}

function PreviewPane({ preview, upsert, onUpsertChange, onImport, onBack }: PreviewPaneProps) {
  const { summary, sheets, errors, warnings } = preview
  const hasErrors  = summary.with_errors > 0
  const canImport  = summary.insertable > 0 || summary.updatable > 0 || (upsert && summary.duplicates > 0)

  return (
    <div>
      {/* Summary cards */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(4, 1fr)',
        gap: '0.5rem',
        marginBottom: '1rem',
      }}>
        {[
          { label: 'Rows read',   value: summary.rows_read,   color: 'var(--cds-blue-60)' },
          { label: 'New records', value: summary.insertable,  color: 'var(--cds-support-success)' },
          { label: 'Duplicates',  value: summary.duplicates,  color: 'var(--cds-blue-50)' },
          {
            label: 'Errors',
            value: summary.with_errors,
            color: hasErrors ? 'var(--cds-support-error)' : 'var(--cds-text-secondary)',
          },
        ].map((s) => (
          <div
            key={s.label}
            style={{
              background: 'var(--cds-layer-02)',
              border: '1px solid var(--cds-border-subtle-00)',
              padding: '0.5rem 0.75rem',
              textAlign: 'center',
            }}
          >
            <div style={{ fontSize: '1.25rem', fontWeight: 700, color: s.color }}>{s.value}</div>
            <div style={{ fontSize: '0.7rem', color: 'var(--cds-text-secondary)' }}>{s.label}</div>
          </div>
        ))}
      </div>

      {/* Additional counters row */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        gap: '0.5rem',
        marginBottom: '1rem',
      }}>
        {[
          { label: 'Laptops',    value: summary.laptop_rows },
          { label: 'Equipment',  value: summary.equipment_rows },
          { label: 'Warnings',   value: summary.with_warnings },
        ].map((s) => (
          <div
            key={s.label}
            style={{
              background: 'var(--cds-layer-01)',
              border: '1px solid var(--cds-border-subtle-00)',
              padding: '0.4rem 0.75rem',
              textAlign: 'center',
            }}
          >
            <div style={{ fontSize: '1rem', fontWeight: 600, color: 'var(--cds-text-secondary)' }}>{s.value}</div>
            <div style={{ fontSize: '0.7rem', color: 'var(--cds-text-placeholder)' }}>{s.label}</div>
          </div>
        ))}
      </div>

      {/* Sheets detected */}
      {sheets?.length > 0 && (
        <div style={{ marginBottom: '0.75rem' }}>
          <p style={{
            fontSize: '0.75rem',
            color: 'var(--cds-text-secondary)',
            fontWeight: 600,
            marginBottom: '0.3rem',
            letterSpacing: '0.04em',
            textTransform: 'uppercase',
          }}>
            Detected sheets
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem' }}>
            {sheets.map((s) => (
              <span key={`${s.file}:${s.sheet}`} className="badge neutral" style={{ fontSize: '0.7rem' }}>
                {s.sheet} ({s.row_count} rows)
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Errors — registros rechazados */}
      {errors?.length > 0 && (
        <div style={{
          marginBottom: '0.75rem',
          maxHeight: 170,
          overflowY: 'auto',
          background: 'var(--cds-support-error-bg)',
          borderLeft: '4px solid var(--cds-support-error)',
          padding: '0.5rem 0.75rem',
        }}>
          <p style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--cds-support-error)', marginBottom: '0.4rem' }}>
            Records with errors ({errors.length}) — will not be imported
          </p>
          {errors.map((r, i) => (
            <ErrorRowDetail key={i} row={r} />
          ))}
        </div>
      )}

      {/* Warnings — importados con ajustes */}
      {warnings?.length > 0 && (
        <div style={{
          marginBottom: '0.75rem',
          maxHeight: 120,
          overflowY: 'auto',
          background: 'var(--cds-support-warning-bg)',
          borderLeft: '4px solid var(--cds-support-warning)',
          padding: '0.5rem 0.75rem',
        }}>
          <p style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--cds-support-warning-text)', marginBottom: '0.4rem' }}>
            Warnings ({warnings.length}) — will be imported with normalised fields
          </p>
          {warnings.slice(0, 5).map((r, i) => (
            <div key={i} style={{ marginBottom: '0.3rem' }}>
              <span style={{ fontSize: '0.7rem', color: 'var(--cds-text-primary)' }}>
                Row {r.row}
                {r.sheet && <> · Sheet: <strong>{r.sheet}</strong></>}
              </span>
              {r.issues?.filter((x) => x.severity === 'WARNING').map((x, j) => (
                <div key={j} style={{ paddingLeft: '0.75rem', fontSize: '0.68rem', color: 'var(--cds-text-secondary)' }}>
                  Campo: <code style={{ fontSize: '0.68rem' }}>{x.field}</code> · {x.message}
                </div>
              ))}
            </div>
          ))}
          {warnings.length > 5 && (
            <p style={{ fontSize: '0.7rem', color: 'var(--cds-text-secondary)', marginTop: '0.25rem' }}>
              … and {warnings.length - 5} more
            </p>
          )}
        </div>
      )}

      {/* Upsert toggle */}
      <label style={{
        display: 'flex', alignItems: 'center', gap: '0.5rem',
        fontSize: '0.8rem', marginBottom: '1rem', cursor: 'pointer',
      }}>
        <input
          type="checkbox"
          checked={upsert}
          onChange={(e) => onUpsertChange(e.target.checked)}
          style={{ cursor: 'pointer' }}
        />
        Update existing records (upsert)
      </label>

      {/* Confirmation text */}
      <p style={{ fontSize: '0.8rem', color: 'var(--cds-text-secondary)', marginBottom: '1rem' }}>
        <strong>{summary.insertable}</strong> new record(s) will be inserted
        {summary.updatable > 0 && <>, <strong>{summary.updatable}</strong> will be updated</>}.
        {summary.duplicates > 0 && <> <strong>{summary.duplicates}</strong> duplicate(s) will be skipped.</>}
        {hasErrors && (
          <>
            {' '}<span style={{ color: 'var(--cds-support-error)' }}>
              <strong>{summary.with_errors}</strong> row(s) with errors will not be imported.
            </span>
          </>
        )}
      </p>

      {/* Actions */}
      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
        {canImport && (
          <button type="button" className="btn btn-primary" onClick={onImport}>
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2">
              <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Import valid records
          </button>
        )}
        {!canImport && (
          <p style={{ fontSize: '0.8rem', color: 'var(--cds-support-error)', alignSelf: 'center' }}>
            No valid records to import.
          </p>
        )}
        <button type="button" className="btn btn-ghost" onClick={onBack}>
          Choose another file
        </button>
      </div>
    </div>
  )
}

// ── ErrorRowDetail ─────────────────────────────────────────────────────────────

/**
 * Renders a single error row with file/sheet/row/serial/field/message.
 * Serial is already masked by the backend (****XXXX).
 */
function ErrorRowDetail({ row }: { row: ImportRow }) {
  const errorIssues = row.issues?.filter((x) => x.severity === 'ERROR') ?? []
  const serial = row.fields?.serial ?? ''

  return (
    <div style={{ marginBottom: '0.4rem', paddingBottom: '0.35rem', borderBottom: '1px solid var(--cds-support-error)' }}>
      <span style={{ fontSize: '0.72rem', fontWeight: 600, color: 'var(--cds-text-primary)' }}>
        Row {row.row}
        {row.sheet && <> · Sheet: <strong>{row.sheet}</strong></>}
        {serial && <> · Serial: <strong>{serial}</strong></>}
      </span>
      {errorIssues.map((issue, i) => (
        <div key={i} style={{ paddingLeft: '0.75rem', marginTop: '0.1rem' }}>
          <span style={{ fontSize: '0.68rem', color: 'var(--cds-support-error)' }}>
            Campo: <code style={{ fontSize: '0.68rem' }}>{issue.field}</code>
          </span>
          <span style={{ fontSize: '0.68rem', color: 'var(--cds-text-primary)', marginLeft: '0.4rem' }}>
            · {issue.message}
          </span>
        </div>
      ))}
    </div>
  )
}

// ── Result pane ────────────────────────────────────────────────────────────────

interface ResultPaneProps {
  result: ApplyResult
  previewErrors?: ImportRow[]
  onClose: () => void
  onReset: () => void
}

function ResultPane({ result, previewErrors, onClose, onReset }: ResultPaneProps) {
  const { summary } = result
  const [showErrors, setShowErrors] = useState(false)
  const rejectedRows = previewErrors ?? []

  return (
    <div>
      <div style={{
        display: 'flex', alignItems: 'center', gap: '0.5rem',
        marginBottom: '1rem',
        padding: '0.75rem',
        background: 'var(--cds-support-success-bg)',
        borderLeft: '4px solid var(--cds-support-success)',
      }}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--cds-support-success)" strokeWidth="2.2">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        <span style={{ fontWeight: 600, fontSize: '0.95rem', color: 'var(--cds-support-success)' }}>
          Import completed
        </span>
      </div>

      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        gap: '0.5rem',
        marginBottom: '1.25rem',
      }}>
        {[
          { label: 'Inserted',  value: summary.inserted,      color: 'var(--cds-support-success)' },
          { label: 'Updated',   value: summary.updated,       color: 'var(--cds-blue-60)' },
          { label: 'Skipped',   value: summary.skipped,       color: 'var(--cds-text-secondary)' },
          { label: 'Errors',    value: summary.errored,       color: summary.errored > 0 ? 'var(--cds-support-error)' : 'var(--cds-text-secondary)' },
          { label: 'Warnings',  value: summary.with_warnings, color: 'var(--cds-support-warning-text)' },
          { label: 'Duration',  value: `${summary.duration_ms} ms`, color: 'var(--cds-text-secondary)' },
        ].map((s) => (
          <div
            key={s.label}
            style={{
              background: 'var(--cds-layer-01)',
              border: '1px solid var(--cds-border-subtle-00)',
              padding: '0.5rem 0.75rem',
              textAlign: 'center',
            }}
          >
            <div style={{ fontSize: '1.1rem', fontWeight: 700, color: s.color }}>{s.value}</div>
            <div style={{ fontSize: '0.7rem', color: 'var(--cds-text-secondary)' }}>{s.label}</div>
          </div>
        ))}
      </div>

      {/* Rejected records notice */}
      {rejectedRows.length > 0 && (
        <div style={{
          marginBottom: '1rem',
          background: 'var(--cds-support-error-bg)',
          borderLeft: '4px solid var(--cds-support-error)',
          padding: '0.5rem 0.75rem',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '0.8rem', color: 'var(--cds-support-error)', fontWeight: 600 }}>
              {rejectedRows.length} record(s) could not be imported
            </span>
            <button
              type="button"
              className="btn btn-ghost"
              style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem' }}
              onClick={() => setShowErrors((v) => !v)}
            >
              {showErrors ? 'Hide' : 'Show details'}
            </button>
          </div>
          {showErrors && (
            <div style={{ marginTop: '0.5rem', maxHeight: 160, overflowY: 'auto' }}>
              {rejectedRows.map((r, i) => (
                <ErrorRowDetail key={i} row={r} />
              ))}
              <p style={{ fontSize: '0.68rem', color: 'var(--cds-text-placeholder)', marginTop: '0.4rem' }}>
                Note: CSV error download will be available in a future version.
              </p>
            </div>
          )}
        </div>
      )}

      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
        <button type="button" className="btn btn-primary" onClick={onClose}>
          Close
        </button>
        <button type="button" className="btn btn-ghost" onClick={onReset}>
          Import another file
        </button>
      </div>
    </div>
  )
}

// Spinner
function Spinner() {
  return (
    <svg
      width="28" height="28" viewBox="0 0 50 50"
      style={{ animation: 'spin 0.75s linear infinite', display: 'block', margin: '0 auto' }}
    >
      <circle
        cx="25" cy="25" r="20"
        fill="none"
        stroke="var(--cds-interactive)"
        strokeWidth="4"
        strokeDasharray="31.4 94.2"
      />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </svg>
  )
}
