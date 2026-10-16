import { safeGet, safeSet, type StorageLike } from './storage';

export type Theme = 'light' | 'dark';

/** Тот же ключ читает встроенный скрипт в index.html до загрузки React. */
export const THEME_KEY = 'family-theme';

export function isTheme(value: unknown): value is Theme {
  return value === 'light' || value === 'dark';
}

export function readStoredTheme(storage: StorageLike | null): Theme | null {
  const v = safeGet(storage, THEME_KEY);
  return isTheme(v) ? v : null;
}

export function storeTheme(storage: StorageLike | null, theme: Theme): boolean {
  return safeSet(storage, THEME_KEY, theme);
}

/**
 * Принимает: сохранённый выбор (или null) и признак системной тёмной темы.
 * Делает: явный выбор пользователя важнее системной настройки.
 * Возвращает: тему для страницы.
 */
export function resolveTheme(stored: Theme | null, prefersDark: boolean): Theme {
  return stored ?? (prefersDark ? 'dark' : 'light');
}
