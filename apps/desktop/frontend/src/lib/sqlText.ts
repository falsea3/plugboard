import type { CellValue, Driver } from './wire';

export function quoteIdent(driver: Driver, name: string): string {
  if (driver === 'mysql') return '`' + name.replaceAll('`', '``') + '`';
  return '"' + name.replaceAll('"', '""') + '"';
}

export function literal(driver: Driver, v: CellValue): string {
  if (v === null) return 'NULL';
  if (typeof v === 'number') return String(v);
  if (typeof v === 'boolean') return driver === 'postgres' ? String(v) : v ? '1' : '0';
  let s = v.replaceAll("'", "''");
  if (driver === 'mysql') s = s.replaceAll('\\', '\\\\');
  return `'${s}'`;
}

export function insertStatement(driver: Driver, schema: string, table: string, columns: string[], rows: CellValue[][]): string {
  const target = (schema ? quoteIdent(driver, schema) + '.' : '') + quoteIdent(driver, table);
  const cols = columns.map(c => quoteIdent(driver, c)).join(', ');
  const values = rows.map(r => '(' + r.map(v => literal(driver, v)).join(', ') + ')');
  return `INSERT INTO ${target} (${cols}) VALUES\n  ${values.join(',\n  ')};`;
}
