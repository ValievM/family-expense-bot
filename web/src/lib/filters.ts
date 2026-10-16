import type { ListParams } from '../api/types';

/** Пустая строка означает "все". */
export type Filters = ListParams;

export const EMPTY_FILTERS: Filters = { user_id: '', category: '', from: '', to: '' };

const KEYS = Object.keys(EMPTY_FILTERS) as (keyof Filters)[];

/**
 * Принимает: query-строка (с "?" или без).
 * Делает: достаёт известные параметры фильтра.
 * Возвращает: Filters; отсутствующие параметры пустые.
 */
export function filtersFromQuery(search: string): Filters {
  const q = new URLSearchParams(search);
  const out = { ...EMPTY_FILTERS };
  for (const k of KEYS) out[k] = q.get(k) ?? '';
  return out;
}

/**
 * Принимает: Filters.
 * Делает: сериализует непустые значения.
 * Возвращает: "?user_id=1" или пустую строку.
 */
export function filtersToQuery(filters: Filters): string {
  const q = new URLSearchParams();
  for (const k of KEYS) if (filters[k] !== '') q.set(k, filters[k]);
  const s = q.toString();
  return s === '' ? '' : `?${s}`;
}

export function hasActiveFilters(filters: Filters): boolean {
  return KEYS.some((k) => filters[k] !== '');
}

/**
 * Принимает: трату и фильтры.
 * Делает: повторяет серверную фильтрацию (границы периода включительно).
 * Возвращает: true, если трата попадёт в текущую выборку.
 */
export function matchesFilters(
  e: { user_id: number; category: string; spent_at: string },
  filters: Filters,
): boolean {
  if (filters.user_id !== '' && String(e.user_id) !== filters.user_id) return false;
  if (filters.category !== '' && e.category !== filters.category) return false;
  if (filters.from !== '' && e.spent_at < filters.from) return false;
  if (filters.to !== '' && e.spent_at > filters.to) return false;
  return true;
}
