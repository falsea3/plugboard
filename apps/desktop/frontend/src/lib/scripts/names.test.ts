import { describe, expect, it } from 'vitest';
import { isDirty, nameProblem, sortNames, suggestName, toState, untitled } from './names';

describe('isDirty', () => {
  it('compares the text with what was last saved', () => {
    expect(isDirty({ title: 'Query 1', sql: '', saved: '' })).toBe(false);
    expect(isDirty({ title: 'Query 1', sql: 'select 1', saved: 'select 1' })).toBe(false);
    expect(isDirty({ title: 'a', sql: 'select 2', saved: 'select 1', script: 'a' })).toBe(true);
  });
});

describe('nameProblem', () => {
  it('accepts plain names and the current one', () => {
    expect(nameProblem('monthly revenue', [])).toBe('');
    expect(nameProblem('Report', ['report'], 'report')).toBe('');
  });

  it('refuses what the backend refuses', () => {
    for (const bad of ['', ' a', 'a ', '.hidden', 'a/b', 'a\\b', 'a:b', 'a\nb', 'x'.repeat(121)]) {
      expect(nameProblem(bad, []), bad).not.toBe('');
    }
  });

  it('refuses a taken name whatever its case', () => {
    expect(nameProblem('Orders', ['orders'])).toMatch(/already exists/);
  });
});

describe('suggestName', () => {
  it('starts from the tab title and avoids taken names', () => {
    expect(suggestName('Query 1', [])).toBe('Query 1');
    expect(suggestName('Query 1', ['query 1', 'Query 1 2'])).toBe('Query 1 3');
    expect(suggestName('a/b:c', [])).toBe('a b c');
    expect(suggestName('..', [])).toBe('Query');
  });
});

describe('untitled', () => {
  it('takes the first free number', () => {
    expect(untitled([])).toBe('Query 1');
    expect(untitled(['Query 1', 'Query 3'])).toBe('Query 2');
  });
});

describe('toState', () => {
  it('keeps what the backend stores', () => {
    expect(toState({ title: 'a', sql: 'x', saved: 'y', script: 'a' }, true)).toEqual({ title: 'a', sql: 'x', script: 'a', saved: 'y', active: true });
    expect(toState({ title: 'Query 1', sql: '', saved: '' }, false).script).toBe('');
  });
});

it('sortNames ignores case', () => {
  expect(sortNames(['b', 'A', 'c'])).toEqual(['A', 'b', 'c']);
});
