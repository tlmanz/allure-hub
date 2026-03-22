import React, { useState } from 'react'
import { Modal, Button } from './ui'
import { useUI } from '../context/UIContext'
import { api } from '../api/client'

interface FormState {
  name: string
  projectId: string
}

interface FormErrors {
  name?: string
  projectId?: string
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
  if (!values.projectId.trim()) {
    errors.projectId = 'Project ID is required.'
  } else if (!ID_PATTERN.test(values.projectId)) {
    errors.projectId = 'ID must be lowercase letters, numbers, and hyphens only.'
  } else if (values.projectId.length > MAX_ID_LEN) {
    errors.projectId = `Project ID must be ${MAX_ID_LEN} characters or fewer.`
  }
  return errors
}

interface NewProjectModalProps {
  onCreated?: () => void
}

const NewProjectModal: React.FC<NewProjectModalProps> = ({ onCreated }) => {
  const { isNewProjectModalOpen, activeEnvId, closeNewProjectModal } = useUI()
  const [form, setForm] = useState<FormState>({ name: '', projectId: '' })
  const [errors, setErrors] = useState<FormErrors>({})
  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  const handleChange = (field: keyof FormState, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) setErrors((prev) => ({ ...prev, [field]: undefined }))
    if (serverError) setServerError(null)
  }

  const handleClose = () => {
    setForm({ name: '', projectId: '' })
    setErrors({})
    setServerError(null)
    closeNewProjectModal()
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
      if (!activeEnvId) throw new Error('No environment selected.')
      await api.createProject(activeEnvId, form.projectId.trim(), form.name.trim())
      onCreated?.()
      handleClose()
    } catch (err) {
      setServerError(err instanceof Error ? err.message : 'Failed to create project.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      isOpen={isNewProjectModalOpen}
      onClose={handleClose}
      title="New Project"
      subtitle="Initialize a new test repository environment."
    >
      <form className="space-y-6" onSubmit={handleSubmit} noValidate>
        <div className="space-y-1.5">
          <label htmlFor="new-project-name" className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Display Name
          </label>
          <input
            id="new-project-name"
            type="text"
            value={form.name}
            onChange={(e) => handleChange('name', e.target.value)}
            placeholder="Acme Logistics UI"
            maxLength={MAX_NAME_LEN}
            className="w-full bg-surface-container border border-outline-variant/50 rounded-lg text-sm px-4 py-3 focus:outline-none focus:border-primary transition-colors placeholder:text-on-surface-variant/40 text-on-surface"
          />
          {errors.name && (
            <p className="text-[10px] text-error font-medium">{errors.name}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="new-project-id" className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Project ID
          </label>
          <input
            id="new-project-id"
            type="text"
            value={form.projectId}
            onChange={(e) => handleChange('projectId', e.target.value.toLowerCase())}
            placeholder="e.g. acme-logistics-ui"
            maxLength={MAX_ID_LEN}
            className="w-full bg-surface-container border border-outline-variant/50 rounded-lg font-mono text-sm px-4 py-3 focus:outline-none focus:border-primary transition-colors placeholder:text-on-surface-variant/40 text-on-surface"
          />
          {errors.projectId ? (
            <p className="text-[10px] text-error font-medium">{errors.projectId}</p>
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
            {submitting ? 'Creating…' : 'Create Project'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export default NewProjectModal
