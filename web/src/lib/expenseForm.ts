import type { CategoryKey, Expense } from '../api/types';
import { kopecksToInput, parseRubles } from './money';
import type { ExpenseValues } from './patch';

export interface ExpenseFormState {
  userId: string;
  amount: string;
  category: string;
  spentAt: string;
  comment: string;
}

export type FormErrors = Partial<Record<keyof ExpenseFormState, string>>;

function todayISO(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function initialForm(expense: Expense | null): ExpenseFormState {
  if (!expense) {
    return { userId: '', amount: '', category: '', spentAt: todayISO(), comment: '' };
  }
  return {
    userId: String(expense.user_id),
    amount: kopecksToInput(expense.amount),
    category: expense.category,
    spentAt: expense.spent_at,
    comment: expense.comment,
  };
}

/**
 * Принимает: значения формы и допустимые ключи категорий.
 * Делает: повторяет проверки API (user_id > 0, сумма > 0, категория, дата).
 * Возвращает: {values} или {errors}.
 */
export function validateForm(
  form: ExpenseFormState,
  categoryKeys: string[],
): { values: ExpenseValues; errors?: undefined } | { errors: FormErrors; values?: undefined } {
  const errors: FormErrors = {};
  const userId = /^\d+$/.test(form.userId.trim()) ? Number(form.userId.trim()) : 0;
  if (!Number.isSafeInteger(userId) || userId <= 0) {
    errors.userId = 'Укажите Telegram ID — целое число больше нуля';
  }
  const amount = parseRubles(form.amount);
  if (amount === null) errors.amount = 'Введите сумму больше нуля, например 350 или 350,50';
  if (!categoryKeys.includes(form.category)) errors.category = 'Выберите категорию';
  if (!/^\d{4}-\d{2}-\d{2}$/.test(form.spentAt)) errors.spentAt = 'Укажите дату';
  if (Object.keys(errors).length > 0 || amount === null) return { errors };
  return {
    values: {
      userId,
      amount,
      category: form.category as CategoryKey,
      spentAt: form.spentAt,
      comment: form.comment.trim(),
    },
  };
}
