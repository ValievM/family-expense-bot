import { describe, expect, it } from 'vitest';
import { validateForm } from './expenseForm';
import { totalKopecks } from './total';
import type { Expense } from '../api/types';

const keys = ['food', 'other'];
const ok = { userId: '12', amount: '350,5', category: 'food', spentAt: '2026-01-10', comment: ' x ' };

describe('validateForm', () => {
  it('returns parsed values', () => {
    expect(validateForm(ok, keys).values).toEqual({
      userId: 12,
      amount: 35050,
      category: 'food',
      spentAt: '2026-01-10',
      comment: 'x',
    });
  });

  it('reports field errors', () => {
    const r = validateForm({ ...ok, userId: '0', amount: '0', category: '', spentAt: '' }, keys);
    expect(Object.keys(r.errors ?? {}).sort()).toEqual(['amount', 'category', 'spentAt', 'userId']);
  });
});

describe('totalKopecks', () => {
  it('sums amounts', () => {
    const items = [{ amount: 100 }, { amount: 250 }] as Expense[];
    expect(totalKopecks(items)).toBe(350);
  });
});
