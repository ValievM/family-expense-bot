import { describe, expect, it } from 'vitest';
import { addDays, daysInMonth, todayISO, weekdayMondayFirst } from './date';

describe('date helpers', () => {
  it.each([
    ['2026-10-16', -1, '2026-10-15'],
    ['2026-03-01', -1, '2026-02-28'],
    ['2024-02-28', 1, '2024-02-29'],
    ['2026-12-31', 1, '2027-01-01'],
    ['2026-10-16', -29, '2026-09-17'],
  ])('addDays(%s, %i) = %s', (iso, n, want) => {
    expect(addDays(iso, n)).toBe(want);
  });

  it('counts days in month including leap years', () => {
    expect(daysInMonth(2026, 2)).toBe(28);
    expect(daysInMonth(2024, 2)).toBe(29);
    expect(daysInMonth(2026, 10)).toBe(31);
  });

  it('starts the week on Monday', () => {
    expect(weekdayMondayFirst('2026-10-12')).toBe(0);
    expect(weekdayMondayFirst('2026-10-16')).toBe(4);
    expect(weekdayMondayFirst('2026-10-18')).toBe(6);
  });

  it('formats a local date as ISO', () => {
    expect(todayISO(new Date(2026, 9, 6, 23, 59))).toBe('2026-10-06');
  });
});
