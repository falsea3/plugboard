import { describe, expect, it } from 'vitest';
import { DEFAULT, TableEdits } from './edits.svelte';
import type { CellValue, ResultColumn } from '../wire';

const columns: ResultColumn[] = [
  { name: 'id', type: 'int' },
  { name: 'email', type: 'text' },
  { name: 'note', type: 'text' },
];

function setup() {
  const rows: CellValue[][] = [
    [1, 'a@x', null],
    [2, 'b@x', 'hi'],
    [3, 'c@x', null],
  ];
  return new TableEdits(() => rows, () => columns.length);
}

describe('TableEdits', () => {
  it('tracks cell edits and forgets edits back to the original', () => {
    const e = setup();
    e.set(0, 1, 'new@x');
    expect(e.value(0, 1)).toBe('new@x');
    expect(e.cellState(0, 1)).toBe('edited');
    expect(e.count).toBe(1);
    e.set(0, 1, 'a@x');
    expect(e.cellState(0, 1)).toBe('');
    expect(e.dirty).toBe(false);
    e.set(1, 0, '2'); // same value typed as text
    expect(e.dirty).toBe(false);
  });

  it('treats NULL and empty string as different values', () => {
    const e = setup();
    e.set(0, 2, '');
    expect(e.dirty).toBe(true);
    e.set(1, 2, null);
    expect(e.changes(columns, ['id']).changes[1]).toEqual({ kind: 'update', key: { id: 2 }, values: { note: null } });
  });

  it('adds rows with DEFAULT cells and drops them on delete', () => {
    const e = setup();
    const r = e.addRow();
    expect(r).toBe(3);
    expect(e.value(r, 0)).toBe(DEFAULT);
    expect(e.cellState(r, 0)).toBe('default');
    e.set(r, 1, 'd@x');
    expect(e.rowState(r)).toBe('new');
    e.deleteRows([r]);
    expect(e.rowCount).toBe(3);
    expect(e.dirty).toBe(false);
  });

  it('orders changes deletes → updates → inserts and maps them back to rows', () => {
    const e = setup();
    const added = e.addRow();
    e.set(added, 1, 'b@x'); // reuses the email of the row deleted below
    e.set(2, 2, 'note');
    e.set(1, 2, 'edited but deleted');
    e.deleteRows([1]);
    const { changes, rows } = e.changes(columns, ['id']);
    expect(changes).toEqual([
      { kind: 'delete', key: { id: 2 }, values: {} },
      { kind: 'update', key: { id: 3 }, values: { note: 'note' } },
      { kind: 'insert', key: {}, values: { email: 'b@x' } },
    ]);
    expect(rows).toEqual([1, 2, 3]);
    expect(e.count).toBe(3);
  });

  it('restores deleted rows and discards everything', () => {
    const e = setup();
    e.deleteRows([0, 2]);
    e.restoreRows([2]);
    expect(e.rowState(0)).toBe('deleted');
    expect(e.rowState(2)).toBe('');
    e.addRow();
    e.discard();
    expect(e.dirty).toBe(false);
    expect(e.rowCount).toBe(3);
  });
});

describe('TableEdits added rows', () => {
  it('keeps each added row independent', () => {
    const e = setup();
    const a = e.addRow();
    const b = e.addRow();
    e.set(b, 1, 'only b');
    expect(e.value(a, 1)).toBe(DEFAULT);
    expect(e.value(b, 1)).toBe('only b');
    e.set(a, 2, 'only a');
    expect(e.value(b, 2)).toBe(DEFAULT);
  });
});

describe('TableEdits failures', () => {
  it('keeps a failed deleted row restorable', () => {
    const e = setup();
    e.deleteRows([1]);
    e.failedRow = 1;
    expect(e.rowState(1)).toBe('deleted');
    expect(e.isFailed(1)).toBe(true);
    e.restoreRows([1]);
    expect(e.rowState(1)).toBe('');
    expect(e.isFailed(1)).toBe(false);
  });
});
