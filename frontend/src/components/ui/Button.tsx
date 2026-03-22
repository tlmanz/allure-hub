import React from 'react'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline'
export type ButtonSize = 'sm' | 'md' | 'lg'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  children: React.ReactNode
}

const variantStyles: Record<ButtonVariant, string> = {
  primary:
    'bg-primary text-on-primary font-headline font-bold hover:brightness-110 active:brightness-90',
  secondary:
    'bg-surface-container-high text-on-surface font-semibold border border-outline hover:bg-surface-container-highest',
  ghost:
    'text-primary font-semibold hover:bg-primary/10 active:bg-primary/20',
  danger:
    'text-error font-semibold hover:bg-error/10 active:bg-error/20',
  outline:
    'text-primary font-semibold border border-primary hover:bg-primary/10 active:bg-primary/20',
}

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'text-xs px-3 py-1.5 rounded-lg',
  md: 'text-sm px-4 py-2 rounded-lg',
  lg: 'text-sm px-5 py-2.5 rounded-lg',
}

const Button: React.FC<ButtonProps> = ({
  variant = 'secondary',
  size = 'md',
  className = '',
  children,
  ...props
}) => (
  <button
    className={[
      'inline-flex items-center justify-center gap-2',
      'transition-colors duration-150',
      'active:scale-[0.98]',
      'disabled:opacity-40 disabled:cursor-not-allowed disabled:pointer-events-none',
      variantStyles[variant],
      sizeStyles[size],
      className,
    ].join(' ')}
    {...props}
  >
    {children}
  </button>
)

export default Button
