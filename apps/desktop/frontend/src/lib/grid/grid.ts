import type { CellValue } from '../api/wire';
import type { IconName } from '../ui/Icon.svelte';

export interface GridEditing {
  canEdit(r: number, c: number): boolean;
  cellState(r: number, c: number): '' | 'edited' | 'default' | 'expr';
  rowState(r: number): '' | 'new' | 'deleted';
  isFailed(r: number): boolean;
  commit(r: number, c: number, value: CellValue): void;
  options?(r: number, c: number): string[] | null;
  open?(r: number, c: number): boolean;
}

export type MenuItem = { id: string; label: string; icon?: IconName; kbd?: string; danger?: boolean; disabled?: boolean; items?: MenuItem[] } | 'sep';

export type MenuAt = { r: number; c: number; rows: number[] };

export type Cursor = { row: number; col: number };

export const inCellEditor = (e: Event) => (e.target as HTMLElement).closest('.cell-editor') !== null;

export const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));

export function rowsSelected(cursor: Cursor | null, anchor: number | null, picked: ReadonlySet<number>): number[] {
  const out = new Set(picked);
  if (cursor && anchor !== null) {
    for (let r = Math.min(anchor, cursor.row); r <= Math.max(anchor, cursor.row); r++) out.add(r);
  } else if (cursor && out.size === 0) {
    out.add(cursor.row);
  }
  return [...out].sort((a, b) => a - b);
}

export function toggled(rows: Iterable<number>, row: number): Set<number> {
  const out = new Set(rows);
  if (!out.delete(row)) out.add(row);
  return out;
}

export interface KeyPress {
  key: string;
  mod: boolean;
  shift: boolean;
  alt: boolean;
}

export interface KeyContext {
  cursor: Cursor | null;
  rows: number;
  cols: number;
  pageRows: number;
  editable: boolean;
}

export type KeyAction =
  | { kind: 'move'; row: number; col: number; extend: boolean }
  | { kind: 'page'; row: number | null; scroll: number | 'top' | 'bottom' }
  | { kind: 'edit'; text?: string }
  | { kind: 'menu'; id: 'null' | 'toggle' | 'delete-rows' | 'value' }
  | { kind: 'copy'; whole: 'cell' | 'rows' | 'all' };

const MOVES: Record<string, [number, number]> = { ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] };

export function keyAction(k: KeyPress, ctx: KeyContext): KeyAction | null {
  const { cursor } = ctx;
  if (cursor && cursor.col >= 0 && k.shift && !k.mod && k.key === 'Enter') return { kind: 'menu', id: 'value' };
  if (ctx.editable && cursor) {
    if (k.mod && k.key === 'Backspace') return { kind: 'menu', id: 'delete-rows' };
    if (cursor.col >= 0 && !k.mod) {
      if (k.alt && k.key === 'Backspace') return { kind: 'menu', id: 'null' };
      if (k.key === ' ') return { kind: 'menu', id: 'toggle' };
      if (k.key === 'Enter') return { kind: 'edit' };
      if (k.key === 'Backspace' || k.key === 'Delete') return { kind: 'edit', text: '' };
      if (k.key.length === 1 && !k.alt) return { kind: 'edit', text: k.key };
    }
  }
  if (k.mod && k.key.toLowerCase() === 'c') {
    if (!cursor) return null;
    return { kind: 'copy', whole: cursor.col >= 0 && !k.shift ? 'cell' : 'rows' };
  }
  if (k.mod && k.key.toLowerCase() === 'a') return { kind: 'copy', whole: 'all' };
  if (ctx.rows === 0) return null;
  const last = ctx.rows - 1;
  switch (k.key) {
    case 'Home':
      return { kind: 'page', row: cursor ? 0 : null, scroll: 'top' };
    case 'End':
      return { kind: 'page', row: cursor ? last : null, scroll: 'bottom' };
    case 'PageDown':
    case 'PageUp': {
      const step = k.key === 'PageDown' ? ctx.pageRows : -ctx.pageRows;
      return { kind: 'page', row: cursor ? clamp(cursor.row + step, 0, last) : null, scroll: step };
    }
    case 'Tab': {
      const cur = cursor ?? { row: 0, col: 0 };
      return { kind: 'move', row: cur.row, col: clamp(Math.max(cur.col, 0) + (k.shift ? -1 : 1), 0, ctx.cols - 1), extend: false };
    }
  }
  const move = MOVES[k.key];
  if (!move) return null;
  const cur = cursor ?? { row: 0, col: 0 };
  const jump = k.mod ? (move[0] !== 0 ? ctx.rows : ctx.cols) : 1;
  return {
    kind: 'move',
    row: clamp(cur.row + move[0] * jump, 0, last),
    col: clamp(Math.max(cur.col, 0) + move[1] * jump, 0, ctx.cols - 1),
    extend: k.shift && move[0] !== 0,
  };
}

export interface MenuParts {
  rows: number;
  canEdit: boolean;
  header: () => MenuItem[];
  cell: () => MenuItem[];
}

export function gridMenuItems(at: { r: number; c: number }, parts: MenuParts): MenuItem[] {
  let items: MenuItem[];
  if (at.r === -1) {
    items = [...parts.header(), 'sep', { id: 'fit', label: 'Fit width' }, { id: 'copy-name', label: 'Copy column name' }];
  } else {
    const copyRows: MenuItem = { id: 'copy-rows', label: parts.rows > 1 ? `Copy ${parts.rows} rows` : 'Copy row', kbd: '⇧⌘C' };
    items = at.c >= 0
      ? [{ id: 'edit', label: 'Edit', kbd: '↵', disabled: !parts.canEdit }, { id: 'copy-value', label: 'Copy value', kbd: '⌘C' }, copyRows]
      : [copyRows];
    items.push('sep', ...parts.cell());
  }
  return items.filter((it, i, all) => it !== 'sep' || (i > 0 && i < all.length - 1 && all[i - 1] !== 'sep'));
}
