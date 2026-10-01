import {
  api, onTunnelState, DEFAULT_SETTINGS, emptyConnection,
  type Connection, type ConnectSecrets, type HostKeyChange, type SessionInfo, type Settings, type TableInfo, type TunnelState, type UpdateInfo,
} from '../backend';
import { setThemeMode } from '../theme';
import { sqliteName } from '../format';

export type TableTab = {
  id: string;
  kind: 'table';
  schema: string;
  table: string;
  tableKind: TableInfo['kind'];
  /** has edits that aren't committed yet (kept current by TableView) */
  dirty?: boolean;
};

export type QueryTab = {
  id: string;
  kind: 'query';
  title: string;
  sql: string;
};

export type Tab = TableTab | QueryTab;

export type Toast = { id: number; kind: 'error' | 'info'; text: string };

export type SettingsSection = 'general' | 'editor' | 'about';

/** A newer release, from found to installed. */
export type Update =
  | { status: 'available' | 'installing' | 'installed'; info: UpdateInfo }
  | { status: 'failed'; info: UpdateInfo; error: string };

let seq = 0;
const nextId = (prefix: string) => `${prefix}-${Date.now().toString(36)}-${(++seq).toString(36)}`;

/** One open connection: its catalog and its tabs. Several can be open at once. */
export class Workspace {
  readonly id = nextId('ws');
  session = $state<SessionInfo>() as SessionInfo;
  switchingReadOnly = $state(false);
  /** 'ok', or the SSH tunnel is down and being re-established */
  tunnel = $state<'ok' | 'lost' | 'failed'>('ok');
  schema = $state('');
  tables = $state<TableInfo[]>([]);
  tablesLoading = $state(false);
  tablesError = $state('');
  tabs = $state<Tab[]>([]);
  activeTabId = $state('');
  activeTab = $derived(this.tabs.find(t => t.id === this.activeTabId) ?? null);

  constructor(session: SessionInfo) {
    this.session = session;
    this.schema = session.defaultSchema;
  }

  get connection() {
    return this.session.connection;
  }

  get dirtyTabs(): TableTab[] {
    return this.tabs.filter((t): t is TableTab => t.kind === 'table' && !!t.dirty);
  }

  get readOnly() {
    return this.session.connection.readOnly;
  }

  /** Reopens the session with read-only on or off; open tabs stay as they are. */
  async setReadOnly(on: boolean) {
    if (this.switchingReadOnly || on === this.readOnly) return;
    this.switchingReadOnly = true;
    try {
      this.session = await api.setReadOnly(this.session.sessionId, on);
    } catch (err) {
      app.notify(err);
    } finally {
      this.switchingReadOnly = false;
    }
  }

  async setSchema(schema: string) {
    if (schema === this.schema) return;
    this.schema = schema;
    await this.loadTables();
  }

  async loadTables() {
    this.tablesLoading = true;
    this.tablesError = '';
    try {
      this.tables = await api.listTables(this.session.sessionId, this.schema);
    } catch (err) {
      this.tablesError = err instanceof Error ? err.message : String(err);
      this.tables = [];
    } finally {
      this.tablesLoading = false;
    }
  }

  openTable(t: TableInfo) {
    const existing = this.tabs.find(tab => tab.kind === 'table' && tab.schema === t.schema && tab.table === t.name);
    if (existing) {
      this.activeTabId = existing.id;
      return;
    }
    const tab: TableTab = { id: nextId('table'), kind: 'table', schema: t.schema, table: t.name, tableKind: t.kind };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  newQuery(sql = '') {
    const n = this.tabs.filter(t => t.kind === 'query').length + 1;
    const tab: QueryTab = { id: nextId('query'), kind: 'query', title: `Query ${n}`, sql };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  closeTab(id: string) {
    const i = this.tabs.findIndex(t => t.id === id);
    if (i < 0) return;
    this.tabs.splice(i, 1);
    if (this.activeTabId === id) {
      this.activeTabId = this.tabs[Math.min(i, this.tabs.length - 1)]?.id ?? '';
    }
  }

  selectTabByOffset(delta: number) {
    if (this.tabs.length === 0) return;
    const i = this.tabs.findIndex(t => t.id === this.activeTabId);
    this.activeTabId = this.tabs[(i + delta + this.tabs.length) % this.tabs.length].id;
  }
}

class AppState {
  connections = $state<Connection[]>([]);
  connectionsLoaded = $state(false);
  connectingId = $state('');

  workspaces = $state<Workspace[]>([]);
  activeId = $state('');
  /** The connections list is showing even though workspaces are open. */
  showHome = $state(false);
  active = $derived(this.showHome ? null : (this.workspaces.find(w => w.id === this.activeId) ?? null));

  settings = $state<Settings>({ ...DEFAULT_SETTINGS });

  // Overlays live here so the menu and shortcuts can open them from anywhere.
  editing = $state<Connection | null | undefined>(undefined); // undefined = form closed, null = new
  passwordFor = $state<Connection | null>(null);
  switcherOpen = $state(false);
  settingsOpen = $state<SettingsSection | null>(null);

  /** An SSH server presented a different key; waits for the user to trust it or not. */
  hostKeyChange = $state<(HostKeyChange & { connection: Connection; secrets?: Partial<ConnectSecrets> }) | null>(null);

  /** Closing something with uncommitted edits waits here for confirmation. */
  pendingClose = $state<{ ws: Workspace; tabId?: string; tables: string[] } | null>(null);

  toasts = $state<Toast[]>([]);

  update = $state<Update | null>(null);
  /** The update banner was closed; the update stays in Settings ▸ About. */
  updateDismissed = $state(false);

  async init() {
    onTunnelState(ev => this.onTunnel(ev));
    await Promise.all([this.loadConnections(), this.loadSettings()]);
    // Let the window settle before going to the network.
    setTimeout(() => this.checkForUpdate(), 3000);
  }

  /** Looks for a newer release; installs it straight away if the setting says so. Returns the check's error, if any. */
  async checkForUpdate(): Promise<string> {
    if (this.update?.status === 'installing' || this.update?.status === 'installed') return '';
    try {
      const { available, error } = await api.checkForUpdate();
      if (error) return error;
      this.update = available ? { status: 'available', info: available } : null;
      if (available && this.settings.autoUpdate) await this.installUpdate();
      return '';
    } catch (err) {
      return err instanceof Error ? err.message : String(err);
    }
  }

  async installUpdate() {
    const u = this.update;
    if (!u || u.status === 'installing' || u.status === 'installed') return;
    this.update = { status: 'installing', info: u.info };
    try {
      await api.installUpdate();
      this.update = { status: 'installed', info: u.info };
      this.updateDismissed = false;
    } catch (err) {
      this.update = { status: 'failed', info: u.info, error: err instanceof Error ? err.message : String(err) };
    }
  }

  /** Restarts into the installed update, after the same check closing a connection does. */
  restartToUpdate() {
    const dirty = this.workspaces.flatMap(ws => ws.dirtyTabs.map(t => t.table));
    if (dirty.length > 0) {
      this.notify(`Commit or discard the edits in ${dirty.join(', ')} first.`, 'info');
      return;
    }
    api.restartApp().catch(err => this.notify(err));
  }

  private onTunnel(ev: TunnelState) {
    const ws = this.workspaces.find(w => w.session.sessionId === ev.sessionId);
    if (!ws) return;
    const name = ws.connection.name;
    if (ev.state === 'reconnected') {
      if (ws.tunnel !== 'ok') this.notify(`SSH connection to ${name} is back.`, 'info');
      ws.tunnel = 'ok';
    } else if (ev.state === 'lost') {
      if (ws.tunnel === 'ok') this.notify(`SSH connection to ${name} dropped — reconnecting…`, 'info');
      ws.tunnel = 'lost';
    } else {
      ws.tunnel = 'failed';
    }
  }

  async loadSettings() {
    try {
      this.settings = await api.getSettings();
    } catch {
      // keep defaults
    }
    this.applySettings();
  }

  async updateSettings(patch: Partial<Settings>) {
    const next = { ...this.settings, ...patch };
    this.settings = next;
    this.applySettings();
    try {
      this.settings = await api.saveSettings(next);
    } catch (err) {
      this.notify(err);
    }
  }

  private applySettings() {
    setThemeMode(this.settings.theme);
    document.documentElement.style.setProperty('--editor-font-size', `${this.settings.editorFontSize}px`);
  }

  async loadConnections() {
    try {
      this.connections = await api.listConnections();
    } catch (err) {
      this.notify(err);
    } finally {
      this.connectionsLoaded = true;
    }
  }

  async saveConnection(c: Connection): Promise<Connection> {
    const saved = await api.saveConnection(c);
    await this.loadConnections();
    return saved;
  }

  async deleteConnection(id: string) {
    try {
      for (const ws of this.workspaces.filter(w => w.connection.id === id)) await this.close(ws);
      await api.deleteConnection(id);
      await this.loadConnections();
    } catch (err) {
      this.notify(err);
    }
  }

  workspaceFor(connectionId: string) {
    return this.workspaces.find(w => w.connection.id === connectionId);
  }

  /**
   * Opens a saved connection, or switches to it when it is already open.
   * secrets is undefined until the user has been asked for the ones the
   * profile doesn't save.
   */
  async open(c: Connection, secrets?: Partial<ConnectSecrets>): Promise<boolean> {
    const existing = this.workspaceFor(c.id);
    if (existing) {
      this.activate(existing);
      return true;
    }
    if (secrets === undefined && needsSecrets(c)) {
      this.passwordFor = c;
      return false;
    }
    this.connectingId = c.id;
    try {
      const { session, hostKeyChange } = await api.connect(c.id, secrets);
      if (!session) {
        if (hostKeyChange) this.hostKeyChange = { ...hostKeyChange, connection: c, secrets };
        return false;
      }
      const ws = new Workspace(session);
      this.workspaces.push(ws);
      this.activate(ws);
      await ws.loadTables();
      return true;
    } catch (err) {
      this.notify(err);
      return false;
    } finally {
      this.connectingId = '';
    }
  }

  activate(ws: Workspace) {
    this.activeId = ws.id;
    this.showHome = false;
  }

  goHome() {
    this.showHome = true;
  }

  leaveHome() {
    if (this.workspaces.length > 0) this.showHome = false;
  }

  async trustHostKeyAndConnect() {
    const h = this.hostKeyChange;
    this.hostKeyChange = null;
    if (!h) return;
    try {
      await api.trustHostKey(h.host, h.fingerprint);
      await this.open(h.connection, h.secrets ?? {});
    } catch (err) {
      this.notify(err);
    }
  }

  requestCloseTab(ws: Workspace, tabId: string) {
    const tab = ws.tabs.find(t => t.id === tabId);
    if (tab?.kind === 'table' && tab.dirty) {
      this.pendingClose = { ws, tabId, tables: [tab.table] };
      return;
    }
    ws.closeTab(tabId);
  }

  requestClose(ws: Workspace) {
    const dirty = ws.dirtyTabs;
    if (dirty.length > 0) {
      this.pendingClose = { ws, tables: dirty.map(t => t.table) };
      return;
    }
    this.close(ws);
  }

  confirmClose() {
    const p = this.pendingClose;
    this.pendingClose = null;
    if (!p) return;
    if (p.tabId) p.ws.closeTab(p.tabId);
    else this.close(p.ws);
  }

  async close(ws: Workspace) {
    const i = this.workspaces.indexOf(ws);
    if (i < 0) return;
    this.workspaces.splice(i, 1);
    if (this.activeId === ws.id) {
      const next = this.workspaces[Math.min(i, this.workspaces.length - 1)];
      this.activeId = next?.id ?? '';
    }
    if (this.workspaces.length === 0) this.showHome = false;
    await api.disconnect(ws.session.sessionId).catch(() => {});
  }

  async openSQLiteFile() {
    try {
      const file = await api.chooseSQLiteFile();
      if (!file) return;
      const existing = this.connections.find(c => c.driver === 'sqlite' && c.file === file);
      if (existing) {
        await this.open(existing);
        return;
      }
      const saved = await this.saveConnection({ ...emptyConnection('sqlite'), name: sqliteName(file), file });
      await this.open(saved);
    } catch (err) {
      this.notify(err);
    }
  }

  notify(err: unknown, kind: Toast['kind'] = 'error') {
    const text = err instanceof Error ? err.message : String(err);
    const toast = { id: ++seq, kind, text };
    this.toasts.push(toast);
    setTimeout(() => this.dismiss(toast.id), kind === 'error' ? 7000 : 3000);
  }

  dismiss(id: number) {
    this.toasts = this.toasts.filter(t => t.id !== id);
  }
}

/** Profiles that don't save secrets ask for them on every connect. */
export function needsSecrets(c: Connection): boolean {
  return c.driver !== 'sqlite' && !c.savePassword;
}

export const app = new AppState();
