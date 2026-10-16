import type { FormField } from './expenseForm';

export type ErrorTarget =
  | { kind: 'field'; field: FormField }
  | { kind: 'period' }
  | { kind: 'notFound' }
  | { kind: 'banner' };

const FIELD_BY_CODE: Record<string, FormField> = {
  invalid_amount: 'amount',
  invalid_user: 'userId',
  unknown_category: 'category',
  invalid_date: 'spentAt',
  comment_too_long: 'comment',
};

export const PERIOD_MESSAGE = 'Дата «с» должна быть не позже даты «по».';
export const NOT_FOUND_MESSAGE = 'Запись не найдена, список обновлён';

/**
 * Принимает: код ошибки сервера.
 * Делает: решает, где показать ошибку: у поля, у фильтра периода, тостом или баннером.
 * Возвращает: цель показа; неизвестные коды уходят в общий баннер.
 */
export function errorTarget(code: string): ErrorTarget {
  const field = FIELD_BY_CODE[code];
  if (field) return { kind: 'field', field };
  if (code === 'invalid_period') return { kind: 'period' };
  if (code === 'not_found') return { kind: 'notFound' };
  return { kind: 'banner' };
}

/**
 * Принимает: код ошибки.
 * Делает: подбирает спокойный текст без технических терминов.
 * Возвращает: текст общего баннера.
 */
export function bannerMessage(code: string): string {
  switch (code) {
    case 'network':
      return 'Нет связи с сервером. Проверьте соединение и попробуйте ещё раз.';
    case 'internal':
      return 'Что-то пошло не так. Попробуйте ещё раз.';
    default:
      return 'Не удалось выполнить запрос. Попробуйте ещё раз.';
  }
}

/** Коды, у которых полезно показать технические детали в свёрнутом виде. */
export function hasTechnicalDetails(code: string): boolean {
  return code !== 'network' && code !== 'internal';
}
