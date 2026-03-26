export interface Environment {
  id: string
  name: string
  icon: string
  createdAt: string
  projectCount: number
}

export interface Project {
  id: string
  environmentId: string
  name: string
  createdAt: string
  buildCount: number
  lastStatus: 'passed' | 'failed' | 'inactive' | 'active'
  lastBuildAt?: string
  lastTotal: number
  lastPassed: number
  lastFailed: number
}

export interface Report {
  buildId: string
  createdAt: string
  reportUrl: string
  passed: number
  failed: number
  skipped: number
  total: number
  status: string
  configSnapshot: Record<string, unknown>
  generationWarnings?: string[]
  uploadedBy?: string
}

export interface ReportStats {
  totalRuns: number
  latestRate: number
  avgRate: number
  totalFailed: number
}

export interface PagedReports {
  builds: Report[]
  total: number
  limit: number
  offset: number
}

export type ProjectStatus = 'passed' | 'failed' | 'inactive' | 'running'

export type UploadPhase = 'uploading' | 'assembling' | 'generating' | 'done' | 'failed'

export interface UploadSession {
  id: string
  uploadId: string
  buildId: string
  projectId: string
  envId: string
  fileName: string
  totalSize: number
  totalChunks: number
  receivedChunks: number
  phase: UploadPhase
  failedAtPhase?: UploadPhase
  error?: string
  startedAt: string
  completedAt?: string
  reportUrl?: string
  passed: number
  failed: number
  skipped: number
  total: number
  uploadedBy?: string
}

export interface APIKey {
  id: string
  name: string
  createdBy: string
  role: string
  lastUsedAt?: string
  createdAt: string
  expiresAt?: string
  isActive: boolean
}

export interface TrackedUser {
  email: string
  name: string
  avatarUrl: string
  provider: string
  role: string
  firstLoginAt: string
  lastLoginAt: string
}

export interface PagedAPIKeys {
  keys: APIKey[]
  total: number
  limit: number
  offset: number
}

export interface PagedUsers {
  users: TrackedUser[]
  total: number
  limit: number
  offset: number
}

export interface RetentionSettings {
  retentionDays: number
  intervalHours: number
  dryRun: boolean
}

export interface CleanupRun {
  id: string
  startedAt: string
  finishedAt: string
  status: 'success' | 'failed'
  deletedCount: number
  skippedCount: number
  dryRun: boolean
  errorMessage?: string
}
