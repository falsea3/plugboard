import { api } from '../api/backend';
import type { QueryTab, Workspace } from '../app/workspace.svelte';
import { isDirty, sortNames, toState } from './names';

export class Scripts {
  names = $state<string[]>([]);
  closing = $state<QueryTab | null>(null);
  naming = $state<{ tab: QueryTab; close: boolean } | null>(null);
  private ready = false;
  private saving = false;
  private again = false;
  private texts = new Map<string, () => string>();

  constructor(
    private ws: Workspace,
    private notify: (err: unknown) => void,
  ) {}

  private get conn() {
    return this.ws.connection.id;
  }

  get busy() {
    return this.closing !== null || this.naming !== null;
  }

  async restore() {
    try {
      const [tabs, names] = await Promise.all([api.queryTabs(this.conn), api.listScripts(this.conn)]);
      this.names = names;
      const restored = tabs.map(t => this.ws.addQuery({ title: t.title, sql: t.sql, saved: t.saved, script: t.script || undefined }, false));
      const active = restored[tabs.findIndex(t => t.active)];
      if (active && !this.ws.activeTabId) this.ws.activeTabId = active.id;
    } catch (err) {
      this.notify(err);
    } finally {
      this.ready = true;
    }
  }

  track(id: string, text: () => string) {
    this.texts.set(id, text);
    return () => this.texts.delete(id);
  }

  private flush(tab: QueryTab) {
    const text = this.texts.get(tab.id);
    if (text) tab.sql = text();
  }

  persist() {
    if (!this.ready) return;
    if (this.saving) {
      this.again = true;
      return;
    }
    this.saving = true;
    const tabs = this.ws.tabs.filter((t): t is QueryTab => t.kind === 'query').map(t => toState(t, t.id === this.ws.activeTabId));
    api
      .saveQueryTabs(this.conn, tabs)
      .catch(err => this.notify(err))
      .finally(() => {
        this.saving = false;
        if (this.again) {
          this.again = false;
          this.persist();
        }
      });
  }

  async open(name: string) {
    const open = this.ws.tabs.find(t => t.kind === 'query' && t.script === name);
    if (open) {
      this.ws.activeTabId = open.id;
      return;
    }
    try {
      const sql = await api.readScript(this.conn, name);
      this.ws.addQuery({ title: name, sql, saved: sql, script: name }, true);
    } catch (err) {
      this.notify(err);
      this.names = await api.listScripts(this.conn).catch(() => this.names);
    }
  }

  async save(tab: QueryTab, close = false) {
    this.flush(tab);
    if (!tab.script) {
      this.naming = { tab, close };
      return;
    }
    const sql = tab.sql;
    try {
      await api.writeScript(this.conn, tab.script, sql);
      tab.saved = sql;
      this.persist();
      if (close) this.ws.closeTab(tab.id);
    } catch (err) {
      this.notify(err);
    }
  }

  async saveAs(tab: QueryTab, name: string, close: boolean) {
    this.flush(tab);
    const sql = tab.sql;
    await api.createScript(this.conn, name, sql);
    this.names = sortNames([...this.names, name]);
    tab.script = name;
    tab.title = name;
    tab.saved = sql;
    this.persist();
    if (close) this.ws.closeTab(tab.id);
  }

  requestClose(tab: QueryTab) {
    this.flush(tab);
    if (isDirty(tab)) this.closing = tab;
    else this.ws.closeTab(tab.id);
  }

  async rename(name: string, to: string) {
    await api.renameScript(this.conn, name, to);
    this.names = sortNames(this.names.map(n => (n === name ? to : n)));
    for (const t of this.ws.tabs) {
      if (t.kind === 'query' && t.script === name) {
        t.script = to;
        t.title = to;
      }
    }
    this.persist();
  }

  async remove(name: string) {
    try {
      await api.deleteScript(this.conn, name);
      this.names = this.names.filter(n => n !== name);
      for (const t of this.ws.tabs) {
        if (t.kind === 'query' && t.script === name) {
          t.script = undefined;
          t.saved = '';
        }
      }
      this.persist();
    } catch (err) {
      this.notify(err);
    }
  }
}
