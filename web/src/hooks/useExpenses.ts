import { fetchExpenses } from '../api/client';
import type { Expense } from '../api/types';
import type { Filters } from '../lib/filters';
import { useFetch, type FetchState } from './useFetch';

/**
 * Принимает: текущие фильтры.
 * Делает: загружает расходы и перезагружает их при смене фильтров.
 * Возвращает: FetchState со списком расходов.
 */
export function useExpenses(filters: Filters): FetchState<Expense[]> {
  return useFetch(
    () => fetchExpenses(filters),
    [filters.user_id, filters.category, filters.from, filters.to],
  );
}
