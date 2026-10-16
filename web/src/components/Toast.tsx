import type { ToastMessage } from '../hooks/useToast';

export function Toast({ toast }: { toast: ToastMessage | null }) {
  return (
    <div className="toast-region" role="status" aria-live="polite">
      {toast && (
        <div className="toast" key={toast.id}>
          {toast.text}
        </div>
      )}
    </div>
  );
}
