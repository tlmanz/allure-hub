import React, { useState } from 'react'
import NavBar from './NavBar'
import SupportModal from './SupportModal'
import { ToastStack } from './ui'
import { useUI } from '../context/UIContext'
import { APP_NAME } from '../design-system/tokens'
import { useVersion } from '../hooks/useVersion'

const GITHUB_URL = 'https://github.com/tlmanz/allure-hub'

function formatBuildTime(iso: string): string {
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
}

const Footer: React.FC = () => {
  const version = useVersion()
  const [supportOpen, setSupportOpen] = useState(false)

  return (
    <>
      <footer
        className="border-t px-8 py-2 flex items-center justify-between relative"
        style={{
          background: 'rgb(var(--color-surface-container-lowest))',
          borderColor: 'rgb(var(--color-outline-variant) / 0.4)',
        }}
      >
        {/* Left: app name + build info */}
        <span className="flex items-center gap-1.5 text-[11px] text-on-surface-variant font-label">
          <span className="font-bold text-on-surface">{APP_NAME}</span>
          {version && (
            <>
              <span className="opacity-40">·</span>
              <span className="font-medium text-on-surface">{version.version}</span>
              <span className="opacity-40">·</span>
              <span>Built {formatBuildTime(version.buildTime)}</span>
              <span className="opacity-40">·</span>
              <span>{version.goVersion}</span>
            </>
          )}
        </span>

        {/* Center: credit */}
        <span className="absolute left-1/2 -translate-x-1/2 text-[12px] text-on-surface-variant font-label">
          Made with <span className="text-error">♥</span> by tlmanz
        </span>

        {/* Right: links + support */}
        <div className="flex items-center gap-3 text-[12px] text-on-surface-variant font-label">
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-on-surface transition-colors flex items-center gap-1.5"
          >
            <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 fill-current" aria-hidden="true">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
            </svg>
            GitHub
          </a>
          <span className="opacity-30">·</span>
          <span>MIT License</span>
          <span className="opacity-30">·</span>
          <button
            onClick={() => setSupportOpen(true)}
            className="flex items-center gap-1 hover:text-on-surface transition-colors group"
            aria-label="Open support"
          >
            <span className="material-symbols-outlined text-[13px] group-hover:text-primary transition-colors" aria-hidden="true">
              help_outline
            </span>
            <span className="group-hover:text-primary transition-colors">Support</span>
          </button>
        </div>
      </footer>

      <SupportModal isOpen={supportOpen} onClose={() => setSupportOpen(false)} />
    </>
  )
}

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { toasts, removeToast } = useUI()

  return (
    <div className="h-screen bg-background text-on-surface font-body flex flex-col overflow-hidden">
      <NavBar />
      <main className="flex-1 overflow-y-auto px-8 py-6">{children}</main>
      <Footer />
      <ToastStack toasts={toasts} onCancel={removeToast} />
    </div>
  )
}

export default Layout
