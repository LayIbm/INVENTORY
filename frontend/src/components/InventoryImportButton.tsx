import { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import InventoryImportModal from './InventoryImportModal'

export interface InventoryImportButtonProps {
  onImportDone: () => void
}

export default function InventoryImportButton({ onImportDone }: InventoryImportButtonProps) {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)

  if (user?.role === 'viewer') return null

  return (
    <>
      <button type="button" className="btn btn-secondary" onClick={() => setOpen(true)}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
        Import Excel
      </button>

      {open && (
        <InventoryImportModal
          onClose={() => setOpen(false)}
          onImportDone={onImportDone}
        />
      )}
    </>
  )
}
