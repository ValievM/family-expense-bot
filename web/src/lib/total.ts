import type { Expense } from '../api/types';

/**
 * Принимает: список расходов.
 * Делает: суммирует amount.
 * Возвращает: итог в копейках.
 */
export function totalKopecks(items: Expense[]): number {
  return items.reduce((sum, e) => sum + e.amount, 0);
}
