import type { ChangeSet, ConnectSecrets, Connection, GoApp, Settings, TableQuery, TunnelState } from './wire';

export * from './wire';

function bridge(): GoApp {
  const app = window.go?.api?.App;
  if (!app) throw new Error('Desktop bridge unavailable — open the app through `make dev`.');
  return app;
}

// Wails rejects with the Go error string; normalise it into an Error.
async function call<T>(fn: (app: GoApp) => Promise<T>): Promise<T> {
  try {
    return await fn(bridge());
  } catch (err) {
    throw err instanceof Error ? err : new Error(String(err));
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
  fetchTablePage: (sessionId: string, q: TableQuery) => call(a => a.FetchTablePage(sessionId, q)),
  countRows: (sessionId: string, queryId: string, q: TableQuery, exact: boolean) =>
    call(a => a.CountRows(sessionId, queryId, q, exact)),
  runQuery: (sessionId: string, queryId: string, sql: string) => call(a => a.RunQuery(sessionId, queryId, sql)),
  writeStatements: (sessionId: string, script: string) => call(a => a.WriteStatements(sessionId, script)),
  runMore: (sessionId: string, queryId: string, statement: string, offset: number) =>
    call(a => a.RunMore(sessionId, queryId, statement, offset)),
  cancelQuery: (queryId: string) => call(a => a.CancelQuery(queryId)),
  chooseSQLiteFile: () => call(a => a.ChooseSQLiteFile()),
  applyChanges: (sessionId: string, cs: ChangeSet) => call(a => a.ApplyChanges(sessionId, cs)),
  previewChanges: (sessionId: string, cs: ChangeSet) => call(a => a.PreviewChanges(sessionId, cs)),
  getSettings: () => call(a => a.GetSettings()),
  saveSettings: (s: Settings) => call(a => a.SaveSettings(s)),
  openDataFolder: () => call(a => a.OpenDataFolder()),
  checkForUpdate: () => call(a => a.CheckForUpdate()),
  installUpdate: () => call(a => a.InstallUpdate()),
  restartApp: () => call(a => a.RestartApp()),
};

/** Subscribe to native menu commands; a no-op outside the desktop app. */
export function onMenuCommand(cb: (command: string) => void): () => void {
  return window.runtime?.EventsOn?.('menu:command', data => cb(String(data))) ?? (() => {});
}

/** SSH tunnel drops and reconnects for open sessions. */
export function onTunnelState(cb: (s: TunnelState) => void): () => void {
  return window.runtime?.EventsOn?.('session:tunnel', data => cb(data as TunnelState)) ?? (() => {});
}
