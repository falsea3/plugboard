<script lang="ts">
  import { onDestroy, tick, untrack } from 'svelte';
  import { api, type CellValue, type ChangeSet, type Column, type ResultColumn, type RowCount, type TablePage } from '../backend';
  import { app, type TableTab, type Workspace } from '../stores/app.svelte';
  import { DEFAULT, NOW, SET_DEFAULT, TableEdits, exprLabel, isExpr } from '../stores/edits.svelte';
  import { ADD_ROW_EVENT, COMMIT_EVENT, FILTER_ROWS_EVENT, REFRESH_EVENT } from '../commands';
  import { formatCount, formatDuration, isTrue } from '../format';
  import { rowsStatement, type RowStatement } from '../sqlText';
  import { copyToClipboard } from '../clipboard';
  import DataGrid, { type GridEditing, type MenuAt, type MenuItem } from './DataGrid.svelte';
  import ValueBar from './ValueBar.svelte';
  import StructureView from './StructureView.svelte';
  import Icon from './Icon.svelte';
  import LoadBar from './LoadBar.svelte';
  import Spinner from './Spinner.svelte';
  import Modal from './Modal.svelte';
  import FilterBar, { newFilter, noValue, type FilterRow } from './FilterBar.svelte';
  import type { Filter, FilterOp, Relation } from '../wire';

  let {
    ws,
    tab,
    active,
    ondirty,
  }: { ws: Workspace; tab: TableTab; active: boolean; ondirty: (dirty: boolean) => void } = $props();

  const sessionId = untrack(() => ws.session.sessionId);

  let mode = $state<'data' | 'structure'>('data');
  let page = $state<TablePage | null>(null);
  let columns = $state<Column[]>([]);
  let sort = $state<{ column: string; desc: boolean } | null>(null);
  let loading = $state(false);
  let columnsLoading = $state(false);
  let error = $state('');
  let selected = $state<{ value: CellValue; column: ResultColumn } | null>(null);
  let loadSeq = 0;
  let pageSize = $state(app.settings.pageSize);
  let grid = $state<DataGrid>();

  const edits = new TableEdits(
    () => page?.result.rows ?? [],
    () => page?.result.columns.length ?? 0,
  );
  let saving = $state(false);
  let saveError = $state('');
  let preview = $state<{ sql: string[]; confirm: boolean } | null>(null);

  let relations = $state<Relation[]>([]);
  let filterRows = $state<FilterRow[]>([]);
  let filters = $state<Filter[]>([]);
  let showFilters = $state(false);
  let filterBar = $state<FilterBar>();

  const shownSort = $derived(sort ?? (page?.defaultOrder?.length ? { column: page.defaultOrder[0], desc: false } : null));

  const keyColumns = $derived(columns.filter(c => c.primaryKey).map(c => c.name));
  const binary = $derived(new Set(columns.filter(c => c.kind === 'binary').map(c => c.name)));
  const canFindRows = $derived(keyColumns.length > 0 && keyColumns.every(k => !binary.has(k)));

  const readOnlyReason = $derived.by(() => {
    if (ws.readOnly) return 'Read-only session';
    if (tab.tableKind === 'view') return 'Views are read-only';
    if (columns.length > 0 && keyColumns.length === 0) return 'No primary key — rows can’t be identified safely';
    return '';
  });
  const canEdit = $derived(!readOnlyReason && columns.length > 0 && !!page);

  $effect(() => {
    ondirty(edits.dirty || structureDirty);
  });

  const features = untrack(() => ws.session.engine);

  const info = $derived(new Map(columns.map(c => [c.name, c])));
  const colAt = (c: number): Column | undefined => (page ? info.get(page.result.columns[c]?.name) : undefined);

  const displayRows = $derived.by(() => {
    const base = page?.result.rows ?? [];
    const n = edits.rowCount;
    const cols = page?.result.columns.length ?? 0;
    const out: CellValue[][] = new Array(n);
    for (let r = 0; r < n; r++) {
      if (r < base.length && !edits.updates.has(r)) {
        out[r] = base[r];
        continue;
      }
      const row: CellValue[] = new Array(cols);
      for (let c = 0; c < cols; c++) {
        const v = edits.value(r, c);
        row[c] = v === DEFAULT ? null : isExpr(v) ? exprLabel(v) : v;
      }
      out[r] = row;
    }
    return out;
  });

  const cellEditable = (r: number, c: number) =>
    canEdit && edits.rowState(r) !== 'deleted' && !!page && !binary.has(page.result.columns[c].name);

  const gridEditing: GridEditing = {
    canEdit: cellEditable,
    cellState: (r, c) => edits.cellState(r, c),
    rowState: r => edits.rowState(r),
    isFailed: r => edits.isFailed(r),
    commit: (r, c, v) => edits.set(r, c, v),
    options: (r, c) => colAt(c)?.enum ?? null,
  };

  const boolValue = (on: boolean): CellValue => (features.booleanType ? on : on ? 1 : 0);

  const links = $derived.by(() => {
    const names = page?.result.columns.map(c => c.name) ?? [];
    const out = new Set<number>();
    for (const rel of relations) for (const col of rel.columns) if (names.includes(col)) out.add(names.indexOf(col));
    return out;
  });

  function jumpFor(r: number, c: number): { rel: Relation; filters: Filter[] } | null {
    if (!page || r >= page.result.rows.length) return null;
    const names = page.result.columns.map(col => col.name);
    const rel = relations.find(x => x.columns.includes(names[c]));
    if (!rel) return null;
    const values = rel.columns.map(col => page!.result.rows[r][names.indexOf(col)]);
    if (values.some(v => v === null || v === undefined)) return null;
    return { rel, filters: rel.refColumns.map((col, i) => ({ column: col, op: '=' as FilterOp, value: String(values[i]) })) };
  }

  function follow(r: number, c: number) {
    const j = jumpFor(r, c);
    if (j) ws.openTable({ schema: j.rel.refSchema, name: j.rel.refTable, kind: 'table' }, j.filters);
  }

  function takeJump(): boolean {
    const jump = ws.takeJump(tab);
    if (!jump) return false;
    filters = jump;
    filterRows = jump.map(f => newFilter(columns, f.column, f.op, f.value));
    showFilters = true;
    return true;
  }

  function cellMenu(r: number, c: number): MenuItem[] {
    const items: MenuItem[] = [];
    const rowsLabel = (n: number) => (n > 1 ? `${n} rows` : 'row');
    if (c >= 0 && page) {
      const col = colAt(c);
      const v = displayRows[r]?.[c] ?? null;
      const name = page.result.columns[c].name;
      if (cellEditable(r, c) && col) {
        if (col.kind === 'bool') {
          if (v === null || !isTrue(v)) items.push({ id: 'set-true', label: 'Set true', kbd: 'Space' });
          if (v === null || isTrue(v)) items.push({ id: 'set-false', label: 'Set false', kbd: 'Space' });
        }
        if (col.kind === 'datetime') items.push({ id: 'set-now', label: 'Set to now()' });
        if (col.nullable && v !== null) items.push({ id: 'null', label: 'Set NULL', kbd: '⌥⌫' });
        const isNew = edits.rowState(r) === 'new';
        if (isNew ? edits.cellState(r, c) !== 'default' : col.default !== null && features.canUpdateToDefault) {
          items.push({ id: 'set-default', label: 'Set DEFAULT' });
        }
      }
      const jump = jumpFor(r, c);
      if (jump) items.push('sep', { id: 'follow', label: `Go to ${jump.rel.refTable}` });
      if (edits.rowState(r) !== 'new') {
        items.push('sep');
        items.push(
          { id: 'filter-value', label: v === null ? `Filter: ${name} is NULL` : 'Filter by this value' },
          { id: 'exclude-value', label: v === null ? `Filter: ${name} is not NULL` : 'Exclude this value' },
        );
      }
    }
    const sqlRows = (grid?.selectedRowsFor(r) ?? [r]).filter(i => edits.rowState(i) !== 'new');
    if (sqlRows.length > 0) {
      const statements = (to: string): MenuItem[] =>
        (['select', 'insert', 'update', 'delete'] as const).map(k => ({ id: `${to}:${k}`, label: k.toUpperCase(), disabled: k !== 'insert' && !canFindRows }));
      items.push('sep', { id: 'sql-copy', label: 'Copy as SQL', items: statements('sql-copy') }, { id: 'sql-open', label: 'Open SQL in new query', items: statements('sql-open') });
    }
    if (canEdit) {
      const rows = grid?.selectedRowsFor(r) ?? [r];
      const allDeleted = rows.every(i => edits.rowState(i) === 'deleted');
      items.push(
        'sep',
        allDeleted
          ? { id: 'restore', label: `Restore ${rowsLabel(rows.length)}`, kbd: '⌘⌫' }
          : { id: 'delete', label: `Delete ${rowsLabel(rows.length)}`, kbd: '⌘⌫', danger: true },
      );
    }
    return items;
  }

  function headerMenu(c: number): MenuItem[] {
    if (!page) return [];
    const name = page.result.columns[c].name;
    const col = info.get(name);
    const cur = shownSort?.column === name ? shownSort : null;
    const items: MenuItem[] = [
      { id: 'sort-asc', label: 'Sort ascending', disabled: cur?.desc === false },
      { id: 'sort-desc', label: 'Sort descending', disabled: cur?.desc === true },
    ];
    if (sort && page.defaultOrder?.length) items.push({ id: 'sort-default', label: `Default order (${page.defaultOrder.join(', ')})` });
    items.push('sep');
    if (col && col.kind === 'bool') items.push({ id: 'f-true', label: 'Is true' }, { id: 'f-false', label: 'Is false' });
    if (col && col.enum?.length) for (const e of col.enum.slice(0, 8)) items.push({ id: 'f-enum:' + e, label: `Is “${e}”` });
    if (!col || col.nullable) items.push({ id: 'f-null', label: 'Is NULL' }, { id: 'f-not_null', label: 'Is not NULL' });
    if (col && col.kind === 'text') items.push({ id: 'f-empty', label: 'Is empty' }, { id: 'f-not_empty', label: 'Is not empty' });
    items.push('sep', { id: 'fv-=', label: 'Equals…' }, { id: 'fv-!=', label: 'Not equals…' });
    if (!col || col.kind === 'text') items.push({ id: 'fv-contains', label: 'Contains…' });
    else items.push({ id: 'fv->', label: 'Greater than…' }, { id: 'fv-<', label: 'Less than…' });
    return items;
  }

  function rowsSQL(kind: RowStatement, rows: number[]): string {
    const names = page!.result.columns.map(c => c.name);
    const keep = names.flatMap((n, i) => (binary.has(n) ? [] : [i]));
    const target = { schema: tab.schema, table: tab.table, columns: keep.map(i => names[i]), key: keyColumns };
    return rowsStatement(kind, features, target, rows.map(r => keep.map(i => page!.result.rows[r][i])));
  }

  function onmenu(id: string, at: MenuAt): boolean {
    if (!page) return false;
    const name = at.c >= 0 ? page.result.columns[at.c].name : '';
    const col = at.c >= 0 ? colAt(at.c) : undefined;
    const v = at.r >= 0 && at.c >= 0 ? (displayRows[at.r]?.[at.c] ?? null) : null;

    if (id === 'sort-asc' || id === 'sort-desc' || id === 'sort-default') {
      if (!canLeavePage()) return true;
      sort = id === 'sort-default' ? null : { column: name, desc: id === 'sort-desc' };
      loadPage('start');
      return true;
    }
    if (id.startsWith('f-')) {
      const op = id.slice(2);
      if (op === 'true' || op === 'false') addFilter(name, '=', String(boolValue(op === 'true')));
      else if (op.startsWith('enum:')) addFilter(name, '=', op.slice(5));
      else addFilter(name, op as FilterOp, '');
      return true;
    }
    if (id.startsWith('fv-')) {
      addFilter(name, id.slice(3) as FilterOp, '', false);
      return true;
    }

    switch (id) {
      case 'follow':
        follow(at.r, at.c);
        return true;
      case 'filter-value':
        addFilter(name, v === null ? 'null' : '=', v === null ? '' : String(v));
        return true;
      case 'exclude-value':
        addFilter(name, v === null ? 'not_null' : '!=', v === null ? '' : String(v));
        return true;
    }
    if (id.startsWith('sql-copy:') || id.startsWith('sql-open:')) {
      const [to, kind] = id.split(':');
      const rows = (at.rows.length ? at.rows : [at.r]).filter(r => edits.rowState(r) !== 'new');
      if (rows.length === 0) return true;
      const sql = rowsSQL(kind as RowStatement, rows);
      if (to === 'sql-open') ws.newQuery(sql);
      else copyToClipboard(sql);
      return true;
    }

    if (!canEdit) {
      if (readOnlyReason && ['null', 'delete', 'restore', 'toggle'].includes(id)) app.notify(`Can’t edit: ${readOnlyReason.toLowerCase()}.`, 'info');
      return false;
    }
    switch (id) {
      case 'delete':
        edits.deleteRows(at.rows);
        return true;
      case 'restore':
        edits.restoreRows(at.rows);
        return true;
    }
    if (at.c < 0 || !cellEditable(at.r, at.c) || !col) return false;
    switch (id) {
      case 'null':
        if (!col.nullable) {
          app.notify(`${name} is NOT NULL.`, 'info');
          return true;
        }
        edits.set(at.r, at.c, null);
        return true;
      case 'set-true':
      case 'set-false':
        edits.set(at.r, at.c, boolValue(id === 'set-true'));
        return true;
      case 'toggle':
        if (col.kind !== 'bool') return false;
        edits.set(at.r, at.c, boolValue(v === null ? true : !isTrue(v)));
        return true;
      case 'set-now':
        edits.set(at.r, at.c, NOW);
        return true;
      case 'set-default':
        edits.set(at.r, at.c, edits.rowState(at.r) === 'new' ? DEFAULT : SET_DEFAULT);
        return true;
    }
    return false;
  }

  let structure = $state<StructureView>();
  let structureDirty = $state(false);
  const structureReadOnlyReason = $derived(ws.readOnly ? 'Read-only session' : tab.tableKind === 'view' ? 'Views can’t be altered here' : '');

  const MAX_ROWS = 100_000;

  let count = $state<RowCount | null>(null);
  let counting = $state(false);
  let loadingMore = $state(false);

  const loadedRows = $derived(page?.result.rows.length ?? 0);
  const newRowsPending = $derived(edits.rowCount > loadedRows);
  const atLimit = $derived(loadedRows >= MAX_ROWS);

  const keyOf = (row: CellValue[]) =>
    (page?.defaultOrder ?? []).map(k => row[page!.result.columns.findIndex(c => c.name === k)]);

  const pageQuery = () => ({
    schema: tab.schema,
    table: tab.table,
    orderBy: sort?.column ?? '',
    orderDesc: sort?.desc ?? false,
    filters,
  });

  async function loadPage(from: 'start' | 'reload' = 'reload') {
    const seq = ++loadSeq;
    const prev = page;
    const limit = from === 'reload' ? Math.min(MAX_ROWS, Math.max(pageSize, loadedRows)) : pageSize;
    loading = true;
    loadingMore = false;
    error = '';
    try {
      const p = await api.fetchTablePage(sessionId, { ...pageQuery(), offset: 0, limit });
      if (seq !== loadSeq) return;
      page = p;
      edits.discard();
      saveError = '';
      if (from === 'start') grid?.scrollToTop();
      if (!prev) loadQuickCount();
    } catch (err) {
      if (seq === loadSeq) error = err instanceof Error ? err.message : String(err);
    } finally {
      if (seq === loadSeq) loading = false;
    }
  }

  async function loadMore() {
    if (!page || loading || loadingMore || !page.hasMore || newRowsPending || atLimit) return;
    const seq = loadSeq;
    const base = page;
    const rows = base.result.rows;
    loadingMore = true;
    try {
      const p = await api.fetchTablePage(sessionId, {
        ...pageQuery(),
        offset: base.keyset ? 0 : rows.length,
        limit: Math.min(pageSize, MAX_ROWS - rows.length),
        ...(base.keyset && rows.length > 0 ? { after: keyOf(rows[rows.length - 1]) } : {}),
      });
      if (seq !== loadSeq || page !== base) return;
      page = { ...base, hasMore: p.hasMore, result: { ...base.result, durationMs: p.result.durationMs, rows: [...rows, ...p.result.rows] } };
    } catch (err) {
      if (seq === loadSeq) app.notify(err);
    } finally {
      if (seq === loadSeq) loadingMore = false;
    }
  }

  async function loadQuickCount() {
    count = null;
    try {
      const c = await api.countRows(sessionId, `${tab.id}-quick`, countQuery(), false);
      if (c.known && !(count as RowCount | null)?.exact) count = c;
    } catch {
    }
  }

  const countId = untrack(() => `${tab.id}-count`);
  onDestroy(() => {
    if (counting) api.cancelQuery(countId);
  });

  async function countExactly() {
    counting = true;
    try {
      count = await api.countRows(sessionId, countId, countQuery(), true);
    } catch (err) {
      app.notify(err);
    } finally {
      counting = false;
    }
  }

  const countQuery = () => ({ schema: tab.schema, table: tab.table, offset: 0, limit: 0, orderBy: '', orderDesc: false, filters });

  async function loadColumns() {
    columnsLoading = true;
    try {
      columns = await api.describeTable(sessionId, tab.schema, tab.table);
    } catch (err) {
      app.notify(err);
    } finally {
      columnsLoading = false;
    }
  }

  function canLeavePage(): boolean {
    if (!edits.dirty) return true;
    app.notify('Commit (⌘S) or discard your changes first.', 'info');
    return false;
  }

  function refresh() {
    if (!canLeavePage()) return;
    if (structureDirty) {
      app.notify('Commit (⌘S) or discard the structure changes first.', 'info');
      return;
    }
    loadColumns();
    loadPage();
  }

  function onsort(column: string) {
    if (!canLeavePage()) return;
    const cur = shownSort;
    if (cur?.column !== column) sort = { column, desc: false };
    else if (!cur.desc) sort = { column, desc: true };
    else sort = null;
    loadPage('start');
  }

  function applyFilters() {
    if (!canLeavePage()) return;
    filters = filterRows
      .filter(r => r.on && r.column && (noValue(r.op) || r.value.trim() !== ''))
      .map(r => ({ column: r.column, op: r.op, value: r.value.trim() }));
    loadPage('start');
    loadQuickCount();
  }

  async function openFilters() {
    mode = 'data';
    showFilters = true;
    if (filterRows.length === 0) filterRows = [newFilter(page?.result.columns ?? columns)];
    await tick();
    filterBar?.focus();
  }

  async function addFilter(column: string, op: FilterOp, value: string, apply = true) {
    const row = newFilter(columns, column, op, value);
    filterRows = [...filterRows, row];
    showFilters = true;
    if (apply) {
      applyFilters();
    } else {
      await tick();
      filterBar?.focus(row.id);
    }
  }

  function addRow() {
    if (mode === 'structure') {
      structure?.addColumn();
      return;
    }
    if (!canEdit) {
      if (readOnlyReason) app.notify(`Can’t add rows: ${readOnlyReason.toLowerCase()}.`, 'info');
      return;
    }
    mode = 'data';
    const r = edits.addRow();
    const first = page!.result.columns.findIndex(c => !binary.has(c.name) && !keyColumns.includes(c.name));
    grid?.showRow(r, Math.max(first, 0), true);
  }

  function changeSet(): { cs: ChangeSet; rows: number[] } {
    const { changes, rows } = edits.changes(page!.result.columns, keyColumns);
    return { cs: { schema: tab.schema, table: tab.table, changes }, rows };
  }

  async function showPreview(confirm: boolean) {
    try {
      preview = { sql: await api.previewChanges(sessionId, changeSet().cs), confirm };
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    }
  }

  function commit() {
    if (mode === 'structure') {
      structure?.commit();
      return;
    }
    if (!edits.dirty || saving) return;
    if (ws.connection.env === 'prod' && app.settings.confirmProdWrites) {
      showPreview(true);
      return;
    }
    apply();
  }

  async function apply() {
    preview = null;
    saving = true;
    saveError = '';
    const { cs, rows } = changeSet();
    try {
      const res = await api.applyChanges(sessionId, cs);
      if (res.error) {
        const failed = res.failedIndex >= 0 ? cs.changes[res.failedIndex] : null;
        edits.failedRow = failed ? rows[res.failedIndex] : null;
        if (failed && edits.failedRow !== null) {
          const verb = { delete: 'Deleting', update: 'Updating', insert: 'Inserting' }[failed.kind];
          const which = failed.kind === 'insert' ? 'a new row' : `row ${edits.failedRow + 1}`;
          saveError = `${verb} ${which} failed — nothing was saved: ${res.error}`;
        } else {
          saveError = res.error;
        }
        if (edits.failedRow !== null) grid?.showRow(edits.failedRow, -1, false);
        return;
      }
      app.notify(`Saved ${formatCount(res.applied, 'change')} to ${tab.table}.`, 'info');
      if (count?.exact) count = { ...count, exact: false };
      await loadPage();
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    } finally {
      saving = false;
    }
  }

  function discard() {
    edits.discard();
    saveError = '';
  }

  $effect(() => {
    const on = (name: string, fn: () => void) => {
      const h = () => active && fn();
      window.addEventListener(name, h);
      return () => window.removeEventListener(name, h);
    };
    const offs = [on(REFRESH_EVENT, refresh), on(COMMIT_EVENT, commit), on(ADD_ROW_EVENT, addRow), on(FILTER_ROWS_EVENT, openFilters)];
    return () => offs.forEach(f => f());
  });

  $effect(() => {
    if (!tab.jump) return;
    untrack(() => {
      if (!canLeavePage()) {
        ws.takeJump(tab);
        return;
      }
      if (takeJump()) {
        mode = 'data';
        loadPage('start');
        loadQuickCount();
      }
    });
  });

  untrack(() => {
    const { schema, table } = tab;
    api.relations(sessionId, schema).then(
      rs => (relations = rs.filter(r => r.table === table)),
      () => {},
    );
  });
  untrack(takeJump);
  refresh();

  const fmt = (n: number) => n.toLocaleString('en-US');
  const rangeLabel = $derived.by(() => {
    if (!page) return '';
    const n = page.result.rows.length;
    if (n === 0) return 'No rows';
    if (!page.hasMore) return formatCount(n, 'row');
    if (count?.known) return `${fmt(n)} of ${count.exact ? '' : '~'}${fmt(count.count)} rows loaded`;
    return `${fmt(n)} rows loaded`;
  });
</script>

<div class="view">
  <div class="toolbar">
    <div class="segmented" role="tablist">
      <button role="tab" aria-selected={mode === 'data'} class:on={mode === 'data'} onclick={() => (mode = 'data')}><Icon name="rows" size={13} />Data</button>
      <button role="tab" aria-selected={mode === 'structure'} class:on={mode === 'structure'} onclick={() => (mode = 'structure')}><Icon name="columns" size={13} />Structure</button>
    </div>
    <span class="title mono faint">{tab.schema}.<span class="muted">{tab.table}</span></span>
    <span style="flex:1"></span>
    <button
      class="btn sm ghost filter-btn"
      class:on={showFilters || filters.length > 0}
      onclick={() => (showFilters ? (showFilters = false) : openFilters())}
      title="Filter rows (⌘F)"
    >
      <Icon name="funnel" size={13} />Filter{#if filters.length > 0}<span class="count">{filters.length}</span>{/if}
    </button>
    {#if mode === 'structure'}
      {#if structureReadOnlyReason}
        <span class="ro-reason" title="The structure can't be changed here"><Icon name="lock" size={11} />{structureReadOnlyReason}</span>
      {:else}
        <button class="btn sm ghost" onclick={addRow} title="Add column (⌘I)"><Icon name="plus" size={13} />Column</button>
      {/if}
    {:else if readOnlyReason && columns.length > 0}
      <span class="ro-reason" title="Editing is off for this table"><Icon name="lock" size={11} />{readOnlyReason}</span>
    {:else}
      <button class="btn sm ghost" onclick={addRow} disabled={!canEdit} title="Add row (⌘I)"><Icon name="plus" size={13} />Row</button>
    {/if}
    <button class="btn icon sm ghost" title="Refresh (⌘R)" onclick={refresh}><Icon name="refresh" size={13} /></button>
  </div>

  {#if showFilters && mode === 'data' && columns.length > 0}
    <FilterBar bind:this={filterBar} columns={page?.result.columns ?? columns} bind:rows={filterRows} onapply={applyFilters} onclose={() => (showFilters = false)} />
  {/if}

  <div class="content">
    {#if loading || columnsLoading}<LoadBar label="Loading {tab.table}" />{/if}
    {#if error}
      <div class="error" role="alert"><Icon name="alert" /><span class="message">{error}</span></div>
    {:else if mode === 'data'}
      {#if !page && loading}
        <div class="waiting faint"><Spinner />Loading rows…</div>
      {:else if page}
        <DataGrid
          bind:this={grid}
          columns={page.result.columns}
          rows={displayRows}
          onend={loadMore}
          {links}
          onfollow={follow}
          sort={shownSort}
          {keyColumns}
          {onsort}
          editing={canEdit ? gridEditing : null}
          {cellMenu}
          {headerMenu}
          {onmenu}
          onselect={(value, column) => (selected = column && value !== undefined ? { value, column } : null)}
        />
      {/if}
    {:else}
      <StructureView
        bind:this={structure}
        {sessionId}
        schema={tab.schema}
        table={tab.table}
        {features}
        {columns}
        prod={ws.connection.env === 'prod'}
        readOnlyReason={structureReadOnlyReason}
        rowChanges={edits.count}
        queryId={`${tab.id}-structure`}
        onchanged={() => { loadColumns(); loadPage(); }}
        ondirty={d => (structureDirty = d)}
      />
    {/if}
  </div>

  {#if edits.dirty && mode === 'data'}
    <div class="footer pending" class:has-error={!!saveError}>
      <span class="dot"></span>
      <span class="small strong">{formatCount(edits.count, 'change')}</span>
      {#if saveError}
        <span class="save-error" title={saveError}><Icon name="alert" size={12} />{saveError}</span>
      {:else}
        <span class="small faint">not saved</span>
      {/if}
      <span style="flex:1"></span>
      <button class="btn sm ghost" onclick={discard} disabled={saving}>Discard</button>
      <button class="btn sm" onclick={() => showPreview(false)} disabled={saving}><Icon name="code" size={12} />Preview SQL</button>
      <button class="btn sm primary" onclick={commit} disabled={saving}>{#if saving}<Spinner size={11} />Saving…{:else}Commit{/if}<span class="kbd on-accent">⌘S</span></button>
    </div>
  {:else if mode === 'data'}
    <div class="footer">
      <span class="small muted">{rangeLabel}</span>
      {#if loadingMore}
        <span class="small faint loading-more"><Spinner size={10} />Loading more…</span>
      {:else if page?.hasMore && atLimit}
        <span class="small faint" title="Sort or filter to reach other rows">first {fmt(MAX_ROWS)} rows only</span>
      {/if}
      {#if page && !count?.exact && page.hasMore}
        <button class="link-btn" onclick={countExactly} disabled={counting} title="Run COUNT(*) — can take a while on big tables">{#if counting}<Spinner size={10} />Counting…{:else}Count{/if}</button>
      {/if}
      {#if filters.length > 0}<span class="filtered">filtered</span>{/if}
      {#if page}<span class="small faint" title="How long the last fetch took">· {formatDuration(page.result.durationMs)}</span>{/if}
      <span style="flex:1"></span>
      {#if selected}
        <ValueBar value={selected.value} column={selected.column} />
      {/if}
    </div>
  {/if}
</div>

{#if preview}
  {@const p = preview}
  <Modal title={p.confirm ? 'Commit to Production?' : 'Pending changes'} width={620} onclose={() => (preview = null)}>
    {#if p.confirm}
      <p class="confirm-text"><strong>{ws.connection.name}</strong> is tagged Production. These statements will run in one transaction:</p>
    {:else}
      <p class="confirm-text">Commit runs these statements in one transaction — all of them apply, or none do.</p>
    {/if}
    <div class="sql-list" class:prod={p.confirm}>
      {#each p.sql as stmt, i (i)}<pre>{stmt};</pre>{/each}
    </div>
    {#snippet footer()}
      <span class="faint" style="font-size:11.5px">{formatCount(p.sql.length, 'statement')}</span>
      <span style="flex:1"></span>
      {#if p.confirm}
        <!-- svelte-ignore a11y_autofocus -->
        <button class="btn" autofocus onclick={() => (preview = null)}>Cancel</button>
        <button class="btn primary danger-fill" onclick={apply}>Commit to Production</button>
      {:else}
        <button class="btn" onclick={() => (preview = null)}>Close</button>
        <button class="btn primary" onclick={() => { preview = null; commit(); }}>Commit</button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .toolbar, .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 36px;
    padding: 0 10px;
    background: var(--surface);
  }
  .toolbar { border-bottom: 1px solid var(--border); }
  .footer { height: 37px; border-top: 1px solid var(--border); gap: 4px; min-width: 0; }
  .content { flex: 1; min-height: 0; position: relative; }
  .loading-more { display: inline-flex; align-items: center; gap: 6px; }
  .waiting { display: flex; align-items: center; justify-content: center; gap: 8px; height: 100%; font-size: 12.5px; }
  .title { margin-left: 4px; font-size: 12px; }
  .small { font-size: 12px; padding: 0 4px; white-space: nowrap; }
  .strong { color: var(--text); font-weight: 600; }

  .link-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    border: 0;
    padding: 0 4px;
    background: transparent;
    color: var(--accent);
    font-size: 12px;
  }
  .link-btn:hover:not(:disabled) { text-decoration: underline; }
  .link-btn:disabled { color: var(--text-3); }
  .filter-btn.on { color: var(--accent); background: var(--accent-dim); }
  .count {
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--on-accent);
    font-size: 10.5px;
    line-height: 16px;
    text-align: center;
  }
  .filtered {
    padding: 0 6px;
    border-radius: 4px;
    background: var(--accent-dim);
    color: var(--accent);
    font-size: 11px;
    line-height: 17px;
  }
  .ro-reason {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 11.5px;
    color: var(--text-3);
    white-space: nowrap;
  }

  .pending {
    gap: 6px;
    background: color-mix(in srgb, var(--warn) 9%, var(--surface));
    border-top-color: color-mix(in srgb, var(--warn) 40%, var(--border));
  }
  .pending.has-error {
    background: color-mix(in srgb, var(--danger) 9%, var(--surface));
    border-top-color: color-mix(in srgb, var(--danger) 40%, var(--border));
  }
  .dot { width: 7px; height: 7px; margin-left: 4px; border-radius: 50%; background: var(--warn); }
  .has-error .dot { background: var(--danger); }
  .save-error {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    overflow: hidden;
    color: var(--danger);
    font-size: 12px;
    white-space: nowrap;
    text-overflow: ellipsis;
    user-select: text;
    -webkit-user-select: text;
  }
  .kbd.on-accent { border-color: rgba(255, 255, 255, 0.35); color: rgba(255, 255, 255, 0.85); }

  .segmented {
    display: flex;
    padding: 2px;
    border-radius: 7px;
    background: var(--elevated);
    border: 1px solid var(--border-subtle);
  }
  .segmented button {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 22px;
    padding: 0 10px;
    border: 0;
    border-radius: 5px;
    background: transparent;
    color: var(--text-2);
    font-size: 12px;
    font-weight: 500;
  }
  .segmented button.on { background: var(--bg); color: var(--text); box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2); }

  .error {
    display: flex;
    gap: 8px;
    margin: 16px;
    padding: 10px 12px;
    border-radius: var(--radius);
    color: var(--danger);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
    font-family: var(--font-mono);
    font-size: 12px;
    user-select: text;
    -webkit-user-select: text;
  }
  .error .message { white-space: pre-wrap; }

  .confirm-text { margin: 0 0 12px; color: var(--text-2); line-height: 1.5; }
  .confirm-text strong { color: var(--text); }
  .sql-list { display: flex; flex-direction: column; gap: 6px; max-height: 320px; overflow: auto; }
  .sql-list pre {
    margin: 0;
    padding: 8px 10px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border-subtle);
    border-left: 3px solid var(--accent);
    font-family: var(--font-mono);
    font-size: 12px;
    white-space: pre-wrap;
    word-break: break-word;
    user-select: text;
    -webkit-user-select: text;
  }
  .sql-list.prod pre { border-left-color: var(--env-prod); }
</style>
