export const APP_VERSION = 'v2.4.1' as const
export const APP_VERSION_STABLE = 'v2.4.1-stable' as const
export const APP_NAME = 'allure-hub' as const

export const COLOR = {
  // Obsidian Architect — deep navy dark palette
  background:               '#040e1f',
  surface:                  '#040e1f',
  surfaceContainerLowest:   '#000000',
  surfaceContainerLow:      '#061326',
  surfaceContainer:         '#0b1a2f',
  surfaceContainerHigh:     '#102036',
  surfaceContainerHighest:  '#15263f',
  // Text
  onSurface:                '#dbe6fe',
  onSurfaceVariant:         '#a0abc2',
  // Primary — neon green
  primary:                  '#5cfd80',
  onPrimary:                '#005d22',
  // Secondary — electric blue
  secondary:                '#1db1f1',
  onSecondary:              '#002b3f',
  // Tertiary — burnt orange
  tertiary:                 '#ff8762',
  onTertiary:               '#531300',
  // Error — coral red
  error:                    '#ff716c',
  onError:                  '#490006',
  // Borders
  outline:                  '#6b768b',
  outlineVariant:           '#3d485c',
} as const

export const FONT = {
  headline: 'font-headline',
  body: 'font-body',
  label: 'font-label',
  mono: 'font-mono',
} as const

export const TRANSITION = {
  base: 'transition-all duration-200 ease-in-out',
  colors: 'transition-colors duration-150',
  transform: 'transition-transform duration-200',
} as const
