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
      const before = this.ws.activeTabId;
      let active = '';
      for (const t of tabs) {
        if (t.kind === 'table' && t.schema && t.table) {
          this.ws.openTable({ schema: t.schema, name: t.table, kind: t.tableKind ?? 'table' });
          if (t.active) active = this.ws.activeTabId;
        } else {
          const q = this.ws.addQuery({ title: t.title, sql: t.sql, saved: t.saved, script: t.script || undefined }, false);
          if (t.active) active = q.id;
        }
      }
      this.ws.activeTabId = before || active || this.ws.tabs[0]?.id || '';
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
    const tabs = this.ws.tabs.flatMap(t => {
      const active = t.id === this.ws.activeTabId;
      if (t.kind === 'query') return [toState(t, active)];
      if (t.kind === 'table') return [{ kind: 'table' as const, title: t.table, sql: '', saved: '', schema: t.schema, table: t.table, tableKind: t.tableKind, active }];
      return [];
    });
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
