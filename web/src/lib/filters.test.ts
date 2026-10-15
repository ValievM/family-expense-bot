import { describe, expect, it } from 'vitest';
import { EMPTY_FILTERS, filtersFromQuery, filtersToQuery, hasActiveFilters } from './filters';

describe('filters query string', () => {
  it('round-trips all fields', () => {
    const f = { user_id: '42', category: 'food', from: '2026-01-01', to: '2026-01-31' };
    expect(filtersFromQuery(filtersToQuery(f))).toEqual(f);
  });

  it('omits empty values', () => {
    expect(filtersToQuery(EMPTY_FILTERS)).toBe('');
    expect(filtersToQuery({ ...EMPTY_FILTERS, category: 'food' })).toBe('?category=food');
  });

  it('ignores unknown params', () => {
    expect(filtersFromQuery('?x=1&user_id=7')).toEqual({ ...EMPTY_FILTERS, user_id: '7' });
  });

  it('detects active filters', () => {
    expect(hasActiveFilters(EMPTY_FILTERS)).toBe(false);
    expect(hasActiveFilters({ ...EMPTY_FILTERS, to: '2026-01-01' })).toBe(true);
  });
});
