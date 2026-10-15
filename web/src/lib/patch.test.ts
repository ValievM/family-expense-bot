import { describe, expect, it } from 'vitest';
import type { Expense } from '../api/types';
import { buildPatch } from './patch';

const original: Expense = {
  id: 1,
  user_id: 5,
  amount: 35000,
  category: 'food',
  category_label: 'еда',
  spent_at: '2026-01-10',
  comment: 'хлеб',
  created_at: '',
  updated_at: '',
};
const same = { userId: 5, amount: 35000, category: 'food' as const, spentAt: '2026-01-10', comment: 'хлеб' };

describe('buildPatch', () => {
  it('is empty when nothing changed', () => {
    expect(buildPatch(original, same)).toEqual({});
  });

  it('contains only changed fields', () => {
    expect(buildPatch(original, { ...same, amount: 100 })).toEqual({ amount: 100 });
    expect(buildPatch(original, { ...same, category: 'health', comment: '' })).toEqual({
      category: 'health',
      comment: '',
    });
  });
});
