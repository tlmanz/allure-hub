import type { Environment, Project, Report, ReportStats, PagedReports, PagedAPIKeys, PagedUsers, UploadSession, APIKey, RetentionSettings, CleanupRun, OverviewStats, DiskUsage, NotificationItem } from '../types'

const BASE = '/api'

const STATE_CHANGING = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? 'GET').toUpperCase()
  const headers = new Headers(init?.headers)
  if (STATE_CHANGING.has(method)) {
    headers.set('X-Requested-With', 'XMLHttpRequest')
  }
  const res = await fetch(`${BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
    throw new Error('Unauthenticated')
  }
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body.trim() || `Request failed (${res.status})`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

// Encode a single URL path segment so slashes / dots / special chars can't
// alter the request path (M-15).
const enc = encodeURIComponent

const CHUNK_SIZE = 5 * 1024 * 1024 // 5 MB per chunk

export const api = {
  // Environments
  listEnvironments: (signal?: AbortSignal) =>
    request<Environment[]>('/environments', { signal }).then(d => d ?? []),

  createEnvironment: (id: string, name: string, icon: string) =>
    request<Environment>('/environments', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, name, icon }),
    }),

  updateEnvironment: (id: string, name: string, icon: string) =>
    request<Environment>(`/environments/${enc(id)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, icon }),
    }),

  deleteEnvironment: (id: string) =>
    request<void>(`/environments/${enc(id)}`, { method: 'DELETE' }),

  // Projects (scoped by environment)
  listProjects: (envId: string, signal?: AbortSignal) =>
    request<Project[]>(`/environments/${enc(envId)}/projects`, { signal }).then(d => d ?? []),

  createProject: (envId: string, id: string, name: string) =>
    request<Project>(`/environments/${enc(envId)}/projects`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, name }),
    }),

  deleteProject: (envId: string, id: string) =>
    request<void>(`/environments/${enc(envId)}/projects/${enc(id)}`, { method: 'DELETE' }),

  // Reports (scoped by environment + project)
  listReports: (envId: string, projectId: string) =>
    request<Report[]>(`/environments/${enc(envId)}/projects/${enc(projectId)}/reports`).then(d => d ?? []),

  listReportsPaged: (envId: string, projectId: string, limit: number, offset: number, filter = 'all', signal?: AbortSignal) =>
    request<PagedReports>(`/environments/${enc(envId)}/projects/${enc(projectId)}/reports?limit=${limit}&offset=${offset}&filter=${enc(filter)}`, { signal }),

  getReportStats: (envId: string, projectId: string, signal?: AbortSignal) =>
    request<ReportStats>(`/environments/${enc(envId)}/projects/${enc(projectId)}/reports/stats`, { signal }),

  deleteReport: (envId: string, projectId: string, buildId: string) =>
    request<void>(`/environments/${enc(envId)}/projects/${enc(projectId)}/reports/${enc(buildId)}`, { method: 'DELETE' }),

  generateReport: (envId: string, projectId: string, buildId: string, asyncGenerate = false) =>
    request<{ reportUrl: string; status?: string }>(
      `/environments/${enc(envId)}/projects/${enc(projectId)}/reports${asyncGenerate ? '?async=true' : ''}`,
      {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ buildId }),
      },
    ),

  retryAssembly: (envId: string, projectId: string, uploadId: string) =>
    request<{ status: string }>(
      `/environments/${enc(envId)}/projects/${enc(projectId)}/uploads/${enc(uploadId)}/complete`,
      { method: 'POST' },
    ),

  // Upload sessions
  listUploadSessions: () =>
    request<UploadSession[]>('/uploads').then(d => d ?? []),

  deleteUploadSession: (id: string) =>
    request<void>(`/uploads/${enc(id)}`, { method: 'DELETE' }),

  // Notifications
  listNotifications: (limit = 50) =>
    request<NotificationItem[]>(`/notifications?limit=${limit}`).then(d => d ?? []),

  getUnreadNotificationCount: () =>
    request<{ count: number }>('/notifications/unread').then(d => d?.count ?? 0),

  markNotificationRead: (id: string) =>
    request<void>(`/notifications/${enc(id)}/read`, { method: 'PATCH' }),

  markAllNotificationsRead: () =>
    request<void>('/notifications/read', { method: 'POST' }),

  // Settings — API keys
  listAPIKeys: (search = '', offset = 0) =>
    request<PagedAPIKeys>(`/settings/apikeys?search=${enc(search)}&offset=${offset}`),

  createAPIKey: (name: string, role: string) =>
    request<{ key: APIKey; plaintext: string }>('/settings/apikeys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, role }),
    }),

  revokeAPIKey: (id: string) =>
    request<void>(`/settings/apikeys/${enc(id)}`, { method: 'DELETE' }),

  deleteAPIKey: (id: string) =>
    request<void>(`/settings/apikeys/${enc(id)}?action=delete`, { method: 'DELETE' }),

  // Settings — users
  listUsers: (search = '', offset = 0) =>
    request<PagedUsers>(`/settings/users?search=${enc(search)}&offset=${offset}`),

  setUserRole: (email: string, role: string) =>
    request<void>(`/settings/users/${enc(email)}/role`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role }),
    }),

  resetUserRole: (email: string) =>
    request<void>(`/settings/users/${enc(email)}/role`, { method: 'DELETE' }),

  // Overview analytics
  getOverviewStats: (signal?: AbortSignal) =>
    request<OverviewStats>('/overview', { signal }),

  // Settings — data retention
  getRetentionSettings: () =>
    request<RetentionSettings>('/settings/retention'),

  getCleanupRuns: (limit = 5) =>
    request<CleanupRun[]>(`/settings/retention/runs?limit=${limit}`).then(d => d ?? []),

  setRetentionSettings: (settings: RetentionSettings) =>
    request<void>('/settings/retention', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(settings),
    }),

  // Settings — Allure CLI version
  getAllureVersion: () =>
    request<{ version: string; latest: string }>('/settings/allure'),

  updateAllureVersion: (version: string) =>
    request<{ version: string }>('/settings/allure', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version }),
    }),

  // Settings — disk usage
  getDiskUsage: () =>
    request<DiskUsage>('/settings/disk'),

  getDiskNotificationThreshold: () =>
    request<{ thresholdPercent: number }>('/settings/disk/notification-threshold'),

  setDiskNotificationThreshold: (thresholdPercent: number) =>
    request<void>('/settings/disk/notification-threshold', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ thresholdPercent }),
    }),

  // Chunked upload — drives Init → Chunks → Complete → Generate.
  // Pass an AbortSignal to cancel mid-flight (M-27).
  uploadResults: async (
    envId: string,
    projectId: string,
    buildId: string,
    file: File,
    onProgress: (receivedChunks: number, totalChunks: number) => void,
    signal?: AbortSignal,
    reportConfig?: Record<string, unknown>,
  ): Promise<void> => {
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
    const uploadsBase = `${BASE}/environments/${enc(envId)}/projects/${enc(projectId)}/uploads`

    // 1. Init
    const { uploadId } = await request<{ uploadId: string }>(
      `/environments/${enc(envId)}/projects/${enc(projectId)}/uploads`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          buildId,
          fileName: file.name,
          totalSize: file.size,
          totalChunks,
        }),
        signal,
      },
    )

    // 2. Upload chunks
    for (let i = 0; i < totalChunks; i++) {
      signal?.throwIfAborted()
      const start = i * CHUNK_SIZE
      const chunk = file.slice(start, start + CHUNK_SIZE)
      await fetch(`${uploadsBase}/${enc(uploadId)}`, {
        method: 'PUT',
        headers: {
          'X-Chunk-Index': String(i),
          'X-Total-Chunks': String(totalChunks),
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: chunk,
        signal,
      }).then(r => { if (!r.ok) throw new Error(`chunk ${i} failed: ${r.status}`) })
      onProgress(i + 1, totalChunks)
    }

    // 3. Complete (assemble)
    await request<{ status: string }>(
      `/environments/${enc(envId)}/projects/${enc(projectId)}/uploads/${enc(uploadId)}/complete`,
      { method: 'POST', signal },
    )

    // 4. Generate report
    await request<{ reportUrl: string }>(
      `/environments/${enc(envId)}/projects/${enc(projectId)}/reports?async=true`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ buildId, ...(reportConfig ? { reportConfig } : {}) }),
        signal,
      },
    )
  },
}
