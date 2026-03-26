import React, { useState } from 'react'
import type { UploadSession } from '../types'
import { formatBytes } from '../utils/format'

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ago`
}

interface Props {
  session: UploadSession
  onRetry?: (session: UploadSession) => void
}

const PHASE_META = {
  uploading:  { label: 'Uploading',   icon: 'cloud_upload',  accent: 'bg-primary',   text: 'text-primary',   ring: 'border-primary/25',   bg: 'bg-primary/5'   },
  assembling: { label: 'Assembling',  icon: 'manufacturing', accent: 'bg-tertiary',  text: 'text-tertiary',  ring: 'border-tertiary/25',  bg: 'bg-tertiary/5'  },
  generating: { label: 'Generating', icon: 'auto_graph',    accent: 'bg-secondary', text: 'text-secondary', ring: 'border-secondary/25', bg: 'bg-secondary/5' },
  done:       { label: 'Done',        icon: 'check_circle',  accent: 'bg-emerald-500', text: 'text-emerald-500', ring: 'border-emerald-500/20', bg: 'bg-emerald-500/5' },
  failed:     { label: 'Failed',      icon: 'cancel',        accent: 'bg-error',     text: 'text-error',     ring: 'border-error/25',     bg: 'bg-error/5'     },
} as const

const RETRY_LABEL: Partial<Record<string, { label: string; icon: string }>> = {
  assembling: { label: 'Retry Assembly',   icon: 'manufacturing' },
  generating: { label: 'Retry Generation', icon: 'auto_graph'    },
}

function UploadSessionCard({ session, onRetry }: Props) {
  const [warningExpanded, setWarningExpanded] = useState(false)
  const meta = PHASE_META[session.phase as keyof typeof PHASE_META] ?? PHASE_META.uploading
  const isActive = session.phase === 'uploading' || session.phase === 'assembling' || session.phase === 'generating'
  const pct = session.totalChunks > 0
    ? Math.round((session.receivedChunks / session.totalChunks) * 100)
    : 0
  const retry = session.phase === 'failed' && session.failedAtPhase
    ? RETRY_LABEL[session.failedAtPhase]
    : undefined

  return (
    <div className={`relative rounded-2xl border overflow-hidden transition-all duration-300 ${meta.ring} ${meta.bg}`}>
      {/* Animated left accent bar */}
      <div className={`absolute left-0 top-0 bottom-0 w-[3px] ${meta.accent} ${isActive ? 'opacity-100' : 'opacity-60'}`} />

      <div className="pl-4 pr-4 pt-3.5 pb-3">
        {/* Top row: filename + status chip */}
        <div className="flex items-start justify-between gap-3 mb-2">
          <div className="min-w-0 flex-1">
            <p className="text-[13px] font-headline font-bold text-on-surface truncate leading-tight">
              {session.fileName || 'results.zip'}
            </p>
            <p className="text-[11px] text-on-surface-variant font-mono mt-0.5 truncate">
              {session.buildId}
              {session.totalSize > 0 && (
                <span className="font-sans opacity-50 ml-1.5">· {formatBytes(session.totalSize)}</span>
              )}
            </p>
          </div>

          {/* Status pill */}
          <span className={`flex items-center gap-1 text-[10px] font-label font-bold uppercase tracking-widest
                            px-2 py-1 rounded-full shrink-0 ${meta.text}`}
            style={{ background: 'rgb(var(--color-surface-container-high) / 0.6)' }}
          >
            {isActive && (
              <span className={`w-1.5 h-1.5 rounded-full ${meta.accent} animate-pulse shrink-0`} />
            )}
            <span className="material-symbols-outlined text-[12px]">{meta.icon}</span>
            {meta.label}
          </span>
        </div>

        {/* ── Phase-specific content ── */}

        {/* Uploading: progress bar */}
        {session.phase === 'uploading' && (
          <div className="mt-1">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] text-on-surface-variant font-label">
                {session.totalChunks > 1
                  ? `Chunk ${session.receivedChunks} / ${session.totalChunks}`
                  : 'Uploading…'}
              </span>
              <span className={`text-[11px] font-label font-bold ${meta.text}`}>
                {pct > 0 ? `${pct}%` : ''}
              </span>
            </div>
            <div className="h-1 rounded-full overflow-hidden"
              style={{ background: 'rgb(var(--color-surface-container-highest))' }}
            >
              {pct === 0 ? (
                <div className={`h-full w-full rounded-full ${meta.accent} opacity-60 animate-pulse`} />
              ) : (
                <div
                  className={`h-full rounded-full transition-all duration-500 ${meta.accent}`}
                  style={{ width: `${pct}%` }}
                />
              )}
            </div>
          </div>
        )}

        {/* Assembling / Generating: animated spinner + message */}
        {(session.phase === 'assembling' || session.phase === 'generating') && (
          <div className="flex items-center gap-2 mt-1">
            <svg className={`w-3 h-3 animate-spin shrink-0 ${meta.text}`} viewBox="0 0 24 24" fill="none">
              <circle className="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
              <path className="opacity-80" fill="currentColor" d="M4 12a8 8 0 018-8v8z" />
            </svg>
            <span className="text-[11px] text-on-surface-variant font-label">
              {session.phase === 'assembling' ? 'Stitching chunks together…' : 'Running allure generate…'}
            </span>
          </div>
        )}

        {/* Done: stats row + open link */}
        {session.phase === 'done' && (
          <div className="mt-1">
            <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-3 flex-wrap">
              <span className="flex items-center gap-1 text-[11px] font-label text-emerald-500 font-semibold">
                <span className="material-symbols-outlined text-[13px]">check_circle</span>
                {session.passed} passed
              </span>
              {session.failed > 0 && (
                <span className="flex items-center gap-1 text-[11px] font-label text-error font-semibold">
                  <span className="material-symbols-outlined text-[13px]">cancel</span>
                  {session.failed} failed
                </span>
              )}
              <span className="text-[10px] text-on-surface-variant/50 font-label">
                {timeAgo(session.completedAt ?? session.startedAt)}
              </span>
            </div>
            {session.reportUrl && (
              <a
                href={session.reportUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-0.5 text-[11px] font-label font-bold text-primary
                           hover:underline shrink-0 transition-colors"
              >
                Open
                <span className="material-symbols-outlined text-[13px]">arrow_forward</span>
              </a>
            )}
            </div>
            {session.error && (
              <div className="mt-2">
                <button
                  onClick={() => setWarningExpanded(v => !v)}
                  className="flex items-center gap-1.5 text-[11px] text-amber-600 font-label font-semibold hover:text-amber-700 transition-colors"
                >
                  <span className="material-symbols-outlined text-[13px]">warning</span>
                  Warnings
                  <span className="material-symbols-outlined text-[13px]">
                    {warningExpanded ? 'expand_less' : 'expand_more'}
                  </span>
                </button>
                {warningExpanded && (
                  <pre className="mt-1 text-[11px] text-amber-700 font-mono whitespace-pre-wrap break-all leading-snug">
                    {session.error}
                  </pre>
                )}
              </div>
            )}
          </div>
        )}

        {/* Failed: error + retry */}
        {session.phase === 'failed' && (
          <div className="mt-1">
            <p className="text-[11px] text-error/80 font-label leading-snug line-clamp-2">
              {session.error || 'Unknown error'}
            </p>
            <div className="flex items-center justify-between mt-2 gap-2">
              <span className="text-[10px] text-on-surface-variant/50 font-label">
                {timeAgo(session.completedAt ?? session.startedAt)}
              </span>
              {retry && onRetry && (
                <button
                  onClick={() => onRetry(session)}
                  className="flex items-center gap-1 text-[11px] font-label font-bold
                             text-on-surface-variant border border-outline-variant/30 rounded-lg px-2.5 py-1
                             hover:text-primary hover:border-primary/40 hover:bg-primary/5 transition-all"
                >
                  <span className="material-symbols-outlined text-[13px]">refresh</span>
                  {retry.label}
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default React.memo(UploadSessionCard)
