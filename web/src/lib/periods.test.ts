import { describe, expect, it } from 'vitest';
import { detectPreset, formatRange, periodChipLabel, presetRange, type PeriodPreset } from './periods';

const today = '2026-10-16';

describe('presetRange', () => {
  it.each<[PeriodPreset, string, string]>([
    ['today', '2026-10-16', '2026-10-16'],
    ['yesterday', '2026-10-15', '2026-10-15'],
    ['week', '2026-10-12', '2026-10-18'],
    ['month', '2026-10-01', '2026-10-31'],
    ['prevMonth', '2026-09-01', '2026-09-30'],
    ['last30', '2026-09-17', '2026-10-16'],
    ['year', '2026-01-01', '2026-12-31'],
    ['all', '', ''],
  ])('%s → %s..%s', (preset, from, to) => {
    expect(presetRange(preset, today)).toEqual({ from, to });
  });

  it('returns null for a custom period', () => {
    expect(presetRange('custom', today)).toBeNull();
  });

  it('wraps the previous month across the year boundary', () => {
    expect(presetRange('prevMonth', '2026-01-10')).toEqual({ from: '2025-12-01', to: '2025-12-31' });
  });
});

describe('detectPreset', () => {
  it.each<[string, string, PeriodPreset]>([
    ['2026-10-01', '2026-10-31', 'month'],
    ['2026-10-16', '2026-10-16', 'today'],
    ['', '', 'all'],
    ['2026-10-02', '2026-10-31', 'custom'],
    ['2026-10-01', '', 'custom'],
  ])('%s..%s → %s', (from, to, want) => {
    expect(detectPreset(from, to, today)).toBe(want);
  });
});

describe('period labels', () => {
  it.each([
    ['', '', 'За всё время'],
    ['2026-10-01', '2026-10-31', '1–31 октября 2026'],
    ['2026-09-28', '2026-10-04', '28 сентября — 4 октября 2026'],
    ['2025-12-28', '2026-01-04', '28 декабря 2025 — 4 января 2026'],
    ['2026-10-16', '2026-10-16', '16 октября 2026'],
    ['2026-10-01', '', 'С 1 октября 2026'],
    ['', '2026-10-31', 'По 31 октября 2026'],
  ])('formatRange(%s, %s) = %s', (from, to, want) => {
    expect(formatRange(from, to)).toBe(want);
  });

  it('shortens whole months and years in chips', () => {
    expect(periodChipLabel('2026-10-01', '2026-10-31')).toBe('Октябрь 2026');
    expect(periodChipLabel('2026-01-01', '2026-12-31')).toBe('2026 год');
    expect(periodChipLabel('2026-10-01', '2026-10-30')).toBe('1–30 октября 2026');
  });
});
