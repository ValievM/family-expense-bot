export const MONTHS_GENITIVE = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря',
];

export const MONTHS_NOMINATIVE = [
  'Январь',
  'Февраль',
  'Март',
  'Апрель',
  'Май',
  'Июнь',
  'Июль',
  'Август',
  'Сентябрь',
  'Октябрь',
  'Ноябрь',
  'Декабрь',
];

export interface DateParts {
  year: number;
  month: number;
  day: number;
}

const pad = (n: number) => String(n).padStart(2, '0');

export function formatDate(iso: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  return m ? `${m[3]}.${m[2]}.${m[1]}` : iso;
}

export function parseISODate(iso: string): DateParts | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso);
  return m ? { year: Number(m[1]), month: Number(m[2]), day: Number(m[3]) } : null;
}

export function isoFromParts(year: number, month: number, day: number): string {
  return `${String(year).padStart(4, '0')}-${pad(month)}-${pad(day)}`;
}

export function todayISO(now: Date = new Date()): string {
  return isoFromParts(now.getFullYear(), now.getMonth() + 1, now.getDate());
}

export function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

/**
 * Принимает: дату ГГГГ-ММ-ДД и число дней (может быть отрицательным).
 * Делает: сдвигает дату в UTC, чтобы переход на летнее время не терял сутки.
 * Возвращает: новую дату ГГГГ-ММ-ДД.
 */
export function addDays(iso: string, days: number): string {
  const p = parseISODate(iso);
  if (!p) return iso;
  const d = new Date(Date.UTC(p.year, p.month - 1, p.day + days));
  return isoFromParts(d.getUTCFullYear(), d.getUTCMonth() + 1, d.getUTCDate());
}

/**
 * Принимает: дату ГГГГ-ММ-ДД.
 * Делает: считает день недели с понедельника, как принято в России.
 * Возвращает: 0 для понедельника … 6 для воскресенья.
 */
export function weekdayMondayFirst(iso: string): number {
  const p = parseISODate(iso);
  if (!p) return 0;
  return (new Date(Date.UTC(p.year, p.month - 1, p.day)).getUTCDay() + 6) % 7;
}
