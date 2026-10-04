import { rowsSelected, toggled, type Cursor } from './grid';

export type Pointer = { shiftKey: boolean; metaKey: boolean; ctrlKey: boolean };

export class GridSelection {
  cursor = $state<Cursor | null>(null);
  anchor = $state<number | null>(null);
  picked = $state<Set<number>>(new Set());
  marked = $derived(
    this.anchor !== null || this.picked.size > 0 || this.cursor?.col === -1 ? new Set(rowsSelected(this.cursor, this.anchor, this.picked)) : null,
  );

  rows(): number[] {
    return rowsSelected(this.cursor, this.anchor, this.picked);
  }

  rowsFor(r: number): number[] {
    const rs = this.rows();
    return rs.includes(r) ? rs : [r];
  }

  clear() {
    this.cursor = null;
    this.anchor = null;
    this.picked = new Set();
  }

  only(row: number, col: number) {
    this.cursor = { row, col };
    this.anchor = null;
    this.picked = new Set();
  }

  point(row: number, col: number, e?: Pointer) {
    if (e?.shiftKey && this.cursor) {
      this.anchor ??= this.cursor.row;
    } else if (e && (e.metaKey || e.ctrlKey)) {
      this.picked = toggled(this.picked.size ? this.picked : this.rows(), row);
      this.anchor = null;
    } else {
      this.anchor = null;
      this.picked = new Set();
    }
    this.cursor = { row, col };
  }

  move(row: number, col: number, extend: boolean): Cursor {
    if (extend && this.cursor) {
      this.anchor ??= this.cursor.row;
      col = this.cursor.col;
    } else {
      this.anchor = null;
      this.picked = new Set();
    }
    this.cursor = { row, col };
    return this.cursor;
  }

  menuAt(row: number, col: number) {
    if (this.marked?.has(row)) this.cursor = { row, col };
    else this.only(row, col);
  }
}
