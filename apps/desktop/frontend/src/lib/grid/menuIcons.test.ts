import { describe, expect, it } from 'vitest';
import { menuIcon } from './menuIcons';
import type { MenuItem } from './grid';
import { treeMenu } from '../app/tree';
import { objectMenu } from '../objects/kinds';
import { keyMenu } from '../keys/folders';
import { scriptMenu } from '../scripts/names';

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

describe('sidebar menus', () => {
  it('give every item an icon, in every engine', () => {
    const table = { schema: 's', name: 't', kind: 'table' as const };
    const column = { name: 'c', type: 'int', nullable: true, default: null, primaryKey: false, enum: null, kind: 'number' as const };
    const index = { name: 'i', columns: ['c'], unique: false, primary: false, method: 'btree', where: '', definition: '' };
    const menus: MenuItem[][] = [
      treeMenu({ table }, false, true, true),
      treeMenu({ table, column }, false, true, true),
      treeMenu({ table, index }, false, true, true),
      objectMenu(),
      keyMenu(false),
      scriptMenu(),
    ];
    for (const item of menus.flat()) {
      if (item !== 'sep') expect(menuIcon(item.id), item.id).toBeDefined();
    }
  });
});
