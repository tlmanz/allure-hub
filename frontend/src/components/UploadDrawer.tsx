import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { useUpload } from '../context/UploadContext'
import UploadSessionCard from './UploadSessionCard'

const DRAWER_LIMIT = 10

export default function UploadDrawer() {
  const { sessions, drawerOpen, closeDrawer, retryUpload } = useUpload()
  const { pathname } = useLocation()
  const match = pathname.match(/^\/environments\/[^/]+\/projects\/([^/]+)/)
  const projectId = match?.[1] ?? null
  const drawerRef = useRef<HTMLDivElement>(null)

  // Only show sessions for the current project.
  const projectSessions = projectId
    ? sessions.filter(s => s.projectId === projectId)
    : sessions

  const activeCount = projectSessions.filter(
    s => s.phase === 'uploading' || s.phase === 'assembling' || s.phase === 'generating',
  ).length
  const doneCount = projectSessions.filter(s => s.phase === 'done').length
  const failedCount = projectSessions.filter(s => s.phase === 'failed').length
  const visible = projectSessions.slice(0, DRAWER_LIMIT)
  const overflow = projectSessions.length - DRAWER_LIMIT

  // Close on Escape key.
  useEffect(() => {
    if (!drawerOpen) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') closeDrawer() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [drawerOpen, closeDrawer])

  // Auto-close when navigating away from a project page.
  useEffect(() => {
    if (!projectId) closeDrawer()
  }, [projectId, closeDrawer])

  return (
    <>
      {/* Backdrop */}
      {drawerOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px]"
          onClick={closeDrawer}
        />
      )}

      {/* Drawer panel */}
      <div
        ref={drawerRef}
        className={`fixed top-14 right-0 h-[calc(100vh-3.5rem)] w-[400px] z-50 flex flex-col
                    bg-surface-container-lowest border-l border-outline-variant/20
                    shadow-2xl transition-transform duration-300 ease-in-out
                    ${drawerOpen ? 'translate-x-0' : 'translate-x-full'}`}
      >
        {/* ── Header ── */}
        <div
          className="relative flex items-center justify-between px-5 py-4 shrink-0
                     border-b border-outline-variant/15"
          style={{ background: 'rgb(var(--color-surface-container) / 0.6)' }}
        >
          {/* Left: icon + title */}
          <div className="flex items-center gap-3 min-w-0">
            <div
              className="w-8 h-8 rounded-xl flex items-center justify-center shrink-0"
              style={{ background: 'rgb(var(--color-primary) / 0.12)' }}
            >
              <span className="material-symbols-outlined text-[18px] text-primary">cloud_upload</span>
            </div>
            <div className="min-w-0">
              <h2 className="text-[13px] font-headline font-bold text-on-surface leading-tight">
                Upload Activity
              </h2>
              {projectId && (
                <p className="text-[10px] font-mono text-on-surface-variant/60 truncate mt-0.5">
                  {projectId}
                </p>
              )}
            </div>
          </div>

          {/* Right: stats pills + close */}
          <div className="flex items-center gap-1.5 shrink-0 ml-3">
            {activeCount > 0 && (
              <span
                className="flex items-center gap-1 text-[10px] font-label font-bold
                           text-primary px-2 py-0.5 rounded-full"
                style={{ background: 'rgb(var(--color-primary) / 0.12)' }}
              >
                <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                {activeCount} active
              </span>
            )}
            {doneCount > 0 && activeCount === 0 && (
              <span className="text-[10px] font-label text-emerald-500 font-semibold
                               px-2 py-0.5 rounded-full bg-emerald-500/10">
                {doneCount} done
              </span>
            )}
            {failedCount > 0 && (
              <span className="text-[10px] font-label text-error font-semibold
                               px-2 py-0.5 rounded-full bg-error/10">
                {failedCount} failed
              </span>
            )}
            <button
              onClick={closeDrawer}
              className="w-7 h-7 flex items-center justify-center rounded-full ml-1
                         hover:bg-surface-container-highest transition-colors text-on-surface-variant"
              aria-label="Close uploads drawer"
            >
              <span className="material-symbols-outlined text-[18px]">close</span>
            </button>
          </div>
        </div>

        {/* ── Session list ── */}
        <div className="flex-1 overflow-y-auto">
          {projectSessions.length === 0 ? (
            /* Empty state */
            <div className="flex flex-col items-center justify-center h-full gap-4 px-8 text-center">
              <div
                className="w-16 h-16 rounded-2xl flex items-center justify-center"
                style={{ background: 'rgb(var(--color-surface-container-high) / 0.6)' }}
              >
                <span className="material-symbols-outlined text-[32px] text-on-surface-variant/40">
                  cloud_upload
                </span>
              </div>
              <div>
                <p className="text-[13px] font-headline font-semibold text-on-surface-variant">
                  No uploads yet
                </p>
                <p className="text-[11px] text-on-surface-variant/50 font-label mt-1 leading-relaxed">
                  Use the Upload Results button or<br />push results via the API
                </p>
              </div>
            </div>
          ) : (
            <div className="p-4 flex flex-col gap-2.5">
              {visible.map(s => (
                <UploadSessionCard key={s.id} session={s} onRetry={retryUpload} />
              ))}

              {/* Overflow notice */}
              {overflow > 0 && (
                <div
                  className="flex items-center gap-2 px-3 py-2.5 rounded-xl
                              border border-outline-variant/15 text-[11px] font-label
                              text-on-surface-variant/60 justify-center"
                  style={{ background: 'rgb(var(--color-surface-container) / 0.4)' }}
                >
                  <span className="material-symbols-outlined text-[13px]">more_horiz</span>
                  {overflow} older {overflow === 1 ? 'session' : 'sessions'} not shown
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Footer ── */}
        <div
          className="shrink-0 px-5 py-3 border-t border-outline-variant/15
                     flex items-center justify-between"
          style={{ background: 'rgb(var(--color-surface-container) / 0.4)' }}
        >
          <p className="text-[10px] font-label text-on-surface-variant/40">
            Updates live via SSE
          </p>
          <button
            onClick={closeDrawer}
            className="flex items-center gap-1 text-[11px] font-label font-semibold
                       text-on-surface-variant/50 hover:text-on-surface transition-colors"
          >
            <span className="material-symbols-outlined text-[13px]">keyboard_tab</span>
            Dismiss
          </button>
        </div>
      </div>
    </>
  )
}
