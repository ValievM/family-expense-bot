import { useCallback, useEffect, useRef, useState } from 'react';

export interface ToastMessage {
  id: number;
  text: string;
}

const TOAST_MS = 3500;

/**
 * Принимает: ничего.
 * Делает: показывает одно сообщение за раз и прячет его через 3,5 секунды.
 * Возвращает: {toast, show}.
 */
export function useToast(): { toast: ToastMessage | null; show: (text: string) => void } {
  const [toast, setToast] = useState<ToastMessage | null>(null);
  const timer = useRef<number | undefined>(undefined);
  const seq = useRef(0);

  const show = useCallback((text: string) => {
    window.clearTimeout(timer.current);
    seq.current += 1;
    setToast({ id: seq.current, text });
    timer.current = window.setTimeout(() => setToast(null), TOAST_MS);
  }, []);

  useEffect(() => () => window.clearTimeout(timer.current), []);
  return { toast, show };
}
