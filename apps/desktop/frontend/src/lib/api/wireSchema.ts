export interface TableInfo {
  schema: string;
  name: string;
  kind: 'table' | 'view';
}

export interface Column {
  name: string;
  type: string;
  nullable: boolean;
  default: string | null;
  primaryKey: boolean;
  enum: string[] | null;
  kind: ColumnKind;
}

export interface Relation {
  name: string;
  table: string;
  columns: string[];
  refSchema: string;
  refTable: string;
  refColumns: string[];
  onDelete: string;
  onUpdate: string;
}

export interface Index {
  name: string;
  columns: string[];
  unique: boolean;
  primary: boolean;
  method: string;
  where: string;
  definition: string;
}

export interface DiagramTable {
  name: string;
  kind: TableInfo['kind'];
  columns: Column[];
}

export interface Diagram {
  schema: string;
  tables: DiagramTable[];
  relations: Relation[];
}

export type ColumnKind = '' | 'number' | 'bool' | 'datetime' | 'text' | 'json' | 'binary';
