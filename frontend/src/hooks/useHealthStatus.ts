import { useEffect, useState } from 'react'

export type HealthStatus = 'checking' | 'ok' | 'degraded' | 'unreachable'

export interface HealthInfo {
  status: HealthStatus
  uptime?: string
  db?: string
}

const POLL_MS = 30_000

export function useHealthStatus(): HealthInfo {
  const [info, setInfo] = useState<HealthInfo>({ status: 'checking' })

  useEffect(() => {
    let cancelled = false

    async function poll() {
      try {
        const res = await fetch('/api/healthz', { cache: 'no-store' })
        if (cancelled) return
        const raw: unknown = await res.json()
        const data = (raw !== null && typeof raw === 'object') ? raw as Record<string, unknown> : {}
        setInfo({
          status: res.ok ? (data.status === 'ok' ? 'ok' : 'degraded') : 'degraded',
          uptime: typeof data.uptime === 'string' ? data.uptime : undefined,
          db: typeof data.db === 'string' ? data.db : undefined,
        })
      } catch {
        if (!cancelled) setInfo(prev => ({ ...prev, status: 'unreachable' }))
      }
    }

    poll()
    const id = setInterval(poll, POLL_MS)
    return () => { cancelled = true; clearInterval(id) }
  }, [])

  return info
}
