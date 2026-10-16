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
