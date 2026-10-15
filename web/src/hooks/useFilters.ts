import { useCallback, useEffect, useState } from 'react';
import { EMPTY_FILTERS, filtersFromQuery, filtersToQuery, type Filters } from '../lib/filters';

/**
 * Принимает: ничего.
 * Делает: хранит фильтры, читает их из query-строки и записывает обратно через history.replaceState; следит за popstate.
 * Возвращает: кортеж [фильтры, изменить одно поле, сбросить все].
 */
export function useFilters(): [Filters, (patch: Partial<Filters>) => void, () => void] {
  const [filters, setFilters] = useState<Filters>(() => filtersFromQuery(window.location.search));

  useEffect(() => {
    const onPop = () => setFilters(filtersFromQuery(window.location.search));
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  useEffect(() => {
    const url = `${window.location.pathname}${filtersToQuery(filters)}`;
    window.history.replaceState(null, '', url);
  }, [filters]);

  const update = useCallback((patch: Partial<Filters>) => setFilters((f) => ({ ...f, ...patch })), []);
  const reset = useCallback(() => setFilters(EMPTY_FILTERS), []);
  return [filters, update, reset];
}
