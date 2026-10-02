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

export type RowStatement = 'select' | 'insert' | 'update' | 'delete';

export interface RowsTarget {
  schema: string;
  table: string;
  columns: string[];
  key: string[];
}

function tableName(e: EngineFeatures, t: RowsTarget): string {
  return (t.schema ? quoteIdent(e, t.schema) + '.' : '') + quoteIdent(e, t.table);
}

function keyMatch(e: EngineFeatures, t: RowsTarget, row: CellValue[]): string {
  return t.key.map(k => `${quoteIdent(e, k)} = ${literal(e, row[t.columns.indexOf(k)])}`).join(' AND ');
}

function whereKeys(e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  if (rows.length === 1) return keyMatch(e, t, rows[0]);
  if (t.key.length === 1) {
    const i = t.columns.indexOf(t.key[0]);
    return `${quoteIdent(e, t.key[0])} IN (${rows.map(r => literal(e, r[i])).join(', ')})`;
  }
  return rows.map(r => `(${keyMatch(e, t, r)})`).join('\n   OR ');
}

export function insertStatement(e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  const cols = t.columns.map(c => quoteIdent(e, c)).join(', ');
  const values = rows.map(r => '(' + r.map(v => literal(e, v)).join(', ') + ')');
  return `INSERT INTO ${tableName(e, t)} (${cols}) VALUES\n  ${values.join(',\n  ')};`;
}

export function selectStatement(e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  return `SELECT * FROM ${tableName(e, t)}\nWHERE ${whereKeys(e, t, rows)};`;
}

export function deleteStatement(e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  return `DELETE FROM ${tableName(e, t)}\nWHERE ${whereKeys(e, t, rows)};`;
}

export function updateStatement(e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  const rest = t.columns.filter(c => !t.key.includes(c));
  const set = rest.length > 0 ? rest : t.key;
  return rows
    .map(r => {
      const pairs = set.map(c => `${quoteIdent(e, c)} = ${literal(e, r[t.columns.indexOf(c)])}`);
      return `UPDATE ${tableName(e, t)} SET\n  ${pairs.join(',\n  ')}\nWHERE ${keyMatch(e, t, r)};`;
    })
    .join('\n\n');
}

export function rowsStatement(kind: RowStatement, e: EngineFeatures, t: RowsTarget, rows: CellValue[][]): string {
  switch (kind) {
    case 'select':
      return selectStatement(e, t, rows);
    case 'insert':
      return insertStatement(e, t, rows);
    case 'update':
      return updateStatement(e, t, rows);
    case 'delete':
      return deleteStatement(e, t, rows);
  }
}
