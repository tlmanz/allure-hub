import React, { useEffect, useRef } from 'react'
import { useFocusTrap } from '../../hooks/useFocusTrap'

interface DeleteConfirmModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description: string
  itemName: string
  isDeleting?: boolean
  errorMessage?: string
}

const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  itemName,
  isDeleting = false,
  errorMessage,
}) => {
  const dialogRef = useRef<HTMLDivElement>(null)
  useFocusTrap(dialogRef, isOpen)

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isDeleting) onClose()
    }
    if (isOpen) document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [isOpen, onClose, isDeleting])

  if (!isOpen) return null

  return (
    <div
      ref={dialogRef}
      className="fixed inset-0 z-[60] flex items-center justify-center px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-modal-title"
      tabIndex={-1}
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-surface-dim/80 backdrop-blur-sm"
        onClick={!isDeleting ? onClose : undefined}
        aria-hidden="true"
      />

      {/* Modal */}
      <div className="relative z-10 w-full max-w-sm bg-surface-container-low border border-outline-variant/20 rounded-2xl p-6 shadow-2xl animate-in">
        {/* Warning icon */}
        <div className="flex items-center justify-center w-12 h-12 rounded-full bg-error/10 mx-auto mb-4">
          <span className="material-symbols-outlined text-[24px] text-error">delete_forever</span>
        </div>

        {/* Title */}
        <h2
          id="delete-modal-title"
          className="text-xl font-headline font-bold text-on-surface text-center mb-1"
        >
          {title}
        </h2>

        {/* Description */}
        <p className="text-sm text-on-surface-variant text-center mb-2">{description}</p>

        {/* Item name chip */}
        <div className="flex justify-center mb-6">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-error/8 border border-error/20 text-error text-sm font-mono font-semibold max-w-full">
            <span className="material-symbols-outlined text-[14px] shrink-0">label</span>
            <span className="truncate">{itemName}</span>
          </span>
        </div>

        {/* Delete error */}
        {errorMessage && (
          <p className="text-xs text-error bg-error/10 rounded-lg px-3 py-2 mb-4 text-center">
            {errorMessage}
          </p>
        )}

        {/* Action buttons */}
        <div className="flex gap-3">
          <button
            onClick={onClose}
            disabled={isDeleting}
            className="flex-1 px-4 py-2.5 rounded-xl border border-outline-variant/30 text-on-surface-variant font-label font-semibold text-sm hover:bg-surface-container hover:text-on-surface transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isDeleting}
            className="flex-1 px-4 py-2.5 rounded-xl bg-error text-on-error font-label font-semibold text-sm hover:bg-error/90 active:scale-95 transition-all disabled:opacity-70 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {isDeleting ? (
              <>
                <span className="material-symbols-outlined text-[16px] animate-spin">progress_activity</span>
                Deleting…
              </>
            ) : (
              <>
                <span className="material-symbols-outlined text-[16px]">delete</span>
                Delete
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

export default DeleteConfirmModal
