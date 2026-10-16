import type { CategoryKey, Expense, UpdateExpense } from '../api/types';

export interface ExpenseValues {
  userId: number;
  amount: number;
  category: CategoryKey;
  spentAt: string;
  comment: string;
}

/**
 * Принимает: исходный расход и новые значения формы.
 * Делает: кладёт в патч только изменившиеся поля.
 * Возвращает: UpdateExpense (пустой, если ничего не менялось).
 */
export function buildPatch(original: Expense, next: ExpenseValues): UpdateExpense {
  const patch: UpdateExpense = {};
  if (next.amount !== original.amount) patch.amount = next.amount;
  if (next.category !== original.category) patch.category = next.category;
  if (next.spentAt !== original.spent_at) patch.spent_at = next.spentAt;
  if (next.comment !== original.comment) patch.comment = next.comment;
  return patch;
}
