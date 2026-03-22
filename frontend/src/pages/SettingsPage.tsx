import React, { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { APIKey, TrackedUser } from '../types'
import { formatDate } from '../utils/format'

// ── Helpers ───────────────────────────────────────────────────────────────────

const ROLES = ['admin', 'developer', 'viewer'] as const
type Role = typeof ROLES[number]

function roleBadge(role: string) {
  switch (role) {
    case 'admin':     return 'bg-primary/10 text-primary'
    case 'developer': return 'bg-secondary/10 text-secondary'
    default:          return 'bg-surface-container text-on-surface-variant'
  }
}

// ── Plaintext reveal modal ────────────────────────────────────────────────────

function PlaintextModal({ plaintext, onClose }: { plaintext: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(plaintext).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-2xl shadow-2xl border p-6 space-y-4"
        style={{
          background: 'rgb(var(--color-surface-container))',
          borderColor: 'rgb(var(--color-outline-variant) / 0.4)',
        }}
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-headline font-bold text-on-surface">API Key Created</h3>
            <p className="text-sm text-on-surface-variant mt-1">
              Copy this key now — it will <span className="text-error font-semibold">never be shown again</span>.
            </p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors">
            <span className="material-symbols-outlined text-[18px]">close</span>
          </button>
        </div>

        <div className="flex items-center gap-2 bg-surface-container-lowest rounded-xl px-4 py-3 border border-outline-variant/20">
          <code className="flex-1 text-sm font-mono text-on-surface break-all select-all">{plaintext}</code>
          <button
            onClick={copy}
            className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary/10 text-primary text-xs font-label font-semibold hover:bg-primary/20 transition-colors"
          >
            <span className="material-symbols-outlined text-[14px]">{copied ? 'check' : 'content_copy'}</span>
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>

        <p className="text-xs text-on-surface-variant">
          Use this key in the <code className="font-mono bg-surface-container-high px-1 rounded">Authorization: Bearer &lt;key&gt;</code> header.
        </p>

        <button
          onClick={onClose}
          className="w-full py-2.5 rounded-xl bg-primary text-on-primary text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all"
        >
          Done
        </button>
      </div>
    </div>
  )
}

// ── Create key form ───────────────────────────────────────────────────────────

function RoleDropdown({ value, onChange }: { value: Role; onChange: (r: Role) => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleOutside)
    return () => document.removeEventListener('mousedown', handleOutside)
  }, [open])

  const ROLE_COLOR: Record<Role, string> = {
    admin:     'text-primary',
    developer: 'text-secondary',
    viewer:    'text-on-surface-variant',
  }

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="flex items-center gap-2 bg-surface-container-low border border-outline-variant/30 rounded-lg px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/40 capitalize min-w-[120px] justify-between"
      >
        <span className={`font-semibold ${ROLE_COLOR[value]}`}>{value}</span>
        <span className="material-symbols-outlined text-[16px] text-on-surface-variant" style={{ transform: open ? 'rotate(180deg)' : undefined, transition: 'transform 0.15s' }}>expand_more</span>
      </button>
      {open && (
        <div
          className="absolute left-0 top-full mt-1 z-50 rounded-xl border shadow-lg py-1 min-w-full overflow-hidden"
          style={{
            background: 'rgb(var(--color-surface-container))',
            borderColor: 'rgb(var(--color-outline-variant) / 0.4)',
          }}
        >
          {ROLES.map(r => (
            <button
              key={r}
              type="button"
              onClick={() => { onChange(r); setOpen(false) }}
              className={`w-full flex items-center gap-2 px-4 py-2 text-sm capitalize transition-colors hover:bg-black/5 dark:hover:bg-white/5 ${
                r === value ? 'font-semibold' : 'font-normal text-on-surface-variant'
              } ${ROLE_COLOR[r]}`}
            >
              {r === value && <span className="material-symbols-outlined text-[14px]">check</span>}
              {r !== value && <span className="w-[14px]" />}
              {r}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function CreateKeyForm({ onCreated }: { onCreated: (plaintext: string) => void }) {
  const [name, setName] = useState('')
  const [role, setRole] = useState<Role>('developer')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setSubmitting(true)
    setError(null)
    try {
      const result = await api.createAPIKey(name.trim(), role)
      setName('')
      setRole('developer')
      onCreated(result.plaintext)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create API key')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-3 flex-wrap">
      <div className="flex-1 min-w-[180px]">
        <label className="block text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1.5">
          Key Name
        </label>
        <input
          type="text"
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="e.g. ci-pipeline"
          required
          className="w-full bg-surface-container-low border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/40 placeholder:text-on-surface-variant/40"
        />
      </div>
      <div>
        <label className="block text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1.5">
          Role
        </label>
        <RoleDropdown value={role} onChange={setRole} />
      </div>
      <button
        type="submit"
        disabled={submitting || !name.trim()}
        className="flex items-center gap-2 bg-primary text-on-primary px-4 py-2 rounded-lg text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
      >
        <span className="material-symbols-outlined text-[16px]">add</span>
        {submitting ? 'Creating…' : 'Create Key'}
      </button>
      {error && <p className="w-full text-xs text-error">{error}</p>}
    </form>
  )
}

// ── API Keys tab ──────────────────────────────────────────────────────────────

function APIKeysTab() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [plaintext, setPlaintext] = useState<string | null>(null)
  const [actionId, setActionId] = useState<string | null>(null)

  const fetchKeys = useCallback(() => {
    setLoading(true)
    api.listAPIKeys()
      .then(setKeys)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { fetchKeys() }, [fetchKeys])

  async function handleRevoke(id: string) {
    setActionId(id)
    try {
      await api.revokeAPIKey(id)
      setKeys(prev => prev.map(k => k.id === id ? { ...k, isActive: false } : k))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to revoke key')
    } finally {
      setActionId(null)
    }
  }

  async function handleDelete(id: string) {
    setActionId(id)
    try {
      await api.deleteAPIKey(id)
      setKeys(prev => prev.filter(k => k.id !== id))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete key')
    } finally {
      setActionId(null)
    }
  }

  function handleCreated(pt: string) {
    setPlaintext(pt)
    fetchKeys()
  }

  return (
    <div className="space-y-6">
      {/* Create form */}
      <div className="bg-surface-container-low rounded-xl p-5 border border-outline-variant/10">
        <h3 className="text-sm font-headline font-bold text-on-surface mb-4">Generate New Key</h3>
        <CreateKeyForm onCreated={handleCreated} />
      </div>

      {/* Keys list */}
      <div>
        <h3 className="text-sm font-headline font-bold text-on-surface mb-3">
          Active Keys
          {!loading && <span className="ml-2 text-on-surface-variant font-normal">({keys.filter(k => k.isActive).length})</span>}
        </h3>

        {error && (
          <p className="text-xs text-error bg-error/10 rounded-lg px-4 py-2 mb-3">{error}</p>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-12">
            <span className="text-on-surface-variant text-sm animate-pulse">Loading keys…</span>
          </div>
        ) : keys.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 bg-surface-container-low/30 rounded-xl border border-dashed border-outline-variant/20">
            <span className="material-symbols-outlined text-[36px] opacity-20 mb-3">key</span>
            <p className="text-sm font-headline font-bold text-on-surface">No API keys yet</p>
            <p className="text-xs text-on-surface-variant mt-1">Create a key above to get started.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {keys.map(k => (
              <div
                key={k.id}
                className={`flex items-center gap-4 px-4 py-3.5 rounded-xl border transition-all ${
                  k.isActive
                    ? 'bg-surface-container-low border-outline-variant/10'
                    : 'bg-surface-container-lowest border-outline-variant/5 opacity-50'
                }`}
              >
                {/* Status dot */}
                <span className={`w-2 h-2 rounded-full shrink-0 ${k.isActive ? 'bg-emerald-500' : 'bg-on-surface-variant/30'}`} />

                {/* Key info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-headline font-semibold text-on-surface">{k.name}</span>
                    <span className={`text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full ${roleBadge(k.role)}`}>
                      {k.role}
                    </span>
                    {!k.isActive && (
                      <span className="text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-error/10 text-error">
                        Revoked
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-4 mt-0.5 text-[11px] text-on-surface-variant font-label flex-wrap">
                    <span>Created by {k.createdBy}</span>
                    <span className="opacity-50">·</span>
                    <span>{formatDate(k.createdAt)}</span>
                    {k.lastUsedAt && (
                      <>
                        <span className="opacity-50">·</span>
                        <span>Last used {formatDate(k.lastUsedAt)}</span>
                      </>
                    )}
                  </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 shrink-0">
                  {k.isActive && (
                    <button
                      onClick={() => handleRevoke(k.id)}
                      disabled={actionId === k.id}
                      className="px-3 py-1.5 rounded-lg text-xs font-label font-semibold text-on-surface-variant border border-outline-variant/20 hover:border-error/30 hover:text-error hover:bg-error/5 transition-colors disabled:opacity-50"
                    >
                      {actionId === k.id ? 'Revoking…' : 'Revoke'}
                    </button>
                  )}
                  <button
                    onClick={() => handleDelete(k.id)}
                    disabled={actionId === k.id}
                    className="p-1.5 rounded-lg text-on-surface-variant/40 hover:text-error hover:bg-error/10 transition-colors disabled:opacity-50"
                    title="Delete permanently"
                    aria-label={`Delete key ${k.name}`}
                  >
                    <span className="material-symbols-outlined text-[16px]">delete</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {plaintext && <PlaintextModal plaintext={plaintext} onClose={() => setPlaintext(null)} />}
    </div>
  )
}

// ── Users tab ─────────────────────────────────────────────────────────────────

function UsersTab() {
  const [users, setUsers] = useState<TrackedUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.listUsers()
      .then(setUsers)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div>
      {error && (
        <p className="text-xs text-error bg-error/10 rounded-lg px-4 py-2 mb-4">{error}</p>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <span className="text-on-surface-variant text-sm animate-pulse">Loading users…</span>
        </div>
      ) : users.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 bg-surface-container-low/30 rounded-xl border border-dashed border-outline-variant/20">
          <span className="material-symbols-outlined text-[36px] opacity-20 mb-3">group</span>
          <p className="text-sm font-headline font-bold text-on-surface">No users yet</p>
          <p className="text-xs text-on-surface-variant mt-1">Users appear here after their first login.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {users.map(u => (
            <div
              key={u.email}
              className="flex items-center gap-4 px-4 py-3.5 rounded-xl bg-surface-container-low border border-outline-variant/10"
            >
              {/* Avatar */}
              {u.avatarUrl ? (
                <img src={u.avatarUrl} alt={u.name} className="w-9 h-9 rounded-full shrink-0" referrerPolicy="no-referrer" />
              ) : (
                <span className="w-9 h-9 rounded-full bg-primary/20 flex items-center justify-center text-primary text-sm font-bold font-headline shrink-0">
                  {u.name?.[0]?.toUpperCase() ?? u.email[0].toUpperCase()}
                </span>
              )}

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-sm font-headline font-semibold text-on-surface">{u.name || u.email}</span>
                  <span className={`text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full ${roleBadge(u.role)}`}>
                    {u.role}
                  </span>
                  <span className="text-[10px] font-label text-on-surface-variant/60 bg-surface-container px-1.5 py-0.5 rounded capitalize">
                    {u.provider}
                  </span>
                </div>
                <p className="text-[11px] text-on-surface-variant font-label mt-0.5 truncate lowercase">{u.email}</p>
              </div>

              {/* Timestamps */}
              <div className="text-right shrink-0 hidden md:block">
                <p className="text-[11px] text-on-surface-variant font-label">Last login: {formatDate(u.lastLoginAt)}</p>
                <p className="text-[10px] text-on-surface-variant/60 font-label mt-0.5">First: {formatDate(u.firstLoginAt)}</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

type Tab = 'apikeys' | 'users'

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('apikeys')

  const TABS: { key: Tab; label: string; icon: string }[] = [
    { key: 'apikeys', label: 'API Keys', icon: 'key' },
    { key: 'users',   label: 'Users',    icon: 'group' },
  ]

  return (
    <div className="flex flex-col h-full -mx-8 -my-6">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-background px-8 pt-6 pb-4 border-b border-outline-variant/15">
        <div className="mb-5">
          <h2 className="text-4xl font-bold font-headline tracking-tight text-on-surface">Settings</h2>
          <p className="text-on-surface-variant font-body mt-2">
            Manage API keys and view users who have accessed this system.
          </p>
        </div>

        {/* Tab bar */}
        <div className="flex items-center gap-1 bg-surface-container-low p-1 rounded-lg w-fit">
          {TABS.map(({ key, label, icon }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={`flex items-center gap-2 px-4 py-1.5 text-xs font-label font-bold rounded-md transition-colors ${
                activeTab === key
                  ? 'bg-surface-container-highest text-on-surface shadow-sm'
                  : 'text-on-surface-variant hover:text-on-surface'
              }`}
            >
              <span className="material-symbols-outlined text-[14px]">{icon}</span>
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto px-8 py-6">
        {activeTab === 'apikeys' ? <APIKeysTab /> : <UsersTab />}
      </div>
    </div>
  )
}
