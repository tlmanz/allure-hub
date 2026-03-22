import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Environment } from '../types'
import EnvironmentCard from '../components/EnvironmentCard'
import NewEnvironmentModal from '../components/NewEnvironmentModal'
import { useUI } from '../context/UIContext'

function AddEnvironmentCard() {
  const { openNewEnvironmentModal } = useUI()
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={openNewEnvironmentModal}
      onKeyDown={(e) => e.key === 'Enter' && openNewEnvironmentModal()}
      className="bg-surface-container-low rounded-2xl p-6 flex flex-col items-center justify-center
                 border-2 border-dashed border-outline-variant/20 min-h-[160px] cursor-pointer group
                 hover:border-primary/40 hover:bg-surface-container transition-all duration-300"
    >
      <div className="w-12 h-12 rounded-full bg-surface-container-highest flex items-center justify-center mb-4
                      group-hover:bg-primary/20 transition-colors">
        <span className="material-symbols-outlined text-[28px] text-on-surface-variant group-hover:text-primary transition-colors">
          add_circle
        </span>
      </div>
      <p className="font-headline font-bold text-on-surface-variant group-hover:text-on-surface transition-colors text-sm">
        New Environment
      </p>
      <p className="text-xs text-on-surface-variant/60 font-label mt-1">
        Production, staging, QA…
      </p>
    </div>
  )
}

export default function EnvironmentsPage() {
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [error, setError] = useState<string | null>(null)
  const load = useCallback(() => {
    api.listEnvironments()
      .then(setEnvironments)
      .catch((e: Error) => setError(e.message))
  }, [])

  // Initial fetch with abort-on-unmount so stale responses are ignored (M-19).
  useEffect(() => {
    const controller = new AbortController()
    api.listEnvironments(controller.signal)
      .then(setEnvironments)
      .catch((e: Error) => { if (e.name !== 'AbortError') setError(e.message) })
    return () => controller.abort()
  }, [])

  return (
    <>
      <div className="flex justify-between items-end mb-12">
        <div>
          <h2 className="text-4xl font-bold mt-6 font-headline tracking-tight text-on-surface">
            Environments
          </h2>
          <p className="text-on-surface-variant font-body mt-2">
            <span className="text-secondary font-semibold">
              {environments.length} environment{environments.length !== 1 ? 's' : ''}
            </span>{' '}
            configured.
          </p>
        </div>
      </div>

      {error && (
        <p className="text-xs text-error bg-error/10 rounded-lg px-3 py-2 mb-4">{error}</p>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
        <AddEnvironmentCard />
        {environments.map((env) => (
          <EnvironmentCard
            key={env.id}
            id={env.id}
            name={env.name}
            icon={env.icon}
            projectCount={env.projectCount}
            createdAt={env.createdAt}
            onDeleted={load}
            onUpdated={load}
          />
        ))}
      </div>

      <NewEnvironmentModal onCreated={load} />
    </>
  )
}
