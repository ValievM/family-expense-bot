import type { CategoryKey, Expense } from '../api/types';
import { todayISO } from './date';
import { kopecksToInput, parseRubles } from './money';
import type { ExpenseValues } from './patch';

export interface ExpenseFormState {
  userId: string;
  amount: string;
  category: string;
  spentAt: string;
  comment: string;
}

export type FormField = keyof ExpenseFormState;
export type FormErrors = Partial<Record<FormField, string>>;

/** Совпадает с ограничениями домена на сервере. */
export const MAX_AMOUNT_KOPECKS = 1_000_000_000 * 100;
export const MAX_COMMENT_LENGTH = 500;

export const FIELD_MESSAGES: Record<FormField, string> = {
  userId: 'Введите Telegram ID: целое положительное число.',
  amount: 'Введите сумму от 0,01 до 1 000 000 000 ₽, не больше двух знаков после запятой.',
  category: 'Выберите категорию.',
  spentAt: 'Выберите дату траты.',
  comment: `Комментарий может быть не длиннее ${MAX_COMMENT_LENGTH} символов.`,
};

export function commentLength(comment: string): number {
  return Array.from(comment).length;
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
 * Делает: повторяет проверки API (user_id > 0, сумма в пределах, категория, дата, длина комментария).
 * Возвращает: {values} или {errors}.
 */
export function validateForm(
  form: ExpenseFormState,
  categoryKeys: string[],
): { values: ExpenseValues; errors?: undefined } | { errors: FormErrors; values?: undefined } {
  const errors: FormErrors = {};
  const userId = /^\d+$/.test(form.userId.trim()) ? Number(form.userId.trim()) : 0;
  if (!Number.isSafeInteger(userId) || userId <= 0) errors.userId = FIELD_MESSAGES.userId;
  const amount = parseRubles(form.amount);
  if (amount === null || amount > MAX_AMOUNT_KOPECKS) errors.amount = FIELD_MESSAGES.amount;
  if (!categoryKeys.includes(form.category)) errors.category = FIELD_MESSAGES.category;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(form.spentAt)) errors.spentAt = FIELD_MESSAGES.spentAt;
  const comment = form.comment.trim();
  if (commentLength(comment) > MAX_COMMENT_LENGTH) errors.comment = FIELD_MESSAGES.comment;
  if (Object.keys(errors).length > 0 || amount === null) return { errors };
  return {
    values: {
      userId,
      amount,
      category: form.category as CategoryKey,
      spentAt: form.spentAt,
      comment,
    },
  };
}
