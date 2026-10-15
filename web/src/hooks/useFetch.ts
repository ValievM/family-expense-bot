import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiError } from '../api/client';

/** Состояние асинхронной загрузки данных. */
export interface FetchState<T> {
  data: T | null;
  loading: boolean;
  error: ApiError | null;
  reload: () => void;
}

/**
 * Принимает: функцию загрузки и список зависимостей, при смене которых нужно перезагрузить.
 * Делает: запускает загрузку, игнорирует устаревшие ответы и хранит data/loading/error.
 * Возвращает: FetchState с функцией reload для ручного обновления.
 */
export function useFetch<T>(load: () => Promise<T>, deps: unknown[]): FetchState<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [tick, setTick] = useState(0);
  const seq = useRef(0);

  useEffect(() => {
    const id = ++seq.current;
    setLoading(true);
    load()
      .then((d) => {
        if (id !== seq.current) return;
        setData(d);
        setError(null);
      })
      .catch((e: unknown) => {
        if (id !== seq.current) return;
        setError(e instanceof ApiError ? e : new ApiError(0, 'unknown', String(e)));
      })
      .finally(() => {
        if (id === seq.current) setLoading(false);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, loading, error, reload };
}
