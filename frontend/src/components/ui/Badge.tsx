import React from 'react'

export type BadgeVariant = 'passed' | 'failed' | 'inactive' | 'active' | 'running' | 'info' | 'mono'

interface BadgeProps {
  variant: BadgeVariant
  label?: string
  className?: string
}

const variantStyles: Record<BadgeVariant, { wrapper: string; dot: string; text: string }> = {
  passed: {
    wrapper: 'bg-primary/10 text-primary',
    dot: 'bg-primary',
    text: 'Passed',
  },
  failed: {
    wrapper: 'bg-error/10 text-error',
    dot: 'bg-error',
    text: 'Failed',
  },
  inactive: {
    wrapper: 'bg-on-surface-variant/10 text-on-surface-variant',
    dot: 'bg-on-surface-variant',
    text: 'Inactive',
  },
  running: {
    wrapper: 'bg-secondary/10 text-secondary',
    dot: 'bg-secondary animate-pulse',
    text: 'Running',
  },
  active: {
    wrapper: 'bg-primary/10 text-primary',
    dot: 'bg-primary',
    text: 'Active',
  },
  info: {
    wrapper: 'bg-primary/10 text-primary',
    dot: 'bg-primary',
    text: 'Info',
  },
  mono: {
    wrapper: 'bg-surface-container-high text-primary',
    dot: '',
    text: '',
  },
}

const Badge: React.FC<BadgeProps> = ({ variant, label, className = '' }) => {
  const style = variantStyles[variant]
  const displayLabel = label ?? style.text

  if (variant === 'mono') {
    return (
      <span className={`${style.wrapper} px-2 py-0.5 rounded-md font-mono text-sm font-bold ${className}`}>
        {displayLabel}
      </span>
    )
  }

  return (
    <div className={`${style.wrapper} px-2 py-1 rounded-full flex items-center gap-1.5 ${className}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${style.dot}`} />
      <span className="text-[10px] font-bold uppercase tracking-wider">{displayLabel}</span>
    </div>
  )
}

export default Badge
