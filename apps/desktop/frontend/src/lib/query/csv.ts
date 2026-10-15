import type { CellValue } from '../api/wire';

function field(v: CellValue): string {
  if (v === null) return '';
  const s = typeof v === 'boolean' ? (v ? 'true' : 'false') : String(v);
  return /[",\r\n]/.test(s) || /^\s|\s$/.test(s) ? `"${s.replaceAll('"', '""')}"` : s;
}

export function toCSV(columns: string[], rows: CellValue[][]): string {
  return [columns, ...rows].map(r => r.map(field).join(',')).join('\r\n') + '\r\n';
}
