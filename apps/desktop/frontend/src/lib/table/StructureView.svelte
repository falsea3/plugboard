<script lang="ts">
  import { api, type CellValue, type Column, type ColumnChange, type EngineFeatures, type ResultColumn, type RowChange } from '../api/backend';
  import { app } from '../app/app.svelte';
  import { DEFAULT, TableEdits } from './edits.svelte';
  import { formatCount, isTrue } from '../ui/format';
  import DataGrid from '../grid/DataGrid.svelte';
  import type { GridEditing, MenuAt, MenuItem } from '../grid/grid';
  import PendingBar from './PendingBar.svelte';
  import SqlPreview from './SqlPreview.svelte';
  import IndexList from './IndexList.svelte';

  let {
    sessionId,
    schema,
    table,
    features,
    columns,
    prod,
    readOnlyReason,
    rowChanges,
    queryId,
    onchanged,
    ondirty,
  }: {
    sessionId: string;
    schema: string;
    table: string;
    features: EngineFeatures;
    columns: Column[];
    prod: boolean;
    readOnlyReason: string;
    rowChanges: number;
    queryId: string;
    onchanged: () => void;
    ondirty: (dirty: boolean) => void;
  } = $props();

  const COL_NAME = 0;
  const COL_NULLABLE = 2;
  const COL_DEFAULT = 3;
  const COL_KEY = 4;

  const gridColumns: ResultColumn[] = [
    { name: 'column', type: '', kind: 'text' },
    { name: 'type', type: '', kind: 'text' },
    { name: 'nullable', type: '', kind: 'bool' },
    { name: 'default', type: '', kind: 'text' },
    { name: 'primary key', type: '', kind: 'bool' },
  ];
  const rows = $derived(columns.map(c => [c.name, c.type, c.nullable, c.default, c.primaryKey] as CellValue[]));
  const edits = new TableEdits(() => rows, () => gridColumns.length);

  let grid = $state<DataGrid>();
  let indexList = $state<IndexList>();

  function changed() {
    onchanged();
    indexList?.reload();
  }
  let saving = $state(false);
  let saveError = $state('');
  let preview = $state<{ sql: string[]; confirm: boolean } | null>(null);

  $effect(() => {
    ondirty(edits.dirty);
  });

  const canAlter = $derived(features.canAlterColumns);

  const displayRows = $derived(edits.shown((v, c) => (c === COL_NULLABLE ? true : c === COL_KEY ? false : null)));

  function canEditCell(r: number, c: number): boolean {
    if (readOnlyReason || edits.rowState(r) === 'deleted' || c === COL_KEY) return false;
    return canAlter || edits.isNew(r) || c === COL_NAME;
  }

  const editing: GridEditing = {
    canEdit: canEditCell,
    cellState: (r, c) => edits.cellState(r, c),
    rowState: r => edits.rowState(r),
    isFailed: r => edits.isFailed(r),
    commit: (r, c, v) => edits.set(r, c, v),
  };

  function cellMenu(r: number, c: number): MenuItem[] {
    const items: MenuItem[] = [];
    if (c === COL_NULLABLE && canEditCell(r, c)) {
      items.push(isTrue(displayRows[r][c]) ? { id: 'not-null', label: 'Make NOT NULL', kbd: 'Space' } : { id: 'nullable', label: 'Allow NULL', kbd: 'Space' });
    }
    if (c === COL_DEFAULT && canEditCell(r, c) && displayRows[r][c] !== null) {
      items.push({ id: 'null', label: 'Remove the default', kbd: '⌥⌫' });
    }
    if (!readOnlyReason) {
      const rs = grid?.selectedRowsFor(r) ?? [r];
      const label = rs.length > 1 ? `${rs.length} columns` : 'column';
      items.push('sep', rs.every(i => edits.rowState(i) === 'deleted')
        ? { id: 'restore', label: `Restore ${label}`, kbd: '⌘⌫' }
        : { id: 'delete', label: `Drop ${label}`, kbd: '⌘⌫', danger: true });
    }
    return items;
  }

  function onmenu(id: string, at: MenuAt): boolean {
    if (readOnlyReason) {
      if (['delete', 'toggle', 'null'].includes(id)) app.notify(`Can’t change the structure: ${readOnlyReason.toLowerCase()}.`, 'info');
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
    if (at.c < 0 || !canEditCell(at.r, at.c)) return false;
    switch (id) {
      case 'toggle':
        if (at.c !== COL_NULLABLE) return false;
        edits.set(at.r, at.c, !isTrue(displayRows[at.r][at.c]));
        return true;
      case 'nullable':
      case 'not-null':
        edits.set(at.r, COL_NULLABLE, id === 'nullable');
        return true;
      case 'null':
        if (at.c === COL_DEFAULT) {
          edits.set(at.r, at.c, edits.isNew(at.r) ? DEFAULT : null);
          return true;
        }
        app.notify(`${gridColumns[at.c].name} can’t be empty.`, 'info');
        return true;
    }
    return false;
  }

  export function addColumn() {
    if (readOnlyReason) {
      app.notify(`Can’t add columns: ${readOnlyReason.toLowerCase()}.`, 'info');
      return;
    }
    const r = edits.addRow();
    edits.set(r, COL_NULLABLE, true);
    edits.set(r, COL_DEFAULT, null);
    edits.set(r, COL_KEY, false);
    grid?.showRow(r, COL_NAME, true);
  }

  function structureChange(): { changes: ColumnChange[]; rows: number[] } {
    const built = edits.changes(gridColumns, ['column']);
    return { changes: built.changes.map(toColumnChange), rows: built.rows };
  }

  function toColumnChange(ch: RowChange): ColumnChange {
    const v = ch.values;
    const out: ColumnChange = {
      kind: ch.kind,
      column: String(ch.key.column ?? ''),
      defaultSet: 'default' in v,
      default: v.default == null || String(v.default).trim() === '' ? null : String(v.default),
    };
    if ('column' in v) out.name = String(v.column ?? '');
    if ('type' in v) out.type = String(v.type ?? '');
    if ('nullable' in v) out.nullable = isTrue(v.nullable as CellValue);
    return out;
  }

  async function showPreview(confirm: boolean) {
    try {
      preview = { sql: await api.previewStructure(sessionId, { schema, table, changes: structureChange().changes }), confirm };
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    }
  }

  export function commit() {
    if (!edits.dirty || saving) return;
    if (rowChanges > 0) {
      app.notify(`Commit or discard the ${formatCount(rowChanges, 'row change')} in Data first — changing the structure reloads the rows.`, 'info');
      return;
    }
    if (prod && app.settings.confirmProdWrites) {
      showPreview(true);
      return;
    }
    apply();
  }

  async function apply() {
    preview = null;
    saving = true;
    saveError = '';
    const { changes, rows: changeRows } = structureChange();
    try {
      const res = await api.applyStructure(sessionId, queryId, { schema, table, changes });
      if (res.cancelled && !features.transactionalDDL) {
        app.notify('Stopped. What ran before stays, and the server may still finish the statement that was running — refresh in a while to see the table as it ends up.', 'info');
        edits.discard();
        changed();
        return;
      }
      if (res.cancelled) {
        saveError = 'Stopped — nothing was changed.';
        return;
      }
      if (res.error) {
        if (res.partial) {
          app.notify(`${formatCount(res.applied, 'change')} applied before one failed — the table now shows what was saved. ${res.error}`);
          edits.discard();
          changed();
          return;
        }
        edits.failedRow = res.failedIndex >= 0 ? changeRows[res.failedIndex] : null;
        saveError = `Nothing was changed: ${res.error}`;
        if (edits.failedRow !== null) grid?.showRow(edits.failedRow, -1, false);
        return;
      }
      app.notify(`Saved ${formatCount(res.applied, 'change')} to the structure of ${table}.`, 'info');
      edits.discard();
      changed();
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
</script>

<div class="structure">
  <div class="grid-wrap">
    <DataGrid bind:this={grid} columns={gridColumns} rows={displayRows} editing={readOnlyReason ? null : editing} {cellMenu} {onmenu} />
  </div>

  <IndexList bind:this={indexList} {sessionId} {schema} {table} />

  {#if edits.dirty}
    <PendingBar
      count={edits.count}
      error={saveError}
      {saving}
      onstop={() => api.cancelQuery(queryId)}
      ondiscard={discard}
      onpreview={() => showPreview(false)}
      oncommit={commit}
    />
  {:else}
    <div class="footer">
      <span class="small muted">{formatCount(columns.length, 'column')}</span>
      {#if !canAlter && !readOnlyReason}<span class="small faint">· This database can rename, add and drop columns here; anything else needs the SQL editor</span>{/if}
    </div>
  {/if}
</div>

{#if preview}
  {@const p = preview}
  <SqlPreview
    title={p.confirm ? 'Change Production?' : 'Pending structure changes'}
    sql={p.sql}
    confirm={p.confirm}
    confirmLabel="Change Production"
    onapply={apply}
    oncommit={() => ((preview = null), commit())}
    onclose={() => (preview = null)}
  >
    {#if p.confirm}<strong>{table}</strong> is on a connection tagged Production.{/if}
    {#if !features.transactionalDDL}
      This database applies each statement on its own: if one fails, the ones before it stay applied.
    {:else}
      These statements run in one transaction — all of them apply, or none do.
    {/if}
  </SqlPreview>
{/if}

<style>
  .structure { height: 100%; display: flex; flex-direction: column; }
  .grid-wrap { flex: 1; min-height: 0; position: relative; }
  .footer {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 37px;
    padding: 0 10px;
    border-top: 1px solid var(--border);
    background: var(--surface);
  }
  .small { font-size: 12px; padding: 0 4px; white-space: nowrap; }
</style>
