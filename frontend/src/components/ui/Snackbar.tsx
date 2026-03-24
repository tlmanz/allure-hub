import { useCallback, useEffect, useRef, useState } from 'react'

export type SnackbarVariant = 'success' | 'error'

interface SnackbarState {
  visible: boolean
  message: string
  variant: SnackbarVariant
}

const VARIANT_STYLES: Record<SnackbarVariant, { container: string; icon: string; iconName: string }> = {
  success: {
    container: 'bg-emerald-500/10 border-emerald-500/25 text-emerald-700 dark:text-emerald-400',
    icon: 'text-emerald-500',
    iconName: 'check_circle',
  },
  error: {
    container: 'bg-error/10 border-error/25 text-error',
    icon: 'text-error',
    iconName: 'error',
  },
}

interface SnackbarProps {
  visible: boolean
  message: string
  variant: SnackbarVariant
  onClose: () => void
}

function Snackbar({ visible, message, variant, onClose }: SnackbarProps) {
  const s = VARIANT_STYLES[variant]
  return (
    <div
      className={`fixed top-20 left-1/2 -translate-x-1/2 z-[80] flex items-center gap-3 px-4 py-3 rounded-xl border shadow-lg text-sm font-label font-semibold transition-all duration-300 ${s.container} ${
        visible ? 'opacity-100 translate-y-0 pointer-events-auto' : 'opacity-0 -translate-y-2 pointer-events-none'
      }`}
      role="status"
      aria-live="polite"
    >
      <span className={`material-symbols-outlined text-[18px] shrink-0 ${s.icon}`}>{s.iconName}</span>
      <span>{message}</span>
      <button
        onClick={onClose}
        className="ml-1 p-0.5 rounded opacity-60 hover:opacity-100 transition-opacity"
        aria-label="Dismiss"
      >
        <span className="material-symbols-outlined text-[14px]">close</span>
      </button>
    </div>
  )
}

export function useSnackbar() {
  const [state, setState] = useState<SnackbarState>({ visible: false, message: '', variant: 'success' })
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = useCallback((message: string, variant: SnackbarVariant = 'success') => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setState({ visible: true, message, variant })
    timerRef.current = setTimeout(() => setState(s => ({ ...s, visible: false })), 3500)
  }, [])

  const dismiss = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setState(s => ({ ...s, visible: false }))
  }, [])

  useEffect(() => () => { if (timerRef.current) clearTimeout(timerRef.current) }, [])

  const SnackbarNode = (
    <Snackbar
      visible={state.visible}
      message={state.message}
      variant={state.variant}
      onClose={dismiss}
    />
  )

  return { show, SnackbarNode }
}

export default Snackbar
