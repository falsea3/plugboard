import { engine } from '../engines';
import type { CellValue, ColumnKind, Connection } from '../api/wire';

export const isTrue = (v: CellValue) => v === true || v === 1 || v === '1' || v === 't' || v === 'true';

export type CellKind = 'null' | 'number' | 'bool' | 'text';

export function cellKind(value: CellValue, kind: ColumnKind): CellKind {
  if (value === null) return 'null';
  if (typeof value === 'boolean') return 'bool';
  if (typeof value === 'number' || kind === 'number') return 'number';
  return 'text';
}

export function formatCell(value: CellValue): string {
  if (value === null) return 'NULL';
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  const s = String(value);
  const nl = s.search(/[\r\n]/);
  const line = nl >= 0 ? s.slice(0, nl) + ' ↵' : s;
  return line.length > 300 ? line.slice(0, 300) + '…' : line;
}

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

export function connectionTarget(c: Connection): string {
  if (engine(c.driver).file) {
    return c.file.split(/[\\/]/).pop() || c.file;
  }
  const port = c.port ? `:${c.port}` : '';
  const db = c.database ? `/${c.database}` : '';
  return `${c.host}${port}${db}`;
}

export function toTSV(columns: string[], rows: CellValue[][]): string {
  const esc = (v: string) => (/[\t\n\r"]/.test(v) ? `"${v.replaceAll('"', '""')}"` : v);
  const lines = [columns.map(esc).join('\t')];
  for (const r of rows) lines.push(r.map(v => esc(copyText(v))).join('\t'));
  return lines.join('\n');
}

export function formatBytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return i === 0 ? `${v} B` : `${v < 10 ? v.toFixed(1) : Math.round(v)} ${units[i]}`;
}
