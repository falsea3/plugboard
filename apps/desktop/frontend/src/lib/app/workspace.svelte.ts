import { SvelteMap } from 'svelte/reactivity';
import { api, type Column, type DBObject, type Filter, type SessionInfo, type TableInfo } from '../api/backend';

export type TableTab = {
  id: string;
  kind: 'table';
  schema: string;
  table: string;
  tableKind: TableInfo['kind'];
  dirty?: boolean;
  jump?: Filter[];
  view?: 'structure' | 'ddl';
};

export type QueryTab = {
  id: string;
  kind: 'query';
  title: string;
  sql: string;
};

export type DiagramTab = {
  id: string;
  kind: 'diagram';
  title: string;
  schema: string;
};

export type DDLTab = {
  id: string;
  kind: 'ddl';
  title: string;
  object: DBObject;
};

export type Tab = TableTab | QueryTab | DiagramTab | DDLTab;

let seq = 0;
const nextId = (prefix: string) => `${prefix}-${Date.now().toString(36)}-${(++seq).toString(36)}`;

export class Workspace {
  readonly id = nextId('ws');
  session = $state<SessionInfo>() as SessionInfo;
  switchingReadOnly = $state(false);
  tunnel = $state<'ok' | 'lost' | 'failed'>('ok');
  schema = $state('');
  tables = $state<TableInfo[]>([]);
  tablesLoading = $state(false);
  tablesError = $state('');
  tabs = $state<Tab[]>([]);
  activeTabId = $state('');
  objects = $state<DBObject[]>([]);
  objectsError = $state('');
  columns = new SvelteMap<string, Column[] | 'loading' | Error>();
  activeTab = $derived(this.tabs.find(t => t.id === this.activeTabId) ?? null);

  constructor(
    session: SessionInfo,
    private notify: (err: unknown) => void,
  ) {
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

  async setReadOnly(on: boolean) {
    if (this.switchingReadOnly || on === this.readOnly) return;
    this.switchingReadOnly = true;
    try {
      this.session = await api.setReadOnly(this.session.sessionId, on);
    } catch (err) {
      this.notify(err);
    } finally {
      this.switchingReadOnly = false;
    }
  }

  async setSchema(schema: string) {
    if (schema === this.schema) return;
    this.schema = schema;
    await this.loadTables();
  }

  loadColumns(t: TableInfo) {
    const key = `${t.schema}.${t.name}`;
    const known = this.columns.get(key);
    if (known && !(known instanceof Error)) return;
    this.columns.set(key, 'loading');
    api.describeTable(this.session.sessionId, t.schema, t.name).then(
      cols => this.columns.set(key, cols),
      err => this.columns.set(key, err instanceof Error ? err : new Error(String(err))),
    );
  }

  async loadTables() {
    this.columns.clear();
    this.tablesLoading = true;
    this.tablesError = '';
    this.objectsError = '';
    try {
      const [tables, objects] = await Promise.all([
        api.listTables(this.session.sessionId, this.schema),
        api.listObjects(this.session.sessionId, this.schema).catch(err => {
          this.objectsError = err instanceof Error ? err.message : String(err);
          return [];
        }),
      ]);
      this.tables = tables;
      this.objects = objects;
    } catch (err) {
      this.tablesError = err instanceof Error ? err.message : String(err);
      this.tables = [];
    } finally {
      this.tablesLoading = false;
    }
  }

  openTable(t: TableInfo, jump?: Filter[], view?: TableTab['view']) {
    const existing = this.tabs.find((tab): tab is TableTab => tab.kind === 'table' && tab.schema === t.schema && tab.table === t.name);
    if (existing) {
      if (jump) existing.jump = jump;
      if (view) existing.view = view;
      this.activeTabId = existing.id;
      return;
    }
    const tab: TableTab = { id: nextId('table'), kind: 'table', schema: t.schema, table: t.name, tableKind: t.kind, jump, view };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  takeView(tab: TableTab): TableTab['view'] {
    const view = tab.view;
    tab.view = undefined;
    return view;
  }

  takeJump(tab: TableTab): Filter[] | undefined {
    const jump = tab.jump;
    tab.jump = undefined;
    return jump;
  }

  openDiagram(schema = this.schema) {
    const existing = this.tabs.find(tab => tab.kind === 'diagram' && tab.schema === schema);
    if (existing) {
      this.activeTabId = existing.id;
      return;
    }
    const title = this.session.schemas.length > 1 ? `Diagram (${schema})` : 'Diagram';
    const tab: DiagramTab = { id: nextId('diagram'), kind: 'diagram', title, schema };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  openDDL(object: DBObject) {
    const same = (o: DBObject) => o.schema === object.schema && o.name === object.name && o.kind === object.kind && o.detail === object.detail;
    const existing = this.tabs.find((tab): tab is DDLTab => tab.kind === 'ddl' && same(tab.object));
    if (existing) {
      this.activeTabId = existing.id;
      return;
    }
    const tab: DDLTab = { id: nextId('ddl'), kind: 'ddl', title: object.name, object };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  newQuery(sql = '') {
    const n = this.tabs.filter(t => t.kind === 'query').length + 1;
    const tab: QueryTab = { id: nextId('query'), kind: 'query', title: `Query ${n}`, sql };
    this.tabs.push(tab);
    this.activeTabId = tab.id;
  }

  saveQuery(id: string, sql: string) {
    const tab = this.tabs.find(t => t.id === id);
    if (tab?.kind === 'query') tab.sql = sql;
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
