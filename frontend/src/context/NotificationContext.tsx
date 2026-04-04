import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import { useSnackbar, type SnackbarVariant } from '../components/ui/Snackbar'
import type { NotificationItem, NotificationSeverity } from '../types'

export interface AppNotification {
  id: string
  message: string
  variant: SnackbarVariant
  timestamp: Date
  read: boolean
}

interface NotificationContextValue {
  notifications: AppNotification[]
  unseenCount: number
  markAsRead: (id: string) => void
  clearUnseen: () => void
}

const NotificationContext = createContext<NotificationContextValue | null>(null)

const MAX_STORED = 50

function severityToVariant(severity: NotificationSeverity): SnackbarVariant {
  switch (severity) {
    case 'success':
      return 'success'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
    case 'info':
    default:
      return 'info'
  }
}

function fromServerNotification(n: NotificationItem): AppNotification {
  return {
    id: n.id,
    message: n.body ? `${n.title} - ${n.body}` : n.title,
    variant: severityToVariant(n.severity),
    timestamp: new Date(n.created_at),
    read: n.read,
  }
}

export function NotificationProvider({ children }: { children: React.ReactNode }) {
  const { show, SnackbarNode } = useSnackbar()
  const [notifications, setNotifications] = useState<AppNotification[]>([])
  const [unseenCount, setUnseenCount] = useState(0)
  const esRef = useRef<EventSource | null>(null)

  const upsert = useCallback((entry: AppNotification) => {
    setNotifications(prev => {
      const idx = prev.findIndex(n => n.id === entry.id)
      if (idx === -1) {
        return [entry, ...prev]
          .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
          .slice(0, MAX_STORED)
      }
      const next = [...prev]
      next[idx] = entry
      return next
        .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
        .slice(0, MAX_STORED)
    })
  }, [])

  const syncFromServer = useCallback(() => {
    api.listNotifications(MAX_STORED)
      .then(items => {
        setNotifications(items.map(fromServerNotification))
      })
      .catch(() => {})
    api.getUnreadNotificationCount()
      .then(count => setUnseenCount(count))
      .catch(() => {})
  }, [])

  useEffect(() => {
    syncFromServer()

    let retryDelay = 1000
    let retryTimer: ReturnType<typeof setTimeout> | null = null
    let stopped = false

    function connect() {
      const es = new EventSource('/api/notifications/stream')
      esRef.current = es

      es.addEventListener('notification', (e: MessageEvent) => {
        try {
          const notif: NotificationItem = JSON.parse(e.data)
          const entry = fromServerNotification(notif)
          show(entry.message, entry.variant)
          upsert(entry)
          if (!notif.read) {
            setUnseenCount(n => n + 1)
          }
        } catch {}
      })

      es.addEventListener('open', () => {
        retryDelay = 1000
      })

      es.onerror = () => {
        es.close()
        esRef.current = null
        if (!stopped) {
          retryTimer = setTimeout(() => {
            retryDelay = Math.min(retryDelay * 2, 30_000)
            connect()
          }, retryDelay)
        }
      }
    }

    connect()

    return () => {
      stopped = true
      if (retryTimer !== null) clearTimeout(retryTimer)
      esRef.current?.close()
      esRef.current = null
    }
  }, [show, syncFromServer, upsert])

  const markAsRead = useCallback((id: string) => {
    const target = notifications.find(n => n.id === id)
    if (!target || target.read) return

    setNotifications(prev => prev.map(n => (
      n.id === id ? { ...n, read: true } : n
    )))
    setUnseenCount(prev => Math.max(0, prev - 1))

    api.markNotificationRead(id).catch(() => {
      syncFromServer()
    })
  }, [notifications, syncFromServer])

  const clearUnseen = useCallback(() => {
    if (unseenCount === 0) return
    setUnseenCount(0)
    setNotifications(prev => prev.map(n => ({ ...n, read: true })))
    api.markAllNotificationsRead().catch(() => {
      syncFromServer()
    })
  }, [syncFromServer, unseenCount])

  return (
    <NotificationContext.Provider value={{ notifications, unseenCount, markAsRead, clearUnseen }}>
      {children}
      {SnackbarNode}
    </NotificationContext.Provider>
  )
}

export function useNotification(): NotificationContextValue {
  const ctx = useContext(NotificationContext)
  if (!ctx) throw new Error('useNotification must be used inside <NotificationProvider>')
  return ctx
}
