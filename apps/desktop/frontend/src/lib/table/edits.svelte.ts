import { SvelteMap, SvelteSet } from 'svelte/reactivity';
import type { CellValue, ResultColumn, RowChange } from '../api/wire';

export const DEFAULT = Symbol('DEFAULT');

export type Expr = { expr: 'now' | 'default' };
export const NOW: Expr = { expr: 'now' };
export const SET_DEFAULT: Expr = { expr: 'default' };
export const isExpr = (v: unknown): v is Expr => typeof v === 'object' && v !== null && 'expr' in v;
export const exprLabel = (e: Expr) => (e.expr === 'now' ? 'now()' : 'DEFAULT');

export type EditValue = CellValue | typeof DEFAULT | Expr;

export type CellState = '' | 'edited' | 'default' | 'expr';
export type RowState = '' | 'new' | 'deleted';

export class TableEdits {
  updates = new SvelteMap<number, SvelteMap<number, CellValue | Expr>>();
  deleted = new SvelteSet<number>();
  inserted = $state<EditValue[][]>([]);
  failedRow = $state<number | null>(null);

  constructor(
    private baseRows: () => CellValue[][],
    private columnCount: () => number,
  ) {}

  get count(): number {
    let n = this.deleted.size + this.inserted.length;
    for (const r of this.updates.keys()) if (!this.deleted.has(r)) n++;
    return n;
  }

  get dirty(): boolean {
    return this.count > 0;
  }

  get rowCount(): number {
    return this.baseRows().length + this.inserted.length;
  }

  isNew(r: number): boolean {
    return r >= this.baseRows().length;
  }

  value(r: number, c: number): EditValue {
    const base = this.baseRows();
    if (r >= base.length) {
      const row = this.inserted[r - base.length];
      return row && c < row.length ? row[c] : DEFAULT;
    }
    const row = this.updates.get(r);
    return row?.has(c) ? (row.get(c) as CellValue | Expr) : base[r][c];
  }

  shown(show: (v: Expr | typeof DEFAULT, c: number) => CellValue): CellValue[][] {
    const base = this.baseRows();
    const cols = this.columnCount();
    const out: CellValue[][] = new Array(this.rowCount);
    for (let r = 0; r < out.length; r++) {
      if (r < base.length && !this.updates.has(r)) {
        out[r] = base[r];
        continue;
      }
      const row: CellValue[] = new Array(cols);
      for (let c = 0; c < cols; c++) {
        const v = this.value(r, c);
        row[c] = v === DEFAULT || isExpr(v) ? show(v, c) : v;
      }
      out[r] = row;
    }
    return out;
  }

  cellState(r: number, c: number): CellState {
    const v = this.value(r, c);
    if (isExpr(v)) return 'expr';
    if (this.isNew(r)) return v === DEFAULT ? 'default' : 'edited';
    return this.updates.get(r)?.has(c) ? 'edited' : '';
  }

  rowState(r: number): RowState {
    if (this.isNew(r)) return 'new';
    return this.deleted.has(r) ? 'deleted' : '';
  }

  isFailed(r: number): boolean {
    return this.failedRow === r;
  }

  set(r: number, c: number, v: EditValue) {
    this.failedRow = null;
    const base = this.baseRows();
    if (r >= base.length) {
      const row = this.inserted[r - base.length];
      if (row) row[c] = v;
      return;
    }
    if (v === DEFAULT) return;
    const original = base[r][c];
    let row = this.updates.get(r);
    if (sameValue(original, v)) {
      row?.delete(c);
      if (row && row.size === 0) this.updates.delete(r);
      return;
    }
    if (!row) {
      row = new SvelteMap();
      this.updates.set(r, row);
    }
    row.set(c, v);
  }

  addRow(): number {
    this.failedRow = null;
    this.inserted.push(Array.from({ length: this.columnCount() }, () => DEFAULT));
    return this.rowCount - 1;
  }

  deleteRows(rows: number[]) {
    this.failedRow = null;
    const base = this.baseRows().length;
    const added = rows.filter(r => r >= base).map(r => r - base).sort((a, b) => b - a);
    for (const i of added) this.inserted.splice(i, 1);
    for (const r of rows) if (r < base) this.deleted.add(r);
  }

  restoreRows(rows: number[]) {
    this.failedRow = null;
    for (const r of rows) this.deleted.delete(r);
  }

  discard() {
    this.updates.clear();
    this.deleted.clear();
    this.inserted = [];
    this.failedRow = null;
  }

  changes(columns: ResultColumn[], keyColumns: string[]): { changes: RowChange[]; rows: number[] } {
    const base = this.baseRows();
    const keyIdx = keyColumns.map(k => columns.findIndex(c => c.name === k));
    const key = (r: number) => Object.fromEntries(keyColumns.map((k, i) => [k, base[r][keyIdx[i]]]));
    const changes: RowChange[] = [];
    const rows: number[] = [];

    for (const r of [...this.deleted].sort((a, b) => a - b)) {
      changes.push({ kind: 'delete', key: key(r), values: {} });
      rows.push(r);
    }
    for (const [r, cells] of [...this.updates].sort((a, b) => a[0] - b[0])) {
      if (this.deleted.has(r) || cells.size === 0) continue;
      const values: Record<string, WireValue> = {};
      for (const [c, v] of cells) values[columns[c].name] = toWire(v);
      changes.push({ kind: 'update', key: key(r), values });
      rows.push(r);
    }
    this.inserted.forEach((row, i) => {
      const values: Record<string, WireValue> = {};
      row.forEach((v, c) => {
        if (v !== DEFAULT) values[columns[c].name] = toWire(v);
      });
      changes.push({ kind: 'insert', key: {}, values });
      rows.push(base.length + i);
    });
    return { changes, rows };
  }
}

function sameValue(a: CellValue, b: EditValue): boolean {
  if (b === DEFAULT || isExpr(b)) return false;
  if (a === null || b === null) return a === b;
  return String(a) === String(b);
}

type WireValue = RowChange['values'][string];

function toWire(v: CellValue | Expr): WireValue {
  return isExpr(v) ? { $expr: v.expr } : v;
}
