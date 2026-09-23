import { Toast } from '../hooks/useToast'

interface Props {
  toasts: Toast[]
}

export default function ToastContainer({ toasts }: Props) {
  return (
    <div className="toast-wrap">
      {toasts.map((t) => (
        <div key={t.id} className={`toast${t.tone === 'danger' ? ' danger' : ''}`}>
          {t.message}
        </div>
      ))}
    </div>
  )
}
