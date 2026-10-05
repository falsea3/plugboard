import { describe, expect, it } from 'vitest';
import type { Column } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import { cellMenu, filterPick, headerMenu, type CellMenu } from './menus';

const column = (over: Partial<Column> = {}): Column => ({
  name: 'c', type: 'text', kind: 'text', nullable: true, default: null, primaryKey: false, ...over,
} as Column);

const ids = (items: MenuItem[]) => items.map(i => (i === 'sep' ? '—' : i.id));

const cell = (over: Partial<NonNullable<CellMenu['cell']>> = {}): CellMenu['cell'] => ({
  col: column(), name: 'c', value: 'x', editable: true, isNew: false, canSetDefault: false, json: false, followTo: null, ...over,
});

describe('cell menu', () => {
  it('offers what the column allows', () => {
    const m = cellMenu({ cell: cell({ col: column({ kind: 'bool' }), value: true }), sqlRows: 0, canFindRows: true, editRows: null });
    expect(ids(m)).toEqual(['set-false', 'null', '—', 'filter-value', 'exclude-value']);
    expect(ids(cellMenu({ cell: cell({ col: column({ kind: 'datetime', nullable: false }), canSetDefault: true }), sqlRows: 0, canFindRows: true, editRows: null })))
      .toEqual(['set-now', 'set-default', 'value', '—', 'filter-value', 'exclude-value']);
  });

  it('leaves editing out of a cell that can’t change', () => {
    const m = cellMenu({ cell: cell({ editable: false, json: true }), sqlRows: 0, canFindRows: true, editRows: null });
    expect(m[0]).toMatchObject({ id: 'json', label: 'View as JSON' });
    expect(ids(m)).not.toContain('null');
  });

  it('names the referenced table and the rows for SQL and deletion', () => {
    const m = cellMenu({ cell: cell({ followTo: 'customers', isNew: false }), sqlRows: 3, canFindRows: false, editRows: { count: 3, allDeleted: false } });
    expect(m).toContainEqual({ id: 'follow', label: 'Go to customers' });
    const copy = m.find(i => i !== 'sep' && i.id === 'sql-copy');
    expect(copy).toMatchObject({ label: 'Copy 3 rows as SQL' });
    expect(copy !== 'sep' && copy?.items?.filter(i => i !== 'sep' && !i.disabled).map(i => i !== 'sep' && i.id)).toEqual(['sql-copy:insert']);
    expect(m[m.length - 1]).toMatchObject({ id: 'delete', label: 'Delete 3 rows', danger: true });
  });

  it('has no value filters for a new row', () => {
    expect(ids(cellMenu({ cell: cell({ isNew: true }), sqlRows: 0, canFindRows: true, editRows: { count: 1, allDeleted: true } })))
      .toEqual(['null', 'value', '—', 'restore']);
  });
});

describe('header menu', () => {
  it('turns off the current sort direction and offers the default order', () => {
    const m = headerMenu({ name: 'id', col: column({ kind: 'number', nullable: false }), sort: { column: 'id', desc: false }, defaultOrder: ['id'] });
    expect(m[0]).toMatchObject({ id: 'sort-asc', disabled: true });
    expect(m[1]).toMatchObject({ id: 'sort-desc', disabled: false });
    expect(ids(m)).toEqual(['sort-asc', 'sort-desc', 'sort-default', '—', '—', 'fv-=', 'fv-!=', 'fv->', 'fv-<']);
  });

  it('lists enum values and text filters', () => {
    const m = headerMenu({ name: 's', col: column({ enum: ['new', 'paid'] }), sort: null, defaultOrder: null });
    expect(ids(m)).toEqual(['sort-asc', 'sort-desc', '—', 'f-enum:new', 'f-enum:paid', 'f-null', 'f-not_null', 'f-empty', 'f-not_empty', '—', 'fv-=', 'fv-!=', 'fv-contains']);
  });
});

describe('filter picks', () => {
  const bool = (on: boolean) => (on ? 1 : 0);
  it('reads the menu item into a filter', () => {
    expect(filterPick('f-true', null, bool)).toEqual({ op: '=', value: '1', apply: true });
    expect(filterPick('f-enum:paid', null, bool)).toEqual({ op: '=', value: 'paid', apply: true });
    expect(filterPick('f-not_null', null, bool)).toEqual({ op: 'not_null', value: '', apply: true });
    expect(filterPick('fv-contains', null, bool)).toEqual({ op: 'contains', value: '', apply: false });
    expect(filterPick('filter-value', 7, bool)).toEqual({ op: '=', value: '7', apply: true });
    expect(filterPick('exclude-value', null, bool)).toEqual({ op: 'not_null', value: '', apply: true });
    expect(filterPick('delete', 1, bool)).toBeNull();
  });
});
