import { useCallback, useEffect, useState } from 'react';
import { browserStorage } from '../lib/storage';
import { isTheme, readStoredTheme, resolveTheme, storeTheme, type Theme } from '../lib/theme';

const DARK_QUERY = '(prefers-color-scheme: dark)';

function systemPrefersDark(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(DARK_QUERY).matches;
}

function initialTheme(): Theme {
  const preset = document.documentElement.dataset.theme;
  return isTheme(preset) ? preset : resolveTheme(readStoredTheme(browserStorage()), systemPrefersDark());
}

/**
 * Принимает: ничего.
 * Делает: держит data-theme и meta color-scheme в синхроне; без явного выбора следует за системой.
 * Возвращает: [текущая тема, переключить и запомнить выбор].
 */
export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(initialTheme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.querySelector('meta[name="color-scheme"]')?.setAttribute('content', theme);
  }, [theme]);

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const mq = window.matchMedia(DARK_QUERY);
    const onChange = () => {
      if (readStoredTheme(browserStorage()) === null) setTheme(mq.matches ? 'dark' : 'light');
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  const toggle = useCallback(() => {
    const next: Theme = theme === 'dark' ? 'light' : 'dark';
    storeTheme(browserStorage(), next);
    setTheme(next);
  }, [theme]);

  return [theme, toggle];
}
