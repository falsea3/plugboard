import type { Column, EngineFeatures, StructureChange, TableInfo } from '../api/wire';
import type { MenuItem } from '../grid/grid';
import { quoteIdent } from '../sql/generate';

export type TreeTarget = { table: TableInfo; column?: Column };

export function treeMenu({ table, column }: TreeTarget, readOnly: boolean, canAlter: boolean): MenuItem[] {
  const view = table.kind === 'view';
  const off = readOnly;
  if (column) {
    return [
      { id: 'tree-copy-name', label: 'Copy column name' },
      { id: 'tree-query', label: `New query: SELECT ${column.name}` },
      { id: 'tree-structure', label: 'Open structure' },
      'sep',
      { id: 'tree-rename', label: 'Rename column…', disabled: off || view },
      { id: 'tree-type', label: 'Change type…', disabled: off || view || !canAlter },
      { id: 'tree-drop', label: 'Drop column…', danger: true, disabled: off || view },
    ];
  }
  return [
    { id: 'tree-open', label: 'Open data' },
    { id: 'tree-structure', label: 'Open structure' },
    { id: 'tree-ddl', label: 'Show DDL' },
    { id: 'tree-query', label: 'New query: SELECT *' },
    { id: 'tree-diagram', label: 'Show in diagram' },
    'sep',
    { id: 'tree-copy-name', label: `Copy ${view ? 'view' : 'table'} name` },
    { id: 'tree-copy-select', label: 'Copy SELECT *' },
    'sep',
    { id: 'tree-rename', label: 'Rename table…', disabled: off || view },
    { id: 'tree-truncate', label: 'Delete all rows…', disabled: off || view },
    { id: 'tree-drop', label: view ? 'Drop view…' : 'Drop table…', danger: true, disabled: off },
  ];
}

export function treeSQL(id: string, e: EngineFeatures, { table, column }: TreeTarget): string | null {
  const q = (name: string) => quoteIdent(e, name);
  const t = (table.schema ? q(table.schema) + '.' : '') + q(table.name);
  if (column) {
    const c = q(column.name);
    if (id === 'tree-query') return `SELECT ${c}\nFROM ${t}\nLIMIT 100;`;
    if (id === 'tree-drop') return `ALTER TABLE ${t}\n  DROP COLUMN ${c};`;
    return null;
  }
  if (id === 'tree-query' || id === 'tree-copy-select') return `SELECT *\nFROM ${t}\nLIMIT 100;`;
  if (id === 'tree-truncate') return `DELETE FROM ${t};`;
  if (id === 'tree-drop') return `DROP ${table.kind === 'view' ? 'VIEW' : 'TABLE'} ${t};`;
  return null;
}

export interface TreePrompt {
  title: string;
  label: string;
  value: string;
  suggestions: string[];
}

const COMMON_TYPES = ['integer', 'bigint', 'smallint', 'numeric(12,2)', 'real', 'varchar(255)', 'text', 'boolean', 'date', 'timestamp'];

export function treePrompt(id: string, { table, column }: TreeTarget, known: Column[] = []): TreePrompt | null {
  if (id === 'tree-rename' && column) return { title: `Rename ${table.name}.${column.name}`, label: 'New name', value: column.name, suggestions: [] };
  if (id === 'tree-rename') return { title: `Rename ${table.name}`, label: 'New name', value: table.name, suggestions: [] };
  if (id === 'tree-type' && column) {
    const used = known.map(c => c.type);
    return { title: `Change the type of ${table.name}.${column.name}`, label: 'New type', value: column.type, suggestions: [...new Set([...used, ...COMMON_TYPES])] };
  }
  return null;
}

export function columnChange(id: string, { table, column }: TreeTarget, value: string): StructureChange | null {
  if (!column || !value.trim() || (id !== 'tree-rename' && id !== 'tree-type')) return null;
  const change = { kind: 'update' as const, column: column.name, defaultSet: false, default: null };
  return {
    schema: table.schema,
    table: table.name,
    changes: [id === 'tree-rename' ? { ...change, name: value.trim() } : { ...change, type: value.trim() }],
  };
}
