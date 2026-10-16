import { useCallback, useEffect, useRef, useState } from 'react';

// С запасом на перезапрос списка: строка может появиться позже, чем закончится анимация.
const HIGHLIGHT_MS = 2500;

/**
 * Принимает: ничего.
 * Делает: помнит id изменённой записи, пока идёт подсветка строки.
 * Возвращает: [id подсвеченной записи или null, подсветить(id)].
 */
export function useHighlight(): [number | null, (id: number) => void] {
  const [id, setId] = useState<number | null>(null);
  const timer = useRef<number | undefined>(undefined);

  const flash = useCallback((next: number) => {
    window.clearTimeout(timer.current);
    setId(next);
    timer.current = window.setTimeout(() => setId(null), HIGHLIGHT_MS);
  }, []);

  useEffect(() => () => window.clearTimeout(timer.current), []);
  return [id, flash];
}
