import { describe, expect, it } from 'vitest';
import type { Column } from '../api/wire';
import { rowsLabel, wherePlaceholder } from './labels';

describe('table labels', () => {
  it('names the row count', () => {
    expect(rowsLabel(16, false, null)).toBe('16 rows');
    expect(rowsLabel(1, false, null)).toBe('1 row');
    expect(rowsLabel(300, true, null)).toBe('300+ rows');
    expect(rowsLabel(300, true, { count: 24813, exact: false, known: true })).toBe('~24,813 rows');
    expect(rowsLabel(null, false, null)).toBe('');
  });

  it('suggests a condition on a text column', () => {
    const cols = [{ name: 'id', kind: 'number', primaryKey: true }, { name: 'country', kind: 'text' }] as Column[];
    expect(wherePlaceholder(cols)).toBe("country = '…'");
    expect(wherePlaceholder(cols.slice(0, 1))).toBe('id = …');
    expect(wherePlaceholder([])).toBe('condition');
  });
});
