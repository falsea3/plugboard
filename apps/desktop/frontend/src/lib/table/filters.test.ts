import { describe, expect, it } from 'vitest';
import { appliedWhere, newFilter, toFilters } from './filters';

const cols = [{ name: 'id' }, { name: 'email' }];

describe('toFilters', () => {
  it('sends finished, enabled conditions', () => {
    const rows = [
      newFilter(cols, 'email', 'contains', ' @x '),
      { ...newFilter(cols, 'id', '=', '1'), on: false },
      newFilter(cols, 'id', '='),
      newFilter(cols, 'email', 'null'),
    ];
    expect(toFilters(rows)).toEqual([
      { column: 'email', op: 'contains', value: '@x' },
      { column: 'email', op: 'null', value: '' },
    ]);
  });
  it('puts the WHERE typed by hand first, and leaves it out when empty', () => {
    const rows = [newFilter(cols, 'id', '>', '5')];
    expect(toFilters(rows, " total > 100 OR status = 'void' ")).toEqual([
      { column: '', op: 'sql', value: "total > 100 OR status = 'void'" },
      { column: 'id', op: '>', value: '5' },
    ]);
    expect(toFilters(rows, '   ')).toEqual([{ column: 'id', op: '>', value: '5' }]);
    expect(appliedWhere(toFilters(rows, 'a = 1'))).toBe('a = 1');
    expect(appliedWhere(toFilters(rows))).toBe('');
  });
});
