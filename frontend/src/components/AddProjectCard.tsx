import React from 'react'
import { useUI } from '../context/UIContext'

interface AddProjectCardProps {
  envId: string
}

const AddProjectCard: React.FC<AddProjectCardProps> = ({ envId }) => {
  const { openNewProjectModal } = useUI()

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => openNewProjectModal(envId)}
      onKeyDown={(e) => e.key === 'Enter' && openNewProjectModal(envId)}
      className="bg-surface-container-low rounded-xl p-6 flex flex-col items-center justify-center
                 border-2 border-dashed border-outline-variant/20
                 min-h-[220px] cursor-pointer group
                 hover:border-primary/40 hover:bg-surface-container transition-all duration-300"
    >
      <div className="w-12 h-12 rounded-full bg-surface-container-highest flex items-center justify-center mb-4
                      group-hover:bg-primary/20 transition-colors">
        <span className="material-symbols-outlined text-[28px] text-on-surface-variant group-hover:text-primary transition-colors">
          add_circle
        </span>
      </div>
      <p className="font-headline font-bold text-on-surface-variant group-hover:text-on-surface transition-colors text-sm">
        Register Project
      </p>
      <p className="text-xs text-on-surface-variant/60 font-label mt-1">
        Allure Collection Name
      </p>
    </div>
  )
}

export default AddProjectCard
