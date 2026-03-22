import React from 'react'

export type IconName =
  | 'folder'
  | 'history'
  | 'bar_chart'
  | 'settings'
  | 'star'
  | 'layers'
  | 'schedule'
  | 'search'
  | 'filter_list'
  | 'add'
  | 'close'
  | 'delete'
  | 'cloud_upload'
  | 'menu'
  | 'arrow_back'
  | 'open_in_new'
  | 'more_vert'
  | 'check_circle'
  | 'cancel'
  | 'warning'
  | string

interface IconProps {
  name: IconName
  filled?: boolean
  className?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
}

const sizeMap: Record<NonNullable<IconProps['size']>, string> = {
  xs: 'text-xs',
  sm: 'text-sm',
  md: 'text-base',
  lg: 'text-2xl',
  xl: 'text-3xl',
}

const Icon: React.FC<IconProps> = ({ name, filled = false, className = '', size = 'md' }) => (
  <span
    className={`material-symbols-outlined ${sizeMap[size]} ${className}`}
    style={filled ? { fontVariationSettings: "'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24" } : undefined}
    aria-hidden="true"
  >
    {name}
  </span>
)

export default Icon
