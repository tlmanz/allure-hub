import { useCallback, useEffect, useRef, useState } from 'react'

export type SnackbarVariant = 'success' | 'error' | 'warning' | 'info'

// Pass a plain string for simple messages, or an object to show a bold title
// with a smaller description line beneath it.
export type SnackbarMessage = string | { title: string; description?: string }

interface SnackbarState {
  visible: boolean
  title: string
  description: string
  variant: SnackbarVariant
}

const VARIANT_STYLES: Record<SnackbarVariant, { accent: string; icon: string; title: string; iconName: string }> = {
  success: {
    accent: 'border-l-emerald-500',
    icon: 'text-emerald-500',
    title: 'text-emerald-700 dark:text-emerald-400',
    iconName: 'check_circle',
  },
  error: {
    accent: 'border-l-red-500',
    icon: 'text-red-500',
    title: 'text-red-700 dark:text-red-400',
    iconName: 'error',
  },
  warning: {
    accent: 'border-l-amber-500',
    icon: 'text-amber-500',
    title: 'text-amber-700 dark:text-amber-400',
    iconName: 'warning',
  },
  info: {
    accent: 'border-l-sky-500',
    icon: 'text-sky-500',
    title: 'text-sky-700 dark:text-sky-400',
    iconName: 'info',
  },
}

interface SnackbarProps {
  visible: boolean
  title: string
  description: string
  variant: SnackbarVariant
  onClose: () => void
}

function Snackbar({ visible, title, description, variant, onClose }: SnackbarProps) {
  const s = VARIANT_STYLES[variant]
  return (
    <div
      className={`fixed top-20 left-1/2 -translate-x-1/2 z-[80] flex items-start gap-3 pl-4 pr-3 py-3 rounded-xl border-l-4 shadow-xl transition-all duration-300 min-w-[280px] max-w-sm ${s.accent} ${
        visible ? 'opacity-100 translate-y-0 pointer-events-auto' : 'opacity-0 -translate-y-2 pointer-events-none'
      }`}
      style={{
        background: 'rgb(var(--color-surface-container-high))',
        boxShadow: '0 8px 32px rgb(0 0 0 / 0.18), 0 2px 8px rgb(0 0 0 / 0.10)',
        borderRight: '1px solid rgb(var(--color-outline-variant) / 0.25)',
        borderTop: '1px solid rgb(var(--color-outline-variant) / 0.25)',
        borderBottom: '1px solid rgb(var(--color-outline-variant) / 0.25)',
      }}
      role="status"
      aria-live="polite"
    >
      <span className={`material-symbols-outlined text-[20px] shrink-0 mt-px ${s.icon}`}>{s.iconName}</span>

      <div className="flex-1 min-w-0">
        <p className={`text-sm font-headline font-bold leading-snug ${s.title}`}>{title}</p>
        {description && (
          <p className="text-xs text-on-surface-variant mt-0.5 leading-relaxed">{description}</p>
        )}
      </div>

      <button
        onClick={onClose}
        className="shrink-0 p-1 rounded text-on-surface-variant opacity-50 hover:opacity-100 transition-opacity"
        aria-label="Dismiss"
      >
        <span className="material-symbols-outlined text-[14px]">close</span>
      </button>
    </div>
  )
}

function parseMessage(msg: SnackbarMessage): { title: string; description: string } {
  if (typeof msg === 'string') return { title: msg, description: '' }
  return { title: msg.title, description: msg.description ?? '' }
}

export function useSnackbar() {
  const [state, setState] = useState<SnackbarState>({
    visible: false,
    title: '',
    description: '',
    variant: 'success',
  })
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = useCallback((msg: SnackbarMessage, variant: SnackbarVariant = 'success') => {
    if (timerRef.current) clearTimeout(timerRef.current)
    const { title, description } = parseMessage(msg)
    setState({ visible: true, title, description, variant })
    timerRef.current = setTimeout(() => setState(s => ({ ...s, visible: false })), 4000)
  }, [])

  const dismiss = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setState(s => ({ ...s, visible: false }))
  }, [])

  useEffect(() => () => { if (timerRef.current) clearTimeout(timerRef.current) }, [])

  const SnackbarNode = (
    <Snackbar
      visible={state.visible}
      title={state.title}
      description={state.description}
      variant={state.variant}
      onClose={dismiss}
    />
  )

  return { show, SnackbarNode }
}

export default Snackbar
