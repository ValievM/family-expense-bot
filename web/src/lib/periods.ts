import {
  addDays,
  daysInMonth,
  isoFromParts,
  MONTHS_GENITIVE,
  MONTHS_NOMINATIVE,
  parseISODate,
  weekdayMondayFirst,
} from './date';

export type PeriodPreset =
  | 'month'
  | 'prevMonth'
  | 'today'
  | 'yesterday'
  | 'week'
  | 'last30'
  | 'year'
  | 'custom'
  | 'all';

export interface PeriodRange {
  from: string;
  to: string;
}

export const PERIOD_OPTIONS: { value: PeriodPreset; label: string }[] = [
  { value: 'month', label: 'Этот месяц' },
  { value: 'prevMonth', label: 'Прошлый месяц' },
  { value: 'today', label: 'Сегодня' },
  { value: 'yesterday', label: 'Вчера' },
  { value: 'week', label: 'Эта неделя' },
  { value: 'last30', label: 'Последние 30 дней' },
  { value: 'year', label: 'Год' },
  { value: 'custom', label: 'Свой период' },
  { value: 'all', label: 'Всё время' },
];

/**
 * Принимает: пресет периода и «сегодня» в виде ГГГГ-ММ-ДД.
 * Делает: переводит пресет в границы from/to включительно; неделя начинается с понедельника.
 * Возвращает: {from, to}; пустые строки для «всё время»; null для «свой период».
 */
export function presetRange(preset: PeriodPreset, today: string): PeriodRange | null {
  const t = parseISODate(today);
  if (!t) return null;
  switch (preset) {
    case 'today':
      return { from: today, to: today };
    case 'yesterday': {
      const y = addDays(today, -1);
      return { from: y, to: y };
    }
    case 'week': {
      const monday = addDays(today, -weekdayMondayFirst(today));
      return { from: monday, to: addDays(monday, 6) };
    }
    case 'month':
      return monthRange(t.year, t.month);
    case 'prevMonth':
      return t.month === 1 ? monthRange(t.year - 1, 12) : monthRange(t.year, t.month - 1);
    case 'last30':
      return { from: addDays(today, -29), to: today };
    case 'year':
      return { from: isoFromParts(t.year, 1, 1), to: isoFromParts(t.year, 12, 31) };
    case 'all':
      return { from: '', to: '' };
    case 'custom':
      return null;
  }
}

function monthRange(year: number, month: number): PeriodRange {
  return { from: isoFromParts(year, month, 1), to: isoFromParts(year, month, daysInMonth(year, month)) };
}

/**
 * Принимает: границы периода и «сегодня».
 * Делает: ищет пресет с точно такими же границами.
 * Возвращает: найденный пресет или 'custom'.
 */
export function detectPreset(from: string, to: string, today: string): PeriodPreset {
  for (const { value } of PERIOD_OPTIONS) {
    const r = presetRange(value, today);
    if (r && r.from === from && r.to === to) return value;
  }
  return 'custom';
}

function dayMonth(iso: string): string {
  const p = parseISODate(iso);
  return p ? `${p.day} ${MONTHS_GENITIVE[p.month - 1]}` : iso;
}

function dayMonthYear(iso: string): string {
  const p = parseISODate(iso);
  return p ? `${dayMonth(iso)} ${p.year}` : iso;
}

/**
 * Принимает: границы периода (каждая может быть пустой).
 * Делает: собирает подпись периода по-русски, сокращая общий месяц и год.
 * Возвращает: «1–31 октября 2026», «28 сентября — 4 октября 2026», «За всё время» и т. п.
 */
export function formatRange(from: string, to: string): string {
  if (!from && !to) return 'За всё время';
  if (!to) return `С ${dayMonthYear(from)}`;
  if (!from) return `По ${dayMonthYear(to)}`;
  if (from === to) return dayMonthYear(from);
  const a = parseISODate(from);
  const b = parseISODate(to);
  if (!a || !b) return `${from} — ${to}`;
  if (a.year === b.year && a.month === b.month) {
    return `${a.day}–${b.day} ${MONTHS_GENITIVE[a.month - 1]} ${a.year}`;
  }
  if (a.year === b.year) return `${dayMonth(from)} — ${dayMonth(to)} ${a.year}`;
  return `${dayMonthYear(from)} — ${dayMonthYear(to)}`;
}

/**
 * Принимает: границы периода.
 * Делает: для целого месяца или года даёт короткую подпись, иначе как formatRange.
 * Возвращает: текст чипа фильтра, например «Октябрь 2026» или «2026 год».
 */
export function periodChipLabel(from: string, to: string): string {
  const a = parseISODate(from);
  const b = parseISODate(to);
  if (a && b && a.year === b.year && a.day === 1) {
    if (a.month === b.month && b.day === daysInMonth(b.year, b.month)) {
      return `${MONTHS_NOMINATIVE[a.month - 1]} ${a.year}`;
    }
    if (a.month === 1 && b.month === 12 && b.day === 31) return `${a.year} год`;
  }
  return formatRange(from, to);
}
