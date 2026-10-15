import { describe, expect, it } from 'vitest';
import { formatRubles, kopecksToInput, parseRubles } from './money';

describe('parseRubles', () => {
  it.each([
    ['350', 35000],
    ['350,5', 35050],
    ['350.50', 35050],
    ['0,05', 5],
    [' 1 200 ', 120000],
  ])('parses %s', (input, want) => {
    expect(parseRubles(input)).toBe(want);
  });

  it.each(['abc', '0', '0,00', '-1', '', '1,234', '1,', '1e3'])('rejects %s', (input) => {
    expect(parseRubles(input)).toBeNull();
  });
});

describe('formatRubles', () => {
  it('formats kopecks as rubles', () => {
    expect(formatRubles(35000).replace(/\u00A0/g, ' ')).toBe('350,00 ₽');
    expect(formatRubles(123456789).replace(/\u00A0/g, ' ')).toBe('1 234 567,89 ₽');
    expect(formatRubles(5).replace(/\u00A0/g, ' ')).toBe('0,05 ₽');
  });
});

describe('kopecksToInput', () => {
  it('round-trips through parseRubles', () => {
    expect(kopecksToInput(35050)).toBe('350,50');
    expect(parseRubles(kopecksToInput(35050))).toBe(35050);
  });
});
