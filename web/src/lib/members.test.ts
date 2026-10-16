import { describe, expect, it } from 'vitest';
import {
  loadNames,
  memberHue,
  memberLabel,
  memberOptionLabel,
  monogram,
  NAMES_KEY,
  saveNames,
  shortId,
  withName,
} from './members';
import type { StorageLike } from './storage';

class MemoryStorage implements StorageLike {
  data = new Map<string, string>();
  getItem(k: string) {
    return this.data.get(k) ?? null;
  }
  setItem(k: string, v: string) {
    this.data.set(k, v);
  }
}

const throwing: StorageLike = {
  getItem() {
    throw new Error('SecurityError');
  },
  setItem() {
    throw new Error('QuotaExceededError');
  },
};

describe('member display', () => {
  it('shortens long IDs to the last four digits', () => {
    expect(shortId(6612214276)).toBe('…4276');
    expect(shortId(42)).toBe('42');
  });

  it('gives the same hue for the same ID and stays in range', () => {
    expect(memberHue(6612214276)).toBe(memberHue(6612214276));
    expect(memberHue(6612214276)).not.toBe(memberHue(5241803182));
    for (const id of [1, 42, 6612214276, 9007199254740991]) {
      const h = memberHue(id);
      expect(h).toBeGreaterThanOrEqual(0);
      expect(h).toBeLessThan(360);
    }
  });

  it.each([
    [6612214276, undefined, '76'],
    [6612214276, 'мама', 'М'],
    [6612214276, ' Анна  Петрова ', 'АП'],
  ])('monogram(%i, %s) = %s', (id, name, want) => {
    expect(monogram(id, name)).toBe(want);
  });

  it('prefers the local name in labels', () => {
    const names = { '6612214276': 'Мама' };
    expect(memberLabel(6612214276, names)).toBe('Мама');
    expect(memberLabel(5241803182, names)).toBe('…3182');
    expect(memberOptionLabel(6612214276, names)).toBe('Мама · …4276');
    expect(memberOptionLabel(5241803182, names)).toBe('ID …3182');
  });
});

describe('local names storage', () => {
  it('round-trips names', () => {
    const s = new MemoryStorage();
    const names = withName({}, 6612214276, '  Мама ');
    expect(saveNames(s, names)).toBe(true);
    expect(loadNames(s)).toEqual({ '6612214276': 'Мама' });
  });

  it('removes a name when it is cleared and trims long names', () => {
    expect(withName({ '1': 'A' }, 1, '   ')).toEqual({});
    expect(withName({}, 1, 'я'.repeat(60))['1']).toHaveLength(40);
  });

  it('survives a storage that throws', () => {
    expect(loadNames(throwing)).toEqual({});
    expect(saveNames(throwing, { '1': 'A' })).toBe(false);
    expect(loadNames(null)).toEqual({});
    expect(saveNames(null, {})).toBe(false);
  });

  it('ignores corrupted data', () => {
    const s = new MemoryStorage();
    s.setItem(NAMES_KEY, '{not json');
    expect(loadNames(s)).toEqual({});
    s.setItem(NAMES_KEY, JSON.stringify({ '1': 'A', x: 'B', '2': 5, '3': ' ' }));
    expect(loadNames(s)).toEqual({ '1': 'A' });
    s.setItem(NAMES_KEY, '[]');
    expect(loadNames(s)).toEqual({});
  });
});
