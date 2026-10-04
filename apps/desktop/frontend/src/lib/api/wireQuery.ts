import type { ColumnKind } from './wireSchema';

export interface ResultColumn {
  name: string;
  type: string;
  kind: ColumnKind;
}

export type CellValue = string | number | boolean | null;

export interface ResultSet {
  statement: string;
  columns: ResultColumn[];
  rows: CellValue[][];
  rowsAffected: number;
  hasRows: boolean;
  truncated: boolean;
  pageable: boolean;
  hasMore: boolean;
  offset: number;
  durationMs: number;
}

export type FilterOp =
  | '=' | '!=' | '<' | '>' | '<=' | '>='
  | 'contains' | 'not_contains' | 'starts' | 'ends'
  | 'in' | 'not_in' | 'null' | 'not_null' | 'empty' | 'not_empty';

export interface Filter {
  column: string;
  op: FilterOp;
  value: string;
}

export interface TableQuery {
  schema: string;
  table: string;
  offset: number;
  limit: number;
  orderBy: string;
  orderDesc: boolean;
  filters: Filter[];
  after?: CellValue[];
  before?: CellValue[];
  last?: boolean;
}

export interface RowCount {
  count: number;
  exact: boolean;
  known: boolean;
}

export interface TablePage {
  result: ResultSet;
  hasMore: boolean;
  defaultOrder: string[] | null;
  hasPrev: boolean;
  keyset: boolean;
  offset: number;
}

export interface QueryRun {
  results: ResultSet[];
  error?: string;
  errorIndex: number;
  errorPosition: number;
  cancelled: boolean;
  rolledBack: boolean;
}

export interface SyntaxProblem {
  index: number;
  position: number;
  message: string;
}

export type ChangeKind = 'update' | 'insert' | 'delete';

export interface RowChange {
  kind: ChangeKind;
  key: Record<string, CellValue>;
  values: Record<string, CellValue | { $expr: 'now' | 'default' }>;
}

export interface ChangeSet {
  schema: string;
  table: string;
  changes: RowChange[];
}

export interface ApplyResult {
  applied: number;
  error?: string;
  failedIndex: number;
  partial: boolean;
  cancelled: boolean;
}

export interface ColumnChange {
  kind: ChangeKind;
  column: string;
  name?: string;
  type?: string;
  nullable?: boolean;
  defaultSet: boolean;
  default: string | null;
}

export interface StructureChange {
  schema: string;
  table: string;
  changes: ColumnChange[];
}
