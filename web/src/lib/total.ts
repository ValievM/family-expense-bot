import type { Expense } from '../api/types';

/**
 * Принимает: список расходов.
 * Делает: суммирует поле amount в копейках.
 * Возвращает: итог в копейках.
 */
export function totalKopecks(items: Expense[]): number {
  return items.reduce((sum, e) => sum + e.amount, 0);
}
