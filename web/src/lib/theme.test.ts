import { describe, expect, it } from 'vitest';
import { readStoredTheme, resolveTheme, storeTheme, THEME_KEY } from './theme';
import type { StorageLike } from './storage';

const throwing: StorageLike = {
  getItem() {
    throw new Error('SecurityError');
  },
  setItem() {
    throw new Error('SecurityError');
  },
};

describe('theme', () => {
  it('prefers the stored choice over the system setting', () => {
    expect(resolveTheme('light', true)).toBe('light');
    expect(resolveTheme(null, true)).toBe('dark');
    expect(resolveTheme(null, false)).toBe('light');
  });

  it('stores and reads back a valid theme only', () => {
    const data = new Map<string, string>();
    const s: StorageLike = { getItem: (k) => data.get(k) ?? null, setItem: (k, v) => void data.set(k, v) };
    expect(storeTheme(s, 'dark')).toBe(true);
    expect(readStoredTheme(s)).toBe('dark');
    data.set(THEME_KEY, 'sepia');
    expect(readStoredTheme(s)).toBeNull();
  });

  it('works without storage', () => {
    expect(readStoredTheme(throwing)).toBeNull();
    expect(storeTheme(throwing, 'dark')).toBe(false);
    expect(readStoredTheme(null)).toBeNull();
  });
});
