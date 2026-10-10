import {
  api, onTunnelState, AppError, DEFAULT_SETTINGS,
  type Connection, type ConnectSecrets, type HostKeyChange, type Settings, type TunnelState, type UpdateInfo,
} from '../api/backend';
import { setThemeMode } from './theme';
import { emptyConnection, engine, nameFromFile, sqlite } from '../engines';
import { scheduleUpdateChecks } from './updateSchedule';
import { Workspace } from './workspace.svelte';

let seq = 0;

export type Toast = { id: number; kind: 'error' | 'info'; text: string; detail?: string; action?: { label: string; run: () => void } };

export type SettingsSection = 'general' | 'editor' | 'agents' | 'about';

export type Update =
  | { status: 'available' | 'installing' | 'installed'; info: UpdateInfo }
  | { status: 'failed'; info: UpdateInfo; error: string };


class AppState {
  connections = $state<Connection[]>([]);
  connectionsLoaded = $state(false);
  connectingId = $state('');

  workspaces = $state<Workspace[]>([]);
  activeId = $state('');
  showHome = $state(false);
  active = $derived(this.showHome ? null : (this.workspaces.find(w => w.id === this.activeId) ?? null));

  settings = $state<Settings>({ ...DEFAULT_SETTINGS });

  editing = $state<Connection | null | undefined>(undefined);
  passwordFor = $state<Connection | null>(null);
  switcherOpen = $state(false);
  settingsOpen = $state<SettingsSection | null>(null);

  hostKeyChange = $state<(HostKeyChange & { connection: Connection; secrets?: Partial<ConnectSecrets> }) | null>(null);

  pendingClose = $state<{ ws: Workspace; tabId?: string; tables: string[] } | null>(null);

  toasts = $state<Toast[]>([]);

  update = $state<Update | null>(null);
  updateDismissed = $state(false);

  async init() {
    onTunnelState(ev => this.onTunnel(ev));
    window.addEventListener('focus', () => this.loadConnections());
    await Promise.all([this.loadConnections(), this.loadSettings()]);
    scheduleUpdateChecks(() => this.checkForUpdate());
    const crashLog = await api.lastCrash().catch(() => '');
    if (crashLog) {
      this.notify('Plugboard quit unexpectedly last time. The crash log can help find out why.', 'error', {
        detail: crashLog,
        action: { label: 'Show log', run: () => api.openLogs().catch(err => this.notify(err)) },
        sticky: true,
      });
    }
  }

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
    }
    this.applySettings();
  }

  private settingsSaves = 0;

  async updateSettings(patch: Partial<Settings>) {
    const next = { ...this.settings, ...patch };
    const save = ++this.settingsSaves;
    this.settings = next;
    this.applySettings();
    try {
      const saved = await api.saveSettings(next);
      if (save === this.settingsSaves) this.settings = saved;
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
      const ws = new Workspace(session, err => this.notify(err));
      this.workspaces.push(ws);
      this.activate(ws);
      await Promise.all([ws.scripts.restore(), ws.loadTables()]);
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
    if (tab?.kind === 'query') ws.scripts.requestClose(tab);
    else ws.closeTab(tabId);
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
      const existing = this.connections.find(c => c.driver === sqlite.driver && c.file === file);
      if (existing) {
        await this.open(existing);
        return;
      }
      const saved = await this.saveConnection({ ...emptyConnection(sqlite), name: nameFromFile(sqlite, file), file });
      await this.open(saved);
    } catch (err) {
      this.notify(err);
    }
  }

  notify(err: unknown, kind: Toast['kind'] = 'error', extra: { detail?: string; action?: Toast['action']; sticky?: boolean } = {}) {
    const text = err instanceof Error ? err.message : String(err);
    const detail = extra.detail ?? (err instanceof AppError ? err.detail : '');
    const toast: Toast = { id: ++seq, kind, text, detail, action: extra.action };
    this.toasts.push(toast);
    if (!extra.sticky) setTimeout(() => this.dismiss(toast.id), kind === 'error' ? 7000 : 3000);
  }

  dismiss(id: number) {
    this.toasts = this.toasts.filter(t => t.id !== id);
  }
}

export function needsSecrets(c: Connection): boolean {
  return !engine(c.driver).file && !c.savePassword;
}

export const app = new AppState();
