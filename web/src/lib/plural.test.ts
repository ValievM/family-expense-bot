import { describe, expect, it } from 'vitest';
import { plural } from './plural';

describe('plural', () => {
  it.each([
    [0, '0 трат'],
    [1, '1 трата'],
    [2, '2 траты'],
    [5, '5 трат'],
    [11, '11 трат'],
    [12, '12 трат'],
    [21, '21 трата'],
    [22, '22 траты'],
    [111, '111 трат'],
  ])('%i', (n, want) => {
    expect(plural(n, ['трата', 'траты', 'трат'])).toBe(want);
  });
});
