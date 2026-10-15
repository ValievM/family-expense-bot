import { Modal } from './Modal';

/** Свойства диалога подтверждения. */
interface ConfirmDialogProps {
  title: string;
  message: string;
  confirmLabel: string;
  saving: boolean;
  error: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * Принимает: тексты, флаг выполнения, ошибку и обработчики подтверждения и отмены.
 * Делает: показывает модальный вопрос; кнопки блокируются, пока идёт запрос.
 * Возвращает: React-элемент диалога.
 */
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
