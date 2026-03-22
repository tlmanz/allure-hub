import { useCallback, useEffect, useState } from 'react'
import { useParams, useLocation, Link } from 'react-router-dom'
import ProjectGrid from '../components/ProjectGrid'
import NewProjectModal from '../components/NewProjectModal'
import { api } from '../api/client'
import type { Project } from '../types'
import type { ProjectCardProps } from '../components/ProjectCard'

function toCardProps(p: Project): ProjectCardProps {
  const status =
    p.lastStatus === 'active' ? 'passed' : (p.lastStatus as ProjectCardProps['status'])
  const lastRun = p.lastBuildAt
    ? new Date(p.lastBuildAt).toLocaleString()
    : 'Never'
  return {
    name: p.name,
    id: p.id,
    envId: p.environmentId,
    builds: p.buildCount,
    status,
    lastRun,
    lastTotal:  p.lastTotal,
    lastPassed: p.lastPassed,
    lastFailed: p.lastFailed,
  }
}

export default function ProjectsPage() {
  const { envId } = useParams<{ envId: string }>()
  const location = useLocation()
  const state = location.state
  const envName = (state !== null && typeof state === 'object' && 'envName' in state && typeof (state as Record<string, unknown>).envName === 'string')
    ? (state as { envName: string }).envName
    : envId ?? ''

  const [projects, setProjects] = useState<Project[]>([])
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!envId) return
    api.listProjects(envId)
      .then(setProjects)
      .catch((e: Error) => setError(e.message))
  }, [envId])

  // Initial fetch with abort-on-unmount / envId-change so stale responses are ignored (M-19).
  useEffect(() => {
    if (!envId) return
    const controller = new AbortController()
    api.listProjects(envId, controller.signal)
      .then(setProjects)
      .catch((e: Error) => { if (e.name !== 'AbortError') setError(e.message) })
    return () => controller.abort()
  }, [envId])

  return (
    <>
      <div className="flex items-center gap-3 mb-2">
        <Link
          to="/"
          className="p-2 text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors rounded-lg"
          aria-label="Back to Environments"
        >
          <span className="material-symbols-outlined text-[20px]">arrow_back</span>
        </Link>
        <span className="text-xs font-label text-on-surface-variant">Environments</span>
        <span className="text-xs font-label text-on-surface-variant opacity-40">/</span>
        <span className="text-xs font-label text-on-surface">{envName}</span>
      </div>

      {error && (
        <p className="text-xs text-error bg-error/10 rounded-lg px-3 py-2 mb-4">{error}</p>
      )}

      <ProjectGrid
        envId={envId ?? ''}
        envName={envName}
        projects={projects.map(toCardProps)}
        onProjectDeleted={load}
      />

      <NewProjectModal onCreated={load} />
    </>
  )
}
