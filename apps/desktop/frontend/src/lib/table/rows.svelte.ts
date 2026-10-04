import { api, type CellValue, type Filter, type RowCount, type TablePage } from '../api/backend';

export const MAX_ROWS = 100_000;

export interface RowsQuery {
  schema: string;
  table: string;
  orderBy: string;
  orderDesc: boolean;
  filters: Filter[];
}

export interface RowsOptions {
  sessionId: string;
  id: string;
  query: () => RowsQuery;
  pageSize: () => number;
  onload: (from: 'start' | 'reload') => void;
  notify: (err: unknown) => void;
}

export class TableRows {
  page = $state<TablePage | null>(null);
  loading = $state(false);
  loadingMore = $state(false);
  error = $state('');
  count = $state<RowCount | null>(null);
  counting = $state(false);
  loaded = $derived(this.page?.result.rows.length ?? 0);
  atLimit = $derived(this.loaded >= MAX_ROWS);
  private seq = 0;

  constructor(private o: RowsOptions) {}

  async load(from: 'start' | 'reload' = 'reload') {
    const seq = ++this.seq;
    const first = !this.page;
    const size = this.o.pageSize();
    const limit = from === 'reload' ? Math.min(MAX_ROWS, Math.max(size, this.loaded)) : size;
    this.loading = true;
    this.loadingMore = false;
    this.error = '';
    try {
      const p = await api.fetchTablePage(this.o.sessionId, { ...this.o.query(), offset: 0, limit });
      if (seq !== this.seq) return;
      this.page = p;
      this.o.onload(from);
      if (first) this.quickCount();
    } catch (err) {
      if (seq === this.seq) this.error = err instanceof Error ? err.message : String(err);
    } finally {
      if (seq === this.seq) this.loading = false;
    }
  }

  async more() {
    const base = this.page;
    if (!base || this.loading || this.loadingMore || !base.hasMore || this.atLimit) return;
    const seq = this.seq;
    const rows = base.result.rows;
    this.loadingMore = true;
    try {
      const p = await api.fetchTablePage(this.o.sessionId, {
        ...this.o.query(),
        offset: base.keyset ? 0 : rows.length,
        limit: Math.min(this.o.pageSize(), MAX_ROWS - rows.length),
        ...(base.keyset && rows.length > 0 ? { after: this.keyOf(rows[rows.length - 1]) } : {}),
      });
      if (seq !== this.seq || this.page !== base) return;
      this.page = { ...base, hasMore: p.hasMore, result: { ...base.result, durationMs: p.result.durationMs, rows: [...rows, ...p.result.rows] } };
    } catch (err) {
      if (seq === this.seq) this.o.notify(err);
    } finally {
      if (seq === this.seq) this.loadingMore = false;
    }
  }

  async quickCount() {
    this.count = null;
    try {
      const c = await api.countRows(this.o.sessionId, `${this.o.id}-quick`, this.countQuery(), false);
      if (c.known && !this.exact()) this.count = c;
    } catch {
    }
  }

  async countExactly() {
    this.counting = true;
    try {
      this.count = await api.countRows(this.o.sessionId, `${this.o.id}-count`, this.countQuery(), true);
    } catch (err) {
      this.o.notify(err);
    } finally {
      this.counting = false;
    }
  }

  private exact() {
    return this.count?.exact ?? false;
  }

  stopCount() {
    if (this.counting) api.cancelQuery(`${this.o.id}-count`);
  }

  changed() {
    if (this.count?.exact) this.count = { ...this.count, exact: false };
  }

  private keyOf(row: CellValue[]) {
    const page = this.page!;
    return (page.defaultOrder ?? []).map(k => row[page.result.columns.findIndex(c => c.name === k)]);
  }

  private countQuery() {
    return { ...this.o.query(), offset: 0, limit: 0, orderBy: '', orderDesc: false };
  }
}
