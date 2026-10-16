import { describe, expect, it } from 'vitest';
import { categoryIconPath, FALLBACK_ICON_PATH } from './categoryIcons';

describe('categoryIconPath', () => {
  it('has a distinct icon for every known category', () => {
    const keys = ['food', 'transport', 'housing', 'entertainment', 'health', 'shopping', 'other'];
    const paths = keys.map(categoryIconPath);
    expect(new Set(paths).size).toBe(keys.length);
    expect(paths).not.toContain(FALLBACK_ICON_PATH);
  });

  it.each(['pets', '', 'toString', '__proto__'])('falls back for unknown key %j', (key) => {
    expect(categoryIconPath(key)).toBe(FALLBACK_ICON_PATH);
  });
});
