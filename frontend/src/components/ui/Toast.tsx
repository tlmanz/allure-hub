import React from 'react'
import Icon from './Icon'

export interface ToastData {
  id: string
  label: string
  progress: number
  detail: string
}

interface ToastProps {
  toast: ToastData
  onCancel: (id: string) => void
}

const Toast: React.FC<ToastProps> = ({ toast, onCancel }) => (
  <div className="w-80 glass-panel rounded-lg shadow-2xl border border-primary/20 overflow-hidden">
    <div className="p-4 bg-primary/10 flex items-center justify-between">
      <div className="flex items-center gap-3">
        <Icon name="cloud_upload" className="text-primary animate-pulse" />
        <span className="text-xs font-bold text-on-surface">{toast.label}</span>
      </div>
      <span className="text-[10px] font-mono text-primary font-bold">{toast.progress}%</span>
    </div>
    <div className="h-1 w-full bg-surface-container-highest">
      <div
        className="h-full bg-primary transition-all duration-300"
        style={{ width: `${toast.progress}%` }}
      />
    </div>
    <div className="p-3 bg-surface-container-low/50 flex items-center justify-between">
      <p className="text-[10px] text-on-surface-variant font-mono truncate max-w-[180px]">
        {toast.detail}
      </p>
      <button
        onClick={() => onCancel(toast.id)}
        className="text-[10px] font-bold text-error uppercase tracking-tighter hover:underline"
      >
        Cancel
      </button>
    </div>
  </div>
)

interface ToastStackProps {
  toasts: ToastData[]
  onCancel: (id: string) => void
}

export const ToastStack: React.FC<ToastStackProps> = ({ toasts, onCancel }) => {
  if (!toasts.length) return null
  return (
    <div className="fixed bottom-6 right-6 z-[70] flex flex-col gap-3">
      {toasts.map((t) => (
        <Toast key={t.id} toast={t} onCancel={onCancel} />
      ))}
    </div>
  )
}

export default Toast
