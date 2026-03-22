import React, { useEffect, useState } from 'react'
import Modal from './ui/Modal'
import Button from './ui/Button'
import IconPicker from './ui/IconPicker'
import { api } from '../api/client'

interface EditEnvironmentModalProps {
  isOpen: boolean
  onClose: () => void
  onUpdated: () => void
  envId: string
  currentName: string
  currentIcon: string
}

const MAX_NAME_LEN = 80

const EditEnvironmentModal: React.FC<EditEnvironmentModalProps> = ({
  isOpen,
  onClose,
  onUpdated,
  envId,
  currentName,
  currentIcon,
}) => {
  const [name, setName] = useState(currentName)
  const [icon, setIcon] = useState(currentIcon)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Sync inputs when the modal opens
  useEffect(() => {
    if (isOpen) {
      setName(currentName)
      setIcon(currentIcon)
      setError(null)
    }
  }, [isOpen, currentName, currentIcon])

  const handleClose = () => {
    setError(null)
    onClose()
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) {
      setError('Display name is required.')
      return
    }
    if (trimmed.length > MAX_NAME_LEN) {
      setError(`Display name must be ${MAX_NAME_LEN} characters or fewer.`)
      return
    }
    if (trimmed === currentName && icon === currentIcon) {
      handleClose()
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await api.updateEnvironment(envId, trimmed, icon)
      onUpdated()
      handleClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update environment.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title="Edit Environment"
      subtitle="Update the name and icon for this environment."
      maxWidth="sm"
    >
      <form className="space-y-6" onSubmit={handleSubmit} noValidate>
        {/* Read-only ID */}
        <div className="space-y-1.5">
          <label className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Environment ID
          </label>
          <div className="w-full bg-surface-container border border-outline-variant/30 rounded-lg font-mono text-sm px-4 py-3 text-on-surface-variant/60 select-all">
            {envId}
          </div>
          <p className="text-[10px] text-on-surface-variant/50">ID cannot be changed after creation.</p>
        </div>

        {/* Editable name */}
        <div className="space-y-1.5">
          <label htmlFor="edit-env-name" className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Display Name
          </label>
          <input
            id="edit-env-name"
            type="text"
            value={name}
            onChange={(e) => { setName(e.target.value); setError(null) }}
            placeholder="Production"
            maxLength={MAX_NAME_LEN}
            autoFocus
            className="w-full bg-surface-container border border-outline-variant/50 rounded-lg text-sm px-4 py-3 focus:outline-none focus:border-primary transition-colors placeholder:text-on-surface-variant/40 text-on-surface"
          />
          {error && <p className="text-[10px] text-error font-medium">{error}</p>}
        </div>

        {/* Icon picker */}
        <div className="space-y-1.5">
          <label className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">
            Icon
          </label>
          <IconPicker value={icon} onChange={setIcon} />
        </div>

        <div className="flex gap-4 pt-2">
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
            {submitting ? 'Saving…' : 'Save Changes'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export default EditEnvironmentModal
