import { useState, useCallback, useRef } from 'react'

export interface Toast {
  id: number
  message: string
  tone?: 'danger'
}

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([])
  const counter = useRef(0)

  const addToast = useCallback((message: string, tone?: 'danger') => {
    const id = ++counter.current
    setToasts((prev) => [...prev, { id, message, tone }])
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 2600)
  }, [])

  return { toasts, addToast }
}
