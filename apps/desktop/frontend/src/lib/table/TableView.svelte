<script lang="ts">
  import { keys } from '../ui/keys';
  import { onDestroy, tick, untrack } from 'svelte';
  import { api, type Column } from '../api/backend';
  import { app } from '../app/app.svelte';
  import type { TableTab, Workspace } from '../app/workspace.svelte';
  import { TableEdits } from './edits.svelte';
  import { TableRows } from './rows.svelte';
  import { ADD_ROW_EVENT, COMMIT_EVENT, FILTER_ROWS_EVENT, REFRESH_EVENT } from '../app/commands';
  import type { FilterPick } from './menus';
  import type { Filter, FilterOp, Relation } from '../api/wire';
  import TableGrid from './TableGrid.svelte';
  import StructureView from './StructureView.svelte';
  import TableToolbar from './TableToolbar.svelte';
  import { rowsLabel, wherePlaceholder } from './labels';
  import DdlView from '../objects/DdlView.svelte';
  import Icon from '../ui/Icon.svelte';
  import LoadBar from '../ui/LoadBar.svelte';
  import Spinner from '../ui/Spinner.svelte';
  import FilterBar from './FilterBar.svelte';
  import WhereBar from './WhereBar.svelte';
  import { rowsLockReason, structureLockReason } from './lock';
  import { appliedWhere, newFilter, toFilters, type FilterRow } from './filters';

  let { ws, tab, active, ondirty }: { ws: Workspace; tab: TableTab; active: boolean; ondirty: (dirty: boolean) => void } = $props();

  const sessionId = untrack(() => ws.session.sessionId);
  const pageSize = app.settings.pageSize;

  let mode = $state<'data' | 'structure' | 'ddl'>('data');
  let columns = $state<Column[]>([]);
  let columnsLoading = $state(false);
  let sort = $state<{ column: string; desc: boolean } | null>(null);
  let relations = $state<Relation[]>([]);
  let filterRows = $state<FilterRow[]>([]);
  let filters = $state<Filter[]>([]);
  let where = $state('');
  let showFilters = $state(false);
  let filterBar = $state<FilterBar>();
  let grid = $state<TableGrid>();
  let structure = $state<StructureView>();
  let ddl = $state<DdlView>();
  let structureDirty = $state(false);

  const rows = new TableRows({
    sessionId,
    id: untrack(() => tab.id),
    query: () => ({ schema: tab.schema, table: tab.table, orderBy: sort?.column ?? '', orderDesc: sort?.desc ?? false, filters }),
    pageSize: () => pageSize,
    onload: from => {
      edits.discard();
      grid?.loaded(from);
    },
    notify: err => app.notify(err),
  });
  const edits = new TableEdits(
    () => rows.page?.result.rows ?? [],
    () => rows.page?.result.columns.length ?? 0,
  );
  onDestroy(() => rows.stopCount());

  const page = $derived(rows.page);
  const shownSort = $derived(sort ?? (page?.defaultOrder?.length ? { column: page.defaultOrder[0], desc: false } : null));
  const keyColumns = $derived(columns.filter(c => c.primaryKey).map(c => c.name));
  const lock = $derived({ readOnly: ws.readOnly, view: tab.tableKind === 'view', editable: ws.session.engine.editable, structureEditable: ws.session.engine.structureEditable });
  const readOnlyReason = $derived(rowsLockReason({ ...lock, hasColumns: columns.length > 0, hasKey: keyColumns.length > 0 }));
  const canEdit = $derived(!readOnlyReason && columns.length > 0 && !!page);
  const structureReadOnlyReason = $derived(structureLockReason(lock));

  $effect(() => {
    ondirty(edits.dirty || structureDirty);
  });

  $effect(() => {
    if (mode !== 'data') ws.selectCell(tab, null);
  });

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
    app.notify(keys('Commit (⌘S) or discard your changes first.'), 'info');
    return false;
  }

  function refresh() {
    if (!canLeavePage()) return;
    if (structureDirty) {
      app.notify(keys('Commit (⌘S) or discard the structure changes first.'), 'info');
      return;
    }
    loadColumns();
    rows.load();
  }

  function sortBy(next: { column: string; desc: boolean } | null) {
    if (next && !ws.session.engine.sortable) {
      app.notify('This database returns rows in primary key order and can’t sort a table — sort in the SQL editor within one partition.', 'info');
      return;
    }
    if (!canLeavePage()) return;
    sort = next;
    rows.load('start');
  }

  function onsort(column: string) {
    const cur = shownSort;
    sortBy(cur?.column !== column ? { column, desc: false } : !cur.desc ? { column, desc: true } : null);
  }

  function applyFilters() {
    if (!canLeavePage()) return;
    filters = toFilters(filterRows, where);
    rows.load('start');
    rows.quickCount();
  }

  async function openFilters() {
    mode = 'data';
    showFilters = true;
    if (filterRows.length === 0) filterRows = [newFilter(page?.result.columns ?? columns)];
    await tick();
    filterBar?.focus();
  }

  async function addFilter(column: string, { op, value, apply }: FilterPick) {
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

  function takeJump(): boolean {
    const jump = ws.takeJump(tab);
    if (!jump) return false;
    filters = jump;
    where = '';
    filterRows = jump.map(f => newFilter(columns, f.column, f.op as FilterOp, f.value));
    showFilters = true;
    return true;
  }

  function addRow() {
    if (mode === 'structure') structure?.addColumn();
    else grid?.addRow();
  }

  function commit() {
    if (mode === 'structure') structure?.commit();
    else grid?.commit();
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
    if (!tab.view) return;
    untrack(() => {
      const view = ws.takeView(tab);
      if (view) mode = view;
    });
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
        rows.load('start');
        rows.quickCount();
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
</script>

<div class="view">
  <TableToolbar
    bind:mode
    schema={tab.schema}
    table={tab.table}
    filterCount={filters.filter(f => f.op !== 'sql').length}
    filtersOpen={showFilters}
    lockReason={mode === 'structure' ? structureReadOnlyReason : columns.length > 0 ? readOnlyReason : ''}
    canAdd={canEdit}
    onfilter={() => (showFilters ? (showFilters = false) : openFilters())}
    onadd={addRow}
    onrefresh={() => (mode === 'ddl' ? ddl?.load() : refresh())}
    oncopy={() => ddl?.copy()}
    onquery={() => ddl?.openInQuery()}
    rowsLabel={rowsLabel(rows.page?.result.rows.length ?? null, !!rows.page?.hasMore, rows.count)}
    dirty={mode === 'structure' ? structureDirty : edits.dirty}
    onsave={canEdit || mode === 'structure' ? commit : undefined}
  />

  {#if mode === 'data'}<WhereBar bind:value={where} applied={appliedWhere(filters)} placeholder={wherePlaceholder(columns)} onapply={applyFilters} />{/if}
  {#if showFilters && mode === 'data' && columns.length > 0}
    <FilterBar bind:this={filterBar} columns={page?.result.columns ?? columns} bind:rows={filterRows} onapply={applyFilters} onclose={() => (showFilters = false)} />
  {/if}

  <div class="content" class:card={mode === 'ddl' || !!rows.error || (mode === 'data' && !page)}>
    {#if rows.loading || rows.loadingMore || columnsLoading}<LoadBar label="Loading {tab.table}" />{/if}
    {#if mode === 'ddl'}
      <DdlView bind:this={ddl} {ws} object={{ schema: tab.schema, name: tab.table, kind: tab.tableKind === 'view' ? 'view' : 'table' }} toolbar={false} />
    {:else if rows.error}
      <div class="error" role="alert"><Icon name="alert" /><span class="message">{rows.error}</span></div>
    {:else if mode === 'data'}
      {#if page}
        <TableGrid
          bind:this={grid}
          {ws}
          {tab}
          {rows}
          {edits}
          {columns}
          {keyColumns}
          {relations}
          {canEdit}
          {readOnlyReason}
          sort={shownSort}
          customSort={!!sort}
          filtered={filters.length > 0}
          {onsort}
          onsortby={sortBy}
          onfilter={addFilter}
        />
      {:else if rows.loading}
        <div class="waiting faint"><Spinner />Loading rows…</div>
      {/if}
    {:else}
      <StructureView
        bind:this={structure}
        {sessionId}
        schema={tab.schema}
        table={tab.table}
        features={ws.session.engine}
        {columns}
        prod={ws.connection.env === 'prod'}
        readOnlyReason={structureReadOnlyReason}
        rowChanges={edits.count}
        queryId={`${tab.id}-structure`}
        onchanged={() => (loadColumns(), rows.load())}
        ondirty={d => (structureDirty = d)}
      />
    {/if}
  </div>
</div>

<style>
  .view { height: 100%; display: flex; flex-direction: column; min-width: 0; }
  .content { flex: 1; min-height: 0; position: relative; display: flex; flex-direction: column; }
  .content.card { margin-top: 10px; background: var(--card); border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); overflow: hidden; }
  .waiting { display: flex; align-items: center; justify-content: center; gap: 8px; height: 100%; font-size: 12.5px; }
  .error { display: flex; gap: 8px; margin: 16px; padding: 10px 12px; border-radius: var(--radius); color: var(--danger); background: color-mix(in srgb, var(--danger) 10%, transparent); font-family: var(--font-mono); font-size: 12px; user-select: text; -webkit-user-select: text; }
  .error .message { white-space: pre-wrap; }
</style>
