import { describe, expect, it } from 'vitest';
import { bannerMessage, errorTarget, hasTechnicalDetails } from './apiErrors';

describe('errorTarget', () => {
  it.each([
    ['invalid_amount', { kind: 'field', field: 'amount' }],
    ['invalid_user', { kind: 'field', field: 'userId' }],
    ['unknown_category', { kind: 'field', field: 'category' }],
    ['invalid_date', { kind: 'field', field: 'spentAt' }],
    ['comment_too_long', { kind: 'field', field: 'comment' }],
    ['invalid_period', { kind: 'period' }],
    ['not_found', { kind: 'notFound' }],
    ['internal', { kind: 'banner' }],
    ['invalid_json', { kind: 'banner' }],
    ['invalid_id', { kind: 'banner' }],
    ['network', { kind: 'banner' }],
    ['something_new', { kind: 'banner' }],
  ])('%s', (code, want) => {
    expect(errorTarget(code)).toEqual(want);
  });
});

describe('banner texts', () => {
  it('uses calm texts without technical terms', () => {
    expect(bannerMessage('internal')).toBe('Что-то пошло не так. Попробуйте ещё раз.');
    expect(bannerMessage('network')).toMatch(/Нет связи/);
    expect(bannerMessage('invalid_json')).toMatch(/Не удалось/);
  });

  it('hides technical details for network and server failures only', () => {
    expect(hasTechnicalDetails('internal')).toBe(false);
    expect(hasTechnicalDetails('network')).toBe(false);
    expect(hasTechnicalDetails('invalid_id')).toBe(true);
  });
});
