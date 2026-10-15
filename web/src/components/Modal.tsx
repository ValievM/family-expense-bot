import { useEffect, useRef, type ReactNode } from 'react';

/** Свойства модального окна. */
interface ModalProps {
  title: string;
  onClose: () => void;
  children: ReactNode;
}

/**
 * Принимает: заголовок, обработчик закрытия и содержимое.
 * Делает: открывает нативный <dialog> как модальный (фокус-ловушка, Esc) и закрывает по клику на фон.
 * Возвращает: React-элемент диалога.
 */
export function Modal({ title, onClose, children }: ModalProps) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dlg = ref.current;
    if (dlg && !dlg.open) dlg.showModal();
  }, []);

  return (
    <dialog
      ref={ref}
      className="modal"
      aria-labelledby="modal-title"
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      onClick={(e) => {
        if (e.target === ref.current) onClose();
      }}
    >
      <div className="modal__body">
        <h2 id="modal-title" className="modal__title">
          {title}
        </h2>
        {children}
      </div>
    </dialog>
  );
}
