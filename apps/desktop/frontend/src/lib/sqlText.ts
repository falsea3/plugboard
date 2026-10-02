import type { CellValue, EngineFeatures } from './wire';

export function quoteIdent(e: EngineFeatures, name: string): string {
  if (e.syntax.doubleQuotedStrings) return '`' + name.replaceAll('`', '``') + '`';
  return '"' + name.replaceAll('"', '""') + '"';
}

export function literal(e: EngineFeatures, v: CellValue): string {
  if (v === null) return 'NULL';
  if (typeof v === 'number') return String(v);
  if (typeof v === 'boolean') return e.booleanType ? String(v) : v ? '1' : '0';
  let s = v.replaceAll("'", "''");
  if (e.syntax.backslashEscapes) s = s.replaceAll('\\', '\\\\');
  return `'${s}'`;
}

export function insertStatement(e: EngineFeatures, schema: string, table: string, columns: string[], rows: CellValue[][]): string {
  const target = (schema ? quoteIdent(e, schema) + '.' : '') + quoteIdent(e, table);
  const cols = columns.map(c => quoteIdent(e, c)).join(', ');
  const values = rows.map(r => '(' + r.map(v => literal(e, v)).join(', ') + ')');
  return `INSERT INTO ${target} (${cols}) VALUES\n  ${values.join(',\n  ')};`;
}
