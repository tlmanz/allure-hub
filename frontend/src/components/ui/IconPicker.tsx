import React from 'react'

export const ENVIRONMENT_ICONS = [
  // Infrastructure / deployment
  { name: 'deployed_code',      label: 'Deployed Code' },
  { name: 'cloud',              label: 'Cloud' },
  { name: 'cloud_upload',       label: 'Cloud Upload' },
  { name: 'dns',                label: 'DNS / Server' },
  { name: 'storage',            label: 'Storage' },
  { name: 'database',           label: 'Database' },
  { name: 'hub',                label: 'Hub' },
  { name: 'lan',                label: 'Network' },
  // Testing / QA
  { name: 'labs',               label: 'Labs' },
  { name: 'science',            label: 'Science' },
  { name: 'bug_report',         label: 'Bug Report' },
  { name: 'pest_control',       label: 'Pest Control' },
  { name: 'verified',           label: 'Verified' },
  { name: 'fact_check',         label: 'Fact Check' },
  { name: 'checklist',          label: 'Checklist' },
  // Environments
  { name: 'rocket_launch',      label: 'Rocket' },
  { name: 'construction',       label: 'Construction' },
  { name: 'build',              label: 'Build' },
  { name: 'tune',               label: 'Tune' },
  { name: 'settings',           label: 'Settings' },
  { name: 'developer_mode',     label: 'Developer' },
  { name: 'code',               label: 'Code' },
  { name: 'terminal',           label: 'Terminal' },
  // Lifecycle stages
  { name: 'release_alert',      label: 'Release' },
  { name: 'flag',               label: 'Flag' },
  { name: 'bolt',               label: 'Bolt' },
  { name: 'new_releases',       label: 'New Releases' },
  { name: 'published_with_changes', label: 'Published' },
  { name: 'history',            label: 'History' },
  { name: 'schedule',           label: 'Schedule' },
  { name: 'pending',            label: 'Pending' },
] as const

export type EnvironmentIconName = typeof ENVIRONMENT_ICONS[number]['name']

interface IconPickerProps {
  value: string
  onChange: (icon: string) => void
}

const IconPicker: React.FC<IconPickerProps> = ({ value, onChange }) => {
  return (
    <div className="grid grid-cols-8 gap-1.5">
      {ENVIRONMENT_ICONS.map(({ name, label }) => (
        <button
          key={name}
          type="button"
          title={label}
          onClick={() => onChange(name)}
          className={`
            flex items-center justify-center w-9 h-9 rounded-lg transition-all
            ${value === name
              ? 'bg-primary text-on-primary shadow-sm'
              : 'bg-surface-container text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
            }
          `}
        >
          <span className="material-symbols-outlined text-[20px]">{name}</span>
        </button>
      ))}
    </div>
  )
}

export default IconPicker
