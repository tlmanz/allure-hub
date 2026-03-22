import { useEffect, useRef } from 'react'
import { useFocusTrap } from '../hooks/useFocusTrap'

const GITHUB_URL = 'https://github.com/tlmanz/allure-hub'

const QUICK_LINKS = [
  { icon: 'star', label: 'Star', sub: 'Show support', href: GITHUB_URL },
  { icon: 'fork_right', label: 'Fork', sub: 'Contribute', href: `${GITHUB_URL}/fork` },
  { icon: 'manage_search', label: 'Issues', sub: 'Browse open', href: `${GITHUB_URL}/issues` },
  { icon: 'call_merge', label: 'Pull Requests', sub: 'Open PRs', href: `${GITHUB_URL}/pulls` },
]

const ACTION_CARDS = [
  {
    icon: 'bug_report',
    label: 'Report a Bug',
    description: 'Found something broken? Open an issue with steps to reproduce.',
    href: `${GITHUB_URL}/issues/new?template=bug_report.md`,
    accent: 'text-error',
    bg: 'bg-error/8 hover:bg-error/12',
    border: 'border-error/20 hover:border-error/40',
    chip: 'Issues',
    chipColor: 'bg-error/10 text-error',
  },
  {
    icon: 'lightbulb',
    label: 'Request a Feature',
    description: 'Have an idea that would make Allure Hub better? Share it.',
    href: `${GITHUB_URL}/issues/new?template=feature_request.md`,
    accent: 'text-tertiary',
    bg: 'bg-tertiary/8 hover:bg-tertiary/12',
    border: 'border-tertiary/20 hover:border-tertiary/40',
    chip: 'Ideas',
    chipColor: 'bg-tertiary/10 text-tertiary',
  },
  {
    icon: 'forum',
    label: 'Community',
    description: 'Ask questions, share tips, or show off how you use Allure Hub.',
    href: `${GITHUB_URL}/discussions`,
    accent: 'text-secondary',
    bg: 'bg-secondary/8 hover:bg-secondary/12',
    border: 'border-secondary/20 hover:border-secondary/40',
    chip: 'Discussions',
    chipColor: 'bg-secondary/10 text-secondary',
  },
  {
    icon: 'menu_book',
    label: 'Documentation',
    description: 'Setup guides, API reference, and configuration options.',
    href: `${GITHUB_URL}#readme`,
    accent: 'text-primary',
    bg: 'bg-primary/8 hover:bg-primary/12',
    border: 'border-primary/20 hover:border-primary/40',
    chip: 'Docs',
    chipColor: 'bg-primary/10 text-primary',
  },
  {
    icon: 'merge',
    label: 'Contribute',
    description: 'Browse open issues, submit a pull request, or improve docs.',
    href: `${GITHUB_URL}/contribute`,
    accent: 'text-on-surface',
    bg: 'bg-surface-container hover:bg-surface-container-high',
    border: 'border-outline-variant/20 hover:border-outline-variant/50',
    chip: 'PRs welcome',
    chipColor: 'bg-surface-container-high text-on-surface-variant',
  },
  {
    icon: 'new_releases',
    label: 'Releases',
    description: "See what's changed in each version and track the roadmap.",
    href: `${GITHUB_URL}/releases`,
    accent: 'text-on-surface',
    bg: 'bg-surface-container hover:bg-surface-container-high',
    border: 'border-outline-variant/20 hover:border-outline-variant/50',
    chip: 'Changelog',
    chipColor: 'bg-surface-container-high text-on-surface-variant',
  },
]

interface Props {
  isOpen: boolean
  onClose: () => void
}

export default function SupportModal({ isOpen, onClose }: Props) {
  const dialogRef = useRef<HTMLDivElement>(null)
  useFocusTrap(dialogRef, isOpen)

  useEffect(() => {
    if (!isOpen) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div
      ref={dialogRef}
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="support-modal-title"
      tabIndex={-1}
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Panel */}
      <div className="relative w-full max-w-2xl rounded-2xl border border-outline-variant/20 shadow-2xl overflow-hidden"
        style={{ background: 'rgb(var(--color-surface-container-low))' }}
      >
        {/* Header */}
        <div
          className="relative flex items-center gap-4 px-6 py-5 flex-shrink-0 overflow-hidden"
          style={{ background: 'rgb(var(--color-surface-container))' }}
        >
          {/* Decorative background glow */}
          <div className="absolute -top-8 -right-8 w-40 h-40 rounded-full bg-primary/8 blur-2xl pointer-events-none" />
          <div className="absolute -bottom-8 -left-4 w-32 h-32 rounded-full bg-secondary/8 blur-2xl pointer-events-none" />

          {/* GitHub icon */}
          <div className="relative w-11 h-11 rounded-xl flex items-center justify-center shrink-0"
            style={{ background: 'rgb(var(--color-on-surface) / 0.08)' }}
          >
            <svg viewBox="0 0 24 24" className="w-6 h-6 fill-on-surface" aria-hidden="true">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
            </svg>
          </div>

          <div className="relative flex-1 min-w-0">
            <p className="text-[10px] font-label font-bold uppercase tracking-[0.18em] text-on-surface-variant mb-0.5">
              Open Source · MIT License
            </p>
            <h2 id="support-modal-title" className="text-xl font-headline font-bold text-on-surface leading-tight">
              Support &amp; Community
            </h2>
          </div>

          <button
            onClick={onClose}
            aria-label="Close support"
            className="relative w-8 h-8 flex items-center justify-center rounded-full text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors"
          >
            <span className="material-symbols-outlined text-[20px]" aria-hidden="true">close</span>
          </button>
        </div>

        {/* Scrollable body */}
        <div className="px-6 py-5 flex flex-col gap-5">
          {/* Repo link */}
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="group flex items-center gap-4 rounded-xl border border-outline-variant/20 px-4 py-3.5
                       hover:border-primary/30 transition-all duration-200"
            style={{ background: 'rgb(var(--color-surface-container))' }}
          >
            <div className="flex-1 min-w-0">
              <p className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-0.5">Repository</p>
              <p className="text-sm font-headline font-bold text-on-surface group-hover:text-primary transition-colors">
                github.com/tlmanz/allure-hub
              </p>
            </div>
            <span className="material-symbols-outlined text-[18px] text-on-surface-variant group-hover:text-primary group-hover:translate-x-0.5 transition-all shrink-0">
              arrow_forward
            </span>
          </a>

          {/* Quick links */}
          <div className="grid grid-cols-4 gap-2">
            {QUICK_LINKS.map(({ icon, label, sub, href }) => (
              <a
                key={label}
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="group flex flex-col items-center gap-2 rounded-xl border border-outline-variant/20 px-3 py-3
                           hover:border-primary/30 hover:bg-primary/5 transition-all duration-200 text-center"
                style={{ background: 'rgb(var(--color-surface-container))' }}
              >
                <span className="material-symbols-outlined text-[22px] text-on-surface-variant group-hover:text-primary transition-colors">
                  {icon}
                </span>
                <div>
                  <p className="text-xs font-headline font-bold text-on-surface group-hover:text-primary transition-colors leading-tight">
                    {label}
                  </p>
                  <p className="text-[10px] text-on-surface-variant mt-0.5">{sub}</p>
                </div>
              </a>
            ))}
          </div>

          {/* Divider */}
          <div className="flex items-center gap-3">
            <div className="flex-1 h-px" style={{ background: 'rgb(var(--color-outline-variant) / 0.3)' }} />
            <span className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant">Get involved</span>
            <div className="flex-1 h-px" style={{ background: 'rgb(var(--color-outline-variant) / 0.3)' }} />
          </div>

          {/* Action cards */}
          <div className="grid grid-cols-2 gap-2.5">
            {ACTION_CARDS.map(({ icon, label, description, href, accent, bg, border, chip, chipColor }) => (
              <a
                key={label}
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className={`group flex flex-col gap-2.5 rounded-xl border p-3.5 transition-all duration-200 ${bg} ${border}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
                    style={{ background: 'rgb(var(--color-surface-container-low))' }}
                  >
                    <span className={`material-symbols-outlined text-[18px] ${accent}`}>{icon}</span>
                  </div>
                  <span className={`text-[9px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full shrink-0 ${chipColor}`}>
                    {chip}
                  </span>
                </div>
                <div>
                  <p className={`font-headline font-bold text-xs ${accent} mb-0.5`}>{label}</p>
                  <p className="text-[11px] text-on-surface-variant leading-relaxed">{description}</p>
                </div>
                <div className="mt-auto flex items-center gap-1 text-on-surface-variant/60 group-hover:text-on-surface-variant transition-colors">
                  <span className="text-[10px] font-label font-semibold">Open on GitHub</span>
                  <span className="material-symbols-outlined text-[12px] group-hover:translate-x-0.5 transition-transform">
                    arrow_forward
                  </span>
                </div>
              </a>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
