import type { Connection, ConnectSecrets, ConnectResult, TestResult, SessionInfo } from './wire';
import type { TableInfo, Column, Relation, Index, Diagram, DBObject, NewIndex } from './wireSchema';
import type { ResultSet, TableQuery, RowCount, TablePage, QueryRun, SyntaxProblem, ChangeSet, ApplyResult, StructureChange } from './wireQuery';
import type { KeyEdit, KeyPage, KeyValue } from './wireKeys';

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
  autoUpdate: boolean;
}

export const DEFAULT_SETTINGS: Settings = {
  theme: 'system',
  pageSize: 300,
  editorFontSize: 13,
  confirmProdWrites: true,
  autoUpdate: false,
};

export interface UpdateInfo {
  version: string;
  notes: string;
  publishedAt: string;
  releaseUrl: string;
}

export interface UpdateCheck {
  available?: UpdateInfo;
  error?: string;
}

export interface QueryTabState {
  title: string;
  sql: string;
  script?: string;
  saved: string;
  active?: boolean;
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
  Diagram(sessionId: string, schema: string): Promise<Diagram>;
  Relations(sessionId: string, schema: string): Promise<Relation[]>;
  Indexes(sessionId: string, schema: string, table: string): Promise<Index[]>;
  ListObjects(sessionId: string, schema: string): Promise<DBObject[]>;
  ObjectDDL(sessionId: string, obj: DBObject): Promise<string>;
  FetchTablePage(sessionId: string, q: TableQuery): Promise<TablePage>;
  CountRows(sessionId: string, queryId: string, q: TableQuery, exact: boolean): Promise<RowCount>;
  RunQuery(sessionId: string, queryId: string, sql: string): Promise<QueryRun>;
  WriteStatements(sessionId: string, script: string): Promise<string[]>;
  CheckSyntax(sessionId: string, script: string): Promise<SyntaxProblem[]>;
  RunMore(sessionId: string, queryId: string, statement: string, offset: number): Promise<ResultSet>;
  CancelQuery(queryId: string): Promise<void>;
  ChooseSQLiteFile(): Promise<string>;
  ApplyChanges(sessionId: string, cs: ChangeSet): Promise<ApplyResult>;
  PreviewChanges(sessionId: string, cs: ChangeSet): Promise<string[]>;
  ApplyStructure(sessionId: string, queryId: string, sc: StructureChange): Promise<ApplyResult>;
  PreviewStructure(sessionId: string, sc: StructureChange): Promise<string[]>;
  RenameTableSQL(sessionId: string, schema: string, from: string, to: string): Promise<string>;
  CreateIndexSQL(sessionId: string, schema: string, table: string, idx: NewIndex): Promise<string>;
  DropIndexSQL(sessionId: string, schema: string, table: string, name: string): Promise<string>;
  ScanKeys(sessionId: string, db: string, pattern: string, cursor: string, count: number): Promise<KeyPage>;
  ReadKey(sessionId: string, db: string, key: string, cursor: string): Promise<KeyValue>;
  EditKey(sessionId: string, db: string, edit: KeyEdit): Promise<void>;
  ListScripts(connId: string): Promise<string[]>;
  ReadScript(connId: string, name: string): Promise<string>;
  CreateScript(connId: string, name: string, sql: string): Promise<void>;
  WriteScript(connId: string, name: string, sql: string): Promise<void>;
  RenameScript(connId: string, from: string, to: string): Promise<void>;
  DeleteScript(connId: string, name: string): Promise<void>;
  QueryTabs(connId: string): Promise<QueryTabState[]>;
  SaveQueryTabs(connId: string, tabs: QueryTabState[]): Promise<void>;
  MCPCommand(): Promise<string>;
  MCPLog(): Promise<string>;
  GetSettings(): Promise<Settings>;
  SaveSettings(s: Settings): Promise<Settings>;
  OpenDataFolder(): Promise<void>;
  OpenLogs(): Promise<void>;
  LastCrash(): Promise<string>;
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
