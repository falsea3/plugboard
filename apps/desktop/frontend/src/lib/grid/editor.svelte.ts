import type { CellValue } from '../api/wire';
import { clamp, type GridEditing } from './grid';

export type OpenEditor = { r: number; c: number; text: string; wasNull: boolean; touched: boolean; options: string[] | null };

export interface EditorHost {
  editing: () => GridEditing | null;
  value: (r: number, c: number) => CellValue | undefined;
  columns: () => number;
  select: (r: number, c: number) => void;
  show: (r: number, c: number) => void;
  focus: () => void;
}

export class GridEditor {
  open = $state<OpenEditor | null>(null);

  constructor(private host: EditorHost) {}

  at(r: number, c: number): boolean {
    return this.open?.r === r && this.open?.c === c;
  }

  start(r: number, c: number, replaceWith?: string) {
    const editing = this.host.editing();
    if (!editing?.canEdit(r, c)) return;
    if (replaceWith === undefined && editing.open?.(r, c)) return;
    const v = this.host.value(r, c);
    const empty = v === null || editing.cellState(r, c) === 'default';
    this.open = {
      r,
      c,
      text: replaceWith ?? (empty ? '' : String(v)),
      wasNull: empty,
      touched: replaceWith !== undefined,
      options: editing.options?.(r, c) ?? null,
    };
    this.host.select(r, c);
  }

  commit() {
    const e = this.open;
    if (!e) return;
    this.open = null;
    if (e.touched) this.host.editing()?.commit(e.r, e.c, e.text);
    this.host.focus();
  }

  cancel() {
    this.open = null;
    this.host.focus();
  }

  type(text: string) {
    if (this.open) this.open.text = text;
  }

  touch() {
    if (this.open) this.open.touched = true;
  }

  blur() {
    const blurred = this.open;
    setTimeout(() => {
      if (document.hasFocus() && this.open === blurred) this.commit();
    });
  }

  key(e: KeyboardEvent) {
    e.stopPropagation();
    if (e.key === 'Escape') {
      e.preventDefault();
      this.cancel();
    } else if (e.key === 'Enter' && !e.shiftKey && !e.altKey) {
      e.preventDefault();
      this.commit();
    } else if (e.key === 'Tab') {
      e.preventDefault();
      this.next(e.shiftKey ? -1 : 1);
    }
  }

  pick(value: string) {
    if (!this.open) return;
    this.open.text = value;
    this.open.touched = true;
    this.commit();
  }

  closeList(chosen: boolean, key?: KeyboardEvent) {
    if (key?.key === 'Tab') {
      key.preventDefault();
      this.next(key.shiftKey ? -1 : 1);
    } else if (chosen) {
      this.commit();
    } else {
      this.cancel();
    }
  }

  private next(delta: number) {
    const ed = this.open;
    if (!ed) return;
    this.commit();
    const c = clamp(ed.c + delta, 0, this.host.columns() - 1);
    this.host.select(ed.r, c);
    this.host.show(ed.r, c);
  }
}
