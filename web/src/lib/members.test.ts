import { describe, expect, it } from 'vitest';
import type { Member } from '../api/types';
import {
  buildDirectory,
  loadNames,
  memberDetails,
  memberHue,
  memberView,
  monogram,
  normalizeMember,
  pickLabel,
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
    expect(shortId(1234567890)).toBe('…7890');
    expect(shortId(42)).toBe('42');
  });

  it('gives the same hue for the same ID and stays in range', () => {
    expect(memberHue(1234567890)).toBe(memberHue(1234567890));
    expect(memberHue(1234567890)).not.toBe(memberHue(5241803182));
    for (const id of [1, 42, 1234567890, 9007199254740991]) {
      const h = memberHue(id);
      expect(h).toBeGreaterThanOrEqual(0);
      expect(h).toBeLessThan(360);
    }
  });

  it.each([
    [1234567890, undefined, '90'],
    [1234567890, '', '90'],
    [1234567890, 'мама', 'М'],
    [1234567890, ' Анна  Петрова ', 'АП'],
    [1234567890, 'Анна Мария Петрова', 'АМ'],
    [1234567890, '@valiev', 'V'],
    [1234567890, '@ivan_petrov', 'IP'],
    [1234567890, '🐱 2', '90'],
  ])('monogram(%i, %s) = %s', (id, name, want) => {
    expect(monogram(id, name)).toBe(want);
  });
});

const member = (user_id: number, name = '', username = ''): Member => ({ user_id, name, username });

describe('pickLabel', () => {
  it.each([
    ['local name wins over everything', 'Мама', member(1, 'Анна', 'anna'), 'Мама', 'local'],
    ['server name when no local name', undefined, member(1, 'Анна', 'anna'), 'Анна', 'telegram'],
    ['blank local name is ignored', '  ', member(1, 'Анна', 'anna'), 'Анна', 'telegram'],
    ['username when no name', undefined, member(1, '', 'anna'), '@anna', 'username'],
    ['short ID when nothing is known', undefined, member(1234567890), '…7890', 'id'],
    ['short ID when the member is missing', undefined, undefined, '…7890', 'id'],
  ] as const)('%s', (_, local, m, label, source) => {
    expect(pickLabel(m?.user_id ?? 1234567890, local, m)).toEqual({ label, source });
  });
});

describe('normalizeMember', () => {
  it('fills missing or wrong fields with empty strings (old server)', () => {
    expect(normalizeMember({ user_id: 42 })).toEqual(member(42));
    expect(normalizeMember({ user_id: 42, name: null, username: 7 })).toEqual(member(42));
  });

  it('trims and strips a leading @ from username', () => {
    expect(normalizeMember({ user_id: 42, name: ' Магомед ', username: '@valiev' })).toEqual(
      member(42, 'Магомед', 'valiev'),
    );
  });

  it.each([null, 'x', {}, { user_id: '42' }, { user_id: 0 }, { user_id: 1.5 }])('rejects %j', (raw) => {
    expect(normalizeMember(raw)).toBeNull();
  });
});

describe('buildDirectory', () => {
  it('labels members by priority and keeps the ID as details', () => {
    const dir = buildDirectory([], [member(1234567890, 'Магомед', 'valiev'), member(5241803182)], {});
    expect(memberView(dir, 1234567890)).toEqual({
      label: 'Магомед',
      name: 'Магомед',
      details: 'Telegram ID 1234567890 · @valiev',
    });
    expect(memberView(dir, 5241803182)).toEqual({ label: '…3182', name: '', details: 'Telegram ID 5241803182' });
  });

  it('includes people known only from expenses and falls back for unknown IDs', () => {
    const dir = buildDirectory([777], [], { '777': 'Бабушка' });
    expect(memberView(dir, 777).label).toBe('Бабушка');
    expect(memberView(dir, 9990001234)).toEqual({ label: '…1234', name: '', details: 'Telegram ID 9990001234' });
  });

  it('appends the short ID when labels collide, case-insensitively', () => {
    const dir = buildDirectory([], [member(1234567890, 'Анна'), member(5241803182, 'анна'), member(1, 'Олег')], {});
    expect(memberView(dir, 1234567890).label).toBe('Анна · …7890');
    expect(memberView(dir, 5241803182).label).toBe('анна · …3182');
    expect(memberView(dir, 1).label).toBe('Олег');
  });

  it('detects a collision between a local name and a server name', () => {
    const dir = buildDirectory([], [member(1234567890, 'Анна'), member(5241803182, 'Мама')], { '5241803182': 'Анна' });
    expect(memberView(dir, 1234567890).label).toBe('Анна · …7890');
    expect(memberView(dir, 5241803182).label).toBe('Анна · …3182');
  });

  it('uses the full ID when the short IDs collide too', () => {
    const dir = buildDirectory([], [member(1110004276, 'Анна'), member(2220004276, 'Анна'), member(3330004276), member(4440004276)], {});
    expect(memberView(dir, 1110004276).label).toBe('Анна · 1110004276');
    expect(memberView(dir, 2220004276).label).toBe('Анна · 2220004276');
    expect(memberView(dir, 3330004276).label).toBe('3330004276');
    expect(memberView(dir, 4440004276).label).toBe('4440004276');
  });

  it('keeps the plain name for the avatar even when the label is disambiguated', () => {
    const dir = buildDirectory([], [member(1234567890, 'Анна Петрова'), member(5241803182, 'Анна Петрова')], {});
    const view = memberView(dir, 1234567890);
    expect(monogram(1234567890, view.name)).toBe('АП');
  });

  it('builds details without username', () => {
    expect(memberDetails(42)).toBe('Telegram ID 42');
    expect(memberDetails(42, { username: 'x' })).toBe('Telegram ID 42 · @x');
  });
});

describe('local names storage', () => {
  it('round-trips names', () => {
    const s = new MemoryStorage();
    const names = withName({}, 1234567890, '  Мама ');
    expect(saveNames(s, names)).toBe(true);
    expect(loadNames(s)).toEqual({ '1234567890': 'Мама' });
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
