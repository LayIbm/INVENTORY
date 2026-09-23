interface Props {
  message: string
  onConfirm: () => void
  onCancel: () => void
  confirmLabel?: string
}

export default function ConfirmDialog({ message, onConfirm, onCancel, confirmLabel = 'Delete' }: Props) {
  return (
    <div className="overlay" style={{ display: 'flex' }}>
      <div className="modal" style={{ maxWidth: 400 }}>
        <div className="modal-body" style={{ paddingTop: 24, paddingBottom: 8 }}>
          <h3 style={{ fontSize: '1.125rem', fontWeight: 400, marginBottom: 12, color: 'var(--cds-text-primary)' }}>
            Confirm action
          </h3>
          <p style={{ color: 'var(--cds-text-secondary)', fontSize: '0.875rem', lineHeight: 1.5, margin: 0 }}>
            {message}
          </p>
        </div>
        <div className="modal-foot" style={{ justifyContent: 'flex-end', gap: 8 }}>
          <button className="btn btn-ghost" onClick={onCancel}>Cancel</button>
          <button
            className="btn"
            style={{
              background: 'var(--cds-support-error)',
              color: 'var(--cds-text-on-color)',
              borderColor: 'var(--cds-support-error)',
            }}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
