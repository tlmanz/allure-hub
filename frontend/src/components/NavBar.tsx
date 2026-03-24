import React, { useEffect, useMemo, useRef, useState } from 'react'
import { NavLink, Link, useLocation } from 'react-router-dom'
import { useTheme } from '../context/ThemeContext'
import { useUpload } from '../context/UploadContext'
import { useHealthStatus } from '../hooks/useHealthStatus'
import { useAuth } from '../context/AuthContext'

interface NavItem {
  label: string
  to: string
  matchPrefix?: string
}

const NAV_ITEMS: NavItem[] = [
  { label: 'Environments', to: '/', matchPrefix: '/environments' },
  { label: 'Uploads', to: '/uploads' },
  { label: 'Settings', to: '/settings' },
]

const ACTIVE_CLS = 'px-3.5 py-1.5 rounded-lg text-[15px] font-semibold font-headline bg-primary/10 text-primary border border-primary/20 transition-colors'
const INACTIVE_CLS = 'px-3.5 py-1.5 rounded-lg text-[15px] font-semibold font-headline text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors'

function useClock() {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])
  return now
}

const NavBar: React.FC = React.memo(() => {
  const { theme, toggleTheme } = useTheme()
  const { pathname } = useLocation()
  const { sessions, drawerOpen, openDrawer, closeDrawer } = useUpload()
  const health = useHealthStatus()
  const { user, logout, can } = useAuth()
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const userMenuRef = useRef<HTMLDivElement>(null)
  const now = useClock()

  useEffect(() => {
    if (!userMenuOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [userMenuOpen])

  // Memoize all pathname-derived values so regex/filter work only reruns when
  // pathname or sessions actually change (M-20).
  const currentProjectId = useMemo(() => {
    const m = pathname.match(/^\/environments\/[^/]+\/projects\/([^/]+)/)
    return m?.[1] ?? null
  }, [pathname])

  const activeCount = useMemo(
    () => sessions.filter(
      s => s.projectId === currentProjectId &&
        (s.phase === 'uploading' || s.phase === 'assembling' || s.phase === 'generating'),
    ).length,
    [sessions, currentProjectId],
  )

  return (
    <nav
      className="sticky top-0 z-30 h-16 flex items-center justify-between px-6 flex-shrink-0"
      style={{
        background: 'rgb(var(--color-surface-container-lowest))',
        borderBottom: '1px solid rgb(var(--color-outline-variant) / 0.4)',
      }}
    >
      {/* Left: brand + nav links */}
      <div className="flex items-center gap-8">
        <Link to="/" className="flex items-center gap-2 select-none">
          <img src="/images/logo.png" alt="" className="h-7" />
          <span className="text-xl font-black tracking-tighter font-headline leading-none">
            <span className="text-gray-900 dark:text-white">Allure</span>
            <span className="text-gray-500 dark:text-gray-400">Hub</span>
          </span>
        </Link>

        <div className="flex items-center gap-1.5">
          {NAV_ITEMS.map(({ label, to, matchPrefix }) => {
            if (label === 'Settings' && !can('manage')) return null
            const isActive =
              pathname === to ||
              (matchPrefix !== undefined && pathname.startsWith(matchPrefix))
            return (
              <NavLink
                key={to}
                to={to}
                className={isActive ? ACTIVE_CLS : INACTIVE_CLS}
              >
                {label === 'Uploads' && activeCount > 0 ? (
                  <span className="flex items-center gap-1.5">
                    Uploads
                    <span className="flex items-center gap-1 text-[10px] font-bold text-primary">
                      <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                      {activeCount}
                    </span>
                  </span>
                ) : label}
              </NavLink>
            )
          })}
        </div>
      </div>

      {/* Right: clock + upload activity toggle + theme toggle + create button */}
      <div className="flex items-center gap-2">
        {/* Live clock */}
        <div className="flex flex-col items-end mr-1 select-none">
          <span className="text-[16px] font-headline font-bold text-on-surface tabular-nums leading-tight">
            {now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </span>
          <span className="text-[11px] font-label text-on-surface-variant leading-tight">
            {now.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })}
          </span>
        </div>
        {/* Health status indicator */}
        <div
          title={
            health.status === 'checking' ? 'Checking server…'
            : health.status === 'ok' ? `Server healthy · uptime ${health.uptime}`
            : health.status === 'degraded' ? `Degraded · DB: ${health.db}`
            : 'Server unreachable'
          }
          className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[12px] font-label font-semibold border transition-colors"
          style={{
            background: health.status === 'ok' ? 'rgb(var(--md-sys-color-tertiary, 0 137 123) / 0.08)'
              : health.status === 'checking' ? 'rgb(var(--color-surface-container-high, 200 200 200) / 0.5)'
              : 'rgb(var(--md-sys-color-error, 176 0 32) / 0.08)',
            borderColor: health.status === 'ok' ? 'rgb(var(--md-sys-color-tertiary, 0 137 123) / 0.25)'
              : health.status === 'checking' ? 'rgb(var(--color-outline-variant, 150 150 150) / 0.3)'
              : 'rgb(var(--md-sys-color-error, 176 0 32) / 0.25)',
            color: health.status === 'ok' ? 'rgb(var(--md-sys-color-tertiary, 0 137 123))'
              : health.status === 'checking' ? 'rgb(var(--color-on-surface-variant, 100 100 100))'
              : 'rgb(var(--md-sys-color-error, 176 0 32))',
          }}
        >
          <span className={`w-1.5 h-1.5 rounded-full ${
            health.status === 'ok'        ? 'bg-emerald-500' :
            health.status === 'checking'  ? 'bg-on-surface-variant animate-pulse' :
                                            'bg-error animate-pulse'
          }`} />
          {health.status === 'ok' ? 'Online' : health.status === 'checking' ? 'Checking' : 'Offline'}
        </div>
        {/* Activity toggle — only visible on project detail pages */}
        {currentProjectId && (
          <button
            onClick={() => drawerOpen ? closeDrawer() : openDrawer()}
            className={`relative flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[15px] font-medium font-headline transition-colors
              ${drawerOpen
                ? 'bg-primary/10 text-primary border border-primary/20'
                : 'text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            aria-label="Toggle upload activity"
            title="Upload activity"
          >
            <span className="material-symbols-outlined text-[20px]">upload_file</span>
            Activity
            {activeCount > 0 && (
              <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
            )}
          </button>
        )}

        <button
          onClick={toggleTheme}
          className="p-2 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
        >
          <span className="material-symbols-outlined text-[20px]">
            {theme === 'dark' ? 'light_mode' : 'dark_mode'}
          </span>
        </button>

        {/* User menu */}
        {user && (
          <div className="relative" ref={userMenuRef}>
            <button
              onClick={() => setUserMenuOpen(o => !o)}
              className="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
              aria-label="User menu"
            >
              {user.avatarUrl ? (
                <img src={user.avatarUrl} alt={user.name} className="w-8 h-8 rounded-full" referrerPolicy="no-referrer" />
              ) : (
                <span className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-primary text-sm font-bold font-headline">
                  {user.name?.[0]?.toUpperCase() ?? user.email[0].toUpperCase()}
                </span>
              )}
              <span className="text-[15px] font-medium font-headline text-on-surface hidden sm:block">{user.name || user.email}</span>
              <span className="material-symbols-outlined text-[15px] text-on-surface-variant">expand_more</span>
            </button>

            {userMenuOpen && (
              <div
                className="absolute right-0 top-full mt-1 w-52 rounded-xl border shadow-lg z-50 py-1 overflow-hidden"
                style={{
                  background: 'rgb(var(--color-surface-container))',
                  borderColor: 'rgb(var(--color-outline-variant) / 0.4)',
                }}
              >
                <div className="px-3 py-2 border-b" style={{ borderColor: 'rgb(var(--color-outline-variant) / 0.3)' }}>
                  <p className="text-xs font-semibold text-on-surface truncate">{user.name}</p>
                  <p className="text-[11px] text-on-surface-variant truncate">{user.email}</p>
                  <span className="inline-block mt-1 text-[10px] font-bold px-1.5 py-0.5 rounded bg-primary/10 text-primary capitalize">{user.role}</span>
                </div>
                <Link
                  to="/profile"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
                >
                  <span className="material-symbols-outlined text-[16px]">manage_accounts</span>
                  View profile
                </Link>
                <div className="my-1 border-t" style={{ borderColor: 'rgb(var(--color-outline-variant) / 0.3)' }} />
                <button
                  onClick={() => { setUserMenuOpen(false); logout() }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
                >
                  <span className="material-symbols-outlined text-[16px]">logout</span>
                  Sign out
                </button>
              </div>
            )}
          </div>
        )}

      </div>
    </nav>
  )
})

export default NavBar
