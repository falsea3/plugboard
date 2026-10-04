import type { CellValue, Filter, Relation } from '../api/wire';

export function linkedColumns(relations: Relation[], names: string[]): Set<number> {
  const out = new Set<number>();
  for (const rel of relations) for (const col of rel.columns) if (names.includes(col)) out.add(names.indexOf(col));
  return out;
}

export function jumpTarget(relations: Relation[], names: string[], row: CellValue[], c: number): { rel: Relation; filters: Filter[] } | null {
  const rel = relations.find(x => x.columns.includes(names[c]));
  if (!rel) return null;
  const values = rel.columns.map(col => row[names.indexOf(col)]);
  if (values.some(v => v === null || v === undefined)) return null;
  return { rel, filters: rel.refColumns.map((col, i) => ({ column: col, op: '=', value: String(values[i]) })) };
}
