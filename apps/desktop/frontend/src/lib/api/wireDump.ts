export interface DumpOptions {
  schemas: string[];
  structure: boolean;
  data: boolean;
  clean: boolean;
  inserts: boolean;
  owners: boolean;
  format: string;
}

export interface DumpFormat {
  id: string;
  label: string;
  ext: string;
  hint: string;
}

export interface DumpProgress {
  phase: string;
  object: string;
  tables: number;
  done: number;
  rows: number;
  bytes: number;
}

export interface DumpResult {
  path: string;
  bytes: number;
  rows: number;
  tables: number;
}

export interface DumpFile {
  path: string;
  name: string;
  size: number;
}

export interface ImportOptions {
  oneTransaction: boolean;
  stopOnError: boolean;
}

export interface ImportProgress {
  bytes: number;
  total: number;
  statements: number;
  errors: number;
  current: string;
}

export interface ImportError {
  index: number;
  statement: string;
  message: string;
}

export interface ImportResult {
  statements: number;
  rows: number;
  skipped: string[];
  errors: ImportError[];
  stopped: boolean;
  rolledBack: boolean;
}
