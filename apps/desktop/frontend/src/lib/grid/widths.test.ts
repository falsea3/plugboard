import { describe, expect, it } from 'vitest';
import { columnWidth } from './widths';

const col = (name: string, type = '') => ({ name, type });

describe('columnWidth', () => {
  it('stays within bounds', () => {
    expect(columnWidth(col('id'), [[1]], 0)).toBe(64);
    expect(columnWidth(col('body', 'text'), [['x'.repeat(1000)]], 0)).toBe(360);
  });

  it('never cuts the name when the key and sort icons sit next to it', () => {
    const plain = columnWidth(col('id', 'int8'), [[1]], 0);
    expect(columnWidth(col('id', 'int8'), [[1]], 0, { key: true })).toBeGreaterThan(plain);
    expect(columnWidth(col('a_really_long_column_name_that_goes_on_and_on'), [['x']], 0)).toBeGreaterThan(360);
  });

  it('fits a column to everything loaded, wider or narrower, and stays put when fitted again', () => {
    const rows = [...Array.from({ length: 300 }, () => ['a']), ['x'.repeat(60)]];
    const fitted = columnWidth(col('note'), rows, 0, { fit: true });
    expect(fitted).toBeGreaterThan(columnWidth(col('note'), rows, 0));
    expect(fitted).toBe(columnWidth(col('note'), rows, 0, { fit: true }));
    expect(columnWidth(col('body'), [['x'.repeat(5000)]], 0, { fit: true })).toBe(800);
  });

  it('leaves room for the type next to the name in the header', () => {
    expect(columnWidth(col('is_active', 'boolean'), [[false]], 0)).toBeGreaterThan(columnWidth(col('is_active'), [[false]], 0) + 40);
  });
});
