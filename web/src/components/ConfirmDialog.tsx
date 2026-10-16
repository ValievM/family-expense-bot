import { Modal } from './Modal';

interface ConfirmDialogProps {
  title: string;
  message: string;
  confirmLabel: string;
  saving: boolean;
  error: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({ title, message, confirmLabel, saving, error, onConfirm, onCancel }: ConfirmDialogProps) {
  return (
    <Modal title={title} onClose={saving ? () => undefined : onCancel}>
      <p>{message}</p>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="modal__actions">
        <button type="button" className="btn" onClick={onCancel} disabled={saving}>
          Отмена
        </button>
        <button type="button" className="btn btn--danger-solid" onClick={onConfirm} disabled={saving} autoFocus>
          {saving ? 'Удаление…' : confirmLabel}
        </button>
      </div>
    </Modal>
  );
}
