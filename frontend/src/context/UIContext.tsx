import React, { createContext, useCallback, useContext, useState } from 'react'
import type { ToastData } from '../components/ui'

interface UIState {
  isNewEnvironmentModalOpen: boolean
  openNewEnvironmentModal: () => void
  closeNewEnvironmentModal: () => void
  isNewProjectModalOpen: boolean
  activeEnvId: string | null
  openNewProjectModal: (envId: string) => void
  closeNewProjectModal: () => void
  toasts: ToastData[]
  addToast: (toast: ToastData) => void
  removeToast: (id: string) => void
}

const UIContext = createContext<UIState | null>(null)

export const UIProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isNewEnvironmentModalOpen, setNewEnvironmentModalOpen] = useState(false)
  const [isNewProjectModalOpen, setNewProjectModalOpen] = useState(false)
  const [activeEnvId, setActiveEnvId] = useState<string | null>(null)
  const [toasts, setToasts] = useState<ToastData[]>([])

  const openNewEnvironmentModal = useCallback(() => setNewEnvironmentModalOpen(true), [])
  const closeNewEnvironmentModal = useCallback(() => setNewEnvironmentModalOpen(false), [])

  const openNewProjectModal = useCallback((envId: string) => {
    setActiveEnvId(envId)
    setNewProjectModalOpen(true)
  }, [])
  const closeNewProjectModal = useCallback(() => {
    setNewProjectModalOpen(false)
    setActiveEnvId(null)
  }, [])

  const addToast = useCallback((toast: ToastData) => {
    setToasts((prev) => [...prev.filter((t) => t.id !== toast.id), toast])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <UIContext.Provider
      value={{
        isNewEnvironmentModalOpen,
        openNewEnvironmentModal,
        closeNewEnvironmentModal,
        isNewProjectModalOpen,
        activeEnvId,
        openNewProjectModal,
        closeNewProjectModal,
        toasts,
        addToast,
        removeToast,
      }}
    >
      {children}
    </UIContext.Provider>
  )
}

export function useUI(): UIState {
  const ctx = useContext(UIContext)
  if (!ctx) throw new Error('useUI must be used within UIProvider')
  return ctx
}
