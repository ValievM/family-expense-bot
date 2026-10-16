import { useEffect, useRef, type ReactNode, type RefObject } from 'react';

interface DialogProps {
  className: string;
  labelledBy: string;
  onRequestClose: () => void;
  initialFocus?: RefObject<HTMLElement | null>;
  children: ReactNode;
}

export function Dialog({ className, labelledBy, onRequestClose, initialFocus, children }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);
  const unmounting = useRef(false);
  const requestClose = useRef(onRequestClose);
  requestClose.current = onRequestClose;

  useEffect(() => {
    const dlg = ref.current;
    if (!dlg) return;
    unmounting.current = false;
    // showModal делает остальную страницу inert: это и есть ловушка фокуса.
    if (!dlg.open) dlg.showModal();
    initialFocus?.current?.focus();
    return () => {
      unmounting.current = true;
      if (dlg.open) dlg.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <dialog
      ref={ref}
      className={className}
      aria-labelledby={labelledBy}
      onCancel={(e) => {
        e.preventDefault();
        requestClose.current();
      }}
      onClose={() => {
        const dlg = ref.current;
        // Браузер может закрыть диалог сам (повторный Esc); тогда спрашиваем владельца и
        // открываем диалог снова, если он решил его не закрывать.
        if (unmounting.current || !dlg || dlg.open) return;
        requestClose.current();
        window.requestAnimationFrame(() => {
          if (dlg.isConnected && !dlg.open && !unmounting.current) dlg.showModal();
        });
      }}
    >
      {children}
    </dialog>
  );
}
