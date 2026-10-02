export type Driver = 'postgres' | 'mysql' | 'sqlite';
export type Env = '' | 'local' | 'dev' | 'staging' | 'prod';

export interface Connection {
  id: string;
  name: string;
  driver: Driver;
  host: string;
  port: number;
  user: string;
  password?: string;
  savePassword: boolean;
  database: string;
  file: string;
  sslMode: string;
  env: Env;
  color: string;
  readOnly: boolean;
  ssh: SSHTunnel;
}

export type SSHAuth = 'password' | 'key' | 'agent';

export interface SSHTunnel {
  enabled: boolean;
  host: string;
  port: number;
  user: string;
  auth: SSHAuth;
  keyFile: string;
  password?: string;
  passphrase?: string;
}

export interface ConnectSecrets {
  password: string;
  sshPassword: string;
  sshPassphrase: string;
}

export interface HostKeyChange {
  host: string;
  fingerprint: string;
}

/** An open session, or the SSH host key change that stopped it opening. */
export interface ConnectResult {
  session?: SessionInfo;
  hostKeyChange?: HostKeyChange;
}

export interface TunnelState {
  sessionId: string;
  state: 'lost' | 'reconnected' | 'failed';
  error?: string;
}

export interface TestResult {
  ok: boolean;
  error?: string;
  serverVersion?: string;
  latencyMs: number;
}

export interface SessionInfo {
  sessionId: string;
  connection: Connection;
  serverVersion: string;
  schemas: string[];
  defaultSchema: string;
}

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
  /** allowed values of an enum column, in declaration order */
  enum: string[] | null;
  /** shown as a hex preview, so the grid can't write it back */
  binary: boolean;
}

export interface ResultColumn {
  name: string;
  type: string;
}

export type CellValue = string | number | boolean | null;

export interface ResultSet {
  statement: string;
  columns: ResultColumn[];
  rows: CellValue[][];
  rowsAffected: number;
  hasRows: boolean;
  truncated: boolean;
  /** came from a wrapped read; more chunks via RunMore from offset + rows.length */
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
  /** keyset paging in primary-key order */
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
  /** primary key columns the rows were sorted by when no sort was given */
  defaultOrder: string[] | null;
  hasPrev: boolean;
  /** pages are found by primary key, not OFFSET */
  keyset: boolean;
  /** first row index when the server worked it out, else -1 */
  offset: number;
}

export interface QueryRun {
  results: ResultSet[];
  error?: string;
  errorIndex: number;
  cancelled: boolean;
  /** the editor's connection closed with a transaction open, so it was rolled back */
  rolledBack: boolean;
}

export type ChangeKind = 'update' | 'insert' | 'delete';

export interface RowChange {
  kind: ChangeKind;
  key: Record<string, CellValue>;
  /** new values; {$expr} asks the server for now() or the column default */
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
  /** changes before the failed one stayed applied (MySQL commits each ALTER TABLE) */
  partial: boolean;
  /** the user stopped it (structure changes) */
  cancelled: boolean;
}

/** One edit from the Structure tab; type and default are SQL as typed. */
export interface ColumnChange {
  kind: ChangeKind;
  /** the existing column (update, delete) */
  column: string;
  name?: string;
  type?: string;
  nullable?: boolean;
  /** default changed; a null default drops it */
  defaultSet: boolean;
  default: string | null;
}

export interface StructureChange {
  schema: string;
  table: string;
  changes: ColumnChange[];
}

export interface AppInfo {
  name: string;
  version: string;
  goVersion: string;
  platform: string;
  dataDir: string;
  copyright: string;
}

export type ThemeMode = 'system' | 'light' | 'dark';

export interface Settings {
  theme: ThemeMode;
  pageSize: number;
  editorFontSize: number;
  confirmProdWrites: boolean;
  /** install a new release as soon as it is found */
  autoUpdate: boolean;
}

export const DEFAULT_SETTINGS: Settings = {
  theme: 'system',
  pageSize: 300,
  editorFontSize: 13,
  confirmProdWrites: true,
  autoUpdate: false,
};

/** A release newer than the running app. */
export interface UpdateInfo {
  version: string;
  /** Markdown from the changelog */
  notes: string;
  publishedAt: string;
  releaseUrl: string;
}

/** available is absent when this is the newest release, or when the check failed (error). */
export interface UpdateCheck {
  available?: UpdateInfo;
  error?: string;
}

export interface GoApp {
  AppInfo(): Promise<AppInfo>;
  ListConnections(): Promise<Connection[]>;
  SaveConnection(c: Connection): Promise<Connection>;
  DeleteConnection(id: string): Promise<void>;
  TestConnection(c: Connection): Promise<TestResult>;
  Connect(id: string, secrets: ConnectSecrets): Promise<ConnectResult>;
  SetReadOnly(sessionId: string, readOnly: boolean): Promise<SessionInfo>;
  ChooseSSHKeyFile(): Promise<string>;
  TrustHostKey(host: string, fingerprint: string): Promise<void>;
  Disconnect(sessionId: string): Promise<void>;
  ListTables(sessionId: string, schema: string): Promise<TableInfo[]>;
  DescribeTable(sessionId: string, schema: string, table: string): Promise<Column[]>;
  FetchTablePage(sessionId: string, q: TableQuery): Promise<TablePage>;
  CountRows(sessionId: string, queryId: string, q: TableQuery, exact: boolean): Promise<RowCount>;
  RunQuery(sessionId: string, queryId: string, sql: string): Promise<QueryRun>;
  WriteStatements(sessionId: string, script: string): Promise<string[]>;
  RunMore(sessionId: string, queryId: string, statement: string, offset: number): Promise<ResultSet>;
  CancelQuery(queryId: string): Promise<void>;
  ChooseSQLiteFile(): Promise<string>;
  ApplyChanges(sessionId: string, cs: ChangeSet): Promise<ApplyResult>;
  PreviewChanges(sessionId: string, cs: ChangeSet): Promise<string[]>;
  ApplyStructure(sessionId: string, queryId: string, sc: StructureChange): Promise<ApplyResult>;
  PreviewStructure(sessionId: string, sc: StructureChange): Promise<string[]>;
  GetSettings(): Promise<Settings>;
  SaveSettings(s: Settings): Promise<Settings>;
  OpenDataFolder(): Promise<void>;
  CheckForUpdate(): Promise<UpdateCheck>;
  InstallUpdate(): Promise<string>;
  RestartApp(): Promise<void>;
}

declare global {
  interface Window {
    go?: { api?: { App?: GoApp } };
    runtime?: {
      ClipboardSetText?: (text: string) => Promise<boolean>;
      BrowserOpenURL?: (url: string) => void;
      EventsOn?: (name: string, cb: (...data: unknown[]) => void) => () => void;
    };
  }
}

export const DEFAULT_PORTS: Record<Driver, number> = {
  postgres: 5432,
  mysql: 3306,
  sqlite: 0,
};

export const DRIVER_LABELS: Record<Driver, string> = {
  postgres: 'PostgreSQL',
  mysql: 'MySQL',
  sqlite: 'SQLite',
};

export function emptyConnection(driver: Driver = 'postgres'): Connection {
  return {
    id: '',
    name: '',
    driver,
    host: driver === 'sqlite' ? '' : '127.0.0.1',
    port: DEFAULT_PORTS[driver],
    user: driver === 'postgres' ? 'postgres' : driver === 'mysql' ? 'root' : '',
    password: '',
    savePassword: true,
    database: '',
    file: '',
    sslMode: '',
    env: '',
    color: '',
    readOnly: false,
    ssh: { enabled: false, host: '', port: 22, user: '', auth: 'password', keyFile: '', password: '', passphrase: '' },
  };
}
