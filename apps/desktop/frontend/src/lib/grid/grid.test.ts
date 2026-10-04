import { describe, expect, it } from 'vitest';
import { gridMenuItems, keyAction, rowsSelected, toggled, type KeyContext, type KeyPress, type MenuItem } from './grid';

const press = (key: string, mods: Partial<Omit<KeyPress, 'key'>> = {}): KeyPress => ({ key, mod: false, shift: false, alt: false, ...mods });
const ctx = (over: Partial<KeyContext> = {}): KeyContext => ({ cursor: { row: 5, col: 2 }, rows: 100, cols: 6, pageRows: 20, editable: true, ...over });

describe('grid keys', () => {
  it('moves the cursor and stops at the edges', () => {
    expect(keyAction(press('ArrowDown'), ctx())).toEqual({ kind: 'move', row: 6, col: 2, extend: false });
    expect(keyAction(press('ArrowUp'), ctx({ cursor: { row: 0, col: 2 } }))).toMatchObject({ row: 0 });
    expect(keyAction(press('ArrowDown', { mod: true }), ctx())).toMatchObject({ row: 99 });
    expect(keyAction(press('ArrowRight', { mod: true }), ctx())).toMatchObject({ col: 5 });
    expect(keyAction(press('Tab', { shift: true }), ctx())).toMatchObject({ col: 1 });
  });

  it('extends the selection with shift and the vertical arrows only', () => {
    expect(keyAction(press('ArrowDown', { shift: true }), ctx())).toMatchObject({ row: 6, extend: true });
    expect(keyAction(press('ArrowRight', { shift: true }), ctx())).toMatchObject({ extend: false });
  });

  it('pages by a screen and jumps to the ends', () => {
    expect(keyAction(press('PageDown'), ctx())).toEqual({ kind: 'page', row: 25, scroll: 20 });
    expect(keyAction(press('PageUp'), ctx())).toEqual({ kind: 'page', row: 0, scroll: -20 });
    expect(keyAction(press('End'), ctx())).toEqual({ kind: 'page', row: 99, scroll: 'bottom' });
    expect(keyAction(press('Home'), ctx({ cursor: null }))).toEqual({ kind: 'page', row: null, scroll: 'top' });
  });

  it('starts editing only where editing is possible', () => {
    expect(keyAction(press('x'), ctx())).toEqual({ kind: 'edit', text: 'x' });
    expect(keyAction(press('Enter'), ctx())).toEqual({ kind: 'edit' });
    expect(keyAction(press('Backspace', { alt: true }), ctx())).toEqual({ kind: 'menu', id: 'null' });
    expect(keyAction(press('Backspace', { mod: true }), ctx())).toEqual({ kind: 'menu', id: 'delete-rows' });
    expect(keyAction(press('x'), ctx({ editable: false }))).toBeNull();
    expect(keyAction(press('x'), ctx({ cursor: { row: 5, col: -1 } }))).toBeNull();
  });

  it('copies a cell, the selected rows or everything', () => {
    expect(keyAction(press('c', { mod: true }), ctx())).toEqual({ kind: 'copy', whole: 'cell' });
    expect(keyAction(press('c', { mod: true, shift: true }), ctx())).toEqual({ kind: 'copy', whole: 'rows' });
    expect(keyAction(press('a', { mod: true }), ctx())).toEqual({ kind: 'copy', whole: 'all' });
    expect(keyAction(press('c', { mod: true }), ctx({ cursor: null }))).toBeNull();
  });
});

describe('selected rows', () => {
  it('joins the shift range and the rows picked with ⌘', () => {
    expect(rowsSelected({ row: 7, col: 1 }, 4, new Set([1, 9]))).toEqual([1, 4, 5, 6, 7, 9]);
    expect(rowsSelected({ row: 3, col: 0 }, null, new Set())).toEqual([3]);
    expect(rowsSelected(null, null, new Set([2, 0]))).toEqual([0, 2]);
    expect(rowsSelected({ row: 5, col: 0 }, null, new Set([1, 9]))).toEqual([1, 9]);
    expect([...toggled([1, 5], 5)]).toEqual([1]);
    expect([...toggled([1], 5)]).toEqual([1, 5]);
  });
});

describe('grid menu', () => {
  const ids = (items: MenuItem[]) => items.map(i => (i === 'sep' ? '—' : i.id));
  const none = () => [];

  it('puts the column tools after the header items', () => {
    expect(ids(gridMenuItems({ r: -1, c: 2 }, { rows: 0, canEdit: false, header: () => [{ id: 'sort-asc', label: '' }], cell: none })))
      .toEqual(['sort-asc', '—', 'fit', 'copy-name']);
  });

  it('copies the picked rows and drops separators with nothing between them', () => {
    const items = gridMenuItems({ r: 3, c: 1 }, { rows: 4, canEdit: false, header: none, cell: () => ['sep', { id: 'json', label: '' }, 'sep'] });
    expect(ids(items)).toEqual(['edit', 'copy-value', 'copy-rows', '—', 'json']);
    expect(items[0]).toMatchObject({ disabled: true });
    expect(items[2]).toMatchObject({ label: 'Copy 4 rows' });
    expect(ids(gridMenuItems({ r: 3, c: -1 }, { rows: 1, canEdit: true, header: none, cell: none }))).toEqual(['copy-rows']);
  });
});
