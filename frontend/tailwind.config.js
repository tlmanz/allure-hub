/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // ── Themed via CSS custom properties – see src/styles/index.css ────────
        'background':               'rgb(var(--color-background) / <alpha-value>)',
        'surface':                  'rgb(var(--color-surface) / <alpha-value>)',
        'surface-container-lowest': 'rgb(var(--color-surface-container-lowest) / <alpha-value>)',
        'surface-container-low':    'rgb(var(--color-surface-container-low) / <alpha-value>)',
        'surface-container':        'rgb(var(--color-surface-container) / <alpha-value>)',
        'surface-container-high':   'rgb(var(--color-surface-container-high) / <alpha-value>)',
        'surface-container-highest':'rgb(var(--color-surface-container-highest) / <alpha-value>)',
        'surface-dim':              'rgb(var(--color-surface-dim) / <alpha-value>)',
        'surface-bright':           'rgb(var(--color-surface-bright) / <alpha-value>)',
        'surface-variant':          'rgb(var(--color-surface-variant) / <alpha-value>)',

        // ── Text ──────────────────────────────────────────────────────────────
        'on-background':            'rgb(var(--color-on-background) / <alpha-value>)',
        'on-surface':               'rgb(var(--color-on-surface) / <alpha-value>)',
        'on-surface-variant':       'rgb(var(--color-on-surface-variant) / <alpha-value>)',
        'inverse-surface':          'rgb(var(--color-inverse-surface) / <alpha-value>)',
        'inverse-on-surface':       'rgb(var(--color-inverse-on-surface) / <alpha-value>)',

        // ── Primary ───────────────────────────────────────────────────────────
        'primary':                  'rgb(var(--color-primary) / <alpha-value>)',
        'on-primary':               'rgb(var(--color-on-primary) / <alpha-value>)',
        'primary-container':        'rgb(var(--color-primary-container) / <alpha-value>)',
        'on-primary-container':     'rgb(var(--color-on-primary-container) / <alpha-value>)',
        'primary-fixed':            'rgb(var(--color-primary-fixed) / <alpha-value>)',
        'primary-fixed-dim':        'rgb(var(--color-primary-fixed-dim) / <alpha-value>)',
        'on-primary-fixed':         'rgb(var(--color-on-primary-fixed) / <alpha-value>)',
        'on-primary-fixed-variant': 'rgb(var(--color-on-primary-fixed-variant) / <alpha-value>)',
        'inverse-primary':          'rgb(var(--color-inverse-primary) / <alpha-value>)',

        // ── Secondary ─────────────────────────────────────────────────────────
        'secondary':                'rgb(var(--color-secondary) / <alpha-value>)',
        'on-secondary':             'rgb(var(--color-on-secondary) / <alpha-value>)',
        'secondary-container':      'rgb(var(--color-secondary-container) / <alpha-value>)',
        'on-secondary-container':   'rgb(var(--color-on-secondary-container) / <alpha-value>)',
        'secondary-fixed':          'rgb(var(--color-secondary-fixed) / <alpha-value>)',
        'secondary-fixed-dim':      'rgb(var(--color-secondary-fixed-dim) / <alpha-value>)',
        'on-secondary-fixed':       'rgb(var(--color-on-secondary-fixed) / <alpha-value>)',
        'on-secondary-fixed-variant':'rgb(var(--color-on-secondary-fixed-variant) / <alpha-value>)',

        // ── Tertiary ──────────────────────────────────────────────────────────
        'tertiary':                 'rgb(var(--color-tertiary) / <alpha-value>)',
        'on-tertiary':              'rgb(var(--color-on-tertiary) / <alpha-value>)',
        'tertiary-container':       'rgb(var(--color-tertiary-container) / <alpha-value>)',
        'on-tertiary-container':    'rgb(var(--color-on-tertiary-container) / <alpha-value>)',
        'tertiary-fixed':           'rgb(var(--color-tertiary-fixed) / <alpha-value>)',
        'tertiary-fixed-dim':       'rgb(var(--color-tertiary-fixed-dim) / <alpha-value>)',
        'on-tertiary-fixed':        'rgb(var(--color-on-tertiary-fixed) / <alpha-value>)',
        'on-tertiary-fixed-variant':'rgb(var(--color-on-tertiary-fixed-variant) / <alpha-value>)',

        // ── Error ─────────────────────────────────────────────────────────────
        'error':                    'rgb(var(--color-error) / <alpha-value>)',
        'on-error':                 'rgb(var(--color-on-error) / <alpha-value>)',
        'error-container':          'rgb(var(--color-error-container) / <alpha-value>)',
        'on-error-container':       'rgb(var(--color-on-error-container) / <alpha-value>)',

        // ── Borders ────────────────────────────────────────────────────────────
        'outline':                  'rgb(var(--color-outline) / <alpha-value>)',
        'outline-variant':          'rgb(var(--color-outline-variant) / <alpha-value>)',

        // ── Surface tint ───────────────────────────────────────────────────────
        'surface-tint':             'rgb(var(--color-surface-tint) / <alpha-value>)',
      },
      fontFamily: {
        headline: ['"Space Grotesk"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        body:     ['"Inter"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        label:    ['"Manrope"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono:     ['"JetBrainsMonoVF"', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        'xs':  ['11px', { lineHeight: '16px' }],
        'sm':  ['12px', { lineHeight: '16px' }],
        'base':['14px', { lineHeight: '20px' }],
        'lg':  ['16px', { lineHeight: '24px' }],
        'xl':  ['18px', { lineHeight: '24px' }],
        '2xl': ['24px', { lineHeight: '32px' }],
        '3xl': ['36px', { lineHeight: '40px' }],
      },
      borderRadius: {
        DEFAULT: '8px',
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px',
        full: '9999px',
      },
      boxShadow: {
        'card':   '0 20px 40px rgba(0,0,0,0.40)',
        'medium': '0 20px 40px rgba(0,0,0,0.40)',
        'raised': '0 2px 6px rgba(0,0,0,0.30)',
        'sm':     '0 1px 4px rgba(0,0,0,0.25)',
        'glow-primary': '0 0 20px rgba(0,173,239,0.20)',
        'glow-error':   '0 0 20px rgba(248,113,113,0.20)',
      },
    },
  },
  plugins: [require('@tailwindcss/forms')],
}
