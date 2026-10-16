import type { Expense } from '../api/types';
import { addDays, MONTHS_GENITIVE, parseISODate } from './date';

export interface DayGroup {
  date: string;
  items: Expense[];
  total: number;
}

export interface DayLabel {
  day: string;
  month: string;
  year: string | null;
  relative: 'сегодня' | 'вчера' | null;
}

/**
 * Принимает: траты в любом порядке.
 * Делает: сортирует новые сверху (день, затем id по убыванию) и группирует по дню.
 * Возвращает: группы дней с суммой за день в копейках.
 */
export function groupByDay(items: Expense[]): DayGroup[] {
  const sorted = [...items].sort((a, b) =>
    a.spent_at === b.spent_at ? b.id - a.id : a.spent_at < b.spent_at ? 1 : -1,
  );
  const groups: DayGroup[] = [];
  for (const e of sorted) {
    const last = groups[groups.length - 1];
    if (last && last.date === e.spent_at) {
      last.items.push(e);
      last.total += e.amount;
    } else {
      groups.push({ date: e.spent_at, items: [e], total: e.amount });
    }
  }
  return groups;
}

/**
 * Принимает: дату дня и «сегодня» (ГГГГ-ММ-ДД).
 * Делает: раскладывает дату для заголовка на полях; год только если он не текущий.
 * Возвращает: число, месяц в родительном падеже, год и «сегодня»/«вчера», если подходит.
 */
export function dayLabel(iso: string, today: string): DayLabel {
  const p = parseISODate(iso);
  const t = parseISODate(today);
  if (!p) return { day: iso, month: '', year: null, relative: null };
  const relative = iso === today ? 'сегодня' : iso === addDays(today, -1) ? 'вчера' : null;
  return {
    day: String(p.day),
    month: MONTHS_GENITIVE[p.month - 1],
    year: t && t.year === p.year ? null : String(p.year),
    relative,
  };
}
