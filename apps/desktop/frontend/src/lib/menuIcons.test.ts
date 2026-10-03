import { describe, expect, it } from 'vitest';
import { menuIcon } from './menuIcons';

describe('menu icons', () => {
  it('gives every action of one kind the same icon', () => {
    for (const id of ['filter-value', 'exclude-value', 'f-null', 'f-enum:pro', 'fv-=', 'fv-contains']) expect(menuIcon(id)).toBe('funnel');
    for (const id of ['edit', 'null', 'set-true', 'set-now', 'set-default', 'nullable']) expect(menuIcon(id)).toBe('pencil');
    for (const id of ['copy-value', 'copy-rows', 'copy-name', 'sql-copy']) expect(menuIcon(id)).toBe('copy');
  });

  it('keeps arrows for sorting and leaves SQL keywords bare', () => {
    expect([menuIcon('sort-asc'), menuIcon('sort-desc')]).toEqual(['arrowUp', 'arrowDown']);
    expect(menuIcon('sql-open:select')).toBeUndefined();
  });
});
