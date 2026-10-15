import type { ListParams } from '../api/types';

/** Состояние фильтров; пустая строка означает "все". */
export type Filters = ListParams;

/** Фильтры по умолчанию (ничего не отфильтровано). */
export const EMPTY_FILTERS: Filters = { user_id: '', category: '', from: '', to: '' };

const KEYS = Object.keys(EMPTY_FILTERS) as (keyof Filters)[];

/**
 * Принимает: query-строку (с "?" или без).
 * Делает: достаёт из неё известные параметры фильтра.
 * Возвращает: объект Filters; отсутствующие параметры пустые.
 */
export function filtersFromQuery(search: string): Filters {
  const q = new URLSearchParams(search);
  const out = { ...EMPTY_FILTERS };
  for (const k of KEYS) out[k] = q.get(k) ?? '';
  return out;
}

/**
 * Принимает: объект Filters.
 * Делает: сериализует непустые значения в query-строку.
 * Возвращает: строку вида "?user_id=1" или пустую строку.
 */
export function filtersToQuery(filters: Filters): string {
  const q = new URLSearchParams();
  for (const k of KEYS) if (filters[k] !== '') q.set(k, filters[k]);
  const s = q.toString();
  return s === '' ? '' : `?${s}`;
}

/**
 * Принимает: объект Filters.
 * Делает: проверяет, что хотя бы один фильтр задан.
 * Возвращает: true, если есть активный фильтр.
 */
export function hasActiveFilters(filters: Filters): boolean {
  return KEYS.some((k) => filters[k] !== '');
}
