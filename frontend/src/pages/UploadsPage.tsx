import React, { useEffect, useMemo, useState, useCallback } from 'react'
import { useUpload } from '../context/UploadContext'
import { useAuth } from '../context/AuthContext'
import type { UploadSession, UploadPhase } from '../types'
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal'
import { formatBytes, formatDate } from '../utils/format'

function useDeleteSession() {
  const { deleteSession } = useUpload()
  const [pending, setPending] = useState<UploadSession | null>(null)
  const [deleting, setDeleting] = useState(false)

  const requestDelete = useCallback((s: UploadSession) => setPending(s), [])
  const cancel = useCallback(() => setPending(null), [])
  const confirm = useCallback(async () => {
    if (!pending) return
    setDeleting(true)
    try { await deleteSession(pending.id) } finally {
      setDeleting(false)
      setPending(null)
    }
  }, [pending, deleteSession])

  return { pending, deleting, requestDelete, cancel, confirm }
}

type Filter = 'all' | 'active' | 'done' | 'failed'

function duration(startedAt: string, completedAt?: string): string {
  const end = completedAt ? new Date(completedAt).getTime() : Date.now()
  const ms = end - new Date(startedAt).getTime()
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m ${s % 60}s`
}

interface PhaseDisplay {
  label: string
  icon: string
  dot: string
  text: string
  badge: string
}

const PHASE: Record<string, PhaseDisplay> = {
  uploading:  { label: 'Uploading',  icon: 'upload',       dot: 'bg-primary animate-pulse',   text: 'text-primary',   badge: 'bg-primary/10 text-primary' },
  assembling: { label: 'Assembling', icon: 'manufacturing', dot: 'bg-tertiary animate-pulse',  text: 'text-tertiary',  badge: 'bg-tertiary/10 text-tertiary' },
  generating: { label: 'Generating', icon: 'bar_chart',     dot: 'bg-secondary animate-pulse', text: 'text-secondary', badge: 'bg-secondary/10 text-secondary' },
  done:       { label: 'Done',       icon: 'check_circle',  dot: 'bg-success',                 text: 'text-success',   badge: 'bg-success/10 text-success' },
  failed:     { label: 'Failed',     icon: 'error',         dot: 'bg-error',                   text: 'text-error',     badge: 'bg-error/10 text-error' },
}

// Tailwind alias for success color using emerald (matches the rest of the app).
const successText = 'text-emerald-500'
const PHASE_DONE: PhaseDisplay = { ...PHASE.done, text: successText, badge: 'bg-emerald-500/10 text-emerald-500' }

function getPhase(phase: UploadPhase): PhaseDisplay {
  if (phase === 'done') return PHASE_DONE
  return PHASE[phase] ?? PHASE.uploading
}

function isActive(phase: UploadPhase) {
  return phase === 'uploading' || phase === 'assembling' || phase === 'generating'
}

function SessionRow({ s, onDelete, canDelete }: { s: UploadSession; onDelete: (s: UploadSession) => void; canDelete: boolean }) {
  const ph = getPhase(s.phase)
  const [errorExpanded, setErrorExpanded] = useState(false)
  const toggleError = useCallback((e: React.MouseEvent) => {
    e.stopPropagation()
    setErrorExpanded(v => !v)
  }, [])

  const pct = s.totalChunks > 0
    ? Math.round((s.receivedChunks / s.totalChunks) * 100)
    : s.phase === 'done' ? 100 : 0

  const isFailed = s.phase === 'failed'
  const hasError = isFailed && !!s.error

  return (
    <div className={`group border rounded-xl overflow-hidden transition-all duration-200 ${
      isFailed
        ? 'bg-error/5 border-error/20 hover:border-error/40'
        : 'bg-surface-container-low hover:bg-surface-container border-outline-variant/10'
    }`}>
      <div className="flex items-center gap-5 px-5 py-4">

        {/* Phase dot */}
        <span className={`w-2.5 h-2.5 rounded-full shrink-0 ${ph.dot}`} />

        {/* File + build */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 flex-wrap">
            <p className="text-sm font-headline font-bold text-on-surface truncate">
              {s.fileName || 'results.zip'}
            </p>
            <span className={`text-[10px] font-label font-bold uppercase tracking-wider px-2 py-0.5 rounded-full shrink-0 ${ph.badge}`}>
              {ph.label}
            </span>
          </div>
          <div className="flex items-center gap-4 mt-1 text-[11px] text-on-surface-variant font-label flex-wrap">
            <span className="font-mono">{s.buildId}</span>
            <span className="opacity-50">·</span>
            <span>{s.projectId} / {s.envId}</span>
            <span className="opacity-50">·</span>
            <span>{formatBytes(s.totalSize)}</span>
            {s.uploadedBy && (
              <>
                <span className="opacity-50">·</span>
                <span className="flex items-center gap-1">
                  <span className="material-symbols-outlined text-[11px]">person</span>
                  {s.uploadedBy}
                </span>
              </>
            )}
          </div>
        </div>

        {/* Upload progress bar — all phases */}
        {s.phase !== 'failed' && (
          <div className="w-36 shrink-0">
            <div className="flex justify-between text-[10px] text-on-surface-variant font-label mb-1">
              <span>{ph.label}</span>
              <span>{s.phase === 'uploading' && pct === 0 ? '…' : `${pct}%`}</span>
            </div>
            <div className="h-1.5 bg-surface-container-highest rounded-full overflow-hidden">
              {s.phase === 'assembling' || s.phase === 'generating' || (s.phase === 'uploading' && pct === 0) ? (
                <div className="h-full w-full rounded-full bg-gradient-to-r from-primary/40 via-primary to-primary/40 animate-pulse" />
              ) : (
                <div
                  className={`h-full rounded-full transition-all duration-300 ${s.phase === 'done' ? 'bg-emerald-500' : 'bg-primary'}`}
                  style={{ width: `${pct}%` }}
                />
              )}
            </div>
          </div>
        )}

        {/* Done: report link */}
        {s.phase === 'done' && s.reportUrl && (
          <a
            href={s.reportUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-[11px] text-primary font-label font-semibold hover:underline shrink-0"
          >
            Open report
            <span className="material-symbols-outlined text-[13px]">arrow_forward</span>
          </a>
        )}

        {/* Failed: inline summary + expand toggle */}
        {isFailed && (
          <div className="flex items-center gap-2 shrink-0">
            <div className="flex items-center gap-1.5 bg-error/10 border border-error/20 rounded-lg px-3 py-1.5">
              <span className="material-symbols-outlined text-[14px] text-error">error</span>
              <span className="text-[11px] text-error font-label font-semibold max-w-[220px] truncate">
                {s.error || 'Unknown error'}
              </span>
            </div>
            {hasError && (
              <button
                onClick={toggleError}
                className="p-1.5 rounded-lg text-error/70 hover:text-error hover:bg-error/10 transition-colors"
                title={errorExpanded ? 'Collapse error' : 'Expand error'}
              >
                <span className="material-symbols-outlined text-[16px]">
                  {errorExpanded ? 'expand_less' : 'expand_more'}
                </span>
              </button>
            )}
          </div>
        )}

        {/* Timing */}
        <div className="text-right shrink-0 hidden lg:block">
          <p className="text-[11px] text-on-surface-variant font-label">{formatDate(s.startedAt)}</p>
          <p className="text-[10px] text-on-surface-variant/60 font-label mt-0.5">
            {duration(s.startedAt, s.completedAt)}
          </p>
        </div>

        {/* Delete */}
        {canDelete && (
          <button
            onClick={() => onDelete(s)}
            className="shrink-0 p-1.5 rounded-lg text-on-surface-variant/40 hover:text-error hover:bg-error/10
                       opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all"
            title="Delete session"
            aria-label={`Delete upload session for ${s.fileName}`}
          >
            <span className="material-symbols-outlined text-[18px]" aria-hidden="true">delete</span>
          </button>
        )}
      </div>

      {/* Expanded error panel */}
      {isFailed && errorExpanded && hasError && (
        <div className="border-t border-error/15 bg-error/5 px-5 py-3">
          <p className="text-[10px] font-label font-bold uppercase tracking-widest text-error/60 mb-1.5">
            Error details
          </p>
          <pre className="text-xs text-error font-mono whitespace-pre-wrap break-all leading-relaxed">
            {s.error}
          </pre>
        </div>
      )}
    </div>
  )
}

const PAGE_SIZE = 20

export default function UploadsPage() {
  const { sessions } = useUpload()
  const { can } = useAuth()
  const { pending, deleting, requestDelete, cancel, confirm } = useDeleteSession()
  const [filter, setFilter] = useState<Filter>('all')
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE)

  const counts = useMemo(() => ({
    all:    sessions.length,
    active: sessions.filter(s => isActive(s.phase)).length,
    done:   sessions.filter(s => s.phase === 'done').length,
    failed: sessions.filter(s => s.phase === 'failed').length,
  }), [sessions])

  // Reset pagination whenever the filter changes — kept in useEffect to avoid
  // a side effect inside useMemo which violates React's rules (M-18).
  useEffect(() => {
    setVisibleCount(PAGE_SIZE)
  }, [filter])

  const filtered = useMemo(() => {
    switch (filter) {
      case 'active': return sessions.filter(s => isActive(s.phase))
      case 'done':   return sessions.filter(s => s.phase === 'done')
      case 'failed': return sessions.filter(s => s.phase === 'failed')
      default:       return sessions
    }
  }, [sessions, filter])

  const visible = filtered.slice(0, visibleCount)
  const hasMore = filtered.length > visibleCount

  const FILTERS: { key: Filter; label: string; activeCls: string }[] = [
    { key: 'all',    label: 'All',    activeCls: 'bg-surface-container-highest text-on-surface' },
    { key: 'active', label: 'Active', activeCls: 'bg-primary/10 text-primary' },
    { key: 'done',   label: 'Done',   activeCls: 'bg-emerald-500/10 text-emerald-500' },
    { key: 'failed', label: 'Failed', activeCls: 'bg-red-500/10 text-red-500' },
  ]

  return (
    <div className="flex flex-col h-full -mx-8 -my-6">

      {/* Sticky header: page title + stats + filters */}
      <div className="sticky top-0 z-10 bg-background px-8 pt-6 pb-4 border-b border-outline-variant/15">
        {/* Page header */}
        <div className="mb-6">
          <h2 className="text-4xl font-bold font-headline tracking-tight text-on-surface">
            Uploads
          </h2>
          <p className="text-on-surface-variant font-body mt-2">
            All upload activity — UI and API uploads in one place.{' '}
            {counts.active > 0 && (
              <span className="text-primary font-semibold">
                {counts.active} active right now.
              </span>
            )}
          </p>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-4">
          {[
            { label: 'Total',  value: counts.all,    color: 'text-on-surface' },
            { label: 'Active', value: counts.active,  color: counts.active > 0 ? 'text-primary' : 'text-on-surface' },
            { label: 'Done',   value: counts.done,    color: counts.done > 0 ? successText : 'text-on-surface' },
            { label: 'Failed', value: counts.failed,  color: counts.failed > 0 ? 'text-error' : 'text-on-surface' },
          ].map(({ label, value, color }) => (
            <div key={label} className="bg-surface-container-low rounded-xl p-4 border border-outline-variant/10">
              <p className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1">{label}</p>
              <p className={`text-3xl font-headline font-bold ${color}`}>{value}</p>
            </div>
          ))}
        </div>

        {/* Filter tabs */}
        <div className="flex items-center gap-1 bg-surface-container-low p-1 rounded-lg w-fit">
          {FILTERS.map(({ key, label, activeCls }) => (
            <button
              key={key}
              onClick={() => setFilter(key)}
              className={`px-4 py-1.5 text-xs font-label font-bold rounded-md transition-colors ${
                filter === key
                  ? `${activeCls} shadow-sm`
                  : 'text-on-surface-variant hover:text-on-surface'
              }`}
            >
              {label}
              {counts[key] > 0 && (
                <span className="ml-1.5 text-[10px] opacity-70">{counts[key]}</span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Scrollable session list */}
      <div className="flex-1 overflow-y-auto px-8 py-6">
        {filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-on-surface-variant bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
            <span className="material-symbols-outlined text-[48px] mb-4 opacity-20">cloud_upload</span>
            <p className="font-headline font-bold text-on-surface text-base">No uploads yet</p>
            <p className="text-sm font-body mt-1">
              {filter === 'all'
                ? 'Upload results from a project page or via the API.'
                : `No ${filter} uploads to show.`}
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {visible.map(s => <SessionRow key={s.id} s={s} onDelete={requestDelete} canDelete={can('manage')} />)}
            {hasMore && (
              <button
                onClick={() => setVisibleCount(c => c + PAGE_SIZE)}
                className="flex items-center justify-center gap-2 py-3 rounded-xl border border-outline-variant/20
                           text-sm font-label font-semibold text-on-surface-variant
                           hover:bg-surface-container hover:text-on-surface transition-colors"
              >
                <span className="material-symbols-outlined text-[18px]">expand_more</span>
                Load more ({filtered.length - visibleCount} remaining)
              </button>
            )}
          </div>
        )}
      </div>

      <DeleteConfirmModal
        isOpen={!!pending}
        onClose={cancel}
        onConfirm={confirm}
        title="Delete Upload Session"
        description="This will permanently remove the session record and all associated files (chunks, results, report)."
        itemName={pending ? (pending.fileName || pending.buildId) : ''}
        isDeleting={deleting}
      />
    </div>
  )
}
