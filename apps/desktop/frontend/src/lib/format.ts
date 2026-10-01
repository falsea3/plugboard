import type { CellValue, Connection } from './wire';

const NUMERIC_TYPE = /^(int|integer|smallint|bigint|tinyint|mediumint|int2|int4|int8|serial|smallserial|bigserial|numeric|decimal|real|double|float|float4|float8|money|oid)\b/i;

/** An array of a type is not that type: int4[]. */
const isArray = (type: string) => type.trim().endsWith('[]');

export function isNumericType(type: string): boolean {
  return NUMERIC_TYPE.test(type.trim()) && !isArray(type);
}

export type CellKind = 'null' | 'number' | 'bool' | 'text';

export function cellKind(value: CellValue, type: string): CellKind {
  if (value === null) return 'null';
  if (typeof value === 'boolean') return 'bool';
  if (typeof value === 'number' || isNumericType(type)) return 'number';
  return 'text';
}

/** Single-line rendering for a grid cell. */
export function formatCell(value: CellValue): string {
  if (value === null) return 'NULL';
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  const s = String(value);
  const nl = s.search(/[\r\n]/);
  const line = nl >= 0 ? s.slice(0, nl) + ' ↵' : s;
  return line.length > 300 ? line.slice(0, 300) + '…' : line;
}

/** Full value for the clipboard: NULL copies as empty. */
export function copyText(value: CellValue): string {
  if (value === null) return '';
  return String(value);
}

export function formatDuration(ms: number): string {
  if (ms < 1) return `${ms.toFixed(2)} ms`;
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(2)} s`;
}

export function formatCount(n: number, one: string, many = one + 's'): string {
  return `${n.toLocaleString('en-US')} ${n === 1 ? one : many}`;
}

/** host:port/db or the SQLite file name, for list rows and the title bar. */
export function connectionTarget(c: Connection): string {
  if (c.driver === 'sqlite') {
    return c.file.split(/[\\/]/).pop() || c.file;
  }
  const port = c.port ? `:${c.port}` : '';
  const db = c.database ? `/${c.database}` : '';
  return `${c.host}${port}${db}`;
}

/** Rows as TSV, ready to paste into a spreadsheet. */
export function toTSV(columns: string[], rows: CellValue[][]): string {
  const esc = (v: string) => (/[\t\n\r"]/.test(v) ? `"${v.replaceAll('"', '""')}"` : v);
  const lines = [columns.map(esc).join('\t')];
  for (const r of rows) lines.push(r.map(v => esc(copyText(v))).join('\t'));
  return lines.join('\n');
}

/** Rough column width from the header and a sample of values. */
export function calcColumnWidth(name: string, type: string, rows: CellValue[][], index: number): number {
  const charW = 7.4;
  let longest = Math.max(name.length + 2, Math.min(type.length, 14));
  const sample = Math.min(rows.length, 200);
  for (let i = 0; i < sample; i++) {
    const len = formatCell(rows[i][index]).length;
    if (len > longest) longest = len;
  }
  return Math.round(Math.min(Math.max(longest * charW + 24, 64), 360));
}

// Column kinds, from catalog type names across PostgreSQL, MySQL and SQLite.

/** boolean, bool, and MySQL's tinyint(1) convention. */
export function isBoolType(type: string): boolean {
  return /^(boolean|bool)$/i.test(type.trim()) || /^tinyint\(1\)/i.test(type.trim());
}

/** date, time, timestamp, datetime with or without time zone (not daterange). */
export function isDateTimeType(type: string): boolean {
  return /^(date|datetime|time|timetz|timestamp|timestamptz)\b/i.test(type.trim()) && !isArray(type);
}

/** Types where "empty" ('') means something: char, varchar, text, clob. */
export function isTextType(type: string): boolean {
  return /char|text|clob|string/i.test(type) && !isArray(type);
}

/** A name for a SQLite file: "/data/shop.sqlite3" → "shop". */
export function sqliteName(path: string): string {
  const file = path.split(/[\\/]/).pop() ?? '';
  return file.replace(/\.(db|sqlite3?|db3)$/i, '') || file || 'SQLite';
}
