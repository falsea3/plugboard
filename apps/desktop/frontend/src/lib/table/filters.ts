import type { Filter, FilterOp } from '../api/wire';

type Col = { name: string };

export type FilterRow = { id: number; on: boolean; column: string; op: FilterOp; value: string };

export const OPS: { value: FilterOp; label: string }[] = [
  { value: '=', label: '=' },
  { value: '!=', label: '≠' },
  { value: '<', label: '<' },
  { value: '>', label: '>' },
  { value: '<=', label: '≤' },
  { value: '>=', label: '≥' },
  { value: 'contains', label: 'contains' },
  { value: 'not_contains', label: 'not contains' },
  { value: 'starts', label: 'starts with' },
  { value: 'ends', label: 'ends with' },
  { value: 'in', label: 'in list' },
  { value: 'not_in', label: 'not in list' },
  { value: 'empty', label: 'is empty' },
  { value: 'not_empty', label: 'is not empty' },
  { value: 'null', label: 'is NULL' },
  { value: 'not_null', label: 'is not NULL' },
];

export const noValue = (op: FilterOp) => op === 'null' || op === 'not_null' || op === 'empty' || op === 'not_empty';

let nextId = 0;
export function newFilter(columns: Col[], column?: string, op: FilterOp = '=', value = ''): FilterRow {
  return { id: ++nextId, on: true, column: column ?? columns[0]?.name ?? '', op, value };
}

export function ready(row: FilterRow): boolean {
  return row.on && !!row.column && (noValue(row.op) || row.value.trim() !== '');
}

export function toFilters(rows: FilterRow[], where = ''): Filter[] {
  const typed: Filter[] = where.trim() ? [{ column: '', op: 'sql', value: where.trim() }] : [];
  return [...typed, ...rows.filter(ready).map(r => ({ column: r.column, op: r.op, value: r.value.trim() }))];
}

export function appliedWhere(filters: Filter[]): string {
  return filters.find(f => f.op === 'sql')?.value ?? '';
}
