import type { CellValue, Column, FilterOp } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import { isTrue } from '../ui/format';

export interface CellMenu {
  cell: {
    col?: Column;
    name: string;
    value: CellValue;
    editable: boolean;
    isNew: boolean;
    canSetDefault: boolean;
    json: boolean;
    followTo: string | null;
  } | null;
  sqlRows: number;
  canFindRows: boolean;
  editRows: { count: number; allDeleted: boolean } | null;
}

const rowsLabel = (n: number) => (n > 1 ? `${n} rows` : 'row');

export function cellMenu(m: CellMenu): MenuItem[] {
  const items: MenuItem[] = [];
  const cell = m.cell;
  if (cell) {
    const { col, name, value: v } = cell;
    if (cell.editable && col) {
      if (col.kind === 'bool') {
        if (v === null || !isTrue(v)) items.push({ id: 'set-true', label: 'Set true', kbd: 'Space' });
        if (v === null || isTrue(v)) items.push({ id: 'set-false', label: 'Set false', kbd: 'Space' });
      }
      if (col.kind === 'datetime') items.push({ id: 'set-now', label: 'Set to now()' });
      if (col.nullable && v !== null) items.push({ id: 'null', label: 'Set NULL', kbd: '⌥⌫' });
      if (cell.canSetDefault) items.push({ id: 'set-default', label: 'Set DEFAULT' });
    }
    if (cell.json) items.push({ id: 'json', label: cell.editable ? 'Edit as JSON' : 'View as JSON', kbd: col?.kind === 'json' ? '↵' : undefined });
    if (col && !['json', 'binary', 'bool'].includes(col.kind)) items.push({ id: 'value', label: cell.editable ? 'Open in editor' : 'View in editor', kbd: '⇧↵' });
    if (cell.followTo) items.push('sep', { id: 'follow', label: `Go to ${cell.followTo}` });
    if (!cell.isNew) {
      items.push(
        'sep',
        { id: 'filter-value', label: v === null ? `Filter: ${name} is NULL` : 'Filter by this value' },
        { id: 'exclude-value', label: v === null ? `Filter: ${name} is not NULL` : 'Exclude this value' },
      );
    }
  }
  if (m.sqlRows > 0) {
    const statements = (to: string): MenuItem[] =>
      (['select', 'insert', 'update', 'delete'] as const).map(k => ({ id: `${to}:${k}`, label: k.toUpperCase(), disabled: k !== 'insert' && !m.canFindRows }));
    const many = m.sqlRows > 1;
    items.push(
      'sep',
      { id: 'sql-copy', label: many ? `Copy ${m.sqlRows} rows as SQL` : 'Copy as SQL', items: statements('sql-copy') },
      { id: 'sql-open', label: many ? `Open ${m.sqlRows} rows as SQL` : 'Open SQL in new query', items: statements('sql-open') },
    );
  }
  if (m.editRows) {
    const { count, allDeleted } = m.editRows;
    items.push(
      'sep',
      allDeleted
        ? { id: 'restore', label: `Restore ${rowsLabel(count)}`, kbd: '⌘⌫' }
        : { id: 'delete', label: `Delete ${rowsLabel(count)}`, kbd: '⌘⌫', danger: true },
    );
  }
  return items;
}

export interface HeaderMenu {
  name: string;
  col?: Column;
  sort: { column: string; desc: boolean } | null;
  defaultOrder: string[] | null;
}

export function headerMenu({ name, col, sort, defaultOrder }: HeaderMenu): MenuItem[] {
  const cur = sort?.column === name ? sort : null;
  const items: MenuItem[] = [
    { id: 'sort-asc', label: 'Sort ascending', disabled: cur?.desc === false },
    { id: 'sort-desc', label: 'Sort descending', disabled: cur?.desc === true },
  ];
  if (defaultOrder?.length) items.push({ id: 'sort-default', label: `Default order (${defaultOrder.join(', ')})` });
  items.push('sep');
  if (col && col.kind === 'bool') items.push({ id: 'f-true', label: 'Is true' }, { id: 'f-false', label: 'Is false' });
  if (col && col.enum?.length) for (const e of col.enum.slice(0, 8)) items.push({ id: 'f-enum:' + e, label: `Is “${e}”` });
  if (!col || col.nullable) items.push({ id: 'f-null', label: 'Is NULL' }, { id: 'f-not_null', label: 'Is not NULL' });
  const textual = !!col && (col.kind === 'text' || col.kind === 'json');
  if (textual) items.push({ id: 'f-empty', label: 'Is empty' }, { id: 'f-not_empty', label: 'Is not empty' });
  items.push('sep', { id: 'fv-=', label: 'Equals…' }, { id: 'fv-!=', label: 'Not equals…' });
  if (!col || textual) items.push({ id: 'fv-contains', label: 'Contains…' });
  else items.push({ id: 'fv->', label: 'Greater than…' }, { id: 'fv-<', label: 'Less than…' });
  return items;
}

export type FilterPick = { op: FilterOp; value: string; apply: boolean };

export function filterPick(id: string, value: CellValue, boolValue: (on: boolean) => CellValue): FilterPick | null {
  if (id.startsWith('fv-')) return { op: id.slice(3) as FilterOp, value: '', apply: false };
  if (id.startsWith('f-')) {
    const op = id.slice(2);
    if (op === 'true' || op === 'false') return { op: '=', value: String(boolValue(op === 'true')), apply: true };
    if (op.startsWith('enum:')) return { op: '=', value: op.slice(5), apply: true };
    return { op: op as FilterOp, value: '', apply: true };
  }
  if (id === 'filter-value') return value === null ? { op: 'null', value: '', apply: true } : { op: '=', value: String(value), apply: true };
  if (id === 'exclude-value') return value === null ? { op: 'not_null', value: '', apply: true } : { op: '!=', value: String(value), apply: true };
  return null;
}
