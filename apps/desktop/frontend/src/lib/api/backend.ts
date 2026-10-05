import type { ChangeSet, ConnectSecrets, Connection, DBObject, GoApp, Settings, StructureChange, TableQuery, TunnelState } from './wire';

export * from './wire';

function bridge(): GoApp {
  const app = window.go?.api?.App;
  if (!app) throw new Error('Desktop bridge unavailable — open the app through `make dev`.');
  return app;
}

export class AppError extends Error {
  constructor(
    message: string,
    readonly code = 'error',
    readonly detail = '',
  ) {
    super(message);
  }
}

export function toError(err: unknown): Error {
  if (err instanceof Error) return err;
  if (err && typeof err === 'object' && 'message' in err) {
    const e = err as { message: unknown; code?: unknown; detail?: unknown };
    return new AppError(String(e.message), String(e.code ?? 'error'), String(e.detail ?? ''));
  }
  return new Error(String(err));
}

async function call<T>(fn: (app: GoApp) => Promise<T>): Promise<T> {
  try {
    return await fn(bridge());
  } catch (err) {
    throw toError(err);
  }
}

export const hasBridge = () => Boolean(window.go?.api?.App);

export const api = {
  appInfo: () => call(a => a.AppInfo()),
  listConnections: () => call(a => a.ListConnections()),
  saveConnection: (c: Connection) => call(a => a.SaveConnection(c)),
  deleteConnection: (id: string) => call(a => a.DeleteConnection(id)),
  testConnection: (c: Connection) => call(a => a.TestConnection(c)),
  connect: (id: string, secrets: Partial<ConnectSecrets> = {}) =>
    call(a => a.Connect(id, { password: '', sshPassword: '', sshPassphrase: '', ...secrets })),
  setReadOnly: (sessionId: string, readOnly: boolean) => call(a => a.SetReadOnly(sessionId, readOnly)),
  chooseSSHKeyFile: () => call(a => a.ChooseSSHKeyFile()),
  trustHostKey: (host: string, fingerprint: string) => call(a => a.TrustHostKey(host, fingerprint)),
  disconnect: (sessionId: string) => call(a => a.Disconnect(sessionId)),
  listTables: (sessionId: string, schema: string) => call(a => a.ListTables(sessionId, schema)),
  describeTable: (sessionId: string, schema: string, table: string) => call(a => a.DescribeTable(sessionId, schema, table)),
  diagram: (sessionId: string, schema: string) => call(a => a.Diagram(sessionId, schema)),
  relations: (sessionId: string, schema: string) => call(a => a.Relations(sessionId, schema)),
  indexes: (sessionId: string, schema: string, table: string) => call(a => a.Indexes(sessionId, schema, table)),
  listObjects: (sessionId: string, schema: string) => call(a => a.ListObjects(sessionId, schema)),
  objectDDL: (sessionId: string, obj: DBObject) => call(a => a.ObjectDDL(sessionId, obj)),
  fetchTablePage: (sessionId: string, q: TableQuery) => call(a => a.FetchTablePage(sessionId, q)),
  countRows: (sessionId: string, queryId: string, q: TableQuery, exact: boolean) =>
    call(a => a.CountRows(sessionId, queryId, q, exact)),
  runQuery: (sessionId: string, queryId: string, sql: string) => call(a => a.RunQuery(sessionId, queryId, sql)),
  writeStatements: (sessionId: string, script: string) => call(a => a.WriteStatements(sessionId, script)),
  checkSyntax: (sessionId: string, script: string) => call(a => a.CheckSyntax(sessionId, script)),
  runMore: (sessionId: string, queryId: string, statement: string, offset: number) =>
    call(a => a.RunMore(sessionId, queryId, statement, offset)),
  cancelQuery: (queryId: string) => call(a => a.CancelQuery(queryId)),
  chooseSQLiteFile: () => call(a => a.ChooseSQLiteFile()),
  applyChanges: (sessionId: string, cs: ChangeSet) => call(a => a.ApplyChanges(sessionId, cs)),
  previewChanges: (sessionId: string, cs: ChangeSet) => call(a => a.PreviewChanges(sessionId, cs)),
  applyStructure: (sessionId: string, queryId: string, sc: StructureChange) => call(a => a.ApplyStructure(sessionId, queryId, sc)),
  previewStructure: (sessionId: string, sc: StructureChange) => call(a => a.PreviewStructure(sessionId, sc)),
  renameTableSQL: (sessionId: string, schema: string, from: string, to: string) => call(a => a.RenameTableSQL(sessionId, schema, from, to)),
  getSettings: () => call(a => a.GetSettings()),
  saveSettings: (s: Settings) => call(a => a.SaveSettings(s)),
  openDataFolder: () => call(a => a.OpenDataFolder()),
  openLogs: () => call(a => a.OpenLogs()),
  lastCrash: () => call(a => a.LastCrash()),
  checkForUpdate: () => call(a => a.CheckForUpdate()),
  installUpdate: () => call(a => a.InstallUpdate()),
  restartApp: () => call(a => a.RestartApp()),
};

export function onMenuCommand(cb: (command: string) => void): () => void {
  return window.runtime?.EventsOn?.('menu:command', data => cb(String(data))) ?? (() => {});
}

export function onTunnelState(cb: (s: TunnelState) => void): () => void {
  return window.runtime?.EventsOn?.('session:tunnel', data => cb(data as TunnelState)) ?? (() => {});
}
