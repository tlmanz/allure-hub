import React, { useState } from 'react'
import { Modal, Button } from './ui'
import IconPicker from './ui/IconPicker'
import { useUI } from '../context/UIContext'
import { api } from '../api/client'

const DEFAULT_ICON = 'deployed_code'

interface FormState {
  name: string
  envId: string
  icon: string
}

interface FormErrors {
  name?: string
  envId?: string
  icon?: string
}

const ID_PATTERN = /^[a-z0-9-]+$/
const MAX_NAME_LEN = 80
const MAX_ID_LEN = 63

function validate(values: FormState): FormErrors {
  const errors: FormErrors = {}
  if (!values.name.trim()) {
    errors.name = 'Display name is required.'
  } else if (values.name.trim().length > MAX_NAME_LEN) {
    errors.name = `Display name must be ${MAX_NAME_LEN} characters or fewer.`
  }
  if (!values.envId.trim()) {
    errors.envId = 'Environment ID is required.'
  } else if (!ID_PATTERN.test(values.envId)) {
    errors.envId = 'ID must be lowercase letters, numbers, and hyphens only.'
  } else if (values.envId.length > MAX_ID_LEN) {
    errors.envId = `Environment ID must be ${MAX_ID_LEN} characters or fewer.`
  }
  return errors
}

interface NewEnvironmentModalProps {
  onCreated?: () => void
}

const NewEnvironmentModal: React.FC<NewEnvironmentModalProps> = ({ onCreated }) => {
  const { isNewEnvironmentModalOpen, closeNewEnvironmentModal } = useUI()
  const [form, setForm] = useState<FormState>({ name: '', envId: '', icon: DEFAULT_ICON })
  const [errors, setErrors] = useState<FormErrors>({})
  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  const handleChange = (field: keyof FormState, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) setErrors((prev) => ({ ...prev, [field]: undefined }))
    if (serverError) setServerError(null)
  }

  const handleClose = () => {
    setForm({ name: '', envId: '', icon: DEFAULT_ICON })
    setErrors({})
    setServerError(null)
    closeNewEnvironmentModal()
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const validationErrors = validate(form)
    if (Object.keys(validationErrors).length) {
      setErrors(validationErrors)
      return
    }
    setSubmitting(true)
    try {
      await api.createEnvironment(form.envId.trim(), form.name.trim(), form.icon)
      onCreated?.()
      handleClose()
    } catch (err) {
      setServerError(err instanceof Error ? err.message : 'Failed to create environment.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      isOpen={isNewEnvironmentModalOpen}
      onClose={handleClose}
      title="New Environment"
      subtitle="Create a logical grouping for your test projects."
    >
      <form className="space-y-6" onSubmit={handleSubmit} noValidate>
        <div className="space-y-1.5">
          <label htmlFor="new-env-name" className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Display Name
          </label>
          <input
            id="new-env-name"
            type="text"
            value={form.name}
            onChange={(e) => handleChange('name', e.target.value)}
            placeholder="Production"
            maxLength={MAX_NAME_LEN}
            className="w-full bg-surface-container border border-outline-variant/50 rounded-lg text-sm px-4 py-3 focus:outline-none focus:border-primary transition-colors placeholder:text-on-surface-variant/40 text-on-surface"
          />
          {errors.name && (
            <p className="text-[10px] text-error font-medium">{errors.name}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Icon
          </label>
          <IconPicker
            value={form.icon}
            onChange={(icon) => setForm((prev) => ({ ...prev, icon }))}
          />
        </div>

        <div className="space-y-1.5">
          <label htmlFor="new-env-id" className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Environment ID
          </label>
          <input
            id="new-env-id"
            type="text"
            value={form.envId}
            onChange={(e) => handleChange('envId', e.target.value.toLowerCase())}
            placeholder="e.g. production"
            maxLength={MAX_ID_LEN}
            className="w-full bg-surface-container border border-outline-variant/50 rounded-lg font-mono text-sm px-4 py-3 focus:outline-none focus:border-primary transition-colors placeholder:text-on-surface-variant/40 text-on-surface"
          />
          {errors.envId ? (
            <p className="text-[10px] text-error font-medium">{errors.envId}</p>
          ) : (
            <p className="text-[10px] text-secondary font-medium">
              ID must be lowercase, no spaces.
            </p>
          )}
        </div>

        {serverError && (
          <p className="text-xs text-error bg-error/10 rounded-lg px-3 py-2">{serverError}</p>
        )}

        <div className="flex gap-4 pt-4">
          <Button
            variant="secondary"
            size="lg"
            type="button"
            className="flex-1"
            onClick={handleClose}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button
            variant="primary"
            size="lg"
            type="submit"
            className="flex-1"
            disabled={submitting}
          >
            {submitting ? 'Creating…' : 'Create Environment'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export default NewEnvironmentModal
