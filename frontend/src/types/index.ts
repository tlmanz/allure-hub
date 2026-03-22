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
}
