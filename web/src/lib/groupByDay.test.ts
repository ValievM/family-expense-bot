import { describe, expect, it } from 'vitest';
import type { Expense } from '../api/types';
import { dayLabel, groupByDay } from './groupByDay';

const e = (id: number, spent_at: string, amount: number): Expense => ({
  id,
  user_id: 1,
  amount,
  category: 'food',
  category_label: 'еда',
  spent_at,
  comment: '',
  created_at: '',
  updated_at: '',
});

describe('groupByDay', () => {
  it('groups by day with per-day totals, newest first', () => {
    const groups = groupByDay([
      e(3, '2026-10-15', 100),
      e(5, '2026-10-16', 250),
      e(4, '2026-10-16', 50),
      e(6, '2026-10-15', 1),
    ]);
    expect(groups.map((g) => g.date)).toEqual(['2026-10-16', '2026-10-15']);
    expect(groups.map((g) => g.total)).toEqual([300, 101]);
    expect(groups[0].items.map((x) => x.id)).toEqual([5, 4]);
    expect(groups[1].items.map((x) => x.id)).toEqual([6, 3]);
  });

  it('returns no groups for an empty list', () => {
    expect(groupByDay([])).toEqual([]);
  });
});

describe('dayLabel', () => {
  const today = '2026-10-16';

  it.each([
    ['2026-10-16', { day: '16', month: 'октября', year: null, relative: 'сегодня' }],
    ['2026-10-15', { day: '15', month: 'октября', year: null, relative: 'вчера' }],
    ['2026-10-01', { day: '1', month: 'октября', year: null, relative: null }],
    ['2025-12-31', { day: '31', month: 'декабря', year: '2025', relative: null }],
  ])('%s', (iso, want) => {
    expect(dayLabel(iso, today)).toEqual(want);
  });

  it('says «вчера» across a month boundary', () => {
    expect(dayLabel('2026-09-30', '2026-10-01').relative).toBe('вчера');
  });
});
